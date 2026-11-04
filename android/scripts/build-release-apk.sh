#!/usr/bin/env bash
# Release APK signed with tailbond's release key → out/android/tailbond-<versionName>.apk
#
#   TB_VERSION=0.2.0 \
#   TB_RELEASE_KEYSTORE=/path/tailbond-release.p12      (or TB_RELEASE_KEYSTORE_B64=<base64 of it>, as in CI)
#   TB_RELEASE_KEYSTORE_PASSWORD=… TB_RELEASE_KEY_ALIAS=tailbond [TB_RELEASE_KEY_PASSWORD=…] \
#   [TB_UPDATE_REPO=owner/name] \
#       android/scripts/build-release-apk.sh
#
# TB_VERSION: versionName (a leading "v" is dropped; default tailbond.version in
# android/gradle.properties). versionCode follows from it (docs/android.md, "Updates").
# TB_UPDATE_REPO: the GitHub repository whose latest release the app checks
# (default kaovilai/tailbond; CI passes github.repository).
# Runs the unit tests and lint first. The keystore and the CI secrets come
# from scripts/release/gen-android-keystore.sh (once, ever).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

die() { echo "build-release-apk: $*" >&2; exit 1; }

missing=()
[[ -n "${TB_RELEASE_KEYSTORE:-}" || -n "${TB_RELEASE_KEYSTORE_B64:-}" ]] || missing+=("TB_RELEASE_KEYSTORE (path) or TB_RELEASE_KEYSTORE_B64")
[[ -n "${TB_RELEASE_KEYSTORE_PASSWORD:-}" ]] || missing+=(TB_RELEASE_KEYSTORE_PASSWORD)
[[ -n "${TB_RELEASE_KEY_ALIAS:-}" ]] || missing+=(TB_RELEASE_KEY_ALIAS)
if ((${#missing[@]})); then
    die "missing ${missing[*]}. A release APK must be signed with the release key (made once by scripts/release/gen-android-keystore.sh, kept in the repo secrets and its encrypted backup); for a test build use build-apk.sh"
fi

TMP_KS=""
# An if, not `[[ ]] &&`: with no temp keystore the test's false would be the
# trap's last status, and the script would exit 1 after a good build.
cleanup() { if [[ -n "$TMP_KS" ]]; then rm -f "$TMP_KS"; fi; }
trap cleanup EXIT
if [[ -z "${TB_RELEASE_KEYSTORE:-}" ]]; then
    TMP_KS="$(mktemp "${TMPDIR:-/tmp}/tailbond-release-XXXXXX.p12")"
    printf '%s' "$TB_RELEASE_KEYSTORE_B64" | base64 -d > "$TMP_KS" || die "TB_RELEASE_KEYSTORE_B64 isn't base64"
    export TB_RELEASE_KEYSTORE="$TMP_KS"
fi
[[ -s "$TB_RELEASE_KEYSTORE" ]] || die "keystore $TB_RELEASE_KEYSTORE not found or empty"
export TB_RELEASE_KEY_PASSWORD="${TB_RELEASE_KEY_PASSWORD:-$TB_RELEASE_KEYSTORE_PASSWORD}"

VERSION="${TB_VERSION:-$(sed -n 's/^tailbond\.version=//p' "$REPO_ROOT/android/gradle.properties")}"
VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || die "TB_VERSION \"$VERSION\" isn't MAJOR.MINOR.PATCH[-pre]"
export TB_VERSION="$VERSION"

"$SCRIPT_DIR/build-native.sh"
"$SCRIPT_DIR/gradle.sh" --console=plain :app:testDebugUnitTest :app:lintRelease :app:assembleRelease

APK="$REPO_ROOT/android/app/build/outputs/apk/release/app-release.apk"
[[ -f "$APK" ]] || die "Gradle made no signed APK (got: $(ls "$REPO_ROOT/android/app/build/outputs/apk/release/" 2>/dev/null | tr '\n' ' '))"
OUT="$REPO_ROOT/out/android/tailbond-$VERSION.apk"
mkdir -p "$(dirname "$OUT")"
cp "$APK" "$OUT"

# show the signer, so a wrong key is caught here and not on a phone
if [[ -z "${JAVA_HOME:-}" ]]; then
    for c in "$HOME/workspace/rea-tools/jdk" /usr/lib/jvm/java-21-openjdk; do
        [[ -x "$c/bin/java" ]] && { export JAVA_HOME="$c"; break; }
    done
fi
JAVA="${JAVA_HOME:+$JAVA_HOME/bin/}java"
ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
APKSIGNER_JAR="$(ls -d "$ANDROID_HOME"/build-tools/*/lib/apksigner.jar 2>/dev/null | sort -V | tail -n1 || true)"
if [[ -n "$APKSIGNER_JAR" ]]; then
    # the jar directly (the apksigner wrapper needs java on PATH); "Signer #1" or "V2 Signer" by version
    CERTS="$(timeout 300 "$JAVA" -jar "$APKSIGNER_JAR" verify --print-certs "$OUT" </dev/null)" || die "apksigner can't verify $OUT"
    grep -E 'Signer.*certificate (DN|SHA-256)' <<<"$CERTS" || die "apksigner printed no signer for $OUT"
else
    echo "build-release-apk: no apksigner in $ANDROID_HOME/build-tools, signer not shown" >&2
fi
echo "sha256 $(sha256sum "$OUT" | cut -d' ' -f1)  size $(stat -c %s "$OUT")"
ls -l "$OUT"
