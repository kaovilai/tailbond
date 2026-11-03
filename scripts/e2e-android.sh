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
# GET through the tunnel with toybox nc (no curl/wget on Android)
fetch_token() {
  sh_ "printf 'GET $TOKEN_URL_PATH HTTP/1.0\\r\\nHost: 10.77.0.1\\r\\n\\r\\n' | nc -w 5 10.77.0.1 8099" 2>/dev/null
}
token_ok() { fetch_token | grep -q "$TOKEN"; }
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
# Wi-Fi (wlan0) and cellular both up before the app starts
wait_for 60 sh -c "adb shell dumpsys connectivity | grep -q 'ni{WIFI CONNECTED'" && ok "Wi-Fi connected" || note "Wi-Fi not connected after 60 s"
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
dump_net connected
grep -qE 'ni\{VPN CONNECTED|Transports: VPN' "$OUT/connectivity-connected.txt" \
  && ok "dumpsys connectivity: a VPN network is up" || bad "no VPN network in dumpsys connectivity"
TUN="$(sh_ ip -o -4 addr show | awk '$4 ~ /^10\.77\.0\./ {print $2; exit}')"
if [ -n "$TUN" ]; then
  ok "$TUN has $(sh_ ip -o -4 addr show dev "$TUN" | awk '{print $4}')"
  # (/sys/class/net is closed to adb's shell by SELinux)
  MTU="$(sh_ ip -o link show dev "$TUN" | sed -n 's/.* mtu \([0-9]*\).*/\1/p')"
  # 1280 (the client default, like tailbondd's client.mtu), less the 8-byte
  # reorder header when the mode negotiates reordering (Speed)
  if [ "$MTU" = 1280 ] || [ "$MTU" = 1272 ]; then ok "$TUN MTU $MTU (1280 cap)"; else bad "$TUN MTU is $MTU, not 1280 (1272 with reorder)"; fi
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
[ "$(srv_client | jq -r .kind)" = android ] && ok "server saw the app's token requests (kind android, version $(srv_client | jq -r .version))" \
  || bad "server has no kind=android for $CLIENT: $(srv_client | jq -c '{kind, version}')"

say "connected home screen"
sh_ am start -n "$PKG/.ui.MainActivity" >/dev/null
sleep 3
shot home-connected

say "multipath: Wi-Fi + cellular"
TWO=false
if wait_for 60 srv_paths_ge 2; then
  TWO=true
  ok "server sees 2 active paths: $(srv_client | jq -c '[.paths[] | {state, srtt_ms}]'); app: $(app_status | jq -c '[.paths[] | {name, kind, status}]')"
else
  dump_net multipath
  note "server sees $(srv_active_paths) active path(s), not 2. App paths: $(app_status | jq -c '[.paths[] | {name, kind, status}]'). Networks: $(grep -oE 'ni\{[A-Z_]+ [A-Z_]+' "$OUT/connectivity-multipath.txt" | sort -u | tr '\n' ' ')"
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
  app_is '.state == "CONNECTED"' && ok "app still connected: $(app_status | jq -c '[.paths[] | {name, kind, status}]')" \
    || bad "app not connected with Wi-Fi off: $(app_status | jq -c '{state, message}')"
  shot wifi-off
  sh_ svc wifi enable
else
  note "failover not tested: only one path came up"
fi

[ "$FAIL" -eq 0 ]
