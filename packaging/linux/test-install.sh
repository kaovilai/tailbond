#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Install test for the generic Linux bundle, run as root inside a throwaway
# container of the target distro (CI: .github/workflows/ci.yml, job
# vps-install). From the host:
#
#   podman run --rm --cap-add NET_ADMIN --device /dev/net/tun \
#     -v "$PWD:/src:ro,Z" almalinux:10 bash /src/packaging/linux/test-install.sh \
#     /src/out/linux/tailbond-linux-x86_64.tar.gz
#
# Checks: the static binaries run; the installer's decisions under --dry-run
# (fake listeners on udp/443 and tcp/80 move the tunnel to 8443 and the phone
# page to 8078; the subnet avoids a real route and injected ones); a real
# install (files, modes, unit limits); a re-run keeps users, subnet and port;
# tailbondd then runs from the installed config (server-up hook, pairing
# link) when the container has NET_ADMIN and /dev/net/tun; uninstall.
set -uo pipefail

BUNDLE_TGZ="${1:?usage: test-install.sh <tailbond-linux-ARCH.tar.gz>}"
PASS=0 FAIL=0
ok()  { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  FAIL %s\n' "$*"; FAIL=$((FAIL + 1)); }
check() { local what="$1"; shift; if "$@"; then ok "$what"; else bad "$what"; fi; }
say() { printf '==> %s\n' "$*"; }

. /etc/os-release
say "deps on $PRETTY_NAME"
if command -v dnf >/dev/null; then
  dnf install -y -q iproute python3 tar gzip nftables procps-ng findutils >/dev/null
elif command -v apt-get >/dev/null; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq && apt-get install -y -qq iproute2 python3 nftables procps >/dev/null
fi

W="$(mktemp -d)"
tar -C "$W" -xzf "$BUNDLE_TGZ"
B="$(echo "$W"/tailbond-linux-*)"
I="$B/install.sh"

say "binaries"
check "mqvpn --help prints usage" sh -c "'$B/bin/mqvpn' --help 2>&1 | grep -q Usage:"
check "tailbondd -version runs" "$B/bin/tailbondd" -version

# A fake tailscale CLI (status works, a 100.x address) so the dry runs take
# the Tailscale path; the real install below runs without it.
mkdir -p "$W/fakebin"
cat >"$W/fakebin/tailscale" <<'EOF'
#!/bin/sh
case "$1" in
status) [ "$2" = --json ] && printf '{\n  "Self": {\n    "DNSName": "vps.tail1234.ts.net.",\n    "TailscaleIPs": ["100.101.1.1"]\n  }\n}\n'; exit 0 ;;
ip) echo 100.101.1.1 ;;
*) exit 0 ;;
esac
EOF
chmod +x "$W/fakebin/tailscale"
dry() { PATH="$W/fakebin:$PATH" "$I" --dry-run --config "$W/none/tailbond.json" --public-addr 203.0.113.7 "$@" 2>&1; }

say "dry run, nothing listening"
out="$(dry)"
check "picks udp/443 when free" grep -q 'tunnel port: udp/443 ' <<<"$out"
check "phone page on 80" grep -q '"landing_port": 80,' <<<"$out"
check "Tailscale on, tailscale cert" grep -q '"use_tailscale_cert": true' <<<"$out"
sub="$(sed -n 's/.*"subnet": "\(10\.[0-9.]*\/24\)".*/\1/p' <<<"$out")"
o2="$(cut -d. -f2 <<<"$sub")"
check "random subnet $sub in 10.78-10.99" test -n "$sub" -a "${o2:-0}" -ge 78 -a "${o2:-0}" -le 99
check "unit: MemoryHigh=96M" grep -q '^MemoryHigh=96M$' <<<"$out"
check "unit: MemoryMax=160M" grep -q '^MemoryMax=160M$' <<<"$out"
check "unit: OOMScoreAdjust=-100" grep -q '^OOMScoreAdjust=-100$' <<<"$out"
check "unit: GOMEMLIMIT" grep -q '^Environment=GOMEMLIMIT=40MiB$' <<<"$out"
has() { grep -Eq -- "$1" <<<"$out"; }
hasnt() { ! grep -Eq -- "$1" <<<"$out"; }
check "unit: Restart=on-failure" has '^Restart=on-failure$'
check "unit: restarts on exit 75 (self-update)" has '^RestartForceExitStatus=75$'
check "unit: nothing forbids running updates from /var/lib/tailbond" hasnt '^(NoExecPaths|ExecPaths|ReadOnlyPaths|InaccessiblePaths|TemporaryFileSystem)='
check "unit: /var/lib/tailbond writable" has '^ReadWritePaths=.*/var/lib/tailbond'
check "no update block unless asked" hasnt '"update":'
out="$(dry --update-policy notify --update-source https://mirror.example/channels)"
check "--update-policy/--update-source -> update block" grep -q '"update": {"policy": "notify", "source": "https://mirror.example/channels"}' <<<"$out"
check "summary names the policy" grep -q 'updates: notify  \[--update-policy\]' <<<"$out"
out="$(dry --update-policy sometimes)"
check "bad --update-policy refused" grep -q 'auto, notify or off' <<<"$out"
check "nothing written by --dry-run" test ! -e /usr/local/bin/tailbondd -a ! -e /etc/tailbond

say "dry run, udp/443 and tcp *:80 taken (an HTTP/3 web server)"
python3 -c 'import socket,time
u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM); u.bind(("0.0.0.0",443))
t=socket.socket(); t.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1); t.bind(("0.0.0.0",80)); t.listen(1)
time.sleep(600)' &
FAKE=$!
sleep 1
out="$(dry)"
check "udp/443 busy -> udp/8443" grep -q 'tunnel port: udp/8443 ' <<<"$out"
check "config port 8443" grep -q '"port": 8443,' <<<"$out"
check "tcp *:80 busy -> landing_port 8078" grep -q '"landing_port": 8078,' <<<"$out"
check "landing_public_port 8078" grep -q '"landing_public_port": 8078,' <<<"$out"
check "Caddy hint printed" grep -q 'reverse_proxy 100.101.1.1:8078' <<<"$out"
out="$(dry --port 443)"
check "--port 443 refused while taken" grep -q 'udp/443 is in use' <<<"$out"
out="$(TB_TEST_LISTEN="udp:8443" dry)"
check "443 and 8443 busy -> 4443" grep -q 'tunnel port: udp/4443 ' <<<"$out"
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null

say "subnet picker avoids routes"
# Everything in 10.78-10.99 is taken except 10.89.0.0/24 and 10.99.255.0/24;
# 10.89.0.0/24 is then taken by a real route (a podman network on the VPS),
# so the only answer is 10.99.255.0/24.
BLOCK="10.64.0.0/12 10.80.0.0/13 10.88.0.0/16 10.89.1.0/24 10.89.2.0/23 10.89.4.0/22 10.89.8.0/21 10.89.16.0/20 10.89.32.0/19 10.89.64.0/18 10.89.128.0/17 10.90.0.0/15 10.92.0.0/14 10.96.0.0/15 10.98.0.0/16 10.99.0.0/17 10.99.128.0/18 10.99.192.0/19 10.99.224.0/20 10.99.240.0/21 10.99.248.0/22 10.99.252.0/23 10.99.254.0/24"
if ip link add tbtest0 type dummy 2>/dev/null && ip addr add 10.89.0.1/24 dev tbtest0 && ip link set tbtest0 up; then
  out="$(TB_TEST_ROUTES="$BLOCK" dry)"
  check "real route 10.89.0.0/24 avoided -> 10.99.255.0/24" grep -q '"subnet": "10.99.255.0/24"' <<<"$out"
  ip link del tbtest0
else
  out="$(TB_TEST_ROUTES="$BLOCK 10.89.0.0/24" dry)"
  check "injected route 10.89.0.0/24 avoided -> 10.99.255.0/24 (no NET_ADMIN for a real one)" grep -q '"subnet": "10.99.255.0/24"' <<<"$out"
fi
out="$(TB_TEST_ROUTES="$BLOCK 10.89.0.0/24 10.99.255.0/24" dry)"
check "no free /24 -> asks for --subnet" grep -q 'pass --subnet' <<<"$out"

say "install (no Tailscale, no systemd in this container)"
out="$("$I" --no-tailscale --public-addr 203.0.113.7 2>&1)"; rc=$?
printf '%s\n' "$out" | sed 's/^/    /' | head -30
check "install.sh exit 0" test "$rc" = 0
check "binaries in /usr/local/bin" test -x /usr/local/bin/mqvpn -a -x /usr/local/bin/tailbondd
check "hooks in /usr/local/lib/tailbond/hooks" test -x /usr/local/lib/tailbond/hooks/server-up -a -x /usr/local/lib/tailbond/hooks/server-down
check "config mode 0600" test "$(stat -c %a /etc/tailbond/tailbond.json)" = 600
check "state dir 0700" test "$(stat -c %a /var/lib/tailbond)" = 700
check "sysctl file enables forwarding" grep -q '^net.ipv4.ip_forward = 1' /etc/sysctl.d/90-tailbond.conf
check "Tailscale off in config" sh -c 'grep -q "\"disable_tailscale\": true" /etc/tailbond/tailbond.json && grep -q "\"use_tailscale_cert\": false" /etc/tailbond/tailbond.json'
check "phone page off without Tailscale" grep -q '"landing_port": -1' /etc/tailbond/tailbond.json
SUB1="$(sed -n 's/.*"subnet": "\([^"]*\)".*/\1/p' /etc/tailbond/tailbond.json)"
PORT1="$(sed -n 's/.*"port": \([0-9]*\).*/\1/p' /etc/tailbond/tailbond.json)"

say "re-run keeps users, subnet, port"
python3 - <<'EOF'
import json
p = "/etc/tailbond/tailbond.json"
c = json.load(open(p))
c["server"]["users"] = [{"name": "slate7", "key": "dGVzdC1rZXk=", "created": 1}]
c["approvers"] = ["alex@example.com"]
json.dump(c, open(p, "w"), indent=2)
EOF
# Something grabbed the port since: kept anyway (paired devices use it), with a warning.
python3 -c 'import socket,time
u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM); u.bind(("0.0.0.0",'"$PORT1"')); time.sleep(600)' &
FAKE=$!
sleep 1
out="$("$I" --memory-high 128M 2>&1)"; rc=$?
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
check "re-run exit 0" test "$rc" = 0
check "re-run warns that udp/$PORT1 is now taken" grep -q "udp/$PORT1 is now used by" <<<"$out"
check "user slate7 kept" grep -q '"slate7"' /etc/tailbond/tailbond.json
check "approvers kept" grep -q 'alex@example.com' /etc/tailbond/tailbond.json
check "subnet kept ($SUB1)" grep -q "\"subnet\": \"$SUB1\"" /etc/tailbond/tailbond.json
check "port kept ($PORT1)" grep -q "\"port\": $PORT1," /etc/tailbond/tailbond.json
check "still no Tailscale" grep -q '"disable_tailscale": true' /etc/tailbond/tailbond.json
check "backup of the previous config" test -f /etc/tailbond/tailbond.json.bak
upd() { python3 -c 'import json,sys; print((json.load(open("/etc/tailbond/tailbond.json")).get("update") or {}).get(sys.argv[1], ""))' "$1"; }
"$I" --update-policy off --update-source https://mirror.example/channels >/dev/null 2>&1
check "re-run with --update-policy off writes the update block" test "$(upd policy) $(upd source)" = "off https://mirror.example/channels"
python3 - <<'EOF2'
import json
p = "/etc/tailbond/tailbond.json"
c = json.load(open(p))
c["update"]["window"] = "02:00-04:00"
json.dump(c, open(p, "w"), indent=2)
EOF2
"$I" >/dev/null 2>&1
check "re-run without update options keeps policy, source and the dashboard's window" \
  test "$(upd policy) $(upd source) $(upd window)" = "off https://mirror.example/channels 02:00-04:00"
"$I" --update-policy auto --update-source "" >/dev/null 2>&1
check "--update-policy auto --update-source '' (back to the built-in source)" test "$(upd policy)|$(upd source)|$(upd window)" = "auto||02:00-04:00"
check "user slate7 still there" grep -q '"slate7"' /etc/tailbond/tailbond.json

say "tailbondd runs from the installed config"
if [ -c /dev/net/tun ] && ip link add tbcap0 type dummy 2>/dev/null; then
  ip link del tbcap0
  /usr/local/bin/tailbondd -config /etc/tailbond/tailbond.json >"$W/agent.log" 2>&1 &
  AG=$!
  get() { # PATH -> body (no curl in minimal images)
    exec 3<>/dev/tcp/127.0.0.1/8077 || return 1
    printf 'GET %s HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n' "$1" >&3
    timeout 3 cat <&3 | sed '1,/^\r$/d'
    exec 3<&-
  }
  post() { # PATH JSON
    exec 3<>/dev/tcp/127.0.0.1/8077 || return 1
    printf 'POST %s HTTP/1.0\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s' "$1" "${#2}" "$2" >&3
    timeout 3 cat <&3 | sed '1,/^\r$/d'
    exec 3<&-
  }
  st=""
  for _ in $(seq 1 20); do
    st="$(get /api/status 2>/dev/null)"
    grep -q '"state":"connected"' <<<"$st" && break
    sleep 1
  done
  check "server connected (mqvpn up on udp/$PORT1)" grep -q '"state":"connected"' <<<"$st"
  check "tb0 has ${SUB1%.0/24}.1" sh -c "ip -4 addr show tb0 | grep -Eq 'inet ${SUB1%.0/24}\.1[/ ]'"
  if command -v nft >/dev/null; then
    for _ in $(seq 1 5); do nft list table inet tailbond >/dev/null 2>&1 && break; sleep 1; done
    check "server-up hook loaded nft table inet tailbond (masquerade $SUB1)" sh -c "nft list table inet tailbond | grep -q 'ip saddr $SUB1'"
  fi
  pair="$(post /api/clients '{"name":"pixel"}')"
  check "POST /api/clients returns a pairing link" grep -q '"uri":"tailbond://pair?d=' <<<"$pair"
  kill "$AG"; wait "$AG" 2>/dev/null
  if command -v nft >/dev/null; then
    check "server-down hook removed the nft table" sh -c '! nft list table inet tailbond >/dev/null 2>&1'
  fi
  [ "$FAIL" = 0 ] || tail -20 "$W/agent.log"
else
  echo "  SKIP no NET_ADMIN or /dev/net/tun in this container"
fi

say "uninstall"
"$B/uninstall.sh" >/dev/null 2>&1
check "binaries removed" test ! -e /usr/local/bin/tailbondd -a ! -e /usr/local/bin/mqvpn
check "hooks removed" test ! -e /usr/local/lib/tailbond
check "config kept" test -f /etc/tailbond/tailbond.json
"$B/uninstall.sh" --purge >/dev/null 2>&1
check "--purge removes config and state" test ! -e /etc/tailbond -a ! -e /var/lib/tailbond

echo
echo "$PRETTY_NAME: $PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]
