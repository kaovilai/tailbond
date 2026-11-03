#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The Android release signing key, made once, kept in GitHub (docs/updates.md,
# "Android signing key"). Android requires every APK to be signed, and an
# update must carry the same key as the installed app: lose the key and every
# phone has to uninstall and reinstall once.
#
#   scripts/release/gen-android-keystore.sh --backup DIR [--repo OWNER/NAME] [--force] [--no-upload]
#   scripts/release/gen-android-keystore.sh --restore FILE.gpg [--repo OWNER/NAME] [--force]
#
# Make: a PKCS12 keystore (RSA 4096, valid 30 years) in a private temporary
# directory, then
#   - GitHub repository secrets (gh secret set): TB_RELEASE_KEYSTORE_B64,
#     TB_RELEASE_KEYSTORE_PASSWORD, TB_RELEASE_KEY_ALIAS,
#     TB_RELEASE_KEY_PASSWORD; CI signs release APKs with them;
#   - repository variable TB_RELEASE_CERT_SHA256 (public: the certificate's
#     fingerprint): CI refuses to publish an APK signed with anything else;
#   - an encrypted backup DIR/tailbond-android-release-key-<date>.tar.gpg
#     (gpg symmetric AES-256; you choose the passphrase; DIR must be outside
#     any git checkout: a password manager's attachment folder, a USB stick).
# The temporary copy is wiped. Nothing is written into the repository.
# Existing secrets are never replaced without --force (a new key means every
# user reinstalls once). --no-upload: make the backup only (tests, offline).
#
# Restore: put the key from a backup back into the repository secrets (new
# repository, deleted secrets).
#
# Env: TB_BACKUP_PASSPHRASE_FILE (else gpg asks), KEYTOOL (default: keytool
# from PATH or $JAVA_HOME/bin), GH (default gh).
set -euo pipefail

die() { echo "gen-android-keystore: $*" >&2; exit 1; }
GH="${GH:-gh}"
ALIAS=tailbond
BACKUP="" RESTORE="" REPO="" FORCE=0 UPLOAD=1
while [ $# -gt 0 ]; do
    case "$1" in
    --backup) BACKUP="${2:?--backup DIR}"; shift 2 ;;
    --restore) RESTORE="${2:?--restore FILE}"; shift 2 ;;
    --repo) REPO="${2:?--repo OWNER/NAME}"; shift 2 ;;
    --force) FORCE=1; shift ;;
    --no-upload) UPLOAD=0; shift ;;
    -h | --help) sed -n '3,30p' "$0"; exit 0 ;;
    *) die "unknown argument $1 (--help)" ;;
    esac
done
[ -n "$BACKUP" ] || [ -n "$RESTORE" ] || die "--backup DIR (where the encrypted copy goes) or --restore FILE"
command -v gpg >/dev/null || die "gpg not found (dnf install gnupg2 / apt install gnupg)"
KEYTOOL="${KEYTOOL:-$(command -v keytool || echo "${JAVA_HOME:-/nonexistent}/bin/keytool")}"

# a directory inside a git work tree would end up committed sooner or later
in_git() { git -C "$1" rev-parse --is-inside-work-tree >/dev/null 2>&1; }

if [ "$UPLOAD" = 1 ]; then
    command -v "$GH" >/dev/null || die "gh not found (https://cli.github.com), or use --no-upload"
    if [ -z "$REPO" ]; then
        REPO="$("$GH" repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" || die "--repo OWNER/NAME (not in a GitHub checkout)"
    fi
    if [ "$FORCE" = 0 ] && "$GH" secret list --repo "$REPO" 2>/dev/null | grep -q '^TB_RELEASE_KEYSTORE_B64\b'; then
        die "$REPO already has a release key (secret TB_RELEASE_KEYSTORE_B64). A different key means every user has to uninstall and reinstall the app once. Restore the old one with --restore, or replace it on purpose with --force."
    fi
fi

W="$(mktemp -d "${TMPDIR:-/tmp}/tb-android-key.XXXXXX")"
chmod 700 "$W"
wipe() { find "$W" -type f -exec sh -c 'command -v shred >/dev/null && shred -u "$@" || rm -f "$@"' _ {} + 2>/dev/null; rm -rf "$W"; }
trap wipe EXIT
trap 'exit 130' INT TERM
gpg_pass=()
if [ -n "${TB_BACKUP_PASSPHRASE_FILE:-}" ]; then
    gpg_pass=(--batch --pinentry-mode loopback --passphrase-file "$TB_BACKUP_PASSPHRASE_FILE")
fi

if [ -n "$RESTORE" ]; then
    [ -f "$RESTORE" ] || die "$RESTORE not found"
    gpg --quiet "${gpg_pass[@]}" --decrypt "$RESTORE" | tar -xf - -C "$W" || die "can't decrypt $RESTORE (wrong passphrase?)"
    [ -s "$W/tailbond-release.p12" ] && [ -s "$W/password" ] || die "$RESTORE doesn't hold a tailbond release key"
    ALIAS="$(cat "$W/alias" 2>/dev/null || echo tailbond)"
else
    [ -x "$KEYTOOL" ] || die "keytool not found (install a JDK or set KEYTOOL)"
    parent="$BACKUP"
    while [ ! -d "$parent" ]; do parent="$(dirname "$parent")"; done
    if in_git "$parent"; then die "$BACKUP is inside a git checkout: choose a folder outside any repository"; fi
    mkdir -p "$BACKUP"
    BACKUP="$(cd "$BACKUP" && pwd)"
    # PKCS12: one password for the store and the key
    (umask 077; openssl rand -base64 33 | tr -d '/+=\n' >"$W/password"; echo "$ALIAS" >"$W/alias")
    "$KEYTOOL" -genkeypair -keystore "$W/tailbond-release.p12" -storetype PKCS12 -alias "$ALIAS" \
        -keyalg RSA -keysize 4096 -validity 10950 -dname "CN=tailbond, O=tailbond" \
        -storepass:file "$W/password" -keypass:file "$W/password" >/dev/null 2>&1 ||
        die "keytool failed"
fi

FP="$("$KEYTOOL" -list -v -keystore "$W/tailbond-release.p12" -storetype PKCS12 -storepass:file "$W/password" -alias "$ALIAS" 2>/dev/null |
    sed -n 's/^[[:space:]]*SHA256: //p' | head -n1 | tr -d ':' | tr 'A-F' 'a-f')"
[ -n "$FP" ] || die "can't read the certificate fingerprint"

if [ -z "$RESTORE" ]; then
    out="$BACKUP/tailbond-android-release-key-$(date +%Y%m%d).tar.gpg"
    [ ! -e "$out" ] || out="$BACKUP/tailbond-android-release-key-$(date +%Y%m%d-%H%M%S).tar.gpg"
    echo "$FP" >"$W/cert-sha256"
    tar -C "$W" -cf - tailbond-release.p12 password alias cert-sha256 |
        gpg --quiet "${gpg_pass[@]}" --symmetric --cipher-algo AES256 --s2k-digest-algo SHA512 --s2k-count 65011712 -o "$out" ||
        die "gpg failed: no backup, no secrets set"
    # prove the backup opens before anything depends on it
    gpg --quiet "${gpg_pass[@]}" --decrypt "$out" | tar -tf - | grep -qx tailbond-release.p12 ||
        die "the backup $out doesn't decrypt: no secrets set"
    echo "backup: $out (keep it and its passphrase; it's the only copy outside GitHub)"
fi

if [ "$UPLOAD" = 1 ]; then
    base64 -w0 "$W/tailbond-release.p12" | "$GH" secret set TB_RELEASE_KEYSTORE_B64 --repo "$REPO"
    "$GH" secret set TB_RELEASE_KEYSTORE_PASSWORD --repo "$REPO" <"$W/password"
    "$GH" secret set TB_RELEASE_KEY_PASSWORD --repo "$REPO" <"$W/password"
    printf '%s' "$ALIAS" | "$GH" secret set TB_RELEASE_KEY_ALIAS --repo "$REPO"
    "$GH" variable set TB_RELEASE_CERT_SHA256 --repo "$REPO" --body "$FP"
    echo "secrets TB_RELEASE_KEYSTORE_B64, TB_RELEASE_KEYSTORE_PASSWORD, TB_RELEASE_KEY_ALIAS, TB_RELEASE_KEY_PASSWORD and variable TB_RELEASE_CERT_SHA256 set on $REPO"
fi
echo "certificate SHA-256: $FP"
