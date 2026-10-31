#!/usr/bin/env bash
# Run the Gradle wrapper with the toolchain from scripts/setup-sdk.sh.
#
#   android/scripts/gradle.sh <tasks...>
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ANDROID_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [[ -z "${JAVA_HOME:-}" ]]; then
    for c in "$HOME/workspace/rea-tools/jdk" /usr/lib/jvm/java-21-openjdk; do
        [[ -x "$c/bin/javac" ]] && { export JAVA_HOME="$c"; break; }
    done
fi
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
[[ -f "$ANDROID_DIR/local.properties" ]] || echo "sdk.dir=$ANDROID_HOME" > "$ANDROID_DIR/local.properties"

cd "$ANDROID_DIR"
exec ./gradlew "$@"
