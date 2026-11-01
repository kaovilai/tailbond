#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build a fully static aarch64 (musl) mqvpn for OpenWrt / GL.iNet routers,
# with tailbond's patches applied. BoringSSL, xquic and libevent are linked
# statically; the result runs on any aarch64 Linux kernel, no libs needed.
#
# Builds natively inside docker.io/library/alpine for linux/arm64 under
# qemu-user-static emulation (binfmt_misc). Slow (tens of minutes cold) but
# needs no cross toolchain; the work tree keeps the build dirs, so later runs
# only rebuild what changed.
#
# Output: out/aarch64/mqvpn (stripped).
#
# Env:
#   TB_WORK_DIR      work tree (default out/build/mqvpn-aarch64)
#   TB_JOBS          make parallelism (default 4)
#   TB_ALPINE_IMAGE  default docker.io/library/alpine:3.22
#   TB_CLEAN=1       wipe the work tree first
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${TB_WORK_DIR:-$ROOT/out/build/mqvpn-aarch64}"
JOBS="${TB_JOBS:-4}"
IMAGE="${TB_ALPINE_IMAGE:-docker.io/library/alpine:3.22}"
OUT="$ROOT/out/aarch64"

die() { echo "build-mqvpn-aarch64-musl: $*" >&2; exit 1; }
# shellcheck source-path=SCRIPTDIR source=lib/mqvpn-src.sh
. "$ROOT/scripts/lib/mqvpn-src.sh"

command -v podman >/dev/null || die "podman not found"

# Emulation: the kernel must hand aarch64 ELF binaries to qemu. The F flag
# (fix-binary) lets containers use the host's interpreter.
binfmt=/proc/sys/fs/binfmt_misc/qemu-aarch64
if [ ! -e "$binfmt" ] || ! grep -q '^enabled' "$binfmt"; then
    echo "qemu-aarch64 binfmt handler not registered; trying to register it (sudo)"
    command -v qemu-aarch64-static >/dev/null \
        || die "install qemu-user-static (dnf install qemu-user-static)"
    sudo systemctl restart systemd-binfmt.service 2>/dev/null || true
    [ -e "$binfmt" ] || die "binfmt registration failed; see qemu-user-static docs"
fi

[ "${TB_CLEAN:-0}" = 1 ] && rm -rf "$WORK"
mqvpn_src_prepare "$WORK"

echo "=== Building in $IMAGE (linux/arm64, make -j$JOBS) ==="
podman run --rm --platform linux/arm64 \
    -e JOBS="$JOBS" \
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
[ -f /w/build-aarch64/CMakeCache.txt ] || cmake -S /w -B /w/build-aarch64 \
    -DCMAKE_BUILD_TYPE=Release -DXQUIC_BUILD_DIR="$XQ/build" \
    -DBUILD_TESTING=OFF \
    -DEVENT_LIB=/usr/lib/libevent.a \
    -DCMAKE_EXE_LINKER_FLAGS="-static" >/dev/null
make -s -C /w/build-aarch64 -j"$JOBS" mqvpn
strip -o /w/build-aarch64/mqvpn.stripped /w/build-aarch64/mqvpn
file /w/build-aarch64/mqvpn.stripped
'

mkdir -p "$OUT"
install -m 0755 "$WORK/src/build-aarch64/mqvpn.stripped" "$OUT/mqvpn"

# Verify: static aarch64 ELF that runs (under qemu here).
info="$(file -b "$OUT/mqvpn")"
echo "$info"
case "$info" in
    *"ARM aarch64"*"statically linked"*) ;;
    *) die "out/aarch64/mqvpn is not a static aarch64 binary: $info" ;;
esac
qemu="qemu-aarch64-static"
command -v "$qemu" >/dev/null || qemu="qemu-aarch64"
# --help exits non-zero; check that it printed the usage text.
help="$("$qemu" "$OUT/mqvpn" --help 2>&1 || true)"
case "$help" in *Usage:*) echo "$help" | head -3 ;; *) die "binary does not run under $qemu: $help" ;; esac
echo "Built: $OUT/mqvpn ($(du -h "$OUT/mqvpn" | cut -f1))"
