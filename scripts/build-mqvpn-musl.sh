#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build a fully static (musl) mqvpn with tailbond's patches applied.
# BoringSSL, xquic and libevent are linked statically; the result runs on
# any Linux kernel of that architecture, no libs needed (OpenWrt / GL.iNet
# routers for aarch64, any VPS distro for x86_64).
#
#   scripts/build-mqvpn-musl.sh aarch64     # -> out/aarch64/mqvpn
#   scripts/build-mqvpn-musl.sh x86_64      # -> out/x86_64/mqvpn
#
# Builds natively inside docker.io/library/alpine for the target platform
# (podman). On a host of another architecture it runs under qemu-user-static
# emulation (binfmt_misc): slow (tens of minutes cold) but needs no cross
# toolchain. x86_64 on an x86_64 host (CI, most laptops) is native and fast.
# The work tree keeps the build dirs, so later runs only rebuild what changed.
#
# Env:
#   TB_WORK_DIR      work tree (default out/build/mqvpn-<arch>)
#   TB_JOBS          make parallelism (default 4)
#   TB_ALPINE_IMAGE  default docker.io/library/alpine:3.22
#   TB_CLEAN=1       wipe the work tree first
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARCH="${1:-}"

die() { echo "build-mqvpn-musl: $*" >&2; exit 1; }

case "$ARCH" in
    aarch64|arm64) ARCH=aarch64; PLATFORM=linux/arm64; FILE_ARCH="ARM aarch64" ;;
    x86_64|amd64)  ARCH=x86_64;  PLATFORM=linux/amd64; FILE_ARCH="x86-64" ;;
    *) die "usage: $0 aarch64|x86_64" ;;
esac

WORK="${TB_WORK_DIR:-$ROOT/out/build/mqvpn-$ARCH}"
JOBS="${TB_JOBS:-4}"
IMAGE="${TB_ALPINE_IMAGE:-docker.io/library/alpine:3.22}"
OUT="$ROOT/out/$ARCH"

# shellcheck source-path=SCRIPTDIR source=lib/mqvpn-src.sh
. "$ROOT/scripts/lib/mqvpn-src.sh"

command -v podman >/dev/null || die "podman not found"

# Emulation, only for a foreign architecture: the kernel must hand the
# target's ELF binaries to qemu. The F flag (fix-binary) lets containers use
# the host's interpreter.
RUN=()
if [ "$(uname -m)" != "$ARCH" ]; then
    binfmt="/proc/sys/fs/binfmt_misc/qemu-$ARCH"
    if [ ! -e "$binfmt" ] || ! grep -q '^enabled' "$binfmt"; then
        echo "qemu-$ARCH binfmt handler not registered; trying to register it (sudo)"
        command -v "qemu-$ARCH-static" >/dev/null \
            || die "install qemu-user-static (dnf install qemu-user-static)"
        sudo systemctl restart systemd-binfmt.service 2>/dev/null || true
        [ -e "$binfmt" ] || die "binfmt registration failed; see qemu-user-static docs"
    fi
    qemu="qemu-$ARCH-static"
    command -v "$qemu" >/dev/null || qemu="qemu-$ARCH"
    RUN=("$qemu")
fi

[ "${TB_CLEAN:-0}" = 1 ] && rm -rf "$WORK"
mqvpn_src_prepare "$WORK"

echo "=== Building in $IMAGE ($PLATFORM, make -j$JOBS) ==="
podman run --rm --platform "$PLATFORM" \
    -e JOBS="$JOBS" -e BDIR="build-$ARCH" \
    -v "$WORK/src:/w:Z" -w /w \
    "$IMAGE" /bin/sh -euc '
apk add --no-cache build-base cmake linux-headers perl \
    libevent-dev libevent-static file >/dev/null

BSSL=/w/third_party/xquic/third_party/boringssl
XQ=/w/third_party/xquic

echo "--- BoringSSL"
[ -f "$BSSL/build/CMakeCache.txt" ] || cmake -S "$BSSL" -B "$BSSL/build" \
    -DBUILD_SHARED_LIBS=0 -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_C_FLAGS=-fPIC -DCMAKE_CXX_FLAGS=-fPIC >/dev/null
make -s -C "$BSSL/build" -j"$JOBS" ssl crypto

echo "--- xquic (FEC/XOR/UNLIMITED/BBR2 as in upstream build.sh)"
[ -f "$XQ/build/CMakeCache.txt" ] || cmake -S "$XQ" -B "$XQ/build" \
    -DCMAKE_BUILD_TYPE=Release -DSSL_TYPE=boringssl -DSSL_PATH="$BSSL" \
    -DXQC_ENABLE_BBR2=ON -DXQC_ENABLE_UNLIMITED=ON \
    -DXQC_ENABLE_FEC=ON -DXQC_ENABLE_XOR=ON >/dev/null
make -s -C "$XQ/build" -j"$JOBS" xquic-static

echo "--- mqvpn (static)"
[ -f "/w/$BDIR/CMakeCache.txt" ] || cmake -S /w -B "/w/$BDIR" \
    -DCMAKE_BUILD_TYPE=Release -DXQUIC_BUILD_DIR="$XQ/build" \
    -DBUILD_TESTING=OFF \
    -DEVENT_LIB=/usr/lib/libevent.a \
    -DCMAKE_EXE_LINKER_FLAGS="-static" >/dev/null
make -s -C "/w/$BDIR" -j"$JOBS" mqvpn
strip -o "/w/$BDIR/mqvpn.stripped" "/w/$BDIR/mqvpn"
file "/w/$BDIR/mqvpn.stripped"
'

mkdir -p "$OUT"
install -m 0755 "$WORK/src/build-$ARCH/mqvpn.stripped" "$OUT/mqvpn"

# Verify: a static ELF of the right architecture that runs (under qemu for a
# foreign one).
info="$(file -b "$OUT/mqvpn")"
echo "$info"
case "$info" in
    *"$FILE_ARCH"*"statically linked"*|*"$FILE_ARCH"*"static-pie linked"*) ;;
    *) die "$OUT/mqvpn is not a static $ARCH binary: $info" ;;
esac
# --help exits non-zero; check that it printed the usage text.
help="$("${RUN[@]}" "$OUT/mqvpn" --help 2>&1 || true)"
case "$help" in *Usage:*) echo "$help" | head -3 ;; *) die "binary does not run${RUN[*]:+ under ${RUN[*]}}: $help" ;; esac
echo "Built: $OUT/mqvpn ($(du -h "$OUT/mqvpn" | cut -f1))"
