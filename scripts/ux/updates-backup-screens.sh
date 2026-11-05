#!/usr/bin/env bash
# Screenshots of the Updates and Backup & restore cards on both surfaces,
# API answered from fixtures (scripts/ux/glinet/ub-fixtures.mjs):
#   out/ux/updates-backup/glinet/  GL admin panel (Slate 7, Flint 3 as home server)
#   out/ux/updates-backup/web/     the agent's dashboard (Fedora homebox, a VPS),
#                                  "Updates & backup" tab
# <name>-desktop.png (1280x800; pages full length, dialogs at window size)
# and <name>-phone.png (390x844) for a few. Manifests:
# docs/ux/updates-backup/{glinet,web}-screens.md.
#
#   scripts/ux/updates-backup-screens.sh [outdir]
#
# Playwright + Vue 2 go into ~/.cache/tailbond-ux on first use; Chromium
# runs in the matching mcr.microsoft.com/playwright image (podman).
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$REPO/out/ux/updates-backup}"
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"
PW_PINNED=1.64.1

# Playwright is pinned, and a cache holding another version is reinstalled:
# the npm version picks the container image tag, and a fresh release can have
# dependencies npm hasn't published yet (playwright-core@1.65.0: ETARGET).
if [[ "$(node -p "try{require('$PW/node_modules/playwright/package.json').version}catch(e){''}")" != "$PW_PINNED" || ! -d "$PW/node_modules/vue" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && { [ -f package.json ] || npm init -y >/dev/null; } && npm i --save-exact "playwright@$PW_PINNED" vue@2.6.14 >/dev/null)
fi
PW_VERSION="$PW_PINNED"
mkdir -p "$OUT"
podman run --rm --network=none --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO:/repo:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules -e TB_UX_ONLY="${TB_UX_ONLY:-}" -e TB_UX_DEBUG="${TB_UX_DEBUG:-}" -w /repo/scripts/ux \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /repo/scripts/ux/updates-backup-screens.mjs /repo /out
echo "$(find "$OUT" -name "*.png" | wc -l) screenshots in $OUT"
