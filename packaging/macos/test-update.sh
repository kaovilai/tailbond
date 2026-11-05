#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
#
# Update test of the macOS install, run as root on a Mac
# (scripts/e2e-macos-update.sh builds the bundles and starts it; CI: job
# macos; the Linux/VPS twin is packaging/linux/test-update.sh). Installs,
# updates and removes tailbond on this Mac: run it on a throwaway one.
#
#   test-update.sh DIR    DIR has tailbond-macos-arm64-<v>.tar.gz for
#                         v = 0.0.1 (installed first), 0.0.2, 0.0.3 (a broken
#                         tailbondd), 0.0.4, 0.0.5
#
# A fake GitHub (python3 http.server on 127.0.0.1:8099: /repos/kaovilai/
# tailbond/releases/latest and the assets + SHA256SUMS) and the real
# install.sh, LaunchDaemon and dashboard. The config is written before the
# daemon's first start, pointing the agent at the fake GitHub with policy
# "Tell me", so nothing installs by itself and the real GitHub is never
# asked. Checks:
# - the agent sees the new release (update.available, notes_url);
# - "Update now" (POST /api/update/install): the download, the SHA-256
#   check, the new bundle's install.sh --update run detached from the
#   LaunchDaemon (it restarts the daemon from under it), the new version
#   serving, .prev kept, install.env, update-result ok, the menu bar app
#   intact;
# - a release whose tailbondd doesn't start is rolled back to .prev and the
#   dashboard says so (state rolled_back, with the reason);
# - a release whose file doesn't match SHA256SUMS is refused (the agent and
#   install.sh --update both);
# - the installed install.sh --update alone (no bundle) downloads the latest
#   release and installs it;
# - uninstall.sh --purge leaves nothing behind.
# Runs with macOS's own bash 3.2.
set -uo pipefail

DIR="${1:?usage: test-update.sh <dir with the bundles>}"
[ "$(id -u)" = 0 ] || { echo "run as root: sudo $0" >&2; exit 2; }
[ "$(uname -s)" = Darwin ] || { echo "this is the macOS test (Linux: packaging/linux/test-update.sh)" >&2; exit 2; }
PASS=0 FAIL=0
ok()  { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  FAIL %s\n' "$*"; FAIL=$((FAIL + 1)); }
check() { local what="$1"; shift; if "$@"; then ok "$what"; else bad "$what"; fi; }
say() { printf '==> %s\n' "$*"; }
wait_for() { # N CMD...
  local n="$1"; shift
  for _ in $(seq 1 "$n"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}

LABEL=com.tailbond.tailbondd
BINDIR=/usr/local/bin LIBDIR=/usr/local/lib/tailbond
ETC=/usr/local/etc/tailbond STATE=/usr/local/var/lib/tailbond
CONF="$ETC/tailbond.json" MANIFEST="$ETC/install.env"
PLIST=/Library/LaunchDaemons/$LABEL.plist
LOG=/usr/local/var/log/tailbond.log
API=http://127.0.0.1:8077
GHURL=http://127.0.0.1:8099
# the broken 0.0.3 tailbondd marks its crashes here (scripts/e2e-macos-update.sh)
BROKENLOG=/tmp/tb-macos-update-broken.log
W="$(mktemp -d /tmp/tb-macos-update.XXXXXX)"
GH="$W/gh"
api() { curl -fsS -m 5 "$@"; }
post() { curl -fsS -m 10 -H 'Content-Type: application/json' -d '{}' "$API$1"; }
status_upd() { api "$API/api/update" | python3 -c "import json,sys; print(json.load(sys.stdin).get('$1') or '')"; }
hello_v() { api "$API/api/hello" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])'; }
is_version() { [ "$(hello_v 2>/dev/null)" = "$1" ]; }
role_is() { api "$API/api/hello" 2>/dev/null | grep -q "\"role\":\"$1\""; }
upd_is() { [ "$(status_upd "$1" 2>/dev/null)" = "$2" ]; }
upd_has() { api "$API/api/update" 2>/dev/null | grep -q -- "$1"; }
gh_up() { api "$GHURL/repos/kaovilai/tailbond/releases/latest" >/dev/null; }
# none_available: "" only counts when /api/update answered (a dead daemon or
# bad JSON fails the pipeline, not the check)
none_available() {
  local v
  v="$(status_upd available 2>/dev/null)" || return 1
  [ -z "$v" ]
}
prev_is() { [ -x "$BINDIR/tailbondd.prev" ] && "$BINDIR/tailbondd.prev" -version 2>&1 | grep -q "tailbondd $1"; }
result_says() { # RESULT SNIPPET: update-result.json holds both
  grep -q "\"result\":\"$1\"" "$STATE/update-result.json" && grep -q "$2" "$STATE/update-result.json"
}
broken_ran() { [ -s "$BROKENLOG" ]; }
daemon_gone() { ! launchctl print "system/$LABEL" >/dev/null 2>&1; }
no_tailbondd() { ! pgrep -f "^$BINDIR/tailbondd" >/dev/null 2>&1; }
trap 'launchctl bootout system/$LABEL 2>/dev/null || true; pkill -f "http.server 8099" 2>/dev/null || true; rm -rf "$W"' EXIT

# publish VERSION [BAD]: make VERSION the latest release; BAD: its bundle
# doesn't match SHA256SUMS
publish() {
  local v="$1" name="tailbond-macos-arm64.tar.gz" d="$GH/dl/v$1"
  mkdir -p "$d" "$GH/repos/kaovilai/tailbond/releases"
  cp "$DIR/tailbond-macos-arm64-$v.tar.gz" "$d/$name"
  (cd "$d" && shasum -a 256 "$name" >SHA256SUMS)
  if [ "${2:-}" = BAD ]; then
    awk '{ print "0000000000000000000000000000000000000000000000000000000000000000  " $2 }' "$d/SHA256SUMS" >"$d/x" && mv "$d/x" "$d/SHA256SUMS"
  fi
  cat >"$GH/repos/kaovilai/tailbond/releases/latest" <<EOF
{"tag_name": "v$v", "html_url": "https://github.com/kaovilai/tailbond/releases/tag/v$v", "draft": false, "prerelease": false,
 "assets": [{"name": "$name", "browser_download_url": "http://127.0.0.1:8099/dl/v$v/$name", "size": 1},
            {"name": "SHA256SUMS", "browser_download_url": "http://127.0.0.1:8099/dl/v$v/SHA256SUMS", "size": 1}]}
EOF
}

# a leftover install from an earlier run (the test ends with uninstall and
# starts from a clean slate itself)
if [ -x "$LIBDIR/uninstall.sh" ]; then "$LIBDIR/uninstall.sh" --purge >"$W/preclean.log" 2>&1 || true; fi

say "install 0.0.1 (the real install.sh, LaunchDaemon)"
publish 0.0.1
(cd "$GH" && exec python3 -m http.server 8099 --bind 127.0.0.1 >/dev/null 2>&1) &
check "fake GitHub up (releases/latest)" wait_for 20 gh_up
mkdir -p "$W/b1"
tar -C "$W/b1" -xzf "$DIR/tailbond-macos-arm64-0.0.1.tar.gz"
B="$W/b1/tailbond-macos-arm64"
# the config before the daemon's first start: the fake GitHub, "Tell me"
# (the agent reads the file once, at start)
install -d -m 0700 "$ETC"
# fatal if this fails: a daemon that starts on the default config (policy
# "Right away", the real GitHub) would update itself mid-test
"$B/bin/tailbondd" -init -role client -config "$CONF" >/dev/null || { echo "test-update.sh: can't write $CONF" >&2; exit 2; }
python3 - "$CONF" <<'EOF' || exit 2
import json, sys
p = sys.argv[1]
c = json.load(open(p))
c["update"] = {"api_url": "http://127.0.0.1:8099", "policy": "notify"}
json.dump(c, open(p, "w"), indent=2)
EOF
chmod 0600 "$CONF"
"$B/install.sh" >"$W/install.log" 2>&1
check "install.sh: tailbond 0.0.1 installed" grep -q 'tailbond 0.0.1 is installed' "$W/install.log"
check "install.env records 0.0.1" grep -qx 'TB_VERSION=0.0.1' "$MANIFEST"
check "agent 0.0.1 up (LaunchDaemon)" wait_for 30 is_version 0.0.1
check "update method static (one-click updates)" upd_is method static
check "policy notify (Tell me: nothing by itself)" upd_is policy notify

say "0.0.2 released: shown, then Update now"
publish 0.0.2
post /api/update/check >/dev/null
check "the agent sees 0.0.2" wait_for 20 upd_is available 0.0.2
check "with its release page" upd_has 'releases/tag/v0.0.2'
check "notify: not installed by itself" is_version 0.0.1
post /api/update/install >/dev/null
check "updated: the agent runs 0.0.2" wait_for 120 is_version 0.0.2
check "result ok" wait_for 20 upd_is last_result ok
check "update-result.json: ok" grep -q '"result":"ok"' "$STATE/update-result.json"
check "previous binary kept as tailbondd.prev" prev_is 0.0.1
check "install.env records 0.0.2" grep -qx 'TB_VERSION=0.0.2' "$MANIFEST"
check "still the client role" role_is client
# the bundle carries the menu bar app (CI's does): it must come through the
# agent's unpack (which rewrites every file) and the install_app copy.
# Checking the unpack matters: install_app skips a bundle without the app,
# so /Applications alone could still be the 0.0.1 one
[ -d "$B/tailbond.app" ] && {
  check "menu bar app intact in the agent's unpack" codesign --verify --strict "$STATE/update/0.0.2/tailbond-macos-arm64/tailbond.app"
  check "menu bar app intact after the update" codesign --verify --strict /Applications/tailbond.app
}
check "nothing available any more" wait_for 30 none_available

say "0.0.3 doesn't start: rolled back"
rm -f "$BROKENLOG"
publish 0.0.3
post /api/update/check >/dev/null
check "the agent sees 0.0.3" wait_for 20 upd_is available 0.0.3
post /api/update/install >/dev/null
check "back on 0.0.2 after the failed start" wait_for 150 upd_is state rolled_back
check "the agent runs 0.0.2" is_version 0.0.2
check "last result: 0.0.3 rolled back" upd_is last_result_version 0.0.3
check "the dashboard says why" upd_has "didn't come up"
check "update-result.json: rolled_back, with the reason" result_says rolled_back "didn't come up"
check "0.0.3 really ran (it marked each crash)" broken_ran
check "install.env back to 0.0.2" grep -qx 'TB_VERSION=0.0.2' "$MANIFEST"
check ".prev holds 0.0.2 (what went back)" prev_is 0.0.2

say "a release that doesn't match SHA256SUMS is refused"
publish 0.0.5 BAD
post /api/update/check >/dev/null
check "the agent sees 0.0.5" wait_for 20 upd_is available 0.0.5
post /api/update/install >/dev/null
check "agent: refused (failed, SHA-256)" wait_for 30 upd_has 'SHA-256'
check "still 0.0.2" is_version 0.0.2
rc=0
TB_API_URL=$GHURL "$LIBDIR/install.sh" --update >"$W/u5.log" 2>&1 || rc=$?
u5_refused() { [ "$rc" != 0 ] && grep -q "doesn't match SHA256SUMS" "$W/u5.log"; }
check "install.sh --update alone: refused too" u5_refused
check "still 0.0.2 after both refusals" is_version 0.0.2

say "install.sh --update on its own (downloads the latest)"
publish 0.0.4
rc4=0
TB_API_URL=$GHURL "$LIBDIR/install.sh" --update >"$W/u4.log" 2>&1 || rc4=$?
check "install.sh --update: 0.0.4 running" wait_for 20 is_version 0.0.4
check "install.sh --update: exit 0" [ "$rc4" = 0 ]
check "it said so" grep -q 'tailbond 0.0.4 is running' "$W/u4.log"
check "install.env records 0.0.4" grep -qx 'TB_VERSION=0.0.4' "$MANIFEST"
rc4b=0
TB_API_URL=$GHURL "$LIBDIR/install.sh" --update >"$W/u4b.log" 2>&1 || rc4b=$?
check "again: already installed" grep -q 'installed already' "$W/u4b.log"
check "again: exit 0" [ "$rc4b" = 0 ]

dump_diag() {
  echo "---- agent log (tail) ----"; tail -n 60 "$LOG" 2>/dev/null
  echo "---- install.env ----"; cat "$MANIFEST" 2>/dev/null
  echo "---- update-result.json ----"; cat "$STATE/update-result.json" 2>/dev/null; echo
  echo "---- install.sh logs ----"; tail -n 20 "$W"/u*.log "$W"/install.log 2>/dev/null
}

say "uninstall.sh --purge"
# before the purge removes the manifest, the state and the log
[ "$FAIL" = 0 ] || dump_diag
"$LIBDIR/uninstall.sh" --purge
check "daemon gone" daemon_gone
check "no tailbondd process" no_tailbondd
left="$(ls -d "$BINDIR/tailbondd" "$BINDIR/mqvpn" "$BINDIR/tailbondd.prev" "$LIBDIR" "$ETC" "$STATE" "$PLIST" /Applications/tailbond.app 2>/dev/null || true)"
[ -z "$left" ] && ok "nothing left behind" || bad "left behind: $left"

[ "$FAIL" = 0 ] || [ ! -e "$ETC" ] || dump_diag
echo
echo "RESULT: $PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]
