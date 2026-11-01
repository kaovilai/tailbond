#!/usr/bin/env bash
# Screenshots of the agent web UI's "Add this phone" flow at phone width
# (headless Chromium via Playwright) → out/ux/web/.
#
#   scripts/ux/web-screens.sh [outdir]
#
# Runs a throwaway server-role tailbondd (a stand-in mqvpn that only reports
# "ready", no Tailscale) on
# 127.0.0.1:18077. The Playwright npm package goes into ~/.cache/tailbond-ux on
# first use; Chromium runs in the matching mcr.microsoft.com/playwright image
# (podman), so the host needs no browser libraries.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$REPO/out/ux/web}"
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"
TMP="$(mktemp -d)"
trap 'kill "${AGENT_PID:-0}" 2>/dev/null || true; rm -rf "$TMP"' EXIT

if [[ ! -d "$PW/node_modules/playwright" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && npm init -y >/dev/null && npm i playwright >/dev/null)
fi
PW_VERSION="$(node -p "require('$PW/node_modules/playwright/package.json').version")"

(cd "$REPO/agent" && CGO_ENABLED=0 go build -o "$TMP/tailbondd" ./cmd/tailbondd)
mkdir -p "$TMP/state" "$TMP/hooks" "$OUT"
printf 'PK\003\004' > "$TMP/tailbond.apk"  # offered as GET /tailbond.apk
printf '#!/bin/sh\necho "mqvpn server ready"\nexec sleep 3600\n' > "$TMP/mqvpn" && chmod +x "$TMP/mqvpn"
cat > "$TMP/tailbond.json" <<JSON
{"role":"server","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/state","hooks_dir":"$TMP/hooks",
 "listen":"127.0.0.1:18077","disable_tailscale":true,"android_apk":"$TMP/tailbond.apk",
 "server":{"port":443,"public_addr":"home.example.net:443","subnet":"10.77.0.0/24","subnet6":"fd77:77::/112","use_tailscale_cert":false}}
JSON
"$TMP/tailbondd" -config "$TMP/tailbond.json" -quiet &
AGENT_PID=$!
for _ in $(seq 50); do curl -fsS -o /dev/null http://127.0.0.1:18077/ && break; sleep 0.2; done

podman run --rm --network=host --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO/scripts/ux:/scripts:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /scripts/web-screens.mjs http://127.0.0.1:18077/ /out
ls -1 "$OUT"
