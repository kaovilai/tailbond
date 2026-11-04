#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build mqvpn for macOS (arm64) with tailbond's patches applied, on a Mac
# (GitHub's macos-latest runner in CI; docs/macos.md).
#
# Like scripts/build-mqvpn-native.sh: the pinned third_party/mqvpn is copied
# to a work tree and patches/mqvpn/*.patch are applied there. BoringSSL,
# xquic and mqvpn are built with upstream's macOS flags (its ci.yml darwin
# job); libevent comes from Homebrew and is linked statically, so the binary
# runs on a Mac without Homebrew.
#
# Output: out/macos/mqvpn
#
# Needs: Xcode command line tools, cmake, Homebrew libevent and GNU patch
# (brew install cmake libevent gpatch).
#
# Env:
#   TB_WORK_DIR  work tree (default out/build/mqvpn-macos; kept between runs)
#   TB_JOBS      make parallelism (default: all cores)
#   TB_TEST=1    also build and run mqvpn's Darwin unit tests (ctest)
#   TB_CLEAN=1   wipe the work tree first
#   LIBEVENT_PREFIX  libevent's prefix (default: brew --prefix libevent)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${TB_WORK_DIR:-$ROOT/out/build/mqvpn-macos}"
JOBS="${TB_JOBS:-$(sysctl -n hw.ncpu 2>/dev/null || echo 4)}"
ARCH="${TB_ARCH:-arm64}"

die() { echo "build-mqvpn-macos: $*" >&2; exit 1; }
# shellcheck source-path=SCRIPTDIR source=lib/mqvpn-src.sh
. "$ROOT/scripts/lib/mqvpn-src.sh"

[ "$(uname -s)" = Darwin ] || die "run this on macOS"
for c in cmake make cc c++; do
    command -v "$c" >/dev/null || die "'$c' not found (xcode-select --install; brew install cmake)"
done
# LIBEVENT_PREFIX: set by the Homebrew formula's source build
LIBEVENT="${LIBEVENT_PREFIX:-$(brew --prefix libevent 2>/dev/null || true)}"
[ -n "$LIBEVENT" ] || die "brew install libevent"
[ -f "$LIBEVENT/lib/libevent.a" ] || die "no $LIBEVENT/lib/libevent.a (brew reinstall libevent)"

[ "${TB_CLEAN:-0}" = 1 ] && rm -rf "$WORK"
mqvpn_src_prepare "$WORK"
SRC="$WORK/src"
BSSL="$SRC/third_party/xquic/third_party/boringssl"
XQ="$SRC/third_party/xquic"

echo "=== BoringSSL ($ARCH) ==="
# CMAKE_BUILD_TYPE is mandatory (upstream build.sh: an unset type builds -O0)
[ -f "$BSSL/build/CMakeCache.txt" ] || cmake -S "$BSSL" -B "$BSSL/build" \
    -DBUILD_SHARED_LIBS=0 -DCMAKE_BUILD_TYPE=Release -DCMAKE_OSX_ARCHITECTURES="$ARCH"
make -C "$BSSL/build" -j"$JOBS" ssl crypto

echo "=== xquic ==="
# -Wno-unknown-warning-option: xquic passes gcc-only warning flags with
# -Werror; AppleClang rejects them otherwise (upstream ci.yml darwin job).
[ -f "$XQ/build/CMakeCache.txt" ] || cmake -S "$XQ" -B "$XQ/build" \
    -DCMAKE_BUILD_TYPE=Release -DSSL_TYPE=boringssl -DSSL_PATH="$BSSL" \
    -DXQC_ENABLE_BBR2=ON -DXQC_ENABLE_UNLIMITED=ON -DXQC_ENABLE_FEC=ON -DXQC_ENABLE_XOR=ON \
    -DCMAKE_OSX_ARCHITECTURES="$ARCH" -DCMAKE_C_FLAGS="-Wno-unknown-warning-option"
make -C "$XQ/build" -j"$JOBS"

echo "=== mqvpn ==="
# EVENT_LIB preset to the static archive: find_library would pick the dylib.
[ -f "$SRC/build/CMakeCache.txt" ] || cmake -S "$SRC" -B "$SRC/build" \
    -DCMAKE_BUILD_TYPE=Release -DXQUIC_BUILD_DIR="$XQ/build" \
    -DCMAKE_OSX_ARCHITECTURES="$ARCH" \
    -DEVENT_INCLUDE_DIR="$LIBEVENT/include" -DEVENT_LIB="$LIBEVENT/lib/libevent.a" \
    -DBUILD_TESTING="$([ "${TB_TEST:-0}" = 1 ] && echo ON || echo OFF)" \
    -DCMAKE_C_FLAGS="-Wall -Wextra -Werror"
if [ "${TB_TEST:-0}" = 1 ]; then
    cmake --build "$SRC/build" -j "$JOBS"
    echo "=== ctest (mqvpn's Darwin unit tests) ==="
    ctest --test-dir "$SRC/build" --no-tests=error --output-on-failure -j "$JOBS"
else
    cmake --build "$SRC/build" --target mqvpn -j "$JOBS"
fi

mkdir -p "$ROOT/out/macos"
install -m 0755 "$SRC/build/mqvpn" "$ROOT/out/macos/mqvpn"
echo "Built: $ROOT/out/macos/mqvpn"
file "$ROOT/out/macos/mqvpn"
# no Homebrew dylib may leak in (the binary must run without brew)
if otool -L "$ROOT/out/macos/mqvpn" | grep -E '/(opt/homebrew|usr/local)/'; then
    die "mqvpn links a Homebrew library dynamically"
fi
"$ROOT/out/macos/mqvpn" --help 2>&1 | head -1 || true
