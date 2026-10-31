#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build mqvpn natively (Fedora) with tailbond's patches applied.
#
# The submodule (third_party/mqvpn, pinned upstream tag) is never modified:
# it is copied to a work tree, patches/mqvpn/*.patch are applied there, and
# upstream's ./build.sh builds BoringSSL + xquic + mqvpn in it.
#
# Output: third_party/mqvpn/build/mqvpn (the path docs/architecture.md
# promises; build/ is gitignored upstream) and out/native/mqvpn.
#
# Env:
#   TB_WORK_DIR  work tree (default out/build/mqvpn-native; build dirs inside
#                it are kept between runs, so rebuilds are incremental)
#   TB_JOBS      make parallelism (default 4)
#   TB_TEST=1    also run upstream's ctest suite in the work tree
#   TB_CLEAN=1   wipe the work tree first
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/third_party/mqvpn"
WORK="${TB_WORK_DIR:-$ROOT/out/build/mqvpn-native}"
JOBS="${TB_JOBS:-4}"

die() { echo "build-mqvpn-native: $*" >&2; exit 1; }
# shellcheck source-path=SCRIPTDIR source=lib/mqvpn-src.sh
. "$ROOT/scripts/lib/mqvpn-src.sh"

for c in cmake make cc c++; do
    command -v "$c" >/dev/null || die "'$c' not found (dnf install cmake make gcc gcc-c++ libevent-devel)"
done

[ "${TB_CLEAN:-0}" = 1 ] && rm -rf "$WORK"
mqvpn_src_prepare "$WORK"

# upstream build.sh uses $(nproc) for -j; cap it without editing build.sh.
shim="$WORK/.shim"
mkdir -p "$shim"
printf '#!/bin/sh\necho %s\n' "$JOBS" >"$shim/nproc"
chmod +x "$shim/nproc"

echo "=== Building (make -j$JOBS) ==="
PATH="$shim:$PATH" "$WORK/src/build.sh"

if [ "${TB_TEST:-0}" = 1 ]; then
    echo "=== ctest ==="
    (cd "$WORK/src/build" && ctest -j"$JOBS" --output-on-failure)
fi

mkdir -p "$ROOT/out/native" "$SRC/build"
install -m 0755 "$WORK/src/build/mqvpn" "$ROOT/out/native/mqvpn"
install -m 0755 "$WORK/src/build/mqvpn" "$SRC/build/mqvpn"
echo "Built: $ROOT/out/native/mqvpn (also $SRC/build/mqvpn)"
"$ROOT/out/native/mqvpn" --help 2>&1 | head -1 || true
