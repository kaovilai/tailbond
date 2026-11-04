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
#    same bundle (binaries swapped, .prev kept, health check); uninstall.sh
#    --purge leaves nothing behind.
# 2. The Homebrew formula (tailbond.rb with a file:// URL to the tarball):
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
  /usr/local/var/lib/tailbond /Library/LaunchDaemons/$LABEL.plist 2>/dev/null || true)"
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
brew uninstall --formula "$name" && ok "brew uninstall" || bad "brew uninstall failed"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
