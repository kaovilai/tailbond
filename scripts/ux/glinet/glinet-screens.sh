#!/usr/bin/env bash
# Screenshots of the GL.iNet admin panel view (VPN -> tailbond) in
# a mock GL 4.x admin shell, every state in fixtures.mjs, desktop 1280x800
# and phone 390x844 → out/ux/glinet/<name>/ (default name: latest).
#
#   scripts/ux/glinet/glinet-screens.sh [round-N]
#   scripts/ux/glinet/glinet-screens.sh bypass-failover
#       bypass rules + more servers only → out/ux/bypass-failover/glinet/
#
# The view file is the real one from packaging/openwrt/gl-ui; the shell
# evaluates it like GL's app does and answers $rpcRequest from fixtures.
# Playwright + Vue 2 go into ~/.cache/tailbond-ux on first use; Chromium
# runs in the matching mcr.microsoft.com/playwright image (podman).
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../../.." && pwd)"
SET=""
if [[ "${1:-}" == bypass-failover ]]; then
    SET=bypass-failover
    OUT="$REPO/out/ux/bypass-failover/glinet"
else
    OUT="$REPO/out/ux/glinet/${1:-latest}"
fi
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"

if [[ ! -d "$PW/node_modules/playwright" || ! -d "$PW/node_modules/vue" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && { [ -f package.json ] || npm init -y >/dev/null; } && npm i playwright vue@2.6.14 >/dev/null)
fi
PW_VERSION="$(node -p "require('$PW/node_modules/playwright/package.json').version")"
mkdir -p "$OUT"
podman run --rm --network=none --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO:/repo:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules -w /repo/scripts/ux/glinet \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /repo/scripts/ux/glinet/glinet-screens.mjs /repo /out $SET
echo "$(find "$OUT" -name "*.png" | wc -l) screenshots in $OUT"
