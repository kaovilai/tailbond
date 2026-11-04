#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Report a problem (docs/reporting.md) in a browser, against a real
# tailbondd: the dashboard (desktop and phone) and the GL.iNet panel (the
# real view in the mock GL shell, its calls sent to the same agent). It
# checks the flow and saves screenshots → out/ux/report/web/ (or [outdir]):
#
#   - the crash banner after a crash (a crash.log left in the state dir);
#   - the preview: the notice, redacted logs and crash (no secret from the
#     fake mqvpn's output or the crash survives);
#   - "Open GitHub issue" stays off until the text and the checkbox;
#   - the issue URL (window.open captured, nothing is opened): GitHub's
#     bug_report.yml form, at most 7000 characters, nothing secret;
#   - "Save report file" downloads tailbond-report-*.txt; "Copy full report"
#     puts the same text on the clipboard.
#
#   scripts/ux/report-screens.sh [outdir]
#
# Playwright and Vue 2 go into ~/.cache/tailbond-ux on first use; Chromium
# runs in the matching mcr.microsoft.com/playwright image (podman).
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$REPO/out/ux/report/web}"
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"
TMP="$(mktemp -d)"
trap 'kill "${AGENT_PID:-0}" 2>/dev/null || true; rm -rf "$TMP"' EXIT

if [[ ! -d "$PW/node_modules/playwright" || ! -d "$PW/node_modules/vue" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && { [ -f package.json ] || npm init -y >/dev/null; } && npm i playwright vue@2.6.14 >/dev/null)
fi
PW_VERSION="$(node -p "require('$PW/node_modules/playwright/package.json').version")"

(cd "$REPO/agent" && CGO_ENABLED=0 go build -ldflags "-X main.version=0.2.0" -o "$TMP/tailbondd" ./cmd/tailbondd)
mkdir -p "$TMP/state" "$TMP/hooks" "$OUT"
# a stand-in mqvpn whose output has things that must not reach a report
cat >"$TMP/mqvpn" <<'SH'
#!/bin/sh
echo "path wifi via 203.0.113.7:443 token=hunter2 user sam@example.com"
echo "peer 2001:db8:1::42 on homebox.tail1234.ts.net"
echo "mqvpn server ready"
exec sleep 3600
SH
chmod +x "$TMP/mqvpn"
# the previous run's fatal error: the crash banner and the crash details
printf 'panic: send on closed channel (key=hunter2 from 198.51.100.9)\n\ngoroutine 42 [running]:\nmain.main()\n\t/src/main.go:12 +0x1d\n' >"$TMP/state/crash.log"
cat >"$TMP/tailbond.json" <<JSON
{"role":"server","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/state","hooks_dir":"$TMP/hooks",
 "listen":"127.0.0.1:8077","landing_port":-1,"disable_tailscale":true,
 "server":{"port":443,"public_addr":"home.example.net:443","subnet":"10.77.0.0/24","subnet6":"fd77:77::/112","use_tailscale_cert":false}}
JSON
"$TMP/tailbondd" -config "$TMP/tailbond.json" -quiet &
AGENT_PID=$!
for _ in $(seq 50); do curl -fsS -o /dev/null http://127.0.0.1:8077/api/status && break; sleep 0.2; done
sleep 1

podman run --rm --network=host --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO:/repo:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /repo/scripts/ux/report-screens.mjs /repo /out http://127.0.0.1:8077
ls -1 "$OUT"
