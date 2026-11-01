#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Generate an ed25519 update-signing key (docs/updates.md, "Signing keys").
#
#   scripts/release/gen-update-key.sh [--out DIR] [--name NAME]
#
# Writes, into DIR (default ${XDG_CONFIG_HOME:-~/.config}/tailbond/update-keys,
# never inside this repository):
#   NAME-<key_id>.pem       private key, PKCS#8 PEM, mode 0600: the CI secret
#                           TB_UPDATE_KEY (gh secret set TB_UPDATE_KEY < file)
#   NAME-<key_id>.pub.b64   raw 32-byte public key, base64: the CI variable
#                           TB_UPDATE_PUBKEY, compiled into tailbondd and the
#                           Android app
# and prints (stdout, one per line, for scripts):
#   TB_UPDATE_KEY_FILE=<path of the .pem>
#   TB_UPDATE_PUBKEY=<base64 raw public key>
#   TB_UPDATE_KEY_ID=<hex of the first 8 bytes of SHA-256(raw public key)>
#
# Non-interactive; needs OpenSSL 3 (ed25519 and pkeyutl -rawin).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${XDG_CONFIG_HOME:-$HOME/.config}/tailbond/update-keys"
NAME="tailbond-update"
die() { echo "gen-update-key: $*" >&2; exit 1; }

while [ $# -gt 0 ]; do
    case "$1" in
        --out) [ $# -ge 2 ] || die "--out needs a directory"; OUT="$2"; shift ;;
        --name) [ $# -ge 2 ] || die "--name needs a value"; NAME="$2"; shift ;;
        -h|--help) sed -n '3,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) die "unknown option $1" ;;
    esac
    shift
done
case "$NAME" in *[!A-Za-z0-9._-]*|"") die "--name: letters, digits, . _ - only" ;; esac

command -v openssl >/dev/null || die "openssl not found"
case "$(openssl version)" in
    "OpenSSL 3."*|"OpenSSL 4."*) ;;
    *) die "OpenSSL 3 needed (ed25519 raw signing), found: $(openssl version)" ;;
esac

# A private key must never land in a working tree that could be committed.
abs="$(realpath -m "$OUT")"
repo="$(realpath -m "$ROOT")"
case "$abs/" in "$repo/"*) die "refusing to write keys inside the repository ($repo); use --out DIR outside it" ;; esac

umask 077
mkdir -p "$abs"
chmod 700 "$abs" 2>/dev/null || true
tmp="$(mktemp "$abs/.key.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
openssl genpkey -algorithm ed25519 -out "$tmp" 2>/dev/null || die "openssl genpkey failed"

# The DER SubjectPublicKeyInfo of an ed25519 key ends with the raw 32 bytes.
pub_b64="$(openssl pkey -in "$tmp" -pubout -outform DER | tail -c 32 | base64 -w0)"
key_id="$(printf '%s' "$pub_b64" | base64 -d | sha256sum | cut -c1-16)"
[ "$(printf '%s' "$pub_b64" | base64 -d | wc -c)" = 32 ] || die "unexpected public key length"

pem="$abs/$NAME-$key_id.pem"
[ -e "$pem" ] && die "$pem exists"
mv "$tmp" "$pem"
chmod 600 "$pem"
printf '%s\n' "$pub_b64" >"$abs/$NAME-$key_id.pub.b64"
chmod 644 "$abs/$NAME-$key_id.pub.b64"

echo "TB_UPDATE_KEY_FILE=$pem"
echo "TB_UPDATE_PUBKEY=$pub_b64"
echo "TB_UPDATE_KEY_ID=$key_id"
