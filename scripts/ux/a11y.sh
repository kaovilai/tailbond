#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Accessibility check of every web surface tailbond ships (docs/accessibility.md):
#   site        site/index.html (light and dark, desktop and phone)
#   phone-page  the agent's "Get tailbond" page (landing.html), every
#               platform tab, phone and desktop
#   dashboard   agent/web: Connections, Devices, Setup, Tailscale, Updates &
#               backup, Logs, Report a problem, Add this phone, phone hub,
#               sharing, sign-in and owner-only dialogs, update/backup/restore
#               dialogs (a real server and client tailbondd, plus fixtures)
#   glinet      the GL.iNet panel view (packaging/openwrt/gl-ui) in the mock
#               GL shell, every fixture state (scripts/ux/glinet/*fixtures.mjs)
#   luci        the LuCI view (packaging/openwrt/luci), in a minimal LuCI page
#
# axe-core on each page and state, plus four checks of its own (keyboard
# reach of clickable things, endless animation under reduced motion). Fails
# on impact serious or critical; moderate and minor are in the report only.
# Writes out/ux/a11y/report.json and report.html (or [outdir]) and prints one
# line per surface.
#
#   scripts/ux/a11y.sh [outdir]
#   TB_A11Y_ONLY=<regex> scripts/ux/a11y.sh   # only pages "surface/name" matching
#   TB_A11Y_JOBS=<n>                          # pages at a time (default 4)
#   TB_A11Y_SHOTS=1                           # also a screenshot of each page → [outdir]/shots/
#
# Like web-screens.sh: a throwaway server-role tailbondd (stand-in mqvpn,
# fake `tailscale` CLI) with the dashboard and the phone page, and an
# unpaired client-role one. Playwright, Vue 2 and axe-core (pinned) go into
# ~/.cache/tailbond-ux on first use; Chromium runs in the matching
# mcr.microsoft.com/playwright image (podman).
set -euo pipefail

AXE_VERSION=4.14.0
PW_PINNED=1.64.1
REPO="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$REPO/out/ux/a11y}"
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"
# ports away from a real tailbondd's 8077 on the same machine
DASH=18277 LAND=18280 CLIENT=18278
TMP="$(mktemp -d)"
# stop_pids: kill only PIDs that were set. `kill 0` (the old default) signals
# the whole process group, which would take the caller down with the test.
stop_pids() {
    local pid
    for pid in "$@"; do [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; done
}
trap 'stop_pids "${AGENT_PID:-}" "${CLIENT_PID:-}"; rm -rf "$TMP"' EXIT

# Playwright is pinned too, and a cache with another version is reinstalled: the
# npm version picks the container image tag, and a brand-new release has none.
if [[ "$(node -p "try{require('$PW/node_modules/playwright/package.json').version}catch(e){''}")" != "$PW_PINNED" || ! -d "$PW/node_modules/vue" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && { [ -f package.json ] || npm init -y >/dev/null; } && npm i --save-exact "playwright@$PW_PINNED" vue@2.6.14 >/dev/null)
fi
if [[ "$(node -p "try{require('$PW/node_modules/axe-core/package.json').version}catch(e){''}")" != "$AXE_VERSION" ]]; then
    (cd "$PW" && npm i --save-exact "axe-core@$AXE_VERSION" >/dev/null)
fi
PW_VERSION="$PW_PINNED"

(cd "$REPO/agent" && CGO_ENABLED=0 go build -ldflags "-X main.version=0.2.0" -o "$TMP/tailbondd" ./cmd/tailbondd)
mkdir -p "$OUT" && rm -f "$OUT"/report.json "$OUT"/report.html "$OUT"/error-*.png "$OUT"/shots/*.png
mkdir -p "$TMP/state" "$TMP/cstate" "$TMP/hooks" "$TMP/bin" "$OUT"
head -c 1000000 /dev/zero >"$TMP/tailbond.apk"
head -c 1000000 /dev/zero >"$TMP/tailbond-macos-arm64.tar.gz"
printf '#!/bin/sh\necho "path wifi via 203.0.113.7:443"\necho "mqvpn server ready"\nexec sleep 3600\n' >"$TMP/mqvpn" && chmod +x "$TMP/mqvpn"
# the previous run's fatal error: the crash banner
printf 'panic: send on closed channel\n\ngoroutine 42 [running]:\nmain.main()\n\t/src/main.go:12 +0x1d\n' >"$TMP/state/crash.log"
cat >"$TMP/bin/tailscale" <<'SH'
#!/bin/sh
# fake tailscale CLI: this machine is homebox (Tailscale IP 127.0.0.1)
case "$1" in
status) echo '{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["127.0.0.1"],"Online":true},
 "Peer":{"n1":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true,"CurAddr":"192.168.1.20:41641"}}}' ;;
debug) echo '{"AdvertiseRoutes":["10.77.0.0/24"]}' ;;
*) exit 1 ;;
esac
SH
chmod +x "$TMP/bin/tailscale"
cat >"$TMP/server.json" <<JSON
{"role":"server","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/state","hooks_dir":"$TMP/hooks",
 "listen":"127.0.0.1:$DASH","android_apk":"$TMP/tailbond.apk","macos_bundle":"$TMP/tailbond-macos-arm64.tar.gz","landing_port":$LAND,"landing_public_port":80,
 "server":{"port":443,"public_addr":"home.example.net:443","subnet":"10.77.0.0/24","subnet6":"fd77:77::/112","use_tailscale_cert":false}}
JSON
cat >"$TMP/client.json" <<JSON
{"role":"client","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/cstate","hooks_dir":"$TMP/hooks","listen":"127.0.0.1:$CLIENT",
 "client":{"pairing_uri":"","paths":[],"captive_portal_detect":false}}
JSON
PATH="$TMP/bin:$PATH" "$TMP/tailbondd" -config "$TMP/server.json" -quiet &
AGENT_PID=$!
PATH="$TMP/bin:$PATH" "$TMP/tailbondd" -config "$TMP/client.json" -quiet &
CLIENT_PID=$!
for u in "http://127.0.0.1:$LAND/" "http://127.0.0.1:$DASH/api/status" "http://127.0.0.1:$CLIENT/api/status"; do
    for _ in $(seq 50); do curl -fs -o /dev/null "$u" && break; sleep 0.2; done
    curl -fsS -o /dev/null "$u" || { echo "a11y: $u never answered" >&2; exit 1; }
done

rc=0
podman run --rm --network=host --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO:/repo:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules -e TB_UX_DEBUG="${TB_UX_DEBUG:-}" -e TB_A11Y_ONLY="${TB_A11Y_ONLY:-}" -e TB_A11Y_JOBS="${TB_A11Y_JOBS:-}" -e TB_A11Y_SHOTS="${TB_A11Y_SHOTS:-}" \
    -w /repo/scripts/ux \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /repo/scripts/ux/a11y.mjs /repo /out "http://127.0.0.1:$DASH/" "http://127.0.0.1:$LAND/" "http://127.0.0.1:$CLIENT/" || rc=$?
echo "report: $OUT/report.html"
exit "$rc"
