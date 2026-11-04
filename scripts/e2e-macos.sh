#!/usr/bin/env bash
# End-to-end test of the macOS client (hub) role on a Mac (CI: GitHub's
# macos-latest runner; docs/macos.md). The hub (tailbondd + patched mqvpn
# on this Mac) bonds an interface path and two phones' shared mobile data:
#
#   server   tailbondd server role + mqvpn in a small aarch64 Linux VM under
#            QEMU (scripts/e2e-macos/build-server-vm.sh): mqvpn's server
#            isn't built for macOS, and a server in this Mac's own network
#            stack would answer the tunnel's traffic locally. QEMU's user
#            network forwards 127.0.0.1:4443/udp (mqvpn) and :18077/tcp (the
#            server's API) to it; its token web server (:8080) is reachable
#            only through the tunnel.
#   hub      tailbondd client role, mqvpn with the control API: path lo0
#            (IP_BOUND_IF-pinned, the only interface that reaches the
#            VM's forwarded port) plus every ready share as a relay path.
#   phones   two tb-sharesim relays (internal/share, the reference sharer)
#            on a fake-Ethernet LAN, feth0 (hub, 192.168.77.1) <-> feth1
#            (phones, .2 and .3): mDNS adverts on feth1 and registrations
#            with the server; upstream to the server over the normal route.
#
# Checks: the server sees the hub with 3 active paths; traffic crosses the
# tunnel (the token, a 2 MB download; bytes on every path); the dashboard
# API lists the shares ("Shared by nearby phones") with kind
# shared-cellular; killing one phone drops its path within ~4 s and the
# tunnel keeps working; "Use this Mac's connections" off leaves only the
# phone; a sharer the server lists but the hub can't reach shows the client
# isolation note.
#
#   MQVPN=out/macos/mqvpn TAILBONDD=out/agent/darwin-arm64/tailbondd \
#   SHARESIM=out/agent/darwin-arm64/tb-sharesim VM=out/e2e-vm scripts/e2e-macos.sh
#
# Needs: macOS, root (sudo: utun, feth, mqvpn's lock), qemu-system-aarch64,
# curl, jq. KEEP=1 leaves everything running.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/out/macos/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/darwin-arm64/tailbondd}"
SHARESIM="${SHARESIM:-$ROOT/out/agent/darwin-arm64/tb-sharesim}"
VM="${VM:-$ROOT/out/e2e-vm}"
API=127.0.0.1:8077   # the hub's dashboard
SAPI=127.0.0.1:18077 # the server's API, forwarded into the VM
PORT=4443

if [ "$(id -u)" -ne 0 ]; then
  exec sudo --preserve-env=KEEP,VERBOSE,PATH MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" SHARESIM="$SHARESIM" VM="$VM" "$0" "$@"
fi
[ "$(uname -s)" = Darwin ] || { echo "run this on macOS"; exit 2; }
for f in "$MQVPN" "$TAILBONDD" "$SHARESIM"; do [ -x "$f" ] || { echo "missing $f"; exit 2; }; done
for f in "$VM/vmlinuz" "$VM/initramfs.gz"; do [ -f "$f" ] || { echo "missing $f (scripts/e2e-macos/build-server-vm.sh)"; exit 2; }; done
for c in qemu-system-aarch64 curl jq nc; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done

WORK="$(mktemp -d /tmp/tailbond-e2e-macos.XXXXXX)"
chmod 755 "$WORK"
PASS=0 FAIL=0
QEMU_PID="" HUB_PID="" PIXEL_PID="" IPHONE_PID=""
say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
info() { printf '  ---- %s\n' "$*"; }
hapi() { curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
sapi() { curl -fsS -m 10 -H 'Content-Type: application/json' "$@"; }
now()  { python3 -c 'import time; print("%.1f" % time.time())'; }
wait_for() { # wait_for <seconds> <description> <command...>
  local t=$1 what=$2; shift 2
  for _ in $(seq 1 "$t"); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "  timed out after ${t}s waiting for: $what"
  return 1
}

cleanup() {
  local rc=$?
  if [ -n "${KEEP:-}" ]; then echo "KEEP set: VM, agents and feth0/feth1 left running; work dir $WORK"; return; fi
  for p in "$HUB_PID" "$PIXEL_PID" "$IPHONE_PID"; do [ -n "$p" ] && kill "$p" 2>/dev/null || true; done
  sleep 2
  pkill -f "$WORK/hub" 2>/dev/null || true
  [ -n "$QEMU_PID" ] && kill "$QEMU_PID" 2>/dev/null || true
  ifconfig feth0 destroy 2>/dev/null || true
  ifconfig feth1 destroy 2>/dev/null || true
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- hub agent log (tail) ---"; tail -n 80 "$WORK/hub.log" 2>/dev/null || true
    for s in pixel iphone; do echo "--- $s sharer log (tail) ---"; tail -n 20 "$WORK/$s.log" 2>/dev/null || true; done
    echo "--- server VM console (tail) ---"; grep -av '^\[ *[0-9.]*\] ' "$WORK/vm.console" | tail -n 80 2>/dev/null || true
  fi
  mkdir -p "$ROOT/out/e2e-macos" && cp "$WORK"/*.log "$WORK/vm.console" "$ROOT/out/e2e-macos/" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

say "server: tailbondd + mqvpn in an aarch64 Linux VM (QEMU, TCG)"
TOKEN="tb-$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')"
accel=tcg,thread=multi
qemu-system-aarch64 -accel help 2>/dev/null | grep -qx hvf && sysctl -n kern.hv_support 2>/dev/null | grep -qx 1 && accel=hvf
cpu=cortex-a72; [ "$accel" = hvf ] && cpu=host
info "accelerator $accel ($(qemu-system-aarch64 --version | head -n1))"
qemu-system-aarch64 -M virt -cpu "$cpu" -accel "$accel" -smp 2 -m 1024 \
  -kernel "$VM/vmlinuz" -initrd "$VM/initramfs.gz" \
  -append "console=ttyAMA0 panic=-1 tb.time=$(date +%s) tb.token=$TOKEN tb.public=127.0.0.1 tb.port=$PORT" \
  -netdev "user,id=n0,hostfwd=udp:127.0.0.1:$PORT-:$PORT,hostfwd=tcp:127.0.0.1:18077-:8077" \
  -device virtio-net-pci,netdev=n0,romfile= -device virtio-rng-pci \
  -display none -monitor none -serial "file:$WORK/vm.console" -no-reboot &
QEMU_PID=$!
T0=$(date +%s)
if wait_for 300 "server up in the VM" sh -c "curl -fsS -m 3 http://$SAPI/api/status | jq -e '.state == \"connected\"'"; then
  ok "server up in $(( $(date +%s) - T0 )) s: $(grep -m1 '^vm: up' "$WORK/vm.console" || true)"
else
  bad "server did not come up: $(curl -sS -m 3 "http://$SAPI/api/status" 2>&1 | head -c 600)"; exit 1
fi
pair() { sapi -X POST -d "{\"name\":\"$1\"}" "http://$SAPI/api/clients" | jq -r .uri; }
HUB_URI="$(pair mac)"; PIXEL_URI="$(pair pixel)"; IPHONE_URI="$(pair iphone)"; ISO_URI="$(pair isolated)"
[ -n "$HUB_URI" ] && [ "$HUB_URI" != null ] && ok "paired the Mac and three phones" || { bad "pairing failed"; exit 1; }

say "LAN: feth0 (this Mac, 192.168.77.1) <-> feth1 (the phones, .2 and .3)"
ifconfig feth0 destroy 2>/dev/null || true; ifconfig feth1 destroy 2>/dev/null || true
if ifconfig feth0 create && ifconfig feth1 create && ifconfig feth0 peer feth1; then
  ifconfig feth0 inet 192.168.77.1/24 up
  ifconfig feth1 inet 192.168.77.2/24 up
  ifconfig feth1 inet 192.168.77.3 netmask 255.255.255.255 alias
  ok "fake Ethernet pair up: $(ifconfig feth0 | head -n1)"
else
  bad "can't create feth interfaces on this Mac"; exit 1
fi

say "phones: two sharers (the reference relay), mDNS on feth1 + registered with the server"
"$SHARESIM" -pairing "$PIXEL_URI" -listen 192.168.77.2:47077 -label "Pixel's 5G" -id sim-pixel \
  -mdns feth1 -register "http://$SAPI" >"$WORK/pixel.log" 2>&1 &
PIXEL_PID=$!
"$SHARESIM" -pairing "$IPHONE_URI" -listen 192.168.77.3:47077 -label "iPhone 16" -id sim-iphone \
  -mdns feth1 -register "http://$SAPI" >"$WORK/iphone.log" 2>&1 &
IPHONE_PID=$!
wait_for 30 "both sharers registered" sh -c "curl -fsS -m 3 http://$SAPI/api/shares | jq -e '.sharers | length == 2'" \
  && ok "server lists them: $(sapi "http://$SAPI/api/shares" | jq -c '[.sharers[] | {label, lan, port}]')" \
  || bad "sharers not registered: $(sapi "http://$SAPI/api/shares" | jq -c .)"

say "hub: tailbondd client role on this Mac (path lo0 + the shares)"
mkdir -p "$WORK/hub"
cat >"$WORK/hub/tailbond.json" <<EOF
{"role": "client", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/hub", "hooks_dir": "$ROOT/packaging/macos/hooks",
 "listen": "$API", "mode": "speed", "disable_tailscale": true,
 "client": {"pairing_uri": "$HUB_URI", "no_dns": true, "manage_routes": false, "captive_portal_detect": false,
            "paths": [{"iface": "lo0", "enabled": true, "priority": "primary", "kind": "ethernet"}]}}
EOF
"$TAILBONDD" -config "$WORK/hub/tailbond.json" >"$WORK/hub.log" 2>&1 &
HUB_PID=$!
wait_for 90 "hub connected" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "hub connected: tunnel $(hapi "http://$API/api/status" | jq -r .tunnel_ip) on $(grep -o 'TUN utun[0-9]* configured' "$WORK/hub.log" | tail -n1 | awk '{print $2}')" \
  || { bad "hub not connected: $(hapi "http://$API/api/status" | jq -c '{state, error}')"; exit 1; }
if wait_for 90 "2 shares ready" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '[.shares[]? | select(.state == \"ready\")] | length == 2'"; then
  ok "hub found both phones: $(hapi "http://$API/api/status" | jq -c '[.shares[] | {label, via, state, path}]')"
else
  bad "shares: $(hapi "http://$API/api/status" | jq -c '{shares, share_note}')"
fi
VIA="$(hapi "http://$API/api/status" | jq -r '[.shares[]?.via] | unique | join(",")')"
info "discovery: $VIA (mdns = Bonjour on the feth LAN; server = the home server's list)"

three() { sapi "http://$SAPI/api/status" | jq -e '.clients[] | select(.name == "mac" and .connected) | [.paths[] | select(.state == "active")] | length >= 3' >/dev/null; }
if wait_for 90 "server sees 3 active paths for the Mac" three; then
  ok "server: the Mac bonds $(sapi "http://$SAPI/api/status" | jq '[.clients[] | select(.name == "mac") | .paths[] | select(.state == "active")] | length') active paths: $(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | select(.name == "mac") | .paths[] | {id, state, srtt_ms}]')"
else
  bad "server paths for the Mac: $(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | select(.name == "mac") | .paths[]]')"
fi

say "dashboard API: Shared by nearby phones, kinds, mqvpn's control API on macOS"
ST="$(hapi "http://$API/api/status")"
jq -e '.control_api == true' <<<"$ST" >/dev/null && ok "per-path stats come from mqvpn's control API (patch 0003)" || bad "no control API: $(jq -c '{control_api, error}' <<<"$ST")"
jq -e '.platform == "darwin" and .use_local == true and .use_shares == true' <<<"$ST" >/dev/null \
  && ok "platform darwin, use_local and use_shares on" || bad "$(jq -c '{platform, use_local, use_shares}' <<<"$ST")"
for l in "Pixel's 5G (shared)" "iPhone 16 (shared)"; do
  jq -e --arg l "$l" '.paths[] | select(.label == $l and .kind == "shared-cellular" and .state == "active")' <<<"$ST" >/dev/null \
    && ok "path \"$l\": kind shared-cellular, active ($(jq -r --arg l "$l" '.paths[] | select(.label == $l) | .iface' <<<"$ST"))" \
    || bad "no active shared-cellular path \"$l\": $(jq -c '[.paths[] | {iface, kind, label, state}]' <<<"$ST")"
done
jq -e '.paths[] | select(.iface == "lo0" and .kind == "ethernet" and .state == "active")' <<<"$ST" >/dev/null \
  && ok "interface path lo0 active" || bad "lo0: $(jq -c '[.paths[] | {iface, kind, state}]' <<<"$ST")"
# mqvpn's own answer: relay paths named, the interface path pinned
CTL="$(printf '{"cmd":"get_status"}\n' | nc -w 3 127.0.0.1 9077 || true)"
jq -e '.state == "established" and ([.clients[0].paths[].iface] | (index("lo0") != null) and (map(select(startswith("sh"))) | length == 2))' <<<"$CTL" >/dev/null \
  && ok "mqvpn get_status on macOS: $(jq -c '[.clients[0].paths[] | {iface, status}]' <<<"$CTL")" \
  || bad "mqvpn get_status: ${CTL:-no answer}"
grep -q "relay path via 127.0.0.1" "$WORK/hub.log" && ok "mqvpn runs the shares as relay paths: $(grep -o 'path\[[0-9]*\] sh[0-9a-f]*: relay path via [0-9.]* port [0-9]*' "$WORK/hub.log" | sort -u | tr '\n' ';')" \
  || bad "no relay path in mqvpn's log"
hapi "http://$API/api/shares" | jq -e '.shares | length == 2' >/dev/null && ok "GET /api/shares lists both" || bad "GET /api/shares: $(hapi "http://$API/api/shares" | jq -c .)"

say "traffic through the tunnel (token server only the tunnel reaches)"
GOT="$(curl -fsS -m 10 http://10.77.0.1:8080/token || true)"
[ "$GOT" = "$TOKEN" ] && ok "token through the tunnel" || bad "token: got '${GOT}'"
curl -fsS -m 60 -o "$WORK/blob" http://10.77.0.1:8080/blob && [ "$(wc -c <"$WORK/blob")" -ge 2000000 ] \
  && ok "2 MB through the tunnel" || bad "download through the tunnel failed"
sleep 3
PB="$(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | select(.name == "mac") | .paths[] | select(.state == "active") | .rx_bytes + .tx_bytes]')"
info "server per-path bytes: $PB"
jq -e 'length >= 3 and all(. > 0)' <<<"$PB" >/dev/null && ok "every path carried traffic" || bad "a path carried nothing: $PB"
SB="$(hapi "http://$API/api/status" | jq -c '[.shares[] | .up_bytes + .down_bytes]')"
jq -e 'all(. > 0)' <<<"$SB" >/dev/null && ok "bytes relayed by each phone: $SB" || bad "share counters: $SB"

say "the Pixel stops (sharer killed): its path goes, the tunnel keeps working"
PIX="$(hapi "http://$API/api/status" | jq -r '.shares[] | select(.label | startswith("Pixel")) | .path')"
( while :; do
    if curl -fsS -m 1 http://10.77.0.1:8080/token >/dev/null 2>&1; then echo "$(now) ok"; else echo "$(now) fail"; fi
    sleep 0.25
  done ) >"$WORK/probe.log" 2>&1 &
PROBE=$!
sleep 2
K="$(now)"
kill -9 "$PIXEL_PID"; PIXEL_PID=""
gone() { curl -fsS -m 1 "http://$API/api/status" | jq -e --arg p "$PIX" '[.shares[]? | select(.path == $p and .state == "ready")] | length == 0'; }
if wait_for 15 "share marked gone" gone; then
  D="$(python3 -c "print(round($(now) - $K, 1))")"
  python3 -c "import sys; sys.exit(0 if $D <= 6 else 1)" && ok "hub dropped the Pixel's path ${D} s after it stopped (link timeout 3.5 s)" \
    || bad "the Pixel's path went only after ${D} s"
else
  bad "share still ready: $(hapi "http://$API/api/status" | jq -c .shares)"
fi
two() { sapi "http://$SAPI/api/status" | jq -e '[.clients[] | select(.name == "mac" and .connected) | .paths[] | select(.state == "active")] | length == 2' >/dev/null; }
wait_for 30 "server back to 2 active paths" two && ok "server: the Mac now bonds lo0 + the iPhone" \
  || bad "server paths: $(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | select(.name == "mac") | .paths[] | {id, state}]')"
sleep 3
kill "$PROBE" 2>/dev/null || true
GAP="$(awk -v k="$K" '$1 >= k && $2 == "fail"' "$WORK/probe.log" | wc -l | tr -d ' ')"
TAIL_OK="$(tail -n 4 "$WORK/probe.log" | grep -c ' ok$' || true)"
info "probe after the kill: $GAP failed request(s) of $(awk -v k="$K" '$1 >= k' "$WORK/probe.log" | wc -l | tr -d ' ') (a share leaving restarts mqvpn, as on Linux)"
[ "$TAIL_OK" -ge 3 ] && curl -fsS -m 10 http://10.77.0.1:8080/token | grep -qx "$TOKEN" \
  && ok "traffic continues without the Pixel" || bad "tunnel not working after the Pixel left: $(tail -n 5 "$WORK/probe.log" | tr '\n' ' ')"

say "\"Use this Mac's connections\" off: only the iPhone is bonded"
hapi -X POST -d '{"use_local": false}' "http://$API/api/share" >/dev/null
one() { sapi "http://$SAPI/api/status" | jq -e '[.clients[] | select(.name == "mac" and .connected) | .paths[] | select(.state == "active")] | length == 1' >/dev/null; }
if wait_for 45 "1 active path" one && wait_for 20 "token over the iPhone only" sh -c "curl -fsS -m 3 http://10.77.0.1:8080/token | grep -qx $TOKEN"; then
  ok "only the iPhone's share carries the tunnel; $(hapi "http://$API/api/status" | jq -c '[.paths[] | select(.state == "active") | .label // .iface]')"
else
  bad "use_local off: server $(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | select(.name == "mac") | .paths[] | {id, state}]')"
fi
hapi -X POST -d '{"use_local": true}' "http://$API/api/share" >/dev/null

say "client isolation: a phone the server lists on our LAN that never answers"
"$SHARESIM" -pairing "$ISO_URI" -listen 192.168.77.9:47077 -label "Sam's phone" -id sim-isolated \
  -register "http://$SAPI" -register-only >>"$WORK/isolated.log" 2>&1 \
  && ok "registered a sharer at 192.168.77.9 (nobody there)" || bad "register-only failed: $(cat "$WORK/isolated.log")"
if wait_for 90 "isolation note" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.share_note | test(\"blocks devices\")'"; then
  ok "hub says: $(hapi "http://$API/api/status" | jq -r .share_note)"
else
  bad "no isolation note: $(hapi "http://$API/api/status" | jq -c '{shares, share_note}')"
fi
hapi "http://$API/api/status" | jq -e '.state == "connected"' >/dev/null && ok "hub still connected" || bad "hub not connected"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
