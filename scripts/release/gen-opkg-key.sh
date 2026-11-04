#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The opkg feed signing key, made once (docs/glinet.md, "Install and update
# from GL's Plug-ins page"). Each release's opkg feed (Packages,
# Packages.gz) is signed with it (Packages.sig); routers check that
# signature against the public key the package installs in
# /etc/opkg/keys/. Lose the key and every router has to add the new public
# key by hand once.
#
#   scripts/release/gen-opkg-key.sh --backup DIR [--repo OWNER/NAME] [--force] [--no-upload] [--keys-dir DIR]
#   scripts/release/gen-opkg-key.sh --restore FILE.gpg [--repo OWNER/NAME] [--force] [--no-upload] [--keys-dir DIR]
#
# Make: a usign (Ed25519) key pair in a private temporary directory, then
#   - GitHub repository secret TB_OPKG_SIGNING_KEY (gh secret set): the
#     secret key; the release job signs the feed with it;
#   - the public key as packaging/openwrt/files/etc/opkg/keys/<fingerprint>
#     (--keys-dir): commit it, the package ships it;
#   - an encrypted backup DIR/tailbond-opkg-key-<date>.tar.gpg (gpg symmetric
#     AES-256; you choose the passphrase; DIR must be outside any git
#     checkout).
# The temporary copy is wiped. An existing secret or key file is never
# replaced without --force. --no-upload: no secret (tests, offline).
#
# Restore: put the key from a backup back (secret and public key file).
#
# Env: TB_BACKUP_PASSPHRASE_FILE (else gpg asks), USIGN (else
# scripts/release/usign.sh builds it), GH (default gh).
set -euo pipefail

die() { echo "gen-opkg-key: $*" >&2; exit 1; }
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GH="${GH:-gh}"
BACKUP="" RESTORE="" REPO="" FORCE=0 UPLOAD=1
KEYS="$ROOT/packaging/openwrt/files/etc/opkg/keys"
while [ $# -gt 0 ]; do
    case "$1" in
    --backup) BACKUP="${2:?--backup DIR}"; shift 2 ;;
    --restore) RESTORE="${2:?--restore FILE}"; shift 2 ;;
    --repo) REPO="${2:?--repo OWNER/NAME}"; shift 2 ;;
    --keys-dir) KEYS="${2:?--keys-dir DIR}"; shift 2 ;;
    --force) FORCE=1; shift ;;
    --no-upload) UPLOAD=0; shift ;;
    -h | --help) sed -n '3,28p' "$0"; exit 0 ;;
    *) die "unknown argument $1 (--help)" ;;
    esac
done
[ -n "$BACKUP" ] || [ -n "$RESTORE" ] || die "--backup DIR (where the encrypted copy goes) or --restore FILE"
command -v gpg >/dev/null || die "gpg not found (dnf install gnupg2 / apt install gnupg)"

# a directory inside a git work tree would end up committed sooner or later
in_git() { git -C "$1" rev-parse --is-inside-work-tree >/dev/null 2>&1; }

if [ "$UPLOAD" = 1 ]; then
    command -v "$GH" >/dev/null || die "gh not found (https://cli.github.com), or use --no-upload"
    if [ -z "$REPO" ]; then
        REPO="$("$GH" repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" || die "--repo OWNER/NAME (not in a GitHub checkout)"
    fi
    if [ "$FORCE" = 0 ] && "$GH" secret list --repo "$REPO" 2>/dev/null | grep -q '^TB_OPKG_SIGNING_KEY\b'; then
        die "$REPO already has an opkg feed key (secret TB_OPKG_SIGNING_KEY). A different key means every router has to add the new public key by hand. Restore the old one with --restore, or replace it on purpose with --force."
    fi
fi
old=()
for k in "$KEYS"/*; do [ -f "$k" ] && old+=("$k"); done

USIGN="$(USIGN="${USIGN:-}" "$ROOT/scripts/release/usign.sh")"
W="$(mktemp -d "${TMPDIR:-/tmp}/tb-opkg-key.XXXXXX")"
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
    [ -s "$W/opkg.sec" ] && [ -s "$W/opkg.pub" ] || die "$RESTORE doesn't hold a tailbond opkg key"
else
    parent="$BACKUP"
    while [ ! -d "$parent" ]; do parent="$(dirname "$parent")"; done
    if in_git "$parent"; then die "$BACKUP is inside a git checkout: choose a folder outside any repository"; fi
    mkdir -p "$BACKUP"
    BACKUP="$(cd "$BACKUP" && pwd)"
    (umask 077; "$USIGN" -G -s "$W/opkg.sec" -p "$W/opkg.pub") || die "usign -G failed"
fi

FP="$("$USIGN" -F -p "$W/opkg.pub")"
[[ "$FP" =~ ^[0-9a-f]{16}$ ]] && [ "$("$USIGN" -F -s "$W/opkg.sec")" = "$FP" ] || die "the secret and public key don't match"
# another key in the package: replacing it locks out routers that have it
for k in ${old[@]+"${old[@]}"}; do
    [ "${k##*/}" = "$FP" ] && cmp -s "$k" "$W/opkg.pub" && continue
    [ "$FORCE" = 1 ] || die "$k is another feed key. Routers trust that one; replace it on purpose with --force."
done

if [ -z "$RESTORE" ]; then
    out="$BACKUP/tailbond-opkg-key-$(date +%Y%m%d).tar.gpg"
    [ ! -e "$out" ] || out="$BACKUP/tailbond-opkg-key-$(date +%Y%m%d-%H%M%S).tar.gpg"
    tar -C "$W" -cf - opkg.sec opkg.pub |
        gpg --quiet "${gpg_pass[@]}" --symmetric --cipher-algo AES256 --s2k-digest-algo SHA512 --s2k-count 65011712 -o "$out" ||
        die "gpg failed: no backup, no secret set"
    # prove the backup opens before anything depends on it
    gpg --quiet "${gpg_pass[@]}" --decrypt "$out" | tar -tf - | grep -qx opkg.sec ||
        die "the backup $out doesn't decrypt: no secret set"
    echo "backup: $out (keep it and its passphrase; it's the only copy outside GitHub)"
fi

if [ "$UPLOAD" = 1 ]; then
    "$GH" secret set TB_OPKG_SIGNING_KEY --repo "$REPO" <"$W/opkg.sec"
    echo "secret TB_OPKG_SIGNING_KEY set on $REPO"
fi
mkdir -p "$KEYS"
for k in ${old[@]+"${old[@]}"}; do [ "${k##*/}" = "$FP" ] || rm -f "$k"; done
install -m 0644 "$W/opkg.pub" "$KEYS/$FP"
echo "public key: $KEYS/$FP (fingerprint $FP)"
if in_git "$KEYS"; then
    echo "commit it: git add '$KEYS/$FP' && git commit -s -m 'openwrt: opkg feed public key $FP'"
fi
