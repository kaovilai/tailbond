#!/usr/bin/env bash
# Unit tests + lint + debug APK → out/android/tailbond-debug.apk
#
#   android/scripts/build-apk.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

"$SCRIPT_DIR/build-native.sh"
"$SCRIPT_DIR/gradle.sh" --console=plain :app:testDebugUnitTest :app:lintDebug :app:assembleDebug

mkdir -p "$REPO_ROOT/out/android"
cp "$REPO_ROOT/android/app/build/outputs/apk/debug/app-debug.apk" "$REPO_ROOT/out/android/tailbond-debug.apk"
ls -l "$REPO_ROOT/out/android/tailbond-debug.apk"
