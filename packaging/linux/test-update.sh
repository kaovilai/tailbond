#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Update test for the static (VPS) install, run as root inside a throwaway
# container (scripts/e2e-update-vps.sh starts it; CI job vps-install):
#
#   test-update.sh DIR    DIR has tailbond-linux-<arch>-<v>.tar.gz for
#                         v = 0.0.1 (installed first), 0.0.2, 0.0.3 (a broken
#                         tailbondd), 0.0.4, 0.0.5
#
# A fake GitHub (python http.server: /repos/kaovilai/tailbond/releases/latest
# and the assets + SHA256SUMS) and no systemd: tailbondd runs under a restart
# loop, TB_TEST_RESTART stands in for systemctl restart. Checks:
# - the agent sees the new release (update.available, notes_url);
# - "Update now" (POST /api/update/install): download, SHA-256 check, the
#   new bundle's install.sh --update, restart on the new version, .prev kept;
# - a release whose tailbondd doesn't start is rolled back to .prev and the
#   dashboard says so (state rolled_back);
# - the installed install.sh --update alone (no bundle) downloads the latest
#   release and installs it;
# - a release whose file doesn't match SHA256SUMS is refused (both paths).
set -uo pipefail

DIR="${1:?usage: test-update.sh <dir with the bundles>}"
PASS=0 FAIL=0
ok()  { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  FAIL %s\n' "$*"; FAIL=$((FAIL + 1)); }
check() { local what="$1"; shift; if "$@"; then ok "$what"; else bad "$what"; fi; }
say() { printf '==> %s\n' "$*"; }
wait_for() { # SECONDS CMD...
  local n="$1"; shift
  for _ in $(seq 1 "$n"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}

. /etc/os-release
say "deps on $PRETTY_NAME"
if command -v dnf >/dev/null; then
  dnf install -y -q iproute python3 tar gzip nftables procps-ng findutils >/dev/null
elif command -v apt-get >/dev/null; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq && apt-get install -y -qq iproute2 python3 nftables procps curl >/dev/null
fi
ARCH="$(uname -m)"
W="$(mktemp -d)"
GH="$W/gh"
API=http://127.0.0.1:8077
api() { curl -fsS -m 5 "$@"; }
post() { curl -fsS -m 10 -H 'Content-Type: application/json' -d '{}' "$API$1"; }
status_upd() { api "$API/api/update" | python3 -c "import json,sys; print(json.load(sys.stdin).get('$1') or '')"; }
hello_v() { api "$API/api/hello" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])'; }
is_version() { [ "$(hello_v 2>/dev/null)" = "$1" ]; }
upd_is() { [ "$(status_upd "$1" 2>/dev/null)" = "$2" ]; }
upd_has() { api "$API/api/update" 2>/dev/null | grep -q -- "$1"; }

# publish VERSION [BAD]: make VERSION the latest release; BAD: its bundle
# doesn't match SHA256SUMS
publish() {
  local v="$1" name="tailbond-linux-$ARCH.tar.gz" d="$GH/dl/v$1"
  mkdir -p "$d" "$GH/repos/kaovilai/tailbond/releases"
  cp "$DIR/tailbond-linux-$ARCH-$v.tar.gz" "$d/$name"
  (cd "$d" && sha256sum "$name" >SHA256SUMS)
  if [ "${2:-}" = BAD ]; then
    awk '{ print "0000000000000000000000000000000000000000000000000000000000000000  " $2 }' "$d/SHA256SUMS" >"$d/x" && mv "$d/x" "$d/SHA256SUMS"
  fi
  cat >"$GH/repos/kaovilai/tailbond/releases/latest" <<EOF
{"tag_name": "v$v", "html_url": "https://github.com/kaovilai/tailbond/releases/tag/v$v", "draft": false, "prerelease": false,
 "assets": [{"name": "$name", "browser_download_url": "http://127.0.0.1:8099/dl/v$v/$name", "size": 1},
            {"name": "SHA256SUMS", "browser_download_url": "http://127.0.0.1:8099/dl/v$v/SHA256SUMS", "size": 1}]}
EOF
}

say "install 0.0.1"
mkdir -p "$W/b1"
tar -C "$W/b1" -xzf "$DIR/tailbond-linux-$ARCH-0.0.1.tar.gz"
"$W/b1/tailbond-linux-$ARCH/install.sh" --no-tailscale --public-addr 203.0.113.7 --no-firewall >"$W/install.log" 2>&1
check "install.sh (0.0.1)" grep -q 'tailbond 0.0.1 server install' "$W/install.log"
check "install.env records 0.0.1" grep -qx 'TB_VERSION=0.0.1' /etc/tailbond/install.env
check "install.sh kept at /usr/local/lib/tailbond/install.sh" test -x /usr/local/lib/tailbond/install.sh
python3 - <<'EOF'
import json
p = "/etc/tailbond/tailbond.json"
c = json.load(open(p))
c["update"] = {"api_url": "http://127.0.0.1:8099"}
json.dump(c, open(p, "w"), indent=2)
EOF

publish 0.0.1
(cd "$GH" && exec python3 -m http.server 8099 --bind 127.0.0.1 >/dev/null 2>&1) &
export TB_TEST_RESTART='pkill -x tailbondd'
( while [ ! -e "$W/stop" ]; do /usr/local/bin/tailbondd -config /etc/tailbond/tailbond.json >>"$W/agent.log" 2>&1; sleep 1; done ) &
trap 'touch "$W/stop"; pkill -x tailbondd; pkill -f "http.server 8099"' EXIT
check "agent 0.0.1 up" wait_for 30 is_version 0.0.1
check "method static" test "$(status_upd method)" = static
check "policy notify by default" test "$(status_upd policy)" = notify

say "0.0.2 released: shown, then Update now"
publish 0.0.2
post /api/update/check >/dev/null
check "the agent sees 0.0.2" wait_for 20 upd_is available 0.0.2
check "with its release page" upd_has 'releases/tag/v0.0.2' 
check "notify: not installed by itself" test "$(hello_v)" = 0.0.1
post /api/update/install >/dev/null
check "updated: the agent runs 0.0.2" wait_for 90 is_version 0.0.2
check "result ok" wait_for 20 upd_is last_result ok
check "previous binary kept as tailbondd.prev" sh -c '/usr/local/bin/tailbondd.prev -version 2>&1 | grep -q 0.0.1'
check "install.env records 0.0.2" grep -qx 'TB_VERSION=0.0.2' /etc/tailbond/install.env
check "nothing available any more" test -z "$(status_upd available)"

say "0.0.3 doesn't start: rolled back"
publish 0.0.3
post /api/update/check >/dev/null
wait_for 20 upd_is available 0.0.3
post /api/update/install >/dev/null
check "back on 0.0.2 after the failed start" wait_for 150 upd_is state rolled_back
check "the agent runs 0.0.2" is_version 0.0.2
check "last result: 0.0.3 rolled back" upd_is last_result_version 0.0.3
check "0.0.3 really ran" grep -q 'broken release' "$W/agent.log"
check "install.env back to 0.0.2" grep -qx 'TB_VERSION=0.0.2' /etc/tailbond/install.env

say "install.sh --update on its own (downloads the latest)"
publish 0.0.4
TB_API_URL=http://127.0.0.1:8099 /usr/local/lib/tailbond/install.sh --update >"$W/u4.log" 2>&1
check "install.sh --update: 0.0.4 running" wait_for 20 is_version 0.0.4
check "it said so" grep -q 'tailbond 0.0.4 is running' "$W/u4.log"
TB_API_URL=http://127.0.0.1:8099 /usr/local/lib/tailbond/install.sh --update >"$W/u4b.log" 2>&1
check "again: already installed" grep -q 'installed already' "$W/u4b.log"

say "a release that doesn't match SHA256SUMS is refused"
publish 0.0.5 BAD
TB_API_URL=http://127.0.0.1:8099 /usr/local/lib/tailbond/install.sh --update >"$W/u5.log" 2>&1
check "install.sh: refused" grep -q "doesn't match SHA256SUMS" "$W/u5.log"
post /api/update/check >/dev/null
wait_for 20 upd_is available 0.0.5
post /api/update/install >/dev/null
check "agent: refused (failed, SHA-256)" wait_for 30 upd_has 'SHA-256'
check "still 0.0.4" is_version 0.0.4

echo
echo "RESULT: $PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]
