#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Render the brand's PNGs from the master SVGs in assets/brand/
# (docs/brand.md), with rsvg-convert and the Inter font in an Alpine
# container (podman), so every machine renders them the same way:
#
#   ios/TailbondShare/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png
#       the iOS app icon: the icon on a full-bleed ink square (iOS rounds it)
#   assets/brand/social-preview.png
#       1280x640 GitHub social preview: the wordmark on ink
#
# These two are the only brand rasters in git (docs/brand.md, Rasters):
# Xcode's asset catalog needs the PNG, and GitHub's settings take the
# social preview as a file. The macOS icons are rendered at build time
# (scripts/brand/render-macos-icons.sh), not kept.
#
#   scripts/brand/render-icons.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PODMAN="${PODMAN:-podman}"
IMAGE="${TB_ALPINE_IMAGE:-docker.io/library/alpine:3.20}"
B="$ROOT/assets/brand"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# iOS: no transparency and no corner radius of our own
sed 's/ rx="15"//' "$B/tailbond-icon.svg" >"$TMP/ios-icon.svg"
# the social preview: the dark wordmark centred on ink, a tagline below
wm="$(sed -e '1d' -e '$d' "$B/tailbond-wordmark-dark.svg")"
cat >"$TMP/social.svg" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 640" width="1280" height="640">
  <rect width="1280" height="640" fill="#071A1D"/>
  <rect x="0" y="616" width="1280" height="24" fill="url(#bar)"/>
  <defs><linearGradient id="bar" x1="0" x2="1"><stop offset="0" stop-color="#2DD4BF"/><stop offset="1" stop-color="#C6F432"/></linearGradient></defs>
  <g transform="translate(250 196) scale(3)">$wm</g>
  <text x="640" y="470" text-anchor="middle" font-family="Inter, sans-serif" font-size="38" fill="#8FB8B2">Many connections, one tunnel to your own server</text>
</svg>
EOF

"$PODMAN" run --rm -v "$TMP:/in:z" -v "$ROOT:/repo:z" "$IMAGE" sh -euc '
  apk add --no-cache rsvg-convert font-inter >/dev/null
  set -x
  out=/repo/ios/TailbondShare/Assets.xcassets/AppIcon.appiconset
  mkdir -p "$out"
  rsvg-convert -w 1024 -h 1024 -b "#071A1D" /in/ios-icon.svg -o "$out/AppIcon-1024.png"
  rsvg-convert -w 1280 -h 640 /in/social.svg -o /repo/assets/brand/social-preview.png
'
cat >"$ROOT/ios/TailbondShare/Assets.xcassets/AppIcon.appiconset/Contents.json" <<'EOF'
{
  "images" : [
    {
      "filename" : "AppIcon-1024.png",
      "idiom" : "universal",
      "platform" : "ios",
      "size" : "1024x1024"
    }
  ],
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
EOF
cat >"$ROOT/ios/TailbondShare/Assets.xcassets/Contents.json" <<'EOF'
{
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
EOF
echo "rendered: iOS AppIcon, social preview"
