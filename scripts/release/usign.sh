#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# OpenWrt's usign (the tool opkg checks feed signatures with), built from a
# pinned commit of its source: the opkg feed (docs/glinet.md, "Install and
# update from GL's Plug-ins page") is signed with the same program the
# router verifies it with. Prints the binary's path; builds it once.
#
#   USIGN="$(scripts/release/usign.sh)"
#
# Env: USIGN (an existing usign binary: used as is), TB_TOOLS (where to
# build, default out/tools), CC (default cc). Needs git and a C compiler;
# usign has no other dependency.
set -euo pipefail

# git.openwrt.org/project/usign.git master, 2025-10-03 ("CMakeLists: update
# cmake minimum required version to 3.10"); the code is that of 2020-05-23
# ("Always pad fingerprints to 16 characters"), which OpenWrt 23.05 ships.
USIGN_COMMIT=c4c72b1b07945ee192361dc751291a7c98d6adcd
USIGN_REPOS="https://git.openwrt.org/project/usign.git https://github.com/openwrt/usign.git"

die() { echo "usign.sh: $*" >&2; exit 1; }

if [ -n "${USIGN:-}" ]; then
    [ -x "$USIGN" ] || die "USIGN=$USIGN is not an executable"
    echo "$USIGN"
    exit 0
fi

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TOOLS="${TB_TOOLS:-$ROOT/out/tools}"
bin="$TOOLS/usign-${USIGN_COMMIT:0:12}"
if [ -x "$bin" ]; then
    echo "$bin"
    exit 0
fi

command -v git >/dev/null || die "git not found"
CC="${CC:-cc}"
command -v "$CC" >/dev/null || die "no C compiler ($CC): install gcc or clang, or set USIGN"
mkdir -p "$TOOLS"
src="$(mktemp -d "${TMPDIR:-/tmp}/usign-src.XXXXXX")"
trap 'rm -rf "$src"' EXIT
got=0
for url in $USIGN_REPOS; do
    rm -rf "$src/usign"
    if git clone -q "$url" "$src/usign" 2>/dev/null && git -C "$src/usign" checkout -q "$USIGN_COMMIT" 2>/dev/null; then
        got=1
        break
    fi
done
[ "$got" = 1 ] || die "can't fetch usign $USIGN_COMMIT from: $USIGN_REPOS"
# the commit id covers the whole tree: this is exactly the pinned source
[ "$(git -C "$src/usign" rev-parse HEAD)" = "$USIGN_COMMIT" ] || die "usign checkout isn't $USIGN_COMMIT"
(cd "$src/usign" && "$CC" -O2 -std=gnu99 -Wall -o "$src/usign-bin" \
    ed25519.c edsign.c f25519.c fprime.c sha512.c main.c base64.c) >&2 || die "building usign failed"
mv "$src/usign-bin" "$bin"
echo "$bin"
