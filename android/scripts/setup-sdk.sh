#!/usr/bin/env bash
# Install the Android toolchain tailbond's android/ build needs (Fedora/Linux x86_64).
#
#   android/scripts/setup-sdk.sh
#
# Installs, idempotently:
#   - JDK 21: reuses $JAVA_HOME, then ~/workspace/rea-tools/jdk, else dnf java-21-openjdk-devel
#   - Android cmdline-tools (latest) under $ANDROID_HOME (default ~/Android/Sdk)
#   - platform-tools, platforms;android-37, build-tools, ndk;28.2.13676358, cmake;3.22.1
#     (versions match third_party/mqvpn/android: compileSdk 37, sdk-native ndkVersion/cmake)
#   - android/local.properties pointing at the SDK
#
# Prints the env lines to export at the end.
set -euo pipefail

ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
NDK_VERSION="28.2.13676358"
CMAKE_VERSION="3.22.1"
PLATFORM="platforms;android-37.0"
BUILD_TOOLS="${BUILD_TOOLS:-build-tools;37.0.0}"
REPO_XML="https://dl.google.com/android/repository/repository2-3.xml"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ANDROID_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── JDK 21 ──
find_jdk() {
    local c
    for c in "${JAVA_HOME:-}" "$HOME/workspace/rea-tools/jdk" /usr/lib/jvm/java-21-openjdk; do
        [[ -n "$c" && -x "$c/bin/javac" ]] || continue
        if "$c/bin/java" -version 2>&1 | grep -q 'version "21'; then
            echo "$c"; return 0
        fi
    done
    return 1
}
if ! JAVA_HOME="$(find_jdk)"; then
    echo "Installing java-21-openjdk-devel via dnf..."
    sudo dnf install -y java-21-openjdk-devel
    JAVA_HOME="$(find_jdk)"
fi
export JAVA_HOME
echo "JDK: $JAVA_HOME"

# ── host tools ──
for t in curl unzip; do
    command -v "$t" >/dev/null || sudo dnf install -y "$t"
done

# ── cmdline-tools ──
SDKMANAGER="$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager"
if [[ ! -x "$SDKMANAGER" ]]; then
    zip="$(curl -fsSL "$REPO_XML" | grep -o 'commandlinetools-linux-[0-9]*_latest.zip' | sort -t- -k3 -n | tail -1)"
    [[ -n "$zip" ]] || { echo "ERROR: cannot find cmdline-tools in $REPO_XML" >&2; exit 1; }
    echo "Downloading $zip..."
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    curl -fsSL -o "$tmp/clt.zip" "https://dl.google.com/android/repository/$zip"
    unzip -q "$tmp/clt.zip" -d "$tmp"
    mkdir -p "$ANDROID_HOME/cmdline-tools"
    rm -rf "$ANDROID_HOME/cmdline-tools/latest"
    mv "$tmp/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest"
fi

# ── SDK packages ──
yes | "$SDKMANAGER" --sdk_root="$ANDROID_HOME" --licenses >/dev/null || true
"$SDKMANAGER" --sdk_root="$ANDROID_HOME" \
    "platform-tools" "$PLATFORM" "$BUILD_TOOLS" "ndk;$NDK_VERSION" "cmake;$CMAKE_VERSION"

# ── local.properties (gitignored) ──
echo "sdk.dir=$ANDROID_HOME" > "$ANDROID_DIR/local.properties"

cat <<EOF

Done. Export before building:
  export JAVA_HOME=$JAVA_HOME
  export ANDROID_HOME=$ANDROID_HOME
  export ANDROID_NDK=$ANDROID_HOME/ndk/$NDK_VERSION
  export PATH=\$ANDROID_HOME/platform-tools:\$JAVA_HOME/bin:\$PATH
EOF
