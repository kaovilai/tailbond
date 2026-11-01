#!/usr/bin/env bash
# Render one review round of every surface into out/ux/final/<round>/
# (web/, glinet/, android/) and assemble the reviewers' packets.
#
#   scripts/ux/final-renders.sh round-N
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
R="$REPO/out/ux/final/${1:?round name, e.g. round-8}"
rm -rf "$R" "$REPO/out/ux/glinet/final-$1"
mkdir -p "$R"

"$REPO/scripts/ux/web-screens.sh" "$R/web" >/dev/null
"$REPO/scripts/ux/glinet/glinet-screens.sh" "final-$1" >/dev/null
mv "$REPO/out/ux/glinet/final-$1" "$R/glinet"
"$REPO/android/scripts/gradle.sh" --console=plain -q :app:testDebugUnitTest --tests 'dev.tailbond.app.ux.*' --rerun-tasks \
    -Ptailbond.uxOut="$R/android" >/dev/null
"$REPO/scripts/ux/final-packets.sh" "$R"
