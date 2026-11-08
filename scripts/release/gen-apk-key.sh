#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The apk feed signing key, made once (docs/glinet.md, "OpenWrt 25.12 and
# apk"). Each release's apk feed (packages.adb, per architecture) is
# signed with it; routers check that signature against the public keys
# the package installs in /etc/apk/keys/. OpenWrt 25.12's apk verifies
# EC P-256 signatures (openssl), unlike opkg's usign/Ed25519 — a separate
# key, not the opkg feed's.
#
#   scripts/release/gen-apk-key.sh --backup DIR [--repo OWNER/NAME] [--force] [--no-upload] [--keys-dir DIR]
#   scripts/release/gen-apk-key.sh --restore FILE.gpg [--repo OWNER/NAME] [--force] [--no-upload] [--keys-dir DIR]
#   scripts/release/gen-apk-key.sh --next --backup DIR [--keys-dir DIR]
#
# Make: an EC P-256 key pair in a private temporary directory, then
#   - GitHub repository secret TB_APK_SIGNING_KEY (gh secret set): the
#     secret key; the release job signs the apk feed with it;
#   - the public key as packaging/openwrt/files/etc/apk/keys/
#     tailbond-<fingerprint>.pem (--keys-dir): commit it, the apk package
#     ships it;
#   - an encrypted backup DIR/tailbond-apk-key-<date>.tar.gpg (gpg
#     symmetric AES-256; you choose the passphrase; DIR must be outside
#     any git checkout).
# The temporary copy is wiped. An existing secret or key file is never
# replaced without --force; when gh can't list the secrets, nothing is
# replaced either. --no-upload: no secret (tests, offline).
#
# Restore: put the key from a backup back (secret and public key file);
# with --force, the other public keys go.
#
# Rotating, while the current secret key still works (as the opkg feed
# key, docs/updates.md, "Rotating the feed key"):
#   1. --next --backup DIR: a new key pair, backed up; its public key goes
#      next to the current one, the secret stays. Commit; releases are
#      still signed with the current key and ship both public keys, so
#      routers that update get the new one.
#   2. Some releases later: --restore NEW.gpg --force. The secret becomes
#      the new key and the old public key goes; commit.
# A lost or leaked key can't do step 1: --force replaces it at once, and
# every router needs the new public key by hand, or one install of the
# package by hand.
#
# Env: TB_BACKUP_PASSPHRASE_FILE (else gpg asks), GH (default gh).
set -euo pipefail

die() { echo "gen-apk-key: $*" >&2; exit 1; }
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GH="${GH:-gh}"
BACKUP="" RESTORE="" REPO="" FORCE=0 UPLOAD=1 NEXT=0
KEYS="$ROOT/packaging/openwrt/files/etc/apk/keys"
while [ $# -gt 0 ]; do
    case "$1" in
    --backup) BACKUP="${2:?--backup DIR}"; shift 2 ;;
    --restore) RESTORE="${2:?--restore FILE}"; shift 2 ;;
    --repo) REPO="${2:?--repo OWNER/NAME}"; shift 2 ;;
    --keys-dir) KEYS="${2:?--keys-dir DIR}"; shift 2 ;;
    --force) FORCE=1; shift ;;
    --no-upload) UPLOAD=0; shift ;;
    --next) NEXT=1; UPLOAD=0; shift ;;
    -h | --help) sed -n '3,44p' "$0"; exit 0 ;;
    *) die "unknown argument $1 (--help)" ;;
    esac
done
[ -n "$BACKUP" ] || [ -n "$RESTORE" ] || die "--backup DIR (where the encrypted copy goes) or --restore FILE"
[ "$NEXT" = 0 ] || { [ -n "$BACKUP" ] && [ -z "$RESTORE" ] && [ "$FORCE" = 0 ]; } || die "--next makes a new key: --backup DIR, without --restore or --force"
command -v gpg >/dev/null || die "gpg not found (dnf install gnupg2 / apt install gnupg)"
command -v openssl >/dev/null || die "openssl not found"

# a directory inside a git work tree would end up committed sooner or later
in_git() { git -C "$1" rev-parse --is-inside-work-tree >/dev/null 2>&1; }

if [ "$UPLOAD" = 1 ]; then
    command -v "$GH" >/dev/null || die "gh not found (https://cli.github.com), or use --no-upload"
    if [ -z "$REPO" ]; then
        REPO="$("$GH" repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" || die "--repo OWNER/NAME (not in a GitHub checkout)"
    fi
    if [ "$FORCE" = 0 ]; then
        # a failed listing (auth, network, rate limit) is not "no secret"
        names="$("$GH" secret list --repo "$REPO" --json name -q '.[].name')" ||
            die "can't list the secrets of $REPO (gh auth status?), so can't tell whether TB_APK_SIGNING_KEY exists: nothing changed"
        if grep -qx TB_APK_SIGNING_KEY <<<"$names"; then
            die "$REPO already has an apk feed key (secret TB_APK_SIGNING_KEY). Routers that only trust it refuse a feed signed by another key. Restore it with --restore, rotate with --next, or replace it at once with --force."
        fi
    fi
fi
old=()
for k in "$KEYS"/*.pem; do [ -f "$k" ] && old+=("$k"); done
[ "$NEXT" = 0 ] || [ ${#old[@]} -gt 0 ] || die "--next adds a key next to the current one, and $KEYS has none: make the first key without --next"

W="$(mktemp -d "${TMPDIR:-/tmp}/tb-apk-key.XXXXXX")"
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
    [ -s "$W/apk.sec" ] && [ -s "$W/apk.pub" ] || die "$RESTORE doesn't hold a tailbond apk key"
else
    parent="$BACKUP"
    while [ ! -d "$parent" ]; do parent="$(dirname "$parent")"; done
    if in_git "$parent"; then die "$BACKUP is inside a git checkout: choose a folder outside any repository"; fi
    mkdir -p "$BACKUP"
    BACKUP="$(cd "$BACKUP" && pwd)"
    (umask 077; openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$W/apk.sec" 2>/dev/null) || die "openssl genpkey failed"
    openssl ec -in "$W/apk.sec" -pubout -out "$W/apk.pub" 2>/dev/null || die "deriving the public key failed"
fi
# the pair matches: the secret signs what the public key verifies
[ "$(openssl ec -in "$W/apk.sec" -pubout 2>/dev/null | sha256sum | cut -d' ' -f1)" = "$(sha256sum "$W/apk.pub" | cut -d' ' -f1)" ] ||
    die "the secret and public key don't match"

FP="tailbond-$(sha256sum "$W/apk.pub" | cut -c1-16)"
# another key in the package: replacing it locks out routers that have it
for k in ${old[@]+"${old[@]}"}; do
    [ "${k##*/}" = "$FP.pem" ] && cmp -s "$k" "$W/apk.pub" && continue
    [ "$FORCE" = 1 ] || [ "$NEXT" = 1 ] || die "$k is another feed key. Routers trust that one: rotate with --next, or replace it at once with --force."
done

if [ -z "$RESTORE" ]; then
    out="$BACKUP/tailbond-apk-key-$(date +%Y%m%d).tar.gpg"
    [ ! -e "$out" ] || out="$BACKUP/tailbond-apk-key-$(date +%Y%m%d-%H%M%S).tar.gpg"
    tar -C "$W" -cf - apk.sec apk.pub |
        gpg --quiet "${gpg_pass[@]}" --symmetric --cipher-algo AES256 --s2k-digest-algo SHA512 --s2k-count 65011712 -o "$out" ||
        die "gpg failed: no backup, no secret set"
    # prove the backup opens before anything depends on it
    gpg --quiet "${gpg_pass[@]}" --decrypt "$out" | tar -tf - | grep -qx apk.sec ||
        die "the backup $out doesn't decrypt: no secret set"
    echo "backup: $out (keep it and its passphrase; it's the only copy outside GitHub)"
fi

if [ "$UPLOAD" = 1 ]; then
    "$GH" secret set TB_APK_SIGNING_KEY --repo "$REPO" <"$W/apk.sec"
    echo "secret TB_APK_SIGNING_KEY set on $REPO"
fi
mkdir -p "$KEYS"
# --next keeps the current public key(s): releases ship both until the switch
removed=""
if [ "$NEXT" = 0 ]; then
    for k in ${old[@]+"${old[@]}"}; do
        [ "${k##*/}" = "$FP.pem" ] || { rm -f "$k"; removed="$removed ${k##*/}"; }
    done
fi
install -m 0644 "$W/apk.pub" "$KEYS/$FP.pem"
echo "public key: $KEYS/$FP.pem"
[ -z "$removed" ] || echo "removed public key(s):$removed"
if [ "$NEXT" = 1 ]; then
    echo "next key: releases stay signed with the current key and now ship both public keys. Switch later: $0 --restore <this backup> --force"
    msg="openwrt: add the next apk feed public key $FP.pem"
elif [ -n "$removed" ]; then
    msg="openwrt: apk feed public key $FP.pem (replaces$removed)"
else
    msg="openwrt: apk feed public key $FP.pem"
fi
if in_git "$KEYS"; then
    # the whole directory: a replaced key's removal must be committed too
    top="$(git -C "$KEYS" rev-parse --show-toplevel)"
    rel="$(cd "$KEYS" && pwd -P)"
    rel="${rel#"$(cd "$top" && pwd -P)"/}"
    echo "commit it: git -C '$top' add -A -- '$rel' && git -C '$top' commit -s -m '$msg' -- '$rel'"
fi
