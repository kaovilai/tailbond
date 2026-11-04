#!/usr/bin/env bash
# x86_64-only debug APK for the emulator end-to-end test
# (scripts/e2e-android.sh) → out/android/tailbond-debug-x86_64.apk
#
#   android/scripts/build-e2e-apk.sh
#
# Phones get arm64-v8a only (build-apk.sh, build-release-apk.sh); this APK
# carries x86_64 only, so neither grows.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

TB_ANDROID_ABIS=x86_64 "$SCRIPT_DIR/build-native.sh"
"$SCRIPT_DIR/gradle.sh" --console=plain -Ptailbond.abis=x86_64 :app:assembleDebug

mkdir -p "$REPO_ROOT/out/android"
cp "$REPO_ROOT/android/app/build/outputs/apk/debug/app-debug.apk" "$REPO_ROOT/out/android/tailbond-debug-x86_64.apk"
ls -l "$REPO_ROOT/out/android/tailbond-debug-x86_64.apk"
