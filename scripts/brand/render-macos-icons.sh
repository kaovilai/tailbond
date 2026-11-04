#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The macOS app icon, rendered at build time from the master SVG
# (assets/brand/tailbond-icon.svg, docs/brand.md); nothing generated is
# kept in git:
#
#   OUTDIR/tailbond_<size>[@2x].png   the iconset sizes (16 … 512, @1x/@2x)
#   OUTDIR/tailbond.icns              on a Mac (iconutil)
#
#   scripts/brand/render-macos-icons.sh OUTDIR
#
# Needs rsvg-convert (brew install librsvg; dnf install librsvg2-tools;
# apt install librsvg2-bin). Used by packaging/macos/build-bundle.sh and the
# Homebrew formula's --HEAD build.
set -euo pipefail

[ $# -eq 1 ] || { echo "usage: $0 OUTDIR" >&2; exit 2; }
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SVG="$ROOT/assets/brand/tailbond-icon.svg"
out="$1"
command -v rsvg-convert >/dev/null ||
    { echo "render-macos-icons: rsvg-convert missing (brew install librsvg)" >&2; exit 1; }
mkdir -p "$out"
for s in 16 32 128 256 512; do
    rsvg-convert -w "$s" -h "$s" "$SVG" -o "$out/tailbond_${s}x${s}.png"
    rsvg-convert -w $((s * 2)) -h $((s * 2)) "$SVG" -o "$out/tailbond_${s}x${s}@2x.png"
done
if command -v iconutil >/dev/null; then
    set_dir="$(mktemp -d)/tailbond.iconset"
    mkdir -p "$set_dir"
    for f in "$out"/tailbond_*.png; do cp "$f" "$set_dir/icon_${f##*/tailbond_}"; done
    iconutil -c icns "$set_dir" -o "$out/tailbond.icns"
    rm -rf "${set_dir%/*}"
fi
