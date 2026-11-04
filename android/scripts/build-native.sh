#!/usr/bin/env bash
# Cross-compile libmqvpn + xquic + BoringSSL with mqvpn's own script, per ABI
# (default arm64-v8a; TB_ANDROID_ABIS="arm64-v8a x86_64" for more, x86_64 is
# the emulator's), from a patched copy of third_party/mqvpn
# (scripts/lib/mqvpn-src.sh: patches/mqvpn/*.patch, e.g. 0002 + 0004's relay
# paths for the Pair & Share hub). Output: out/mqvpn-android/src, whose
# android/sdk-native/prebuilt/<abi>/*.a, headers and reactor.c
# android/mqvpn-sdk/sdk-native builds against. Prebuilt libs are reused
# while the patches are unchanged (.patches stamp).
#
#   [TB_ANDROID_ABIS=...] android/scripts/build-native.sh [--force]
#
# Parallelism is capped at 4 (shared build host): build_android.sh uses
# $(nproc), and GNU nproc honours OMP_NUM_THREADS.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
WORK="$REPO_ROOT/out/mqvpn-android"
MQVPN="$WORK/src"
ABIS="${TB_ANDROID_ABIS:-arm64-v8a}"
# shellcheck disable=SC2034 # read by mqvpn-src.sh
ROOT="$REPO_ROOT"
die() { echo "build-native: $*" >&2; exit 1; }
# shellcheck source=../../scripts/lib/mqvpn-src.sh
. "$REPO_ROOT/scripts/lib/mqvpn-src.sh"
[[ -f "$REPO_ROOT/third_party/mqvpn/third_party/xquic/CMakeLists.txt" ]] || \
    git -C "$REPO_ROOT" submodule update --init --recursive
mqvpn_src_prepare "$WORK"
STAMP="$(cat "$REPO_ROOT"/patches/mqvpn/*.patch | sha256sum | cut -c1-16)"
ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
export ANDROID_NDK="${ANDROID_NDK:-$ANDROID_HOME/ndk/28.2.13676358}"
export OMP_NUM_THREADS="${OMP_NUM_THREADS:-4}"

for ABI in ${ABIS//,/ }; do
    PREBUILT="$MQVPN/android/sdk-native/prebuilt/$ABI"
    if [[ "${1:-}" != "--force" && -f "$PREBUILT/libmqvpn.a" && -f "$PREBUILT/libssl.a" \
          && -f "$PREBUILT/libxquic.a" && -f "$PREBUILT/liblwip_core.a" \
          && "$(cat "$PREBUILT/.patches" 2>/dev/null)" = "$STAMP" ]]; then
        echo "prebuilt libs present in $PREBUILT for these patches (use --force to rebuild)"
        continue
    fi
    "$MQVPN/scripts/build_android.sh" --abi "$ABI" --ndk "$ANDROID_NDK"
    echo "$STAMP" >"$PREBUILT/.patches"
done
