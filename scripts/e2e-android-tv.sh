#!/usr/bin/env bash
# Android TV check (CI job "android-tv", docs/android.md "Android TV"): the
# debug APK on an Android TV emulator, driven only with the remote's keys.
#
#   APK=out/android/tailbond-debug-x86_64.apk scripts/e2e-android-tv.sh
#
# Checks, over adb:
#   - the APK: a TV banner, a LEANBACK_LAUNCHER activity, no required
#     hardware feature (aapt2 dump badging);
#   - the TV's launcher lists it (LEANBACK_LAUNCHER resolves), and it starts
#     from there;
#   - every step by remote: initial focus on Get started, OK → pairing with
#     the server's name field focused and no camera scan offered, D-pad out
#     of the field (a text field doesn't trap the remote), a pairing link →
#     the confirm dialog on Add phone, OK → home on the main switch, up to
#     Settings, down through it to Sharing, Back home, and the switch → the
#     handoff dialog → Android's own VPN dialog (left with Back);
#   - Android's VPN dialog and VPN settings screen exist on this TV image.
# The keys go through the emulator's own key device (adb emu), like a
# remote's; `input keyevent` comes from a virtual device, which text fields
# treat as a keyboard (its arrows move the cursor).
#
# Artifacts in $OUT (default out/e2e-android-tv): a screenshot per step,
# UI dumps, logcat.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-$ROOT/out/e2e-android-tv}"
APK="${APK:-$ROOT/out/android/tailbond-debug-x86_64.apk}"
PKG=dev.tailbond.app

PASS=0 FAIL=0 NOTE=0
say()  { printf '\n==> %s\n' "$*"; }
ok()   { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  FAIL %s\n' "$*"; echo "::error::$*"; FAIL=$((FAIL + 1)); }
note() { printf '  NOTE %s\n' "$*"; echo "::notice::$*"; NOTE=$((NOTE + 1)); }
sh_()  { adb shell "$@" | tr -d '\r'; }
shot() { adb exec-out screencap -p >"$OUT/$1.png" 2>/dev/null && echo "  screenshot $OUT/$1.png"; }

mkdir -p "$OUT"
[ -f "$APK" ] || { echo "APK not found: $APK"; exit 2; }

# The remote: Linux key codes into the emulator's keyboard device (arrows
# are DPAD_*, Enter is OK for Compose and Android's dialogs alike, Back);
# input keyevent if the console isn't there.
declare -A LINUX=([UP]=KEY_UP [DOWN]=KEY_DOWN [LEFT]=KEY_LEFT [RIGHT]=KEY_RIGHT [CENTER]=KEY_ENTER [BACK]=KEY_BACK)
declare -A ANDROID=([UP]=DPAD_UP [DOWN]=DPAD_DOWN [LEFT]=DPAD_LEFT [RIGHT]=DPAD_RIGHT [CENTER]=DPAD_CENTER [BACK]=BACK)
USE_EMU=1
key() {
  local k=$1 n=${2:-1}
  for _ in $(seq "$n"); do
    if [ "$USE_EMU" = 1 ] && r="$(adb emu event send "EV_KEY:${LINUX[$k]}:1" "EV_KEY:${LINUX[$k]}:0" 2>&1)" && ! grep -q KO <<<"$r"; then :
    else
      [ "$USE_EMU" = 1 ] && note "emulator console keys unavailable ($r): input keyevent instead (text fields then keep the arrows)"
      USE_EMU=0
      sh_ input dpad keyevent "${ANDROID[$k]}" >/dev/null
    fi
    sleep 0.7
  done
}

# The UI as uiautomator sees it (Compose exposes its semantics).
dump() { sh_ uiautomator dump /sdcard/ui.xml >/dev/null 2>&1; adb exec-out cat /sdcard/ui.xml 2>/dev/null >"$OUT/ui.xml"; }
# The focused node: its text, content description and its children's texts.
focused() {
  dump
  python3 - "$OUT/ui.xml" <<'PY'
import sys, xml.etree.ElementTree as ET
try:
    root = ET.parse(sys.argv[1]).getroot()
except Exception:
    sys.exit(0)
for n in root.iter("node"):
    if n.get("focused") == "true":
        words = [m.get("text") or m.get("content-desc") or "" for m in n.iter("node")]
        print(n.get("class", ""), "|", " / ".join(w for w in words if w))
        break
PY
}
has_text() { dump; grep -q "text=\"[^\"]*$1" "$OUT/ui.xml"; }
# focus_is PATTERN STEP: the focused node matches (grep -E), else FAIL with what has focus
focus_is() {
  local f; f="$(focused)"
  if grep -qE "$1" <<<"$f"; then ok "$2 (focus: $f)"; return 0; fi
  bad "$2: focus is on \"$f\""; return 1
}
# move KEY PATTERN MAX STEP: press KEY until the focus matches, at most MAX times
move() {
  local k=$1 want=$2 max=$3 step=$4 f
  for _ in $(seq "$max"); do
    f="$(focused)"
    grep -qE "$want" <<<"$f" && { ok "$step (focus: $f)"; return 0; }
    key "$k"
  done
  f="$(focused)"
  grep -qE "$want" <<<"$f" && { ok "$step (focus: $f)"; return 0; }
  bad "$step: after $max × $k focus is on \"$f\""; return 1
}

say "APK: TV banner, leanback launcher, no required hardware"
AAPT2="$(ls -d "${ANDROID_HOME:-$HOME/Android/Sdk}"/build-tools/*/aapt2 2>/dev/null | sort -V | tail -n1)"
if [ -n "$AAPT2" ]; then
  "$AAPT2" dump badging "$APK" >"$OUT/badging.txt" 2>&1
  grep -q "banner=" "$OUT/badging.txt" && ok "application banner" || bad "no android:banner in the APK"
  grep -q "^leanback-launchable-activity: name='$PKG.ui.MainActivity'" "$OUT/badging.txt" && ok "leanback-launchable-activity" || bad "no leanback-launchable-activity"
  # a required feature (declared or implied) keeps the app off TVs
  req="$(grep -E "^ *uses-feature: name=" "$OUT/badging.txt" || true)"
  [ -z "$req" ] && ok "no required feature" || bad "required features: $req"
  grep -E "^ *uses-implied-feature" "$OUT/badging.txt" | sed 's/^/  /'
else
  note "no aapt2: APK badging not checked"
fi

say "Install and the TV launcher"
adb install -r "$APK" | tail -n1
sh_ getprop ro.build.characteristics | grep -q tv && ok "a TV image ($(sh_ getprop ro.build.version.release), $(sh_ wm size | tail -n1))" || note "not a TV image? ro.build.characteristics=$(sh_ getprop ro.build.characteristics)"
sh_ cmd package query-activities --brief -a android.intent.action.MAIN -c android.intent.category.LEANBACK_LAUNCHER >"$OUT/leanback.txt"
grep -q "$PKG/.ui.MainActivity" "$OUT/leanback.txt" && ok "the TV launcher lists tailbond" || bad "LEANBACK_LAUNCHER doesn't resolve to tailbond"
# Android's VPN dialog (VpnService.prepare) and VPN settings on this TV
sh_ pm list packages | grep -q "com.android.vpndialogs" && ok "Android's VPN permission dialog is installed" || note "no com.android.vpndialogs: tailbond says it can't turn on here"
vs="$(sh_ cmd package resolve-activity --brief -a android.settings.VPN_SETTINGS | tail -n1)"
if [ -n "$vs" ] && ! grep -q "No activity" <<<"$vs"; then note "VPN settings screen: $vs"; else note "no VPN settings screen: Settings shows the adb way to set Always-on"; fi

say "Start it like the TV launcher does"
adb logcat -c
sh_ monkey -p "$PKG" -c android.intent.category.LEANBACK_LAUNCHER 1 >/dev/null 2>&1
sleep 5
shot tv-01-onboarding
focus_is "Get started" "first screen starts on Get started"

say "Pairing by remote"
key CENTER
sleep 2
shot tv-02-pairing
focus_is "EditText|Home server's name" "pairing starts on the server's name field"
has_text "Scan code" && bad "a camera scan is offered on a TV with no camera" || ok "no camera scan offered"
key DOWN
f="$(focused)"
if grep -q "Home server's name" <<<"$f"; then bad "D-pad down stays in the text field (focus: $f)"; else ok "D-pad leaves the text field (focus: $f)"; fi
shot tv-03-pairing-moved

say "A pairing link (no server: only the screens are checked)"
D="$(printf '%s' '{"v":1,"name":"living-room-tv","server":"10.0.2.2:443","key":"tv-check-not-a-real-key","server_name":"homebox","dns":["10.77.0.1"]}' | base64 -w0 | tr '+/' '-_' | tr -d '=')"
sh_ am start -W -a android.intent.action.VIEW -d "'tailbond://pair?d=$D'" "$PKG" | grep -E 'Status' | sed 's/^/  /'
sleep 2
shot tv-04-confirm
focus_is "Add phone" "the confirm dialog starts on Add phone"
key CENTER
sleep 3
shot tv-05-home
focus_is "Switch [|] ?$" "home starts on the main switch"

say "Settings, Sharing and back"
key UP
focus_is "Settings" "up from the switch: Settings"
key CENTER
sleep 2
has_text "Internet traffic" && ok "Settings open" || bad "Settings didn't open"
shot tv-06-settings
move DOWN "Sharing" 30 "down through Settings to Sharing"
shot tv-07-settings-sharing
key CENTER
sleep 2
has_text "Share bonded connection" && ok "Sharing open" || bad "Sharing didn't open"
shot tv-08-sharing
key BACK
sleep 1
key BACK
sleep 2
has_text "This device" && ok "Back, Back: home" || bad "Back didn't come home"
shot tv-09-home-again

say "Turning it on: the handoff dialog, then Android's VPN dialog"
move UP "Switch [|] ?$" 15 "back up to the main switch"
key CENTER
sleep 2
shot tv-10-handoff
focus_is "Turn on tailbond" "the handoff dialog starts on Turn on tailbond"
key CENTER
sleep 3
shot tv-11-vpn-consent
win="$(sh_ dumpsys window | grep -E 'mCurrentFocus' | head -n1)"
grep -q vpndialogs <<<"$win" && ok "Android's VPN dialog is up ($win)" || note "focused window after Turn on: $win"
key BACK
sleep 2
shot tv-12-after-consent-back

adb logcat -d -t 2000 >"$OUT/logcat.txt" 2>/dev/null || true
if grep -q "FATAL EXCEPTION" "$OUT/logcat.txt"; then
  grep -A 15 "FATAL EXCEPTION" "$OUT/logcat.txt" | head -n 40
  bad "the app crashed (logcat)"
else
  ok "no crash in logcat"
fi

say "Android TV: $PASS passed, $FAIL failed, $NOTE notes"
[ "$FAIL" -eq 0 ]
