#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# End-to-end test of the Mac as the tailbond home server (docs/macos.md
# "Mac as home server"; CI: job macos-server on GitHub's macos-latest arm64
# runner):
#
#   server   tailbondd (server role) + the patched mqvpn (patch 0006, server
#            mode on a utun) on this Mac, with the real macOS hooks
#            (packaging/macos/hooks: forwarding, pf NAT and firewall
#            anchor, application firewall, power assertion).
#   client   tailbondd (client role) + the static mqvpn in a small aarch64
#            Linux VM under QEMU (scripts/e2e-macos/build-server-vm.sh,
#            booted with tb.role=client), bonding two NICs: two QEMU user
#            networks, each reaching the Mac's own address on its own.
#   tailnet  a stand-in for Tailscale (scripts/e2e-macos/fake-tailscaled.py):
#            a utun with the "Tailscale address" 100.101.0.1, 100.64.0.0/10
#            routed to it, a tailnet machine 100.101.0.2 behind it, and a
#            fake `tailscale` CLI (status, ip, whois, set --advertise-routes).
#
# Checks: pairing over the tailnet (a pair request from a tailnet address,
# whois, approval); the VM connects and the server sees >= 2 active paths;
# the VM reaches the internet over TCP (mqvpn's TCP lane: the server's TCP
# egress connects out) and UDP (through the Mac's NAT: pf's state table shows
# the translation), the tailnet through the tunnel with its source NATed to the
# Mac's Tailscale address, and the server's DNS forwarder; a tailnet
# machine reaches the VM through the advertised route without NAT; the
# subnet is advertised; the dashboard (8077) and the phone page (80)
# answer tailnet sources and the tunnel but not other sources; the
# application firewall lets mqvpn in; a sleep assertion is held and Setup
# health says so; backup and restore of the server's settings; pf, the
# forwarding sysctls and the power assertion are back to what they were
# after a stop, after kill -9 then a restart, and after kill -9 then the
# restore that uninstall.sh runs.
#
#   MQVPN=out/macos/mqvpn TAILBONDD=out/agent/darwin-arm64/tailbondd \
#   VM=out/e2e-vm scripts/e2e-macos-server.sh
#
# Needs: macOS, root (sudo), qemu-system-aarch64, python3, curl, jq.
# KEEP=1 leaves everything running.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/out/macos/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/darwin-arm64/tailbondd}"
VM="${VM:-$ROOT/out/e2e-vm}"
API=127.0.0.1:8077    # the Mac server's dashboard
CAPI=127.0.0.1:28077  # the VM client's dashboard (QEMU forward)
PROBE=127.0.0.1:28080 # the VM's probe CGI (QEMU forward)
PORT=4443
CTL=47999             # fake-tailscaled's control socket
ANCHOR=com.apple/251.tailbond.server
TSIP=100.101.0.1 PEER=100.101.0.2 ASKER=100.101.9.9
STRANGER=198.51.100.77 # a source that is neither tailnet nor tunnel (a lo0 alias)
OTHER=com.apple/200.e2e-otherfw OTHER_TOK="" # another product's pf anchor

if [ "$(id -u)" -ne 0 ]; then
  exec sudo --preserve-env=KEEP,VERBOSE,PATH MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" VM="$VM" "$0" "$@"
fi
[ "$(uname -s)" = Darwin ] || { echo "run this on macOS"; exit 2; }
for f in "$MQVPN" "$TAILBONDD"; do [ -x "$f" ] || { echo "missing $f"; exit 2; }; done
for f in "$VM/vmlinuz" "$VM/initramfs.gz"; do [ -f "$f" ] || { echo "missing $f (scripts/e2e-macos/build-server-vm.sh)"; exit 2; }; done
for c in qemu-system-aarch64 curl jq python3 pfctl; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done

WORK="$(mktemp -d /tmp/tailbond-e2e-macsrv.XXXXXX)"
chmod 755 "$WORK"
SRV="$WORK/srv" CONF="$WORK/srv/tailbond.json"
mkdir -p "$SRV" "$WORK/bin"
PASS=0 FAIL=0
QEMU_PID="" SRV_PID="" FTS_PID="" ALF_WAS=""
ALF=/usr/libexec/ApplicationFirewall/socketfilterfw
say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
info() { printf '  ---- %s\n' "$*"; }
sapi() { curl -fsS -m 10 -H 'Content-Type: application/json' "$@"; }
probe() { curl -sS -m 40 "http://$PROBE/cgi-bin/probe?$1" 2>&1 || true; }
fts() { # a command to fake-tailscaled, its one-line answer
  python3 -c 'import socket, sys
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.settimeout(10)
s.sendto(sys.argv[1].encode(), ("127.0.0.1", int(sys.argv[2]))); print(s.recv(512).decode())' "$1" "$CTL" 2>/dev/null || echo "no answer"
}
wait_for() { # wait_for <seconds> <description> <command...>
  local t=$1 what=$2; shift 2
  for _ in $(seq 1 "$t"); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "  timed out after ${t}s waiting for: $what"
  return 1
}
fwd() { echo "$(sysctl -n net.inet.ip.forwarding)/$(sysctl -n net.inet6.ip6.forwarding)"; }
pfstatus() { pfctl -s info 2>/dev/null | awk '/^Status:/ {print $2; exit}'; }
anchor_rules() { { pfctl -a "$ANCHOR" -s nat 2>/dev/null; pfctl -a "$ANCHOR" -s rules 2>/dev/null; } | grep -v '^$' || true; }
caff() { pgrep -f "caffeinate -i -s -w" || true; }
# the main ruleset (nat and filter; anchors show as references only)
root_rules() { { pfctl -s nat; pfctl -s rules; } 2>/dev/null; }

start_server() {
  PATH="$WORK/bin:$PATH" "$TAILBONDD" -config "$CONF" >>"$WORK/server.log" 2>&1 &
  SRV_PID=$!
}
server_up() { curl -fsS -m 3 "http://$API/api/status" | jq -e '.state == "connected"'; }
client_paths() { # active paths of the VM, as the server sees them
  sapi "http://$API/api/status" | jq '[.clients[]? | select(.name == "linux-vm" and .connected) | .paths[] | select(.state == "active")] | length'
}
# clean: no anchor rules, no restore record, no assertion, sysctls and pf as at the start
check_clean() { # check_clean <when>
  local r
  r="$(anchor_rules)"
  [ -z "$r" ] && ok "$1: pf anchor $ANCHOR is empty" || bad "$1: anchor still has rules: $r"
  [ "$(fwd)" = "$FWD0" ] && ok "$1: forwarding back to $FWD0" || bad "$1: forwarding $(fwd), was $FWD0"
  [ "$(pfstatus)" = "$PF0" ] && ok "$1: pf $PF0 as before" || bad "$1: pf $(pfstatus), was $PF0"
  [ ! -f "$SRV/server-net.state" ] && ok "$1: no restore record left" || bad "$1: restore record left: $(cat "$SRV/server-net.state")"
  [ -z "$(caff)" ] && ok "$1: no power assertion left" || bad "$1: caffeinate still running: $(caff)"
  [ "$(root_rules)" = "$ROOT_RULES0" ] && ok "$1: main pf ruleset as before" || bad "$1: main ruleset changed: $(root_rules | tr '\n' ';')"
  if [ -n "$OTHER_TOK" ]; then
    pfctl -a "$OTHER" -s rules 2>/dev/null | grep -q 'port = 9' && ok "$1: the other firewall's anchor $OTHER is intact" || bad "$1: $OTHER lost its rule"
  fi
}

cleanup() {
  local rc=$?
  if [ -n "${KEEP:-}" ]; then echo "KEEP set: VM, server and fake tailnet left running; work dir $WORK"; return; fi
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null || true
  sleep 3
  [ -f "$SRV/mqvpn.pid" ] && kill "$(cat "$SRV/mqvpn.pid")" 2>/dev/null || true
  [ -f "$SRV/server-net.state" ] && TB_STATE_DIR="$SRV" "$ROOT/packaging/macos/hooks/server-down" >/dev/null 2>&1 || true
  [ -n "$QEMU_PID" ] && kill "$QEMU_PID" 2>/dev/null || true
  [ -n "$FTS_PID" ] && kill "$FTS_PID" 2>/dev/null || true
  ifconfig lo0 -alias "$ASKER" 2>/dev/null || true
  ifconfig lo0 -alias "$STRANGER" 2>/dev/null || true
  if [ -n "$OTHER_TOK" ]; then pfctl -q -a "$OTHER" -F all 2>/dev/null; pfctl -q -X "$OTHER_TOK" 2>/dev/null; fi
  route -q -n delete -host "$TSIP" 2>/dev/null || true
  if [ -n "$ALF_WAS" ]; then
    "$ALF" --remove "$MQVPN" >/dev/null 2>&1 || true
    "$ALF" --remove "$TAILBONDD" >/dev/null 2>&1 || true
    "$ALF" --setglobalstate "$ALF_WAS" >/dev/null 2>&1 || true
  fi
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- server log (tail) ---"; tail -n 120 "$WORK/server.log" 2>/dev/null || true
    echo "--- fake tailnet log (tail) ---"; tail -n 40 "$WORK/fts.log" 2>/dev/null || true
    echo "--- client VM console (tail) ---"; grep -av '^\[ *[0-9.]*\] ' "$WORK/vm.console" 2>/dev/null | tail -n 80 || true
  fi
  mkdir -p "$ROOT/out/e2e-macos-server" && cp "$WORK"/*.log "$WORK/vm.console" "$ROOT/out/e2e-macos-server/" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

say "before: forwarding, pf"
FWD0="$(fwd)" PF0="$(pfstatus)"
info "forwarding (inet/inet6) $FWD0, pf $PF0; anchor $ANCHOR: $(anchor_rules | wc -l | tr -d ' ') rules"
WAN="$(route -n get default | awk '/interface:/ {print $2; exit}')"
PUB="$(ifconfig "$WAN" | awk '$1 == "inet" {print $2; exit}')"
# the installer's own pick (a real CIDR overlap check with the routes here)
SUBNET="$(TB_INSTALL_SOURCE_ONLY=1 bash -c '. "$1/packaging/macos/install.sh"; pick_subnet' _ "$ROOT" || true)"
SUBPFX="${SUBNET%.0/24}"
[ -n "$PUB" ] && [ -n "$SUBNET" ] && ok "this Mac: $PUB on $WAN (the address devices dial), tunnel subnet $SUBNET" || { bad "no address ($PUB) or subnet ($SUBNET)"; exit 1; }

# The stand-in Tailscale is installed now but starts only after the server
# (the Tailscale app starts at login, tailbondd at boot): until $WORK/ts-up
# exists its CLI says Stopped.
TSIF=""
cat >"$WORK/ts-status.json" <<J
{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com","MagicDNSSuffix":"tail1234.ts.net"},
 "Self":{"HostName":"macserver","DNSName":"macserver.tail1234.ts.net.","OS":"macOS","TailscaleIPs":["$TSIP"],"Online":true,"UserID":1},
 "User":{"1":{"LoginName":"alex@example.com"}},
 "Peer":{"n1":{"HostName":"pixel","DNSName":"pixel.tail1234.ts.net.","OS":"android","TailscaleIPs":["$ASKER"],"Online":true},
         "n2":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["$PEER"],"Online":true}}}
J
cat >"$WORK/bin/tailscale" <<EOF
#!/bin/sh
# fake tailscale CLI (scripts/e2e-macos-server.sh)
up() { [ -f "$WORK/ts-up" ]; }
case "\$1" in
status) up || { echo '{"BackendState":"Stopped"}'; exit 1; }; cat "$WORK/ts-status.json" ;;
ip) up || { echo "Tailscale is stopped." >&2; exit 1; }; [ "\$2" = -6 ] || echo "$TSIP" ;;
whois)
  [ "\$3" = "$ASKER" ] || { echo "no such peer" >&2; exit 1; }
  echo '{"Node":{"ID":9,"StableID":"nE2EPIXEL","Name":"pixel.tail1234.ts.net.","ComputedName":"pixel","Hostinfo":{"Hostname":"Pixel 9","OS":"android"}},"UserProfile":{"LoginName":"alex@example.com","DisplayName":"Alex"}}' ;;
debug) printf '{"AdvertiseRoutes":[%s]}\n' "\$(cat "$WORK/ts-routes" 2>/dev/null)" ;;
set)
  for a in "\$@"; do
    case "\$a" in --advertise-routes=*) r="\${a#--advertise-routes=}"; echo "\$r" >>"$WORK/ts-advertised"; printf '"%s"' "\$r" | sed 's/,/","/g' >"$WORK/ts-routes" ;; esac
  done ;;
cert|netcheck) echo "not in this test" >&2; exit 1 ;;
*) exit 0 ;;
esac
EOF
chmod 0755 "$WORK/bin/tailscale"

if [ -x "$ALF" ]; then
  say "application firewall: on for this test"
  if "$ALF" --getglobalstate | grep -qi enabled; then ALF_WAS=on; else ALF_WAS=off; fi
  "$ALF" --setglobalstate on >/dev/null 2>&1 && "$ALF" --getglobalstate | grep -qi enabled \
    && ok "application firewall on (was $ALF_WAS)" || { info "can't turn the application firewall on here"; ALF_WAS=""; }
fi

ROOT_RULES0="$(root_rules)"
info "main pf ruleset: $(tr '\n' ';' <<<"$ROOT_RULES0")"

say "server: tailbondd + mqvpn (server mode, utun) on this Mac, the macOS hooks (Tailscale not up yet)"
cat >"$CONF" <<EOF
{"role": "server", "mqvpn_bin": "$MQVPN", "state_dir": "$SRV", "hooks_dir": "$ROOT/packaging/macos/hooks",
 "listen": ":8077", "mode": "speed",
 "server": {"port": $PORT, "public_addr": "$PUB", "subnet": "$SUBNET", "use_tailscale_cert": false}}
EOF
start_server
T0=$(date +%s)
if wait_for 60 "server running" server_up; then
  ok "server up in $(( $(date +%s) - T0 )) s"
else
  bad "server not up: $(curl -sS -m 3 "http://$API/api/status" | jq -c '{state, error}' 2>&1)"; exit 1
fi
STUN="$(grep -o 'TUN utun[0-9]* configured: [0-9.]*' "$WORK/server.log" | tail -n1)"
[ -n "$STUN" ] && ok "mqvpn server on macOS: $STUN" || bad "no utun line in the log"
TUN="$(echo "$STUN" | awk '{print $2}')"
netstat -rn -f inet | grep -q "^$SUBPFX.*$TUN" && ok "subnet $SUBNET routed to $TUN" || bad "no route for $SUBNET: $(netstat -rn -f inet | grep "$SUBPFX" || true)"
wait_for 10 "server-up hook" grep -q 'server-up: pf: anchor' "$WORK/server.log" && ok "server-up: $(grep -o 'server-up: pf: anchor.*' "$WORK/server.log" | tail -n1)" || bad "server-up hook didn't load pf: $(grep 'server-up' "$WORK/server.log" | tail -n 10)"
[ "$(fwd)" = 1/0 ] || [ "$(fwd)" = 1/1 ] && ok "net.inet.ip.forwarding on ($(fwd))" || bad "forwarding $(fwd)"
[ "$(pfstatus)" = Enabled ] && ok "pf enabled" || bad "pf $(pfstatus)"
NAT="$(pfctl -a "$ANCHOR" -s nat 2>/dev/null)"
grep -q "nat on $WAN inet from $SUBNET to any -> ($WAN)" <<<"$NAT" && ok "NAT $SUBNET out $WAN" || bad "nat rules: $NAT"
grep -q "100.64.0.0/10" <<<"$NAT" && bad "tailnet NAT before Tailscale is up: $NAT" || ok "no tailnet NAT yet (Tailscale not up)"
[ "$(root_rules)" = "$ROOT_RULES0" ] && ok "the main pf ruleset is untouched (only the anchor changed)" || bad "main ruleset changed: $(root_rules | tr '\n' ';')"
[ -s "$SRV/server-net.state" ] && ok "restore record: $(tr '\n' ' ' <"$SRV/server-net.state")" || bad "no restore record"
[ -n "$(caff)" ] && pmset -g assertions | grep -q 'PreventUserIdleSystemSleep *1' && ok "sleep assertion held (caffeinate $(caff))" || bad "no sleep assertion: $(pmset -g assertions | grep -i prevent)"
HL="$(sapi "http://$API/api/health" | jq -c '.checks[] | select(.id == "sleep")')"
jq -e '.status == "ok"' <<<"$HL" >/dev/null && ok "Setup health: $(jq -r .title <<<"$HL")" || bad "health sleep check: $HL"
if [ -n "$ALF_WAS" ]; then
  "$ALF" --listapps 2>/dev/null | grep -A1 -F "$MQVPN" | grep -qi allow && ok "application firewall allows $MQVPN in" || bad "firewall apps: $("$ALF" --listapps 2>&1 | head -n 20)"
fi

say "Tailscale comes up after the server: the rules and the route follow"
python3 "$ROOT/scripts/e2e-macos/fake-tailscaled.py" "$WORK/fts.log" "$CTL" "$WORK/fts.if" >"$WORK/fts.out.log" 2>&1 &
FTS_PID=$!
wait_for 10 "fake tailnet utun" test -s "$WORK/fts.if" && ok "stand-in Tailscale interface $(cat "$WORK/fts.if") ($TSIP)" \
  || { bad "fake tailnet didn't start: $(cat "$WORK/fts.out.log")"; exit 1; }
TSIF="$(cat "$WORK/fts.if")"
touch "$WORK/ts-up"
T0=$(date +%s)
if wait_for 60 "tailnet NAT after Tailscale came up" sh -c "pfctl -a $ANCHOR -s nat 2>/dev/null | grep -q 'nat on $TSIF inet from $SUBNET to 100.64.0.0/10 -> ($TSIF)'"; then
  ok "NAT $SUBNET to the tailnet out $TSIF, $(( $(date +%s) - T0 )) s after Tailscale came up: $(grep -o 'tailscale changed.*' "$WORK/server.log" | tail -n1)"
else
  bad "no tailnet NAT after Tailscale came up: $(pfctl -a "$ANCHOR" -s nat 2>&1 | tr '\n' ';')"
fi
wait_for 20 "advertised" grep -qx "$SUBNET" "$WORK/ts-advertised" && ok "tailscale set --advertise-routes=$SUBNET (after it came up)" || bad "subnet not advertised: $(cat "$WORK/ts-advertised" 2>/dev/null)"
[ "$(tr '\n' ' ' <"$SRV/server-net.state" | grep -c 'FWD4=0')" = 1 ] && [ -n "$(caff)" ] \
  && ok "the refresh kept the restore record and the assertion: $(tr '\n' ' ' <"$SRV/server-net.state")" || bad "record after refresh: $(cat "$SRV/server-net.state")"

say "pairing over the tailnet: a pair request from $ASKER, whois, approval"
ifconfig lo0 alias "$ASKER" netmask 255.255.255.255
wait_for 20 "dashboard on $TSIP" curl -fsS -m 2 "http://$TSIP:8077/api/hello" && ok "dashboard listens on the Tailscale address: $(curl -fsS -m 2 "http://$TSIP:8077/api/hello")" || bad "nothing on $TSIP:8077"
REQ="$(curl -fsS -m 10 --interface "$ASKER" -H 'Content-Type: application/json' -d '{"name":"linux-vm","kind":"computer"}' "http://$TSIP:8077/api/pair-requests" || true)"
RID="$(jq -r .id <<<"$REQ" 2>/dev/null)" RTOK="$(jq -r .token <<<"$REQ" 2>/dev/null)"
jq -e '.status == "pending" and .device == "pixel" and .user == "alex@example.com"' <<<"$REQ" >/dev/null \
  && ok "pair request accepted from the tailnet (whois: pixel, alex@example.com)" || bad "pair request: $REQ"
sapi -X POST -d '{}' "http://$API/api/pair-requests/$RID/approve" | jq -e '.client == "linux-vm"' >/dev/null && ok "approved on the server" || bad "approve failed"
URI="$(curl -fsS -m 10 --interface "$ASKER" -H "Authorization: Bearer $RTOK" "http://$TSIP:8077/api/pair-requests/$RID" | jq -r '.uri // empty')"
[ -n "$URI" ] && ok "the requester got its pairing link (${#URI} chars)" || { bad "no pairing link"; exit 1; }
wait_for 20 "phone page on port 80" sh -c "curl -fsS -m 3 --interface $ASKER http://$TSIP/ | grep -qi tailbond" && ok "phone page on http://$TSIP/ (port 80) for the tailnet" || bad "no phone page on port 80"

say "firewall: dashboard and phone page only from the tailnet, loopback and the tunnel"
ifconfig lo0 alias "$STRANGER" netmask 255.255.255.255
web_from() { curl -s -m 5 -o /dev/null -w '%{http_code}' --interface "$1" "$2" || true; }
for u in "http://$TSIP:8077/api/status" "http://$TSIP/"; do
  c="$(web_from "$STRANGER" "$u")"; [ "$c" = 403 ] && ok "$u from $STRANGER (not tailnet): 403 (tailbondd itself refuses it)" || bad "$u from $STRANGER: HTTP $c"
done
for p in 8077 80; do
  r="$(fts "syn $PEER $TSIP $p")"; [ "$r" = synack ] && ok "tcp/$p from tailnet $PEER: answered" || bad "tcp/$p from tailnet $PEER: $r"
  r="$(fts "syn 192.0.2.9 $TSIP $p")"; [ "$r" = none ] && ok "tcp/$p from 192.0.2.9 (not tailnet): dropped" || bad "tcp/$p from 192.0.2.9: $r"
done

say "client: tailbondd in a Linux VM (QEMU), bonding eth0 + eth1 to $PUB:$PORT"
accel=tcg,thread=multi
qemu-system-aarch64 -accel help 2>/dev/null | grep -qx hvf && sysctl -n kern.hv_support 2>/dev/null | grep -qx 1 && accel=hvf
cpu=cortex-a72; [ "$accel" = hvf ] && cpu=host
info "accelerator $accel ($(qemu-system-aarch64 --version | head -n1))"
qemu-system-aarch64 -M virt -cpu "$cpu" -accel "$accel" -smp 2 -m 512 \
  -kernel "$VM/vmlinuz" -initrd "$VM/initramfs.gz" \
  -append "console=ttyAMA0 panic=-1 tb.time=$(date +%s) tb.role=client tb.pair=$URI" \
  -netdev "user,id=n0,net=192.168.71.0/24,hostfwd=tcp:127.0.0.1:28077-:8077,hostfwd=tcp:127.0.0.1:28080-:8080" \
  -device virtio-net-pci,netdev=n0,romfile= \
  -netdev "user,id=n1,net=192.168.72.0/24" -device virtio-net-pci,netdev=n1,romfile= \
  -device virtio-rng-pci -display none -monitor none -serial "file:$WORK/vm.console" -no-reboot &
QEMU_PID=$!
T0=$(date +%s)
if wait_for 300 "VM client connected" sh -c "curl -fsS -m 3 http://$CAPI/api/status | jq -e '.state == \"connected\"'"; then
  ok "client connected in $(( $(date +%s) - T0 )) s: tunnel $(curl -fsS -m 3 "http://$CAPI/api/status" | jq -r .tunnel_ip)"
else
  bad "client not connected: $(curl -sS -m 3 "http://$CAPI/api/status" | jq -c '{state, error}' 2>&1)"; probe what=log | tail -n 30; exit 1
fi
CIP="$(curl -fsS -m 3 "http://$CAPI/api/status" | jq -r .tunnel_ip)"
if wait_for 60 "server sees 2 active paths" sh -c "[ \"\$(curl -fsS -m 3 http://$API/api/status | jq '[.clients[]? | select(.name == \"linux-vm\" and .connected) | .paths[] | select(.state == \"active\")] | length')\" -ge 2 ]"; then
  ok "server: linux-vm bonds $(client_paths) active paths: $(sapi "http://$API/api/status" | jq -c '[.clients[] | select(.name == "linux-vm") | .paths[] | {state, srtt_ms}]')"
else
  bad "server paths: $(sapi "http://$API/api/status" | jq -c '[.clients[]? | {name, connected, paths}]')"
fi

say "traffic: the internet through the Mac's NAT, the tailnet, DNS, the dashboard via the tunnel"
info "VM routes: $(probe what=routes | grep -E '^(default|0\.0\.0\.0|128\.0|[0-9.]+/[0-9]+ dev tb)' | tr '\n' ';')"
tcpdump -n -l -i "$WAN" -c 40 host 1.1.1.1 >"$WORK/tcpdump-wan.log" 2>&1 &
TD1=$!
tcpdump -n -l -i "$TUN" -c 40 host 1.1.1.1 >"$WORK/tcpdump-tun.log" 2>&1 &
TD2=$!
sleep 1
# TCP rides mqvpn's hybrid TCP lane: the server connects out itself (its
# TCP egress, patch 0006); datagrams (UDP, ICMP) are forwarded and NATed
OUT="$(probe 'what=http&url=http://1.1.1.1/cdn-cgi/trace')"
DOUT="$(probe 'what=dns&name=github.com&server=1.1.1.1')"
STATES="$(pfctl -s state 2>/dev/null | grep '1\.1\.1\.1:53' || true)"
sleep 1; kill "$TD1" "$TD2" 2>/dev/null || true
grep -q "TCP egress\|tcp_egress\|connect-tcp" "$WORK/server.log" && info "server log: $(grep -m1 -i 'egress' "$WORK/server.log")"
grep -q 'Name:' <<<"$DOUT" && ! grep -q PROBE-FAILED <<<"$DOUT" && ok "VM asked 1.1.1.1 (UDP) for github.com through the tunnel" || bad "UDP to the internet: $(head -c 300 <<<"$DOUT")"
if grep -q 'HTTP/1\.[01] [0-9]' <<<"$OUT"; then
  ok "VM fetched http://1.1.1.1/cdn-cgi/trace through the tunnel (TCP, the server's TCP egress)"
else
  bad "internet from the VM: $(head -c 300 <<<"$OUT")"
  info "the Mac itself: $(curl -sS -m 10 -o /dev/null -w '%{http_code}' http://1.1.1.1/cdn-cgi/trace 2>&1)"
  info "on $TUN: $(head -n 8 "$WORK/tcpdump-tun.log" | tr '\n' ';')"
  info "on $WAN: $(head -n 8 "$WORK/tcpdump-wan.log" | tr '\n' ';')"
  info "pf: $(pfctl -s info 2>/dev/null | grep -E 'state-|match|Status' | tr -s ' ' | tr '\n' ';')"
fi
grep -q "$CIP" <<<"$STATES" && ok "pf NATed the UDP: $(grep "$CIP" <<<"$STATES" | head -n1 | tr -s ' ')" || bad "no NAT state for $CIP: $STATES"
OUT="$(probe "what=ping&host=$PEER")"
grep -q ' 0% packet loss\|bytes from' <<<"$OUT" && ok "VM pings tailnet machine $PEER through the tunnel" || bad "tailnet ping: $(tail -n 3 <<<"$OUT")"
grep -q "icmp echo from $TSIP to $PEER" "$WORK/fts.log" && ok "the tailnet saw the Mac's Tailscale address $TSIP (NAT)" || bad "tailnet saw: $(grep 'icmp echo' "$WORK/fts.log" | tail -n 3)"
grep -q "icmp echo from $SUBPFX" "$WORK/fts.log" && bad "a tunnel address leaked into the tailnet: $(grep "icmp echo from $SUBPFX" "$WORK/fts.log" | head -n1)" || ok "no tunnel address leaked into the tailnet"
r="$(fts "ping $PEER $CIP")"
[ "$r" = "reply $CIP" ] && ok "tailnet machine $PEER reaches the VM ($CIP) via the advertised route, reply not NATed" || bad "tailnet -> VM: $r"
OUT="$(probe "what=dns&name=github.com&server=$SUBPFX.1")"
grep -q 'Name:' <<<"$OUT" && ! grep -q PROBE-FAILED <<<"$OUT" && ok "the server's DNS forwarder ($SUBPFX.1) answers the VM" || bad "DNS: $(head -c 300 <<<"$OUT")"
OUT="$(probe "what=http&url=http://$SUBPFX.1:8077/api/hello")"
grep -q '"role":"server"' <<<"$OUT" && ok "dashboard API reachable through the tunnel ($SUBPFX.1:8077)" || bad "dashboard via tunnel: $(head -c 200 <<<"$OUT")"
BYTES="$(sapi "http://$API/api/status" | jq -c '[.clients[] | select(.name == "linux-vm") | .paths[] | {tx_bytes, rx_bytes}]')"
info "per-path bytes on the server: $BYTES"

say "backup and restore of the server's settings"
export TB_BACKUP_PASSPHRASE=e2e-correct-horse
"$TAILBONDD" backup -config "$CONF" --plain | jq -e '.role == "server"' >/dev/null && ok "backup --plain: server role" || bad "backup summary failed"
"$TAILBONDD" backup -config "$CONF" --out "$WORK/srv.tbbackup" >/dev/null && ok "encrypted backup written" || bad "backup failed"
mkdir -p "$WORK/srv2"
jq --arg d "$WORK/srv2" '.state_dir = $d | .server.users = []' "$CONF" >"$WORK/srv2/tailbond.json"
if "$TAILBONDD" restore -config "$WORK/srv2/tailbond.json" --yes "$WORK/srv.tbbackup" >"$WORK/restore.log" 2>&1 \
  && jq -e --arg d "$WORK/srv2" '.state_dir == $d and ([.server.users[] | select(.name == "linux-vm")] | length == 1)' "$WORK/srv2/tailbond.json" >/dev/null; then
  ok "restore into a second config brings linux-vm back"
else
  bad "restore: $(cat "$WORK/restore.log")"
fi
unset TB_BACKUP_PASSPHRASE

say "stop: everything put back"
sapi -X POST "http://$API/api/disconnect" >/dev/null
wait_for 20 "server-down" sh -c "! test -f '$SRV/server-net.state'" || true
check_clean "after a stop"
r="$(fts "syn 192.0.2.9 $TSIP 8077")"; [ "$r" = synack ] && ok "with the server off, the pf rule is gone (192.0.2.9's SYN is answered again)" || info "192.0.2.9 after stop: $r"
for u in "http://$TSIP:8077/api/status" "http://$TSIP/"; do
  c="$(web_from "$STRANGER" "$u")"; [ "$c" = 403 ] && ok "server off, listeners on: $u from $STRANGER still 403" || bad "server off: $u from $STRANGER: HTTP $c"
done
c="$(web_from "$ASKER" "http://$TSIP:8077/api/hello")"; [ "$c" = 200 ] && ok "server off: the tailnet still reaches the dashboard" || bad "server off, tailnet: HTTP $c"

say "another firewall: its own pf anchor and pf reference survive tailbond's start and stop"
echo "block drop in quick proto tcp from any to any port 9" | pfctl -q -a "$OTHER" -f - 2>/dev/null
OTHER_TOK="$(pfctl -E 2>&1 | sed -n 's/.*Token : *\([0-9][0-9]*\).*/\1/p' | head -n1)"
PF0="$(pfstatus)"
[ -n "$OTHER_TOK" ] && [ "$PF0" = Enabled ] && ok "another product: anchor $OTHER, pf enabled with its own reference" || bad "couldn't set up the other product ($OTHER_TOK, $PF0)"
ROOT_RULES0="$(root_rules)"

say "crash: kill -9 tailbondd and mqvpn, then start it again as launchd would"
sapi -X POST "http://$API/api/connect" >/dev/null
wait_for 30 "server running again" server_up && ok "server running again" || bad "server didn't restart"
wait_for 10 "record" test -s "$SRV/server-net.state" || true
MQ_PID="$(cat "$SRV/mqvpn.pid" 2>/dev/null || true)"
kill -9 "$SRV_PID" ${MQ_PID:+"$MQ_PID"} 2>/dev/null || true
wait "$SRV_PID" 2>/dev/null || true
sleep 2
[ -n "$(anchor_rules)" ] && [ -f "$SRV/server-net.state" ] && ok "after kill -9 the rules and the record are still there (as expected)" || bad "unexpected state after kill -9"
start_server
wait_for 60 "server up after the crash" server_up && ok "server up again after kill -9" || bad "server not up after kill -9"
wait_for 10 "stale restore" grep -q 'restore record from an unclean stop' "$WORK/server.log" \
  && ok "server-up put the crashed run's state back first: $(grep -o 'restored (after an unclean stop).*' "$WORK/server.log" | tail -n1)" || bad "no stale restore in the log"
wait_for 120 "client back" sh -c "[ \"\$(curl -fsS -m 3 http://$API/api/status | jq '[.clients[]? | select(.name == \"linux-vm\" and .connected)] | length')\" -ge 1 ]" \
  && ok "the VM reconnected by itself" || bad "the VM didn't reconnect"
sapi -X POST "http://$API/api/disconnect" >/dev/null
wait_for 20 "server-down" sh -c "! test -f '$SRV/server-net.state'" || true
check_clean "after kill -9, restart, stop"

say "crash, then the restore uninstall.sh runs (hooks/server-down)"
sapi -X POST "http://$API/api/connect" >/dev/null
wait_for 30 "server running again" server_up || true
wait_for 10 "record" test -s "$SRV/server-net.state" || true
MQ_PID="$(cat "$SRV/mqvpn.pid" 2>/dev/null || true)"
kill -9 "$SRV_PID" ${MQ_PID:+"$MQ_PID"} 2>/dev/null || true
wait "$SRV_PID" 2>/dev/null || true
SRV_PID=""
sleep 2
TB_STATE_DIR="$SRV" "$ROOT/packaging/macos/hooks/server-down" | sed 's/^/  /'
check_clean "after kill -9 and the uninstall restore"

say "disable_tailscale: the hook leaves Tailscale alone (TB_TAILSCALE=0)"
mkdir -p "$WORK/h"
: >"$WORK/ts-advertised"
env PATH="$WORK/bin:$PATH" TB_STATE_DIR="$WORK/h" TB_TUN=lo0 TB_SUBNET="$SUBNET" TB_PORT="$PORT" TB_WEB_PORTS="8077 80" \
  TB_TAILSCALE=0 TB_ADVERTISE_ROUTE=1 "$ROOT/packaging/macos/hooks/server-up" >"$WORK/hook-nots.log" 2>&1
NAT="$(pfctl -a "$ANCHOR" -s nat 2>/dev/null)"
grep -q "100.64.0.0/10" <<<"$NAT" && bad "tailnet NAT with Tailscale disabled: $NAT" || ok "no tailnet NAT with Tailscale disabled ($(grep -c '^nat' <<<"$NAT") uplink NAT rules)"
[ -s "$WORK/ts-advertised" ] && bad "advertised with Tailscale disabled: $(cat "$WORK/ts-advertised")" || ok "no route advertised with Tailscale disabled"
TB_STATE_DIR="$WORK/h" "$ROOT/packaging/macos/hooks/server-down" | sed 's/^/  /'
SRV="$WORK/h" check_clean "after the disable_tailscale hook run"

echo
echo "e2e-macos-server: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
