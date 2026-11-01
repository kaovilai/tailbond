#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Sign an update manifest: <channel>.json -> <channel>.json.sig
#
#   scripts/release/sign-manifest.sh stable.json KEY.pem
#   TB_UPDATE_KEY="$(cat KEY.pem)" scripts/release/sign-manifest.sh stable.json
#
# The .sig is JSON {"key_id": "<hex of the first 8 bytes of SHA-256(raw
# public key)>", "sig": "<base64 ed25519 signature>"}: plain ed25519 (no
# prehash) over the exact bytes of the manifest file, so never edit it after
# signing. The key is the PEM from gen-update-key.sh (CI secret
# TB_UPDATE_KEY). Prints the .sig path. Non-interactive; OpenSSL 3.
set -euo pipefail

die() { echo "sign-manifest: $*" >&2; exit 1; }
MANIFEST="${1:?usage: sign-manifest.sh <channel>.json [KEY.pem]}"
KEY="${2:-}"
[ -f "$MANIFEST" ] || die "$MANIFEST: no such file"
command -v openssl >/dev/null || die "openssl not found"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
if [ -z "$KEY" ]; then
    [ -n "${TB_UPDATE_KEY:-}" ] || die "no key: pass KEY.pem or set TB_UPDATE_KEY"
    (umask 077; printf '%s\n' "$TB_UPDATE_KEY" >"$tmp/key.pem")
    KEY="$tmp/key.pem"
fi
[ -r "$KEY" ] || die "$KEY: not readable"

openssl pkey -in "$KEY" -noout 2>/dev/null || die "$KEY is not a private key PEM"
openssl pkey -in "$KEY" -pubout -outform DER >"$tmp/pub.der"
# ed25519 SubjectPublicKeyInfo: 12-byte header + 32-byte key
[ "$(wc -c <"$tmp/pub.der")" = 44 ] || die "$KEY is not an ed25519 key"
key_id="$(tail -c 32 "$tmp/pub.der" | sha256sum | cut -c1-16)"

openssl pkeyutl -sign -inkey "$KEY" -rawin -in "$MANIFEST" -out "$tmp/sig.bin"
[ "$(wc -c <"$tmp/sig.bin")" = 64 ] || die "unexpected signature length"
# Check it before publishing.
openssl pkeyutl -verify -pubin -inkey "$tmp/pub.der" -keyform DER -rawin -in "$MANIFEST" \
    -sigfile "$tmp/sig.bin" >/dev/null || die "signature does not verify"

out="$MANIFEST.sig"
printf '{"key_id":"%s","sig":"%s"}\n' "$key_id" "$(base64 -w0 <"$tmp/sig.bin")" >"$out.tmp"
mv "$out.tmp" "$out"
echo "$out"
