#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Before/after pictures of the brand pass: out/ux/brand/{before,after}/<rel>
# side by side → out/ux/brand/compare/<name>.png (ImageMagick in an Alpine
# container). Pairs: "<name> <rel>" lines on stdin.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
B="$ROOT/out/ux/brand"
mkdir -p "$B/compare"
pairs="$(mktemp)"; trap 'rm -f "$pairs"' EXIT
cat >"$pairs"
podman run --rm -v "$B:/b:z" -v "$pairs:/pairs.txt:ro,z" docker.io/library/alpine:3.20 sh -euc '
apk add --no-cache imagemagick font-inter >/dev/null
F=/usr/share/fonts/inter/Inter-Regular.otf
while read -r name rel; do
  [ -f "/b/before/$rel" ] && [ -f "/b/after/$rel" ] || { echo "skip $name ($rel)"; continue; }
  magick \( "/b/before/$rel" -resize "x1000>" -bordercolor "#071A1D" -border 12 \) \
         \( "/b/after/$rel" -resize "x1000>" -bordercolor "#071A1D" -border 12 \) +append \
         -background "#071A1D" -gravity North -splice 0x52 \
         -fill "#E6FFFA" -font "$F" -pointsize 28 -annotate +0+10 "before  ·  after" "/b/compare/$name.png"
  echo "$name"
done </pairs.txt'
