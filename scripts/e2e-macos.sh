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
# Use phone hub (docs/phone-hub.md): a stand-in phone hub (tb-sharesim
# -proxy: internal/proxy, the reference SOCKS5 server, on 192.168.77.4,
# upstream over this Mac's tunnel) is found; "Use" sets the system SOCKS
# proxy of the default route's network service to tailbondd's forwarder;
# curl through the system proxy's values reaches the tunnel-only token and
# the hub logs the traffic; PAC mode sets the auto-proxy URL; turning it off
# and killing tailbondd (-9, then starting it as launchd would) put back the
# original settings.
#
# Other VPNs (docs/macos.md "Using tailbond with NetBird"): the real NetBird
# package, logged out, is detected without fuss; then a NetBird stand-in (a
# wireguard-go utun owning 100.92.0.0/16, an /etc/resolver/corp.example
# answered by tb-dnsstub, a `netbird status --json` stub on PATH): no route
# of tailbond's covers 100.92/16, a tailnet machine inside it is reached by
# a /32, ts.net resolves via the tunnel while corp.example keeps its own
# resolver and the global resolver is untouched, NetBird's endpoints route
# via en0 (Direct), Bonded raises the MTU or falls back, and a disconnect
# and a killed agent take routes and resolver files away.
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
DNSSTUB="${DNSSTUB:-$ROOT/out/agent/darwin-arm64/tb-dnsstub}"
VM="${VM:-$ROOT/out/e2e-vm}"
API=127.0.0.1:8077   # the hub's dashboard
SAPI=127.0.0.1:18077 # the server's API, forwarded into the VM
PORT=4443

if [ "$(id -u)" -ne 0 ]; then
  exec sudo --preserve-env=KEEP,VERBOSE,PATH MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" SHARESIM="$SHARESIM" DNSSTUB="$DNSSTUB" VM="$VM" "$0" "$@"
fi
[ "$(uname -s)" = Darwin ] || { echo "run this on macOS"; exit 2; }
for f in "$MQVPN" "$TAILBONDD" "$SHARESIM"; do [ -x "$f" ] || { echo "missing $f"; exit 2; }; done
for f in "$VM/vmlinuz" "$VM/initramfs.gz"; do [ -f "$f" ] || { echo "missing $f (scripts/e2e-macos/build-server-vm.sh)"; exit 2; }; done
for c in qemu-system-aarch64 curl jq nc; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done

WORK="$(mktemp -d /tmp/tailbond-e2e-macos.XXXXXX)"
chmod 755 "$WORK"
PASS=0 FAIL=0
QEMU_PID="" HUB_PID="" PIXEL_PID="" IPHONE_PID="" PHUB_PID="" SVC="" DNS_PID="" WG=""
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
  for p in "$HUB_PID" "$PIXEL_PID" "$IPHONE_PID" "$PHUB_PID" "$DNS_PID"; do [ -n "$p" ] && kill "$p" 2>/dev/null || true; done
  rm -f /etc/resolver/corp.example
  [ -n "$WG" ] && { rm -f "/var/run/wireguard/$WG.sock"; pkill -f "wireguard-go utun" 2>/dev/null || true; }
  sleep 2
  # never leave the runner's proxy pointing at a dead forwarder
  if [ -n "$SVC" ]; then
    networksetup -setsocksfirewallproxystate "$SVC" off 2>/dev/null || true
    networksetup -setautoproxystate "$SVC" off 2>/dev/null || true
    networksetup -setautoproxyurl "$SVC" "" 2>/dev/null || true
  fi
  pkill -f "$WORK/hub" 2>/dev/null || true
  [ -n "$QEMU_PID" ] && kill "$QEMU_PID" 2>/dev/null || true
  ifconfig feth0 destroy 2>/dev/null || true
  ifconfig feth1 destroy 2>/dev/null || true
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- hub agent log (tail) ---"; tail -n 80 "$WORK/hub.log" 2>/dev/null || true
    for s in pixel iphone phonehub; do echo "--- $s sharer log (tail) ---"; tail -n 20 "$WORK/$s.log" 2>/dev/null || true; done
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
  # the phones' addresses as /32s: feth0 alone routes 192.168.77.0/24, so
  # the hub's datagrams to them stay on this one path
  ifconfig feth1 inet 192.168.77.2 netmask 255.255.255.255 up
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
disown "$PROBE"
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

say "use phone hub: a stand-in hub (SOCKS5, internal/proxy) at 192.168.77.4, upstream through this Mac's tunnel"
ifconfig feth1 inet 192.168.77.4 netmask 255.255.255.255 alias
"$SHARESIM" -pairing "$PIXEL_URI" -proxy 192.168.77.4:1888 -label "Pixel" -id sim-pixel-hub \
  -mdns feth1 -register "http://$SAPI" >"$WORK/phonehub.log" 2>&1 &
PHUB_PID=$!
DEV="$(netstat -rn -f inet | awk '$1 == "default" && $NF !~ /^(utun|ipsec|ppp|gif|stf|lo)/ {print $NF; exit}')"
SVC="$(networksetup -listnetworkserviceorder | awk -v d="$DEV" '/^\([0-9*]+\) / {sub(/^\([0-9*]+\) /, ""); n = $0} index($0, "Device: " d ")") {print n; exit}')"
[ -n "$SVC" ] && ok "default route: $DEV, network service \"$SVC\"" || { bad "no network service for the default route ($DEV)"; exit 1; }
# a recognisable original: a SOCKS server that's set but off; restoring
# must bring back exactly this
networksetup -setsocksfirewallproxy "$SVC" 10.9.9.9 1080 off
networksetup -setsocksfirewallproxystate "$SVC" off
socks() { networksetup -getsocksfirewallproxy "$SVC" | awk -F': ' '/^(Enabled|Server|Port):/ {printf "%s=%s ", $1, $2}'; }
ORIG="$(socks)"
info "original SOCKS setting of \"$SVC\": $ORIG"
found() { hapi -X POST -d '{"refresh": true}' "http://$API/api/phone-hub" | jq -e '.found[] | select(.id == "sim-pixel-hub")'; }
if wait_for 60 "hub finds the phone hub" found; then
  ok "found: $(hapi "http://$API/api/phone-hub" | jq -c '[.found[] | {id, label, addr, via}]')"
else
  bad "phone hub not found: $(hapi "http://$API/api/phone-hub" | jq -c .)"
fi
use_hub() { hapi -X POST -d "{\"use\": \"sim-pixel-hub\", \"pac\": ${1:-false}}" "http://$API/api/phone-hub"; }
PH="$(use_hub false || true)"
jq -e --arg s "$SVC" '.using.mode == "socks" and .using.service == $s and .using.local == "127.0.0.1:1889"' <<<"$PH" >/dev/null \
  && ok "using it: $(jq -c '.using | {label, addr, mode, service, local}' <<<"$PH")" || bad "use: $PH"
wait_for 5 "status shows the hub in use" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.phone_hub.using.label == \"Pixel\"'" \
  && ok "dashboard status: Using Pixel's bonded connection (proxy)" || bad "status: $(hapi "http://$API/api/status" | jq -c .phone_hub)"
info "networksetup: $(socks)"
SP="$(scutil --proxy)"
SH="$(awk '/SOCKSProxy :/ {print $3}' <<<"$SP")"; SPORT="$(awk '/SOCKSPort :/ {print $3}' <<<"$SP")"
awk '/SOCKSEnable :/ {print $3}' <<<"$SP" | grep -qx 1 && [ "$SH" = 127.0.0.1 ] && [ "$SPORT" = 1889 ] \
  && ok "system proxy (scutil --proxy): SOCKS $SH:$SPORT, on" || bad "scutil --proxy: $(tr '\n' ' ' <<<"$SP")"
GOT="$(curl -fsS -m 15 --socks5-hostname "$SH:$SPORT" http://10.77.0.1:8080/token || true)"
[ "$GOT" = "$TOKEN" ] && ok "curl through the system proxy's values ($SH:$SPORT) reached the tunnel-only token" || bad "token through the proxy: got '$GOT'"
# (the stand-in runs on this Mac, so the forwarder's connection comes from
# its own address, .4)
if wait_for 10 "the hub logs the Mac's traffic" grep -Eq 'client 192\.168\.77\.[0-9]+ conns [0-9]+ up [1-9][0-9]* down [1-9]' "$WORK/phonehub.log"; then
  ok "the phone hub carried it: $(grep -E 'client 192\.168\.77\.' "$WORK/phonehub.log" | tail -n1 | sed 's/.*proxy: //')"
else
  bad "no traffic at the phone hub: $(tail -n 5 "$WORK/phonehub.log" | tr '\n' ' ')"
fi
[ -f "$WORK/hub/phone-hub-proxy.json" ] && ok "restore record written: $(jq -c '{service, mode, previous}' "$WORK/hub/phone-hub-proxy.json")" || bad "no restore record"

say "phone hub off: the original SOCKS setting is back"
hapi -X POST -d '{"use": ""}' "http://$API/api/phone-hub" >/dev/null
[ "$(socks)" = "$ORIG" ] && ok "restored: $(socks)" || bad "after off: $(socks), want $ORIG"
[ ! -f "$WORK/hub/phone-hub-proxy.json" ] && ok "restore record removed" || bad "restore record left behind"

say "phone hub, PAC mode: the auto-proxy URL, local and tailnet direct"
# a recognisable original, as for SOCKS: a PAC URL that's set but off
networksetup -setautoproxyurl "$SVC" http://10.9.9.9/original.pac
networksetup -setautoproxystate "$SVC" off
PACORIG="$(networksetup -getautoproxyurl "$SVC" | tr '\n' ' ')"
PH="$(use_hub true || true)"
PACNOW="$(networksetup -getautoproxyurl "$SVC" | tr '\n' ' ')"
jq -e '.using.mode == "pac"' <<<"$PH" >/dev/null && grep -q "URL: http://127.0.0.1:8077/proxy.pac Enabled: Yes" <<<"$PACNOW" \
  && ok "auto-proxy URL set: $PACNOW" || bad "PAC: $(jq -c .using <<<"$PH") / $PACNOW"
curl -fsS -m 5 "http://$API/proxy.pac" | grep -q 'SOCKS5 127.0.0.1:1889' && ok "proxy.pac sends the rest to 127.0.0.1:1889" || bad "proxy.pac: $(curl -sS -m 5 "http://$API/proxy.pac" | head -c 300)"
hapi -X POST -d '{"use": ""}' "http://$API/api/phone-hub" >/dev/null
[ "$(networksetup -getautoproxyurl "$SVC" | tr '\n' ' ')" = "$PACORIG" ] && ok "auto-proxy setting restored" || bad "auto-proxy after off: $(networksetup -getautoproxyurl "$SVC" | tr '\n' ' '), want $PACORIG"

say "phone hub, tailbondd killed (-9) while in use: started again (as launchd does), the original comes back"
use_hub false >/dev/null || true
socks | grep -q 'Server=127.0.0.1' && ok "in use again: $(socks)" || bad "not in use: $(socks)"
info "restore record: $(jq -c '{service, mode, previous}' "$WORK/hub/phone-hub-proxy.json" 2>/dev/null)"
kill -9 "$HUB_PID"; wait "$HUB_PID" 2>/dev/null || true
info "after kill -9 (nobody to restore it yet): $(socks)"
"$TAILBONDD" -config "$WORK/hub/tailbond.json" >>"$WORK/hub.log" 2>&1 &
HUB_PID=$!
if wait_for 30 "restored at start" grep -q "put back the proxy settings of \"$SVC\"" "$WORK/hub.log"; then
  is_orig() { [ "$(socks)" = "$ORIG" ]; }
  # tailbondd checks again at +3, +10, +30 s (other writers at start-up)
  wait_for 20 "the original setting stays" is_orig && sleep 4 && is_orig \
    && ok "restored at start: $(socks)" || bad "after restart: $(socks), want $ORIG"
  grep -q "changed back; putting them back again" "$WORK/hub.log" \
    && info "tailbondd had to put them back a second time: $(grep -c 'changed back' "$WORK/hub.log") time(s)"
else
  bad "no restore at start: $(socks)"
fi
wait_for 90 "hub connected again" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "hub connected again after the restart" || bad "hub not connected after the restart"

say "other VPNs: the real NetBird package, not logged in"
NB_REAL="$(command -v netbird || ls /opt/homebrew/bin/netbird /usr/local/bin/netbird 2>/dev/null | head -n1 || true)"
if [ -n "$NB_REAL" ]; then
  "$NB_REAL" service install >/dev/null 2>&1 || true
  "$NB_REAL" service start >/dev/null 2>&1 || true
  sleep 3
  info "netbird status: $("$NB_REAL" status 2>&1 | head -n 2 | tr '\n' ' ')"
  if wait_for 60 "coexist sees NetBird" sh -c "curl -fsS -m 3 http://$API/api/coexist | jq -e '.netbird.cli == true'"; then
    CX="$(hapi "http://$API/api/coexist")"
    jq -e '.netbird.logged_in == false and ([.foreign[] | select(.kind == "netbird")] | length == 0)' <<<"$CX" >/dev/null \
      && ok "a logged-out NetBird is seen and left be: $(jq -c '.netbird | {cli, running, logged_in, error}' <<<"$CX")" \
      || bad "logged-out NetBird: $(jq -c . <<<"$CX")"
  else
    bad "NetBird's CLI not seen: $(hapi "http://$API/api/coexist" | jq -c .)"
  fi
  hapi "http://$API/api/status" | jq -e '.state == "connected"' >/dev/null && ok "hub still connected" || bad "hub not connected"
  "$NB_REAL" service stop >/dev/null 2>&1 || true
  "$NB_REAL" service uninstall >/dev/null 2>&1 || true
else
  info "NOTE: the NetBird package isn't installed here (brew install netbirdio/tap/netbird failed): smoke test skipped"
fi

say "other VPNs: a NetBird stand-in (wireguard-go utun on 100.92.0.0/16, corp.example resolver, netbird CLI stub)"
WG_TUN_NAME_FILE="$WORK/wg.name" wireguard-go utun >"$WORK/wg.log" 2>&1
WG="$(cat "$WORK/wg.name" 2>/dev/null)"
if [ -n "$WG" ] && ifconfig "$WG" inet 100.92.1.2 100.92.1.2 netmask 255.255.0.0 up && route -n add -net 100.92.0.0/16 -interface "$WG" >/dev/null; then
  ifconfig "$WG" mtu 1280
  ok "second VPN on $WG: $(netstat -rn -f inet | awk -v w="$WG" '$NF == w {print $1}' | tr '\n' ' ')"
else
  bad "can't make a second VPN utun: $(cat "$WORK/wg.log")"
fi
"$DNSSTUB" -listen 127.0.0.1:5390 -a corp.example=10.92.0.5 >"$WORK/dnsstub.log" 2>&1 &
DNS_PID=$!
mkdir -p /etc/resolver
printf 'nameserver 127.0.0.1\nport 5390\n' >/etc/resolver/corp.example
mkdir -p "$WORK/nbstub"
cat >"$WORK/nbstub/netbird" <<'NB'
#!/bin/sh
[ "$1 $2" = "status --json" ] || exit 0
cat <<'J'
{"peers":{"total":1,"connected":1,"details":[{"fqdn":"laptop.netbird.cloud","netbirdIp":"100.92.4.9","status":"Connected",
  "iceCandidateEndpoint":{"local":"192.168.64.2:51820","remote":"203.0.113.50:51820"},"relayAddress":""}]},
 "management":{"url":"https://192.0.2.10:443","connected":true},"signal":{"url":"https://192.0.2.11:443","connected":true},
 "relays":{"total":1,"available":1,"details":[{"uri":"rels://198.51.100.77:443","available":true}]},
 "netbirdIp":"100.92.1.2/16","networks":[],"dnsServers":[{"servers":["127.0.0.1:5390"],"domains":["corp.example"],"enabled":true}]}
J
NB
chmod +x "$WORK/nbstub/netbird"
# the hub with DNS on, the stub first on PATH
kill "$HUB_PID" 2>/dev/null; wait "$HUB_PID" 2>/dev/null || true
jq '.client.no_dns = false' "$WORK/hub/tailbond.json" >"$WORK/hub/tb.tmp" && mv "$WORK/hub/tb.tmp" "$WORK/hub/tailbond.json"
start_hub() { PATH="$WORK/nbstub:$PATH" "$TAILBONDD" -config "$WORK/hub/tailbond.json" >>"$WORK/hub.log" 2>&1 & HUB_PID=$!; }
start_hub
wait_for 90 "hub connected next to the second VPN" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "hub connected" || { bad "hub not connected: $(hapi "http://$API/api/status" | jq -c '{state, error}')"; }
tbtun() { grep -o 'TUN utun[0-9]* configured' "$WORK/hub.log" | tail -n1 | awk '{print $2}'; }
TB="$(tbtun)"
iface_of() { route -n get "$1" 2>/dev/null | awk '/interface:/ {print $2}'; }
cxj() { hapi "http://$API/api/coexist"; }
if wait_for 60 "tailnet /32 on tailbond's utun" sh -c "[ \"\$(route -n get 100.92.5.5 2>/dev/null | awk '/interface:/ {print \$2}')\" = '$TB' ]"; then
  ok "100.92.5.5 (a tailnet machine inside NetBird's /16) via $TB; 100.92.9.9 via $(iface_of 100.92.9.9)"
else
  bad "100.92.5.5 via $(iface_of 100.92.5.5), want $TB; coexist $(cxj | jq -c .)"
fi
[ "$(iface_of 100.92.9.9)" = "$WG" ] && ok "the rest of 100.92.0.0/16 stays with $WG" || bad "100.92.9.9 via $(iface_of 100.92.9.9)"
netstat -rn -f inet | awk -v t="$TB" '$NF == t {print $1}' | grep -Eq '^(100\.92/16|100\.64/10|100/8)$' \
  && bad "a covering route on $TB: $(netstat -rn -f inet | awk -v t="$TB" '$NF == t {print $1}' | tr '\n' ' ')" \
  || ok "nothing on $TB covers 100.92/16: $(netstat -rn -f inet | awk -v t="$TB" '$NF == t {print $1}' | tr '\n' ' ')"
cxj | jq -e --arg w "$WG" '.foreign | any(.iface == $w and .kind == "netbird")' >/dev/null && cxj | jq -e '.split_dns and .work_vpn == "direct"' >/dev/null \
  && ok "API: NetBird on $WG, split DNS, Work VPN traffic Direct" || bad "API: $(cxj | jq -c .)"

say "other VPNs: DNS split, never global"
wait_for 30 "resolver file" test -f /etc/resolver/ts.net && ok "/etc/resolver/ts.net: $(tr '\n' ' ' </etc/resolver/ts.net)" || bad "no /etc/resolver/ts.net"
SD="$(scutil --dns)"
grep -A3 'domain   : ts.net' <<<"$SD" | grep -q 'nameserver\[0\] : 10.77.0.1' && ok "scutil --dns: ts.net → 10.77.0.1" || bad "scutil --dns has no ts.net → 10.77.0.1"
grep -A4 'domain   : corp.example' <<<"$SD" | grep -q '127.0.0.1' && ok "scutil --dns: corp.example → the other VPN's resolver, untouched" || bad "corp.example resolver gone"
awk '/^resolver #1/,/^$/' <<<"$SD" | grep -q '10.77.0.1' && bad "the global resolver points at tailbond: $(awk '/^resolver #1/,/^$/' <<<"$SD" | tr '\n' ' ')" \
  || ok "the global resolver is the Mac's own: $(awk '/^resolver #1/,/^$/' <<<"$SD" | grep nameserver | head -n1 | xargs)"
dscacheutil -q host -a name app.corp.example | grep -q 'ip_address: 10.92.0.5' && ok "app.corp.example resolves via the other VPN's DNS (10.92.0.5)" \
  || bad "corp.example: $(dscacheutil -q host -a name app.corp.example | tr '\n' ' '); stub: $(tail -n 2 "$WORK/dnsstub.log" | tr '\n' ' ')"
tcpdump -n -l -i "$TB" -c 1 udp port 53 >"$WORK/dns.pcap.txt" 2>/dev/null &
TD=$!
sleep 1
dscacheutil -q host -a name build.tail1234.ts.net >/dev/null 2>&1 || true
sleep 3
kill "$TD" 2>/dev/null || true
grep -q '10.77.0.1.53' "$WORK/dns.pcap.txt" && ok "*.ts.net queries go to the tunnel's DNS: $(head -n1 "$WORK/dns.pcap.txt" | cut -c1-120)" \
  || bad "no ts.net query on $TB: $(cat "$WORK/dns.pcap.txt")"
grep -Eq '^DNS *=' "$WORK/hub/client.conf" && bad "mqvpn got DNS servers (global DNS)" || ok "mqvpn runs without DNS servers"

say "other VPNs: Work VPN traffic Direct (host routes via en0) and Bonded (the MTU rule)"
# stand-ins for tailbond's catch-all, for the TEST-NET ranges NetBird's endpoints are in
for n in 203.0.113.0/24 198.51.100.0/24 192.0.2.0/24; do route -n add -net "$n" -interface "$TB" >/dev/null 2>&1; done
UP="$(netstat -rn -f inet | awk '$1 == "default" && $NF !~ /^(utun|ppp|ipsec)/ {print $NF; exit}')"
if wait_for 45 "NetBird's endpoints direct" sh -c "[ \"\$(route -n get 203.0.113.50 | awk '/interface:/ {print \$2}')\" = '$UP' ]"; then
  ok "Direct: NetBird's endpoints via $UP (203.0.113.50 $(iface_of 203.0.113.50), 198.51.100.77 $(iface_of 198.51.100.77), 192.0.2.10 $(iface_of 192.0.2.10)); 203.0.113.99 via $(iface_of 203.0.113.99)"
else
  bad "203.0.113.50 via $(iface_of 203.0.113.50), want $UP; $(cxj | jq -c '{direct_routes, warnings, error}')"
fi
hapi -X POST -d '{"work_vpn": "bonded"}' "http://$API/api/coexist" >/dev/null
if wait_for 90 "Bonded: MTU raised" sh -c "curl -fsS -m 2 http://$API/api/coexist | jq -e '.work_vpn_effective == \"bonded\" and .want_mtu == 1360'" \
   && wait_for 60 "reconnected" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'"; then
  TB="$(tbtun)"
  ifconfig "$TB" | grep -Eq 'mtu 13(52|60)' && ok "Bonded: $TB $(ifconfig "$TB" | grep -o 'mtu [0-9]*') for $WG's 1280 + 80" || bad "Bonded: $TB $(ifconfig "$TB" | grep -o 'mtu [0-9]*')"
  wait_for 20 "no host route" sh -c "! netstat -rn -f inet | grep -q '^203.0.113.50 '" && ok "Bonded: NetBird's endpoints have no host route via $UP" || bad "host route still there"
else
  bad "Bonded: $(cxj | jq -c '{work_vpn_effective, want_mtu, warnings}')"
fi
ifconfig "$WG" mtu 1420
if wait_for 60 "Bonded can't fit" sh -c "curl -fsS -m 2 http://$API/api/coexist | jq -e '.work_vpn_effective == \"direct\" and (.warnings | length) > 0'"; then
  ok "$WG at 1420 needs 1500: back to Direct, warned: $(cxj | jq -r '.warnings[0]' | cut -c1-100)"
else
  bad "no fallback: $(cxj | jq -c '{work_vpn_effective, want_mtu, warnings}')"
fi
hapi -X POST -d '{"work_vpn": "direct"}' "http://$API/api/coexist" >/dev/null
ifconfig "$WG" mtu 1280

say "other VPNs: disconnect, and a killed agent, take routes and resolver files away"
wait_for 60 "connected" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" || true
wait_for 30 "resolver back" test -f /etc/resolver/ts.net || true
hapi -X POST -d '{}' "http://$API/api/disconnect" >/dev/null
if wait_for 30 "resolver gone" sh -c '! test -f /etc/resolver/ts.net'; then
  ok "disconnect: /etc/resolver/ts.net and the host routes gone ($(netstat -rn -f inet | grep -c '^203.0.113.50 ') left); corp.example kept: $(test -f /etc/resolver/corp.example && echo yes)"
else
  bad "after disconnect: $(ls /etc/resolver)"
fi
hapi -X POST -d '{}' "http://$API/api/connect" >/dev/null
wait_for 90 "connected again" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" || true
wait_for 45 "resolver back" test -f /etc/resolver/ts.net && ok "connected again: split DNS back" || bad "split DNS not back"
kill -9 "$HUB_PID"; wait "$HUB_PID" 2>/dev/null || true
pkill -9 -f "$WORK/hub/client.conf" 2>/dev/null || true
test -f /etc/resolver/ts.net && ok "after kill -9 the resolver file is still there (nobody to take it away yet)" || bad "resolver file gone before the restart?"
jq '.client.disconnected = true' "$WORK/hub/tailbond.json" >"$WORK/hub/tb.tmp" && mv "$WORK/hub/tb.tmp" "$WORK/hub/tailbond.json"
N0="$(grep -c 'took away the routes and resolver files' "$WORK/hub.log")"
start_hub
# a new line (the log is appended: earlier stops logged it too)
if wait_for 30 "restored at start" sh -c "[ \$(grep -c 'took away the routes and resolver files' '$WORK/hub.log') -gt $N0 ]"; then
  ! test -f /etc/resolver/ts.net && ! netstat -rn -f inet | grep -q '^203.0.113.50 ' \
    && ok "restart after kill -9 took the resolver file and host routes away" || bad "left after restart: $(ls /etc/resolver) $(netstat -rn -f inet | grep '^203.0.113.50 ')"
else
  bad "no restore at start: $(tail -n 5 "$WORK/hub.log" | tr '\n' ' ')"
fi

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
