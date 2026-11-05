#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Install test of the macOS packaging on a Mac (CI: job macos; docs/macos.md):
#
#   scripts/test-macos-install.sh out/macos/tailbond-macos-arm64.tar.gz out/macos/tailbond.rb
#
# 1. The bundle as a user gets it: quarantined like a browser download,
#    unpacked, sudo ./install.sh: the LaunchDaemon runs tailbondd (client
#    role, unpaired) on 127.0.0.1:8077; install.sh --update --force from the
#    same bundle (binaries swapped, .prev kept, health check); the menu bar
#    app in /Applications (ad-hoc signed, not quarantined, reads the running
#    agent: tailbond --status); uninstall.sh --purge leaves nothing behind.
# 2. The Mac as home server: install.sh --role server --public-addr --port
#    (no Tailscale on the runner): the server runs, the hooks load the pf
#    anchor and turn forwarding on; --update keeps the role; uninstall.sh
#    --purge puts pf and the sysctls back.
# 3. The Homebrew formula (tailbond.rb with a file:// URL to the tarball):
#    brew install, brew test, sudo brew services start, the dashboard
#    answers, brew uninstall.
set -euo pipefail

TGZ="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
FORMULA="$2"
LABEL=com.tailbond.tailbondd
PASS=0 FAIL=0
ok()  { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
say() { printf '\033[1m==> %s\033[0m\n' "$*"; }
hello() { curl -fsS -m 3 http://127.0.0.1:8077/api/hello 2>/dev/null; }
wait_hello() { for _ in $(seq 1 "$1"); do hello >/dev/null && return 0; sleep 1; done; return 1; }
W="$(mktemp -d /tmp/tb-macos-install.XXXXXX)"
trap 'sudo launchctl bootout system/$LABEL 2>/dev/null || true; rm -rf "$W"' EXIT

say "bundle: quarantined download, sudo ./install.sh"
cp "$TGZ" "$W/"
xattr -w com.apple.quarantine "0081;$(printf %x "$(date +%s)");Safari;" "$W/$(basename "$TGZ")"
tar -xzf "$W/$(basename "$TGZ")" -C "$W"
B="$W/tailbond-macos-arm64"
xattr -p com.apple.quarantine "$B/bin/tailbondd" >/dev/null 2>&1 && echo "  (bsdtar kept the quarantine flag on the unpacked files)"
sudo "$B/install.sh"
xattr -p com.apple.quarantine /usr/local/bin/tailbondd >/dev/null 2>&1 && bad "installed tailbondd is still quarantined" || ok "installed binaries carry no quarantine flag"
sudo launchctl print "system/$LABEL" >/dev/null && ok "LaunchDaemon $LABEL loaded" || bad "LaunchDaemon not loaded"
if wait_hello 30; then
  ok "dashboard on 127.0.0.1:8077: $(hello)"
else
  bad "no answer on 127.0.0.1:8077"; sudo tail -n 40 /usr/local/var/log/tailbond.log || true
fi
hello | grep -q '"role":"client"' && ok "client role" || bad "role: $(hello)"
curl -fsS -m 3 http://127.0.0.1:8077/api/status | grep -q '"platform":"darwin"' && ok "status says platform darwin" || bad "status: $(curl -fsS -m 3 http://127.0.0.1:8077/api/status | head -c 300)"
curl -fsS -m 3 http://127.0.0.1:8077/api/update | grep -q '"method":"static"' && ok "update method static (one-click updates)" || bad "update: $(curl -fsS -m 3 http://127.0.0.1:8077/api/update)"
IP="$(ipconfig getifaddr en0 2>/dev/null || true)"
if [ -n "$IP" ]; then
  curl -fsS -m 3 "http://$IP:8077/api/hello" >/dev/null 2>&1 && bad "dashboard reachable from the network ($IP)" || ok "dashboard not reachable on $IP (this Mac only)"
fi
sudo test -f /usr/local/etc/tailbond/install.env && ok "install.env written" || bad "no install.env"
rep="$(curl -fsS -m 5 http://127.0.0.1:8077/api/report)"
echo "$rep" | grep -q '"component":"Mac"' && echo "$rep" | grep -q '"device":"macOS ' && ok "Report a problem fills in Mac and macOS" || bad "report: $(echo "$rep" | head -c 300)"

say "menu bar app (tailbond.app)"
APP=/Applications/tailbond.app
if [ -d "$B/tailbond.app" ]; then
  [ -x "$APP/Contents/MacOS/tailbond" ] && ok "installed in /Applications" || bad "no $APP"
  codesign --verify --strict "$APP" 2>/dev/null && ok "signature verifies (ad-hoc)" || bad "codesign --verify: $(codesign --verify --strict "$APP" 2>&1)"
  codesign -dv "$APP" 2>&1 | grep -q 'Signature=adhoc' && ok "ad-hoc signed" || bad "signature: $(codesign -dv "$APP" 2>&1 | tail -n 3)"
  [ "$(/usr/libexec/PlistBuddy -c 'Print :LSUIElement' "$APP/Contents/Info.plist" 2>/dev/null)" = true ] && ok "menu bar only (LSUIElement)" || bad "LSUIElement not set"
  # without it macOS kills the app on its first CoreBluetooth call (Connect via phone)
  /usr/libexec/PlistBuddy -c 'Print :NSBluetoothAlwaysUsageDescription' "$APP/Contents/Info.plist" 2>/dev/null | grep -q Bluetooth \
    && ok "Bluetooth usage description (Connect via phone)" || bad "NSBluetoothAlwaysUsageDescription missing"
  xattr -p com.apple.quarantine "$APP/Contents/MacOS/tailbond" >/dev/null 2>&1 && bad "app is quarantined" || ok "app carries no quarantine flag"
  out="$("$APP/Contents/MacOS/tailbond" --status 2>&1)" && ok "the app reads the running agent: $(echo "$out" | head -n 1)" || bad "tailbond --status: $out"
  echo "$out" | grep -q '^tailbond: Not paired' && ok "it says Not paired (fresh install)" || bad "--status said: $out"
  echo "$out" | grep -q "Install updates: Right away" && ok "it shows the update policy (Install updates: Right away)" || bad "no update policy in: $out"
  "$APP/Contents/MacOS/tailbond" --status http://127.0.0.1:1 >/dev/null 2>&1 && bad "--status succeeded with no agent" || ok "--status fails when nothing answers"
else
  bad "the bundle has no tailbond.app"
fi

say "install.sh --update --force (same bundle)"
pid0="$(sudo launchctl print "system/$LABEL" | awk '/pid =/{print $3; exit}')"
sudo /usr/local/lib/tailbond/install.sh --update --force --bundle-dir "$B" && ok "update ran" || bad "update failed"
pid1="$(sudo launchctl print "system/$LABEL" | awk '/pid =/{print $3; exit}')"
[ -n "$pid1" ] && [ "$pid0" != "$pid1" ] && ok "tailbondd restarted (pid $pid0 -> $pid1)" || bad "not restarted ($pid0 / $pid1)"
sudo grep -q '"result":"ok"' /usr/local/var/lib/tailbond/update-result.json && ok "update-result ok" || bad "update-result: $(sudo cat /usr/local/var/lib/tailbond/update-result.json 2>/dev/null)"
[ -x /usr/local/bin/tailbondd.prev ] && ok "previous version kept as tailbondd.prev" || bad "no tailbondd.prev"

say "uninstall.sh --purge"
sudo /usr/local/lib/tailbond/uninstall.sh --purge
sudo launchctl print "system/$LABEL" >/dev/null 2>&1 && bad "daemon still loaded" || ok "daemon gone"
left="$(ls -d /usr/local/bin/tailbondd /usr/local/bin/mqvpn /usr/local/lib/tailbond /usr/local/etc/tailbond \
  /usr/local/var/lib/tailbond /Library/LaunchDaemons/$LABEL.plist /Applications/tailbond.app 2>/dev/null || true)"
[ -z "$left" ] && ok "nothing left behind" || bad "left behind: $left"

say "the Mac as home server: install.sh --role server"
ANCHOR=com.apple/251.tailbond.server
fwd() { echo "$(sysctl -n net.inet.ip.forwarding)/$(sysctl -n net.inet6.ip6.forwarding)"; }
anchor_rules() { { sudo pfctl -a "$ANCHOR" -s nat 2>/dev/null; sudo pfctl -a "$ANCHOR" -s rules 2>/dev/null; } | grep -v '^$' || true; }
FWD0="$(fwd)"
sudo "$B/install.sh" --role server --public-addr 203.0.113.7 --port 4443 && ok "install.sh --role server" || bad "server install failed"
wait_hello 30 && hello | grep -q '"role":"server"' && ok "server role: $(hello)" || bad "role: $(hello)"
st() { curl -fsS -m 3 http://127.0.0.1:8077/api/status; }
for _ in $(seq 1 30); do st | grep -q '"state":"connected"' && break; sleep 1; done
st | grep -q '"state":"connected"' && ok "server running (mqvpn server mode on a utun)" || { bad "server state: $(st | head -c 300)"; sudo tail -n 40 /usr/local/var/log/tailbond.log || true; }
C=/usr/local/etc/tailbond/tailbond.json
sudo grep -q '"listen": ":8077"' "$C" && sudo grep -q '"port": 4443' "$C" && sudo grep -q '"public_addr": "203.0.113.7"' "$C" && sudo grep -q '"subnet": "10\.' "$C" \
  && ok "config: listen :8077, port 4443, public_addr, a 10.x subnet" || bad "config: $(sudo cat "$C")"
for _ in $(seq 1 15); do [ -n "$(anchor_rules)" ] && break; sleep 1; done
anchor_rules | grep -q '^nat on ' && ok "pf anchor loaded: $(anchor_rules | head -n1)" || bad "no pf anchor rules"
[ "$(sysctl -n net.inet.ip.forwarding)" = 1 ] && ok "forwarding on" || bad "forwarding $(fwd)"
sudo test -f /usr/local/var/lib/tailbond/server-net.state && ok "restore record written" || bad "no restore record"
if [ -n "$IP" ]; then
  curl -fsS -m 3 "http://$IP:8077/api/hello" >/dev/null 2>&1 && bad "server dashboard reachable from the network ($IP)" || ok "server dashboard not on $IP"
fi
sudo /usr/local/lib/tailbond/install.sh --update --force --bundle-dir "$B" >/dev/null && wait_hello 30 && hello | grep -q '"role":"server"' \
  && ok "--update keeps the server role" || bad "after --update: $(hello)"
sudo /usr/local/lib/tailbond/uninstall.sh --purge
[ -z "$(anchor_rules)" ] && ok "uninstall: pf anchor empty" || bad "uninstall left pf rules: $(anchor_rules)"
[ "$(fwd)" = "$FWD0" ] && ok "uninstall: forwarding back to $FWD0" || bad "forwarding $(fwd), was $FWD0"
pgrep -f "caffeinate -i -s -w" >/dev/null && bad "a power assertion was left" || ok "uninstall: no power assertion left"
left="$(ls -d /usr/local/bin/tailbondd /usr/local/lib/tailbond /usr/local/etc/tailbond /usr/local/var/lib/tailbond 2>/dev/null || true)"
[ -z "$left" ] && ok "nothing left behind" || bad "left behind: $left"

say "Homebrew formula (file:// tarball)"
F="$W/tailbond.rb"
sed "s|^  url \".*\"|  url \"file://$TGZ\"|" "$FORMULA" >"$F"
how=""
if HOMEBREW_NO_AUTO_UPDATE=1 brew install --formula "$F" 2>"$W/brew.err"; then
  how="brew install --formula ./tailbond.rb"
else
  echo "  (brew install --formula from a file: $(tail -n 2 "$W/brew.err" | tr '\n' ' '))"
  brew tap-new --no-git local/tailbond >/dev/null
  cp "$F" "$(brew --repository local/tailbond)/Formula/tailbond.rb"
  HOMEBREW_NO_AUTO_UPDATE=1 brew install local/tailbond/tailbond && how="a local tap (brew tap-new; brew install local/tailbond/tailbond)"
fi
[ -n "$how" ] && ok "installed with $how" || bad "brew install failed"
name=tailbond; brew list --formula | grep -qx tailbond || name=local/tailbond/tailbond
brew test "$name" && ok "brew test" || bad "brew test failed"
conf="$(brew --prefix)/etc/tailbond/tailbond.json"
grep -q "$(brew --prefix)/opt/tailbond/bin/mqvpn" "$conf" && ok "config points at Homebrew's mqvpn" || bad "config: $(cat "$conf" 2>/dev/null)"
sudo brew services start "$name" && ok "sudo brew services start" || bad "brew services start failed"
if wait_hello 30; then ok "dashboard up under brew services: $(hello)"; else bad "no dashboard under brew services"; sudo tail -n 30 "$(brew --prefix)/var/log/tailbond.log" || true; fi
curl -fsS -m 3 http://127.0.0.1:8077/api/update | grep -q '"method":"brew"' && ok "update method brew (the dashboard shows the brew command)" || bad "update: $(curl -fsS -m 3 http://127.0.0.1:8077/api/update)"
sudo brew services stop "$name" >/dev/null || true
# a root service leaves the keg root-owned: hand it back (docs/macos.md)
P="$(brew --prefix)"
sudo chown -R "$(id -un)" "$P/Cellar/tailbond"
sudo chown -h "$(id -un)" "$P/opt/tailbond" "$P/var/homebrew/linked/tailbond" 2>/dev/null || true
brew uninstall --formula "$name" && ok "brew uninstall (after handing the keg back)" || bad "brew uninstall failed"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
