#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The macOS release bundle and Homebrew formula (docs/macos.md):
#
#   out/macos/tailbond-macos-arm64.tar.gz   tailbond-macos-arm64/{bin/tailbondd,
#       bin/mqvpn, hooks/, install.sh, uninstall.sh, the LaunchDaemon plist,
#       icons/ (PNG sizes, tailbond.icns when built on a Mac; rendered from
#       assets/brand/tailbond-icon.svg, needs rsvg-convert), README.md,
#       VERSION, licenses}
#   out/macos/tailbond.rb                   packaging/macos/tailbond.rb with this
#       version and the tarball's SHA-256 (TB_FORMULA_URL: where it downloads
#       the tarball from; default the GitHub release of that version)
#
# Needs out/macos/mqvpn (scripts/build-mqvpn-macos.sh) and
# out/agent/darwin-arm64/tailbondd (agent/build.sh darwin/arm64). Runs on
# macOS or Linux. Env: VERSION (default git describe), TB_UPDATE_REPO.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/macos"
OUT="$ROOT/out/macos"
# shellcheck source=../../scripts/release/ldflags.sh
. "$ROOT/scripts/release/ldflags.sh"
VERSION="$(tb_norm_version "${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}")"
REPO="${TB_UPDATE_REPO:-kaovilai/tailbond}"
die() { echo "build-bundle: $*" >&2; exit 1; }

mq="$OUT/mqvpn"
tbd="$ROOT/out/agent/darwin-arm64/tailbondd"
[ -f "$mq" ] || die "$mq missing: scripts/build-mqvpn-macos.sh"
[ -f "$tbd" ] || die "$tbd missing: agent/build.sh darwin/arm64"
for b in "$mq" "$tbd"; do
    case "$(file -b "$b")" in
        *Mach-O*arm64*) ;;
        *) die "$b is not a macOS arm64 binary: $(file -b "$b")" ;;
    esac
done

name=tailbond-macos-arm64
top="$(mktemp -d)"
trap 'rm -rf "$top"' EXIT
d="$top/$name"
mkdir -p "$d/bin" "$d/hooks"
install -m 0755 "$mq" "$d/bin/mqvpn"
install -m 0755 "$tbd" "$d/bin/tailbondd"
install -m 0755 "$PKG/install.sh" "$PKG/uninstall.sh" "$d/"
install -m 0755 "$PKG"/hooks/* "$d/hooks/"
install -m 0644 "$PKG/com.tailbond.tailbondd.plist" "$d/"
# the app icon (docs/brand.md), rendered from the SVG here (rsvg-convert):
# the PNG sizes, and tailbond.icns on a Mac (iconutil)
"$ROOT/scripts/brand/render-macos-icons.sh" "$d/icons"
install -m 0644 "$ROOT/docs/macos.md" "$d/README.md"
install -m 0644 "$ROOT/LICENSE" "$d/LICENSE"
install -m 0644 "$ROOT/NOTICE" "$d/NOTICE"
install -m 0644 "$ROOT/third_party/mqvpn/LICENSE" "$d/LICENSE.mqvpn"
install -m 0644 "$ROOT/third_party/mqvpn/NOTICE" "$d/NOTICE.mqvpn"
echo "$VERSION" >"$d/VERSION"
mkdir -p "$OUT"
# root-owned entries whichever tar this is (bsdtar on macOS, GNU on Linux)
if tar --version 2>/dev/null | grep -q bsdtar; then
    tar -C "$top" --uid 0 --gid 0 -czf "$OUT/$name.tar.gz" "$name"
else
    tar -C "$top" --owner=0 --group=0 -czf "$OUT/$name.tar.gz" "$name"
fi
if command -v sha256sum >/dev/null; then sum="$(sha256sum "$OUT/$name.tar.gz" | cut -d' ' -f1)"
else sum="$(shasum -a 256 "$OUT/$name.tar.gz" | cut -d' ' -f1)"; fi
url="${TB_FORMULA_URL:-https://github.com/$REPO/releases/download/v$VERSION/$name.tar.gz}"
sed -e "s|^  version \".*\"|  version \"$VERSION\"|" \
    -e "s|^  url \".*\"|  url \"$url\"|" \
    -e "s|^  sha256 \".*\"|  sha256 \"$sum\"|" \
    -e "s|https://github.com/kaovilai/tailbond|https://github.com/$REPO|g" \
    "$PKG/tailbond.rb" >"$OUT/tailbond.rb"
grep -q "sha256 \"$sum\"" "$OUT/tailbond.rb" || die "couldn't fill in the formula"
echo "$OUT/$name.tar.gz ($(du -h "$OUT/$name.tar.gz" | cut -f1)), $OUT/tailbond.rb ($VERSION)"
