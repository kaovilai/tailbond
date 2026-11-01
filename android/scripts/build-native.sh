#!/usr/bin/env bash
# Cross-compile libmqvpn + xquic + BoringSSL for arm64-v8a with mqvpn's own
# script. Output: third_party/mqvpn/android/sdk-native/prebuilt/arm64-v8a/*.a
# (gitignored inside the submodule), which android/mqvpn-sdk/sdk-native links.
#
#   android/scripts/build-native.sh [--force]
#
# Parallelism is capped at 4 (shared build host): build_android.sh uses
# $(nproc), and GNU nproc honours OMP_NUM_THREADS.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MQVPN="$REPO_ROOT/third_party/mqvpn"
ABI=arm64-v8a
ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
export ANDROID_NDK="${ANDROID_NDK:-$ANDROID_HOME/ndk/28.2.13676358}"
export OMP_NUM_THREADS="${OMP_NUM_THREADS:-4}"

PREBUILT="$MQVPN/android/sdk-native/prebuilt/$ABI"
if [[ "${1:-}" != "--force" && -f "$PREBUILT/libmqvpn.a" && -f "$PREBUILT/libssl.a" \
      && -f "$PREBUILT/libxquic.a" && -f "$PREBUILT/liblwip_core.a" ]]; then
    echo "prebuilt libs present in $PREBUILT (use --force to rebuild)"
    exit 0
fi

[[ -f "$MQVPN/third_party/xquic/CMakeLists.txt" ]] || \
    git -C "$REPO_ROOT" submodule update --init --recursive
"$MQVPN/scripts/build_android.sh" --abi "$ABI" --ndk "$ANDROID_NDK"
