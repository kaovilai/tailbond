#!/usr/bin/env bash
# Create tailbond's Android release signing key (once, ever) outside the repo
# and print what to store as CI secrets.
#
#   android/scripts/gen-release-keystore.sh [OUT_DIR]     # default ~/.config/tailbond/android-release
#
# Keep the keystore and its password safe and backed up: every future
# release must be signed with this same key, or phones can't update to it
# (Android refuses an update signed with another key; users would have to
# uninstall, losing their pairing). Debug builds use the debug key and can
# never update a release install, or the other way round.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="${1:-$HOME/.config/tailbond/android-release}"
ALIAS="${TB_RELEASE_KEY_ALIAS:-tailbond}"

mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"
case "$OUT_DIR/" in
    "$REPO_ROOT"/*) echo "refusing to write a signing key inside the repo ($OUT_DIR)" >&2; exit 1 ;;
esac
KS="$OUT_DIR/tailbond-release.p12"
PW_FILE="$OUT_DIR/tailbond-release.password"
if [[ -e "$KS" ]]; then
    echo "$KS already exists. A release key is made once; delete it yourself only if no release was ever signed with it." >&2
    exit 1
fi

if [[ -z "${JAVA_HOME:-}" ]]; then
    for c in "$HOME/workspace/rea-tools/jdk" /usr/lib/jvm/java-21-openjdk; do
        [[ -x "$c/bin/keytool" ]] && { export JAVA_HOME="$c"; break; }
    done
fi
KEYTOOL="${JAVA_HOME:+$JAVA_HOME/bin/}keytool"
command -v "$KEYTOOL" >/dev/null || { echo "keytool not found (install a JDK or set JAVA_HOME)" >&2; exit 1; }

umask 077
# PKCS12 uses one password for the store and the key
PASSWORD="$(head -c 32 /dev/urandom | base64 | tr -d '/+=' | head -c 32)"
printf '%s' "$PASSWORD" > "$PW_FILE"

"$KEYTOOL" -genkeypair -keystore "$KS" -storetype PKCS12 -alias "$ALIAS" \
    -keyalg RSA -keysize 4096 -validity 10000 -dname "CN=tailbond, O=tailbond" \
    -storepass:file "$PW_FILE" -keypass:file "$PW_FILE" >/dev/null 2>&1

FINGERPRINT="$("$KEYTOOL" -list -v -keystore "$KS" -storepass:file "$PW_FILE" -alias "$ALIAS" | sed -n 's/^.*SHA256: //p' | head -n1)"

cat <<EOF
Created the release key:
  keystore  $KS
  password  $PW_FILE   (store and key password; PKCS12 uses one)
  alias     $ALIAS
  SHA-256   $FINGERPRINT
            (the certificate fingerprint: publish it so people can check the APK, e.g. in Obtainium or AppVerifier)

Back up both files somewhere safe and offline. Lose them and phones can't update any more.

CI secrets (GitHub → Settings → Secrets and variables → Actions), set from the files
without printing them:

  base64 -w0 '$KS' | gh secret set TB_RELEASE_KEYSTORE_B64
  gh secret set TB_RELEASE_KEYSTORE_PASSWORD < '$PW_FILE'
  printf '%s' '$ALIAS' | gh secret set TB_RELEASE_KEY_ALIAS
  gh secret set TB_RELEASE_KEY_PASSWORD < '$PW_FILE'

Local release build:

  TB_RELEASE_KEYSTORE='$KS' \\
  TB_RELEASE_KEYSTORE_PASSWORD="\$(cat '$PW_FILE')" TB_RELEASE_KEY_ALIAS='$ALIAS' \\
  TB_VERSION=0.2.0 TB_UPDATE_PUBKEY=<base64 update key> android/scripts/build-release-apk.sh
EOF
