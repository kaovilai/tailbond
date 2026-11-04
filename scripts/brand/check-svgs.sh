#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The brand SVGs draw (docs/brand.md):
#   - the beam's gradient (id tb-beam) is in user space: on the mark's
#     horizontal beam an objectBoundingBox gradient has a zero-height box
#     and the beam isn't drawn at all;
#   - the dashboard's copies (agent/web/) are the masters byte for byte.
#
#   scripts/brand/check-svgs.sh
set -euo pipefail

cd "$(dirname "$0")/../.."
fail=0
while IFS= read -r f; do
    if grep -o '<linearGradient id="tb-beam"[^>]*>' "$f" | grep -qv 'gradientUnits="userSpaceOnUse"'; then
        echo "::error file=$f::tb-beam gradient without gradientUnits=\"userSpaceOnUse\" (the beam doesn't render): use x1=\"36\" y1=\"0\" x2=\"50\" y2=\"0\""
        fail=1
    fi
done < <(git ls-files '*.svg' '*.html' '*.js' | xargs grep -l 'id="tb-beam"' 2>/dev/null || true)
for pair in "agent/web/icon.svg assets/brand/tailbond-icon.svg" \
    "agent/web/wordmark-dark.svg assets/brand/tailbond-wordmark-dark.svg" \
    "agent/web/wordmark-light.svg assets/brand/tailbond-wordmark-light.svg"; do
    set -- $pair
    cmp -s "$1" "$2" || { echo "::error file=$1::$1 differs from $2: copy the master"; fail=1; }
done
[ "$fail" = 0 ] && echo "brand SVGs: beam gradients in user space, web copies match the masters"
exit "$fail"
