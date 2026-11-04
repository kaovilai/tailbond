#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# tailbond.app, the menu bar app (macos/TailbondMenu, docs/macos.md "Menu
# bar"), ad-hoc signed (no Apple Developer ID: fine for software you
# install yourself):
#
#   packaging/macos/build-app.sh      → out/macos/tailbond.app
#
# On a Mac with Xcode (or the command line tools with Swift 5.9+) and
# rsvg-convert (brew install librsvg) for the icon. Env: VERSION (default
# git describe). packaging/macos/build-bundle.sh puts it in the release
# bundle; install.sh copies it to /Applications.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="$ROOT/out/macos"
# shellcheck source=../../scripts/release/ldflags.sh
. "$ROOT/scripts/release/ldflags.sh"
VERSION="$(tb_norm_version "${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}")"
die() { echo "build-app: $*" >&2; exit 1; }
[ "$(uname -s)" = Darwin ] || die "build it on a Mac"

pkg="$ROOT/macos/TailbondMenu"
swift build --package-path "$pkg" -c release --arch arm64 --product tailbond
bin="$(swift build --package-path "$pkg" -c release --arch arm64 --show-bin-path)/tailbond"
[ -x "$bin" ] || die "no $bin"

app="$OUT/tailbond.app"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
install -m 0755 "$bin" "$app/Contents/MacOS/tailbond"
icons="$(mktemp -d)"
trap 'rm -rf "$icons"' EXIT
"$ROOT/scripts/brand/render-macos-icons.sh" "$icons"
[ -f "$icons/tailbond.icns" ] || die "no tailbond.icns (iconutil)"
install -m 0644 "$icons/tailbond.icns" "$app/Contents/Resources/tailbond.icns"
# CFBundleVersion: digits and dots only
build="$(printf '%s' "$VERSION" | sed 's/[^0-9.].*//')"
cat >"$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>tailbond</string>
  <key>CFBundleDisplayName</key><string>tailbond</string>
  <key>CFBundleIdentifier</key><string>dev.tailbond.menu</string>
  <key>CFBundleExecutable</key><string>tailbond</string>
  <key>CFBundleIconFile</key><string>tailbond</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>${build:-0}</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>LSUIElement</key><true/>
  <key>LSApplicationCategoryType</key><string>public.app-category.utilities</string>
  <key>NSHumanReadableCopyright</key><string>Apache-2.0</string>
  <key>NSAppTransportSecurity</key><dict><key>NSAllowsLocalNetworking</key><true/></dict>
</dict>
</plist>
PLIST
plutil -lint "$app/Contents/Info.plist" >/dev/null
# ad-hoc signature: the whole bundle, so macOS checks it as one app
codesign --force --sign - --timestamp=none "$app"
codesign --verify --strict "$app" || die "the signature doesn't verify"
echo "$app ($VERSION, ad-hoc signed)"
