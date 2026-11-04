#!/usr/bin/env bash
# Android emulator end-to-end test (CI job "android-e2e"): the real app on an
# x86_64 emulator, bonded to a tailbond server on the host
# (scripts/e2e-android-server.sh, reached as 10.0.2.2).
#
#   scripts/e2e-android-server.sh          # first, on the host
#   APK=out/android/tailbond-debug-x86_64.apk scripts/e2e-android.sh
#
# Pairs through the tailbond://pair deep link and the confirm dialog's "Add
# phone", connects through the app's own connect action (the one the quick
# settings tile uses), with VPN consent granted up front (appops
# ACTIVATE_VPN). Then checks over adb: a VPN network and the tun address,
# MTU, ping 10.77.0.1, the token web server behind the tunnel, the server's
# /api/status (the phone with its paths, kind android), the app's machine
# list (GET /api/peers), both paths (Wi-Fi + cellular) and Wi-Fi off →
# traffic on cellular. The app's state comes from the debug-only status
# receiver (android/app/src/debug).
#
# Phone hub (docs/phone-hub.md): "Share bonded connection" listens on 1888;
# from the host, through adb forward (a laptop on the phone's hotspot
# stands in), SOCKS5 and HTTP CONNECT requests with the password derived
# from the share key reach the tunnel-only token server, which logs the
# phone's tunnel address as the client; a wrong password is refused.
#
# Pair & Share hub on the phone: a sharer on the host (tb-sharesim, started
# by e2e-android-server.sh with SHARESIM; 10.0.2.2:47077 from the emulator)
# becomes a relay path of the app's tunnel (patches/mqvpn/0002 + 0004): the
# link gets ready, mqvpn carries the path, the sharer relays bytes, and
# when it stops the path goes and the tunnel keeps working.
#
# Artifacts in $OUT (default out/e2e-android): screenshots, logcat, UI dumps,
# dumpsys connectivity, server status and log.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${TB_E2E_DIR:-/tmp/tailbond-e2e-android}"
OUT="${OUT:-$ROOT/out/e2e-android}"
APK="${APK:-$ROOT/out/android/tailbond-debug-x86_64.apk}"
PKG=dev.tailbond.app
API=127.0.0.1:8077
CLIENT=pixel-e2e
TOKEN_URL_PATH=/token.txt

PASS=0 FAIL=0 NOTE=0
say()  { printf '\n==> %s\n' "$*"; }
ok()   { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  FAIL %s\n' "$*"; echo "::error::$*"; FAIL=$((FAIL + 1)); }
note() { printf '  NOTE %s\n' "$*"; echo "::notice::$*"; NOTE=$((NOTE + 1)); }
api()  { curl -fsS -m 5 "$@"; }
sh_()  { adb shell "$@" | tr -d '\r'; }
wait_for() { # wait_for SECONDS cmd...
  local t=$1; shift
  for _ in $(seq "$t"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}

mkdir -p "$OUT"
[ -s "$WORK/pair-uri" ] || { echo "no $WORK/pair-uri: run scripts/e2e-android-server.sh first"; exit 2; }
[ -f "$APK" ] || { echo "APK not found: $APK"; exit 2; }
URI="$(cat "$WORK/pair-uri")"
TOKEN="$(cat "$WORK/www/token.txt")"

# The app's state (debug/E2eReceiver.kt) as JSON.
app_status() {
  sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.STATUS \
    | sed -n 's/^Broadcast completed: result=[-0-9]*, data="\(.*\)"$/\1/p'
}
app_is() { app_status | jq -e "$1" >/dev/null; }
srv_status() { api "http://$API/api/status"; }
srv_client() { srv_status | jq -c --arg n "$CLIENT" '.clients[] | select(.name == $n)'; }
srv_active_paths() { srv_status | jq --arg n "$CLIENT" '[.clients[] | select(.name == $n and .connected) | .paths[] | select(.state == "active")] | length'; }
srv_paths_ge() { [ "$(srv_active_paths)" -ge "$1" ]; }
# Both of the phone's networks carry an active path (mqvpn status 1). The
# server's count alone can't tell: it also lists the handshake path (id 0).
app_two_paths() { app_is '[.paths[] | select(.status == 1) | .kind] | (index("WIFI") != null and index("CELLULAR") != null)'; }
# GET through the tunnel with toybox nc (no curl/wget on Android)
fetch_token() {
  sh_ "printf 'GET $TOKEN_URL_PATH HTTP/1.0\\r\\nHost: 10.77.0.1\\r\\n\\r\\n' | nc -w 5 10.77.0.1 8099" 2>/dev/null
}
token_ok() { fetch_token | grep "$TOKEN" >/dev/null; }
# The emulator's Wi-Fi as dumpsys connectivity lists it (one NetworkAgentInfo
# line per network). Validated = Android's own DNS + HTTP probes went through
# it, i.e. wlan0 carries traffic; connected alone doesn't say that.
wifi_nai() { sh_ dumpsys connectivity | grep -E '^ *NetworkAgentInfo\{.*ni\{WIFI '; }
wifi_connected() { wifi_nai | grep -q 'ni{WIFI CONNECTED'; }
wifi_validated() { wifi_nai | grep -Eq 'Capabilities: ([A-Z_]+&)*VALIDATED[& ]'; }
wifi_gone() { ! wifi_nai >/dev/null; }
# why not: its state, its first DNS config and last validation (logcat)
wifi_diag() {
  local nai id
  nai="$(wifi_nai | head -n 1)"
  [ -n "$nai" ] || { echo "no Wi-Fi network"; return; }
  id="$(grep -oE 'network\{[0-9]+\}' <<<"$nai" | tr -dc 0-9)"
  printf '%s, network %s; first DNS config %s; last %s\n' \
    "$(grep -oE 'ni\{WIFI [A-Z_]+' <<<"$nai" | cut -c4-)" "$id" \
    "$(grep -E "ResolverParamsParcel\{netId: $id," "$OUT/logcat.txt" | head -n 1 | grep -oE 'servers: \[[^]]*\]' || echo '?')" \
    "$(grep -oE "\[$id WIFI\] validation [a-z]+" "$OUT/logcat.txt" | tail -n 1 | cut -d' ' -f3- || echo 'validation ?')"
}
shot() { adb exec-out screencap -p >"$OUT/$1.png" 2>/dev/null && echo "  screenshot $OUT/$1.png"; }
# Tap the first view whose text is $1 (uiautomator dump; Compose exposes its texts).
tap_text() {
  local want=$1 i xy
  for i in $(seq 10); do
    sh_ uiautomator dump /sdcard/ui.xml >/dev/null 2>&1
    adb exec-out cat /sdcard/ui.xml >"$OUT/ui-$i.xml" 2>/dev/null
    xy="$(python3 -I - "$OUT/ui-$i.xml" "$want" <<'PY'
import re, sys, xml.etree.ElementTree as ET
try:
    root = ET.parse(sys.argv[1]).getroot()
except Exception:
    sys.exit(1)
for n in root.iter("node"):
    if n.get("text") == sys.argv[2] or n.get("content-desc") == sys.argv[2]:
        x1, y1, x2, y2 = map(int, re.findall(r"\d+", n.get("bounds", "")))
        print((x1 + x2) // 2, (y1 + y2) // 2)
        sys.exit(0)
sys.exit(1)
PY
)" && { echo "  tap \"$want\" at $xy"; sh_ input tap $xy; return 0; }
    sleep 2
  done
  return 1
}
dump_net() { sh_ dumpsys connectivity >"$OUT/connectivity-$1.txt" 2>&1; sh_ ip addr >"$OUT/ip-addr-$1.txt" 2>&1; }

adb logcat -c 2>/dev/null || true
adb logcat -v threadtime >"$OUT/logcat.txt" 2>&1 &
LOGCAT_PID=$!
finish() {
  shot final
  srv_status >"$OUT/server-status.json" 2>/dev/null || true
  app_status >"$OUT/app-status.json" 2>/dev/null || true
  cp "$WORK/server.log" "$OUT/server.log" 2>/dev/null || true
  kill "$LOGCAT_PID" 2>/dev/null || true
  printf '\n%d passed, %d failed, %d notes\n' "$PASS" "$FAIL" "$NOTE"
}
trap finish EXIT

say "emulator: Android $(sh_ getprop ro.build.version.release) (API $(sh_ getprop ro.build.version.sdk)), $(sh_ getprop ro.product.cpu.abi)"
sh_ svc wifi enable; sh_ svc data enable
# Wi-Fi (wlan0) and cellular both up before the app starts, and Wi-Fi
# validated, not just connected: the Wi-Fi path, the Pair & Share hub's link
# (10.0.2.2 is on wlan0's subnet, so its socket is bound to Wi-Fi) and the
# failover all need wlan0 to carry traffic. Now and then the emulator brings
# Wi-Fi up with its address before DHCP and no DNS server
# (setResolverConfiguration(netId, servers: [])); that network never carries
# traffic (every probe fails for as long as it lives), so it's brought down
# and up again until a bring-up validates. A good one validates within ~2 s.
WIFI_OK=false
for try in 1 2 3; do
  if wait_for 60 wifi_connected && wait_for 20 wifi_validated; then WIFI_OK=true; break; fi
  dump_net "wifi-$try"
  note "Wi-Fi bring-up $try doesn't carry traffic: $(wifi_diag)"
  [ "$try" -lt 3 ] || break
  sh_ svc wifi disable
  wait_for 20 wifi_gone || note "Wi-Fi still listed 20 s after svc wifi disable"
  sh_ svc wifi enable
done
if $WIFI_OK; then ok "Wi-Fi connected and validated (bring-up $try)"
else note "Wi-Fi not validated after 3 bring-ups: no Wi-Fi path or failover; the Pair & Share hub fails below"; fi
dump_net before
grep -E '^ *NetworkAgentInfo\{' "$OUT/connectivity-before.txt" | sed -E 's/^ *(NetworkAgentInfo\{[^ ]+ +[^ ]+ +ni\{[^}]*\}).*/  \1/' | head -n 5

say "install $(basename "$APK") ($(du -h "$APK" | cut -f1))"
adb install -r -g "$APK" >/dev/null && ok "installed" || { bad "adb install failed"; exit 1; }
sh_ appops set "$PKG" ACTIVATE_VPN allow
[ "$(sh_ appops get "$PKG" ACTIVATE_VPN)" = "ACTIVATE_VPN: allow" ] && ok "VPN consent granted (appops ACTIVATE_VPN allow)" \
  || bad "appops ACTIVATE_VPN: $(sh_ appops get "$PKG" ACTIVATE_VPN)"

say "before the tunnel: the token server is out of reach"
if token_ok; then bad "token server answered without the tunnel"; else ok "10.77.0.1:8099 not reachable without the tunnel"; fi

say "pair through the deep link"
sh_ am start -W -a android.intent.action.VIEW -d "'$URI'" "$PKG" | grep -E 'Status|Activity' | sed 's/^/  /'
sleep 3
shot pair-dialog
if tap_text "Add phone"; then
  if wait_for 15 app_is '.servers == 1'; then ok "paired through the deep link's confirm dialog (Add phone)"
  else bad "tapped Add phone, but the app has no server"; fi
else
  bad "no \"Add phone\" button on screen (see pair-dialog.png, ui-*.xml)"
fi
if ! app_is '.servers == 1'; then
  echo "  falling back to the debug pairing hook so the rest still runs"
  sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.PAIR --es uri "'$URI'" | sed 's/^/  /'
fi
echo "  app: $(app_status | jq -c '{servers, server_names, state}')"

say "connect (the app's connect action, as the quick settings tile sends it)"
sh_ am start -n "$PKG/.ui.MainActivity" -a dev.tailbond.app.UI_CONNECT | sed 's/^/  /'
if wait_for 60 app_is '.state == "CONNECTED"'; then
  ok "app connected: $(app_status | jq -c '{state, tunnel_ip, paths: [.paths[] | {name, kind, status}]}')"
else
  bad "app not connected after 60 s: $(app_status | jq -c '{state, message, error_kind}')"
  shot not-connected
  exit 1
fi

say "VPN on the device"
# the app says connected a moment before Android lists the VPN network
# (grep reads everything: grep -q would end the pipe early, and pipefail
# would count dumpsys's SIGPIPE as a failure)
vpn_listed() { sh_ dumpsys connectivity | grep -E 'ni\{VPN CONNECTED|Transports: VPN' >/dev/null; }
wait_for 20 vpn_listed && ok "dumpsys connectivity: a VPN network is up" || bad "no VPN network in dumpsys connectivity"
dump_net connected
TUN="$(sh_ ip -o -4 addr show | awk '$4 ~ /^10\.77\.0\./ {print $2; exit}')"
if [ -n "$TUN" ]; then
  ok "$TUN has $(sh_ ip -o -4 addr show dev "$TUN" | awk '{print $4}')"
  # from the ip addr dump: SELinux closes /sys/class/net to adb's shell
  MTU="$(sed -n "s/^[0-9]*: $TUN: .* mtu \([0-9]*\) .*/\1/p" "$OUT/ip-addr-connected.txt")"
  # 1280 in every mode (like tailbondd's client.mtu, IPv6's minimum): in
  # Speed the cap covers mqvpn's 8-byte reorder header
  if [ "$MTU" = 1280 ]; then ok "$TUN MTU $MTU"; else bad "$TUN MTU is $MTU, not 1280"; fi
else
  bad "no interface with a 10.77.0.x address: $(sh_ ip -o -4 addr show | awk '{print $2, $4}' | tr '\n' ' ')"
fi

say "traffic through the tunnel"
if sh_ ping -c 3 -W 3 10.77.0.1 >"$OUT/ping.txt" 2>&1; then ok "ping 10.77.0.1: $(grep -E 'packets transmitted' "$OUT/ping.txt")"
else bad "ping 10.77.0.1 failed: $(tail -n 2 "$OUT/ping.txt" | tr '\n' ' ')"; fi
if wait_for 10 token_ok; then
  ok "token server behind the tunnel answered the token"
else
  bad "token server didn't answer through the tunnel: $(fetch_token | head -c 200)"
fi

say "server /api/status"
if wait_for 20 srv_paths_ge 1; then
  ok "server lists $CLIENT connected with $(srv_active_paths) active path(s): $(srv_client | jq -c '[.paths[] | {state, srtt_ms}]')"
else
  bad "server doesn't list $CLIENT with an active path: $(srv_client)"
fi

say "machine list (GET /api/peers with the pairing's token)"
if wait_for 45 app_is '.peers.server == "homebox" and .peers.fetched_at_ms > 0 and .peers.count == 2'; then
  ok "app fetched the machine list: $(app_status | jq -c '.peers | {server, count, names}')"
else
  bad "app has no machine list: $(app_status | jq -c .peers)"
fi
grep -q 'machine list from 10.77.0.1: 2 machines' "$OUT/logcat.txt" && ok "logcat: machine list from 10.77.0.1: 2 machines" \
  || bad "logcat has no successful machine list fetch: $(grep 'machine list' "$OUT/logcat.txt" | tail -n 2)"
# the server's status refreshes once a second: give it a few
srv_kind_android() { [ "$(srv_client | jq -r .kind)" = android ]; }
wait_for 10 srv_kind_android && ok "server saw the app's token requests (kind android, version $(srv_client | jq -r .version))" \
  || bad "server has no kind=android for $CLIENT: $(srv_client | jq -c '{kind, version}')"

say "connected home screen"
sh_ am start -n "$PKG/.ui.MainActivity" >/dev/null
sleep 3
shot home-connected

say "multipath: Wi-Fi + cellular"
TWO=false
if wait_for 60 app_two_paths && srv_paths_ge 2; then
  TWO=true
  ok "Wi-Fi and cellular both carry the tunnel; server: $(srv_client | jq -c '[.paths[] | {state, srtt_ms}]'); app: $(app_status | jq -c '[.paths[] | {name, kind, status}]')"
else
  dump_net multipath
  note "server sees $(srv_active_paths) active path(s), not 2. App paths: $(app_status | jq -c '[.paths[] | {name, kind, status}]'). Networks: $(grep -oE 'ni\{[A-Z_]+ [A-Z_]+' "$OUT/connectivity-multipath.txt" | sort -u | tr '\n' ' ')"
fi

say "Pair & Share: \"Share my mobile data\" comes up next to the VPN"
# The emulator can't host a hub on its Wi-Fi (its only other device is the
# emulator's own virtual router), so the relay itself is covered by unit
# tests and scripts/e2e-share.sh; here: the service starts, holds mobile
# data next to Wi-Fi, listens on UDP 47077 and has the share key.
sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.SHARE --ez on true >/dev/null
if wait_for 30 app_is '.share.status == "WAITING"'; then
  ok "sharing ready: $(app_status | jq -c '{share, share_key}')"
else
  bad "sharing not ready: $(app_status | jq -c '{share, share_key}')"
fi
# 47077 = 0xB7E5 in /proc/net/udp
sh_ cat /proc/net/udp /proc/net/udp6 2>/dev/null | grep -qi ':B7E5 ' && ok "the relay listens on UDP 47077" || bad "nothing on UDP 47077"
app_is '.state == "CONNECTED"' && ok "tailbond's own tunnel still connected while sharing" || bad "the tunnel dropped when sharing started"
shot sharing
sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.SHARE --ez on false >/dev/null
wait_for 10 app_is '.share.status == "OFF"' && ok "sharing stops" || bad "sharing didn't stop: $(app_status | jq -c .share)"

say "phone hub: \"Share bonded connection\" (SOCKS5 + HTTP CONNECT on 1888), upstream into the tunnel"
sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.HUB --ez on true >/dev/null
if wait_for 30 app_is '.hub.status == "READY"'; then
  ok "hub ready: $(app_status | jq -c '.hub | {status, up_bytes, down_bytes}')"
else
  bad "hub not ready: $(app_status | jq -c '.hub | del(.pass)')"
fi
# 1888 = 0x0760 in /proc/net/tcp6 (listening on "::")
# (informational: the SOCKS5 round trip below is the proof it listens)
sh_ cat /proc/net/tcp /proc/net/tcp6 2>/dev/null | grep -qi ':0760 .* 0A ' && ok "listening on TCP 1888" || note "TCP 1888 not seen in /proc/net/tcp*"
# the login of a laptop "e2e-laptop" holding its server's pass for this phone
# one status read: the user carries the login's expiry, the password is
# derived from it
HUBST="$(app_status)"
HUBUSER="$(jq -r '.hub.user // empty' <<<"$HUBST")"
HUBPASS="$(jq -r '.hub.pass // empty' <<<"$HUBST")"
TIP="$(app_status | jq -r '.tunnel_ip // empty' | cut -d/ -f1)"
adb forward tcp:18888 tcp:1888 >/dev/null
# the token server answers only through the tunnel, so getting the token
# through the hub means the hub's upstream went into the phone's tunnel
GOT="$(curl -fsS -m 20 --socks5-hostname 127.0.0.1:18888 -U "$HUBUSER:$HUBPASS" "http://10.77.0.1:8099$TOKEN_URL_PATH?via=phone-hub-socks" || true)"
[ "$GOT" = "$TOKEN" ] && ok "SOCKS5 through the phone hub reached the tunnel-only token server" || bad "SOCKS5 through the hub: got '${GOT:0:80}'"
GOT="$(curl -fsS -m 20 -p -x "http://${HUBUSER//\//%2F}:$HUBPASS@127.0.0.1:18888" "http://10.77.0.1:8099$TOKEN_URL_PATH?via=phone-hub-connect" || true)"
[ "$GOT" = "$TOKEN" ] && ok "HTTP CONNECT through the phone hub reached it too" || bad "HTTP CONNECT through the hub: got '${GOT:0:80}'"
if [ -n "$TIP" ] && grep -E "^$TIP .*GET $TOKEN_URL_PATH\?via=phone-hub-socks" "$WORK/www.log" >/dev/null; then
  ok "the token server saw the phone's tunnel address ($TIP) as the client: $(grep -E "via=phone-hub" "$WORK/www.log" | tail -n 1 | cut -c1-120)"
else
  bad "the token server didn't log the phone's tunnel address '$TIP': $(grep -E 'via=phone-hub' "$WORK/www.log" | tail -n 2 | tr '\n' ' ')"
fi
if curl -fsS -m 10 --socks5-hostname 127.0.0.1:18888 -U "$HUBUSER:wrong-password" "http://10.77.0.1:8099$TOKEN_URL_PATH" >/dev/null 2>&1; then
  bad "the hub let a wrong password through"
else
  ok "a wrong password is refused"
fi
if wait_for 10 app_is '.hub.up_bytes > 0 and .hub.down_bytes > 0 and (.hub.clients | length) >= 1'; then
  ok "hub counts the laptop's traffic: $(app_status | jq -c '.hub | {status, up_bytes, down_bytes, clients}')"
else
  bad "hub counters: $(app_status | jq -c '.hub | del(.pass)')"
fi
grep -q 'hub: up [1-9]' "$OUT/logcat.txt" && ok "logcat: $(grep -o 'hub: up [0-9]* down [0-9]*.*' "$OUT/logcat.txt" | tail -n 1 | cut -c1-140)" || note "no hub counter line in logcat yet"
shot hub-sharing
adb forward --remove tcp:18888 >/dev/null 2>&1 || true
sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.HUB --ez on false >/dev/null
wait_for 10 app_is '.hub.status == "OFF"' && ok "hub stops" || bad "hub didn't stop: $(app_status | jq -c '.hub | del(.pass)')"

say "Pair & Share hub: a nearby phone's shared data (tb-sharesim on the host) as a relay path of the app's tunnel"
# the link's socket is bound to Wi-Fi (10.0.2.2 is on wlan0's subnet): with
# a Wi-Fi that carries nothing it can only end up "unreachable"
if [ -s "$WORK/sharer.pid" ] && ! wifi_validated; then
  bad "Pair & Share hub not tested: the emulator's Wi-Fi, the link's network to 10.0.2.2, carries no traffic: $(wifi_diag)"
elif [ -s "$WORK/sharer.pid" ]; then
  # its key id: the app links it with the pass the server gave it for that
  SKID="$(grep -o 'key id [0-9a-f]*' "$WORK/sharer.log" | head -n1 | awk '{print $3}')"
  sh_ am broadcast -f 32 -n "$PKG/.debug.E2eReceiver" -a dev.tailbond.app.debug.SHARE_HUB_ADD --es addr 10.0.2.2:47077 --es keyid "$SKID" >/dev/null
  if wait_for 45 app_is '[.hub_shares[]? | select(.state == "ready")] | length == 1'; then
    ok "link to the sharer ready: $(app_status | jq -c '[.hub_shares[] | {label, addr, via, state, path}]')"
  else
    bad "link not ready: $(app_status | jq -c '.hub_shares')"
  fi
  RP="$(app_status | jq -r '.hub_shares[0].path // empty')"
  if [ -n "$RP" ] && wait_for 60 app_is "[.paths[] | select(.name == \"$RP\" and .status == 1)] | length == 1"; then
    ok "mqvpn carries the relay path $RP: $(app_status | jq -c '[.paths[] | {name, kind, status}]')"
  else
    bad "no active relay path ${RP:-?}: $(app_status | jq -c '[.paths[] | {name, kind, status}]')"
  fi
  grep -q "relay path" "$OUT/logcat.txt" && ok "logcat: $(grep -oE 'Relay path added: [^ ]+ → 127.0.0.1:[0-9]+|android-reactor: path [^ ]+: relay path' "$OUT/logcat.txt" | head -n 2 | tr '\n' ';')" \
    || bad "no relay path in logcat"
  wait_for 30 srv_paths_ge 2 && ok "server: $CLIENT has $(srv_active_paths) active paths: $(srv_client | jq -c '[.paths[] | {state, srtt_ms}]')" \
    || bad "server paths: $(srv_client | jq -c '[.paths[] | {state}]')"
  for _ in 1 2 3 4 5; do token_ok; done
  if wait_for 20 sh -c "grep -Eq 'hub [^ ]+ at [^ ]+: up [1-9][0-9]* down [1-9]' '$WORK/sharer.log'"; then
    ok "the sharer relayed the phone's traffic: $(grep -E 'hub [^ ]+ at' "$WORK/sharer.log" | tail -n 1 | sed 's/^.*: hub /hub /')"
  else
    bad "the sharer relayed nothing: $(tail -n 3 "$WORK/sharer.log" | tr '\n' ' ')"
  fi
  wait_for 10 app_is '.hub_shares[0].up > 0 and .hub_shares[0].down > 0' && ok "the app counts it: $(app_status | jq -c '.hub_shares[0] | {up, down}')" \
    || bad "app counters: $(app_status | jq -c '.hub_shares')"
  shot share-hub
  say "Pair & Share hub: the sharer stops; its path goes, the tunnel keeps working"
  sudo kill "$(cat "$WORK/sharer.pid")" 2>/dev/null || true
  if wait_for 20 app_is "([.hub_shares[]? | select(.state == \"ready\")] | length == 0) and ([.paths[] | select(.name == \"$RP\" and .status == 1)] | length == 0)"; then
    ok "relay path gone: $(app_status | jq -c '{hub_shares: [.hub_shares[] | {state, reason}], paths: [.paths[] | {name, status}]}')"
  else
    bad "relay path still there: $(app_status | jq -c '{hub_shares, paths: [.paths[] | {name, status}]}')"
  fi
  wait_for 20 token_ok && ok "tunnel still works without the sharer" || bad "tunnel broken after the sharer left"
else
  note "no sharer on the host (e2e-android-server.sh without SHARESIM): the Pair & Share hub isn't tested"
fi

say "failover: Wi-Fi off"
if $TWO; then
  sh_ svc wifi disable
  if wait_for 30 token_ok; then
    ok "token server still answers with Wi-Fi off"
  else
    bad "no traffic through the tunnel with Wi-Fi off"
  fi
  if sh_ ping -c 3 -W 3 10.77.0.1 >"$OUT/ping-cellular.txt" 2>&1; then ok "ping 10.77.0.1 over cellular: $(grep -E 'packets transmitted' "$OUT/ping-cellular.txt")"
  else bad "ping 10.77.0.1 with Wi-Fi off failed"; fi
  app_is '.state == "CONNECTED" and ([.paths[] | select(.kind == "CELLULAR" and .status == 1)] | length) == 1' && ok "app still connected on cellular: $(app_status | jq -c '[.paths[] | {name, kind, status}]')" \
    || bad "app not connected with Wi-Fi off: $(app_status | jq -c '{state, message}')"
  shot wifi-off
  sh_ svc wifi enable
else
  note "failover not tested: only one path came up"
fi

[ "$FAIL" -eq 0 ]
