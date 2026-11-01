#!/usr/bin/env bash
# shellcheck disable=SC2329 # helpers are called through check()
# SPDX-License-Identifier: Apache-2.0
#
# Self-test of the release scripts: a throwaway key (gen-update-key.sh), a
# manifest from dummy artifacts (make-manifest.sh), its signature
# (sign-manifest.sh), verified with openssl from the base64 public key
# alone, the way tailbondd does it. Prints the throwaway public key.
#
#   scripts/release/test-release-scripts.sh
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
W="$(mktemp -d)"
trap 'rm -rf "$W"' EXIT
fail=0
ok() { echo "  PASS $*"; }
bad() { echo "  FAIL $*"; fail=1; }
check() { local what="$1"; shift; if "$@"; then ok "$what"; else bad "$what"; fi; }

echo "== gen-update-key.sh"
if (cd "$HERE" && "$HERE/gen-update-key.sh" --out "$HERE/../../out/.key-refusal-test-$$" >/dev/null 2>&1); then
    bad "a key directory inside the repo is refused"
else
    ok "a key directory inside the repo is refused"
fi
rm -rf "$HERE/../../out/.key-refusal-test-$$"
eval "$("$HERE/gen-update-key.sh" --out "$W/keys" --name test)"
echo "  public key (base64): $TB_UPDATE_PUBKEY  key id: $TB_UPDATE_KEY_ID"
check "private key is mode 0600" test "$(stat -c %a "$TB_UPDATE_KEY_FILE")" = 600
check "public key is 32 bytes" test "$(printf '%s' "$TB_UPDATE_PUBKEY" | base64 -d | wc -c)" = 32
check "key id is 16 hex digits" grep -Eqx '[0-9a-f]{16}' <<<"$TB_UPDATE_KEY_ID"
check ".pub.b64 matches" test "$(cat "$W/keys/test-$TB_UPDATE_KEY_ID.pub.b64")" = "$TB_UPDATE_PUBKEY"

echo "== make-manifest.sh"
D="$W/dist"
mkdir -p "$D"
for f in tailbond-linux-x86_64.tar.gz tailbond-linux-aarch64.tar.gz \
    tailbond_0.2.0-r1_aarch64_cortex-a53.ipk tailbond_0.2.0-r1_aarch64_cortex-a53_neon-vfpv4.ipk \
    tailbond-server-0.2.0-1.fc44.x86_64.rpm tailbond-0.2.0.apk \
    tailbond-debug.apk SHA256SUMS tailbondd-v0.2.0-linux-amd64.tar.gz; do
    head -c "$(( ${#f} * 100 ))" /dev/urandom >"$D/$f"
done
m="$("$HERE/make-manifest.sh" --version v0.2.0 --channel stable --dir "$D" --out "$W/channels" \
    --base-url https://example.com/releases/download/v0.2.0/ --protocol 2 --requires-server-protocol 1 \
    --notes-url https://example.com/releases/tag/v0.2.0 --released 2026-11-02T10:00:00Z 2>/dev/null)"
check "writes stable.json" test "$m" = "$W/channels/stable.json"
j() { jq -r "$1" "$m"; }
check "format, channel, version (v stripped)" test "$(j '[.format, .channel, .version] | join(" ")')" = "1 stable 0.2.0"
check "protocol 2, requires_server_protocol 1, security false" \
    test "$(j '[.protocol, .requires_server_protocol, .security] | map(tostring) | join(" ")')" = "2 1 false"
check "six artifacts (debug apk, SHA256SUMS, plain tailbondd ignored)" test "$(j '.artifacts | length')" = 6
check "kinds and arches from file names" test "$(j '[.artifacts[] | .kind + ":" + .arch] | sort | join(" ")')" = \
    "apk:arm64-v8a ipk:aarch64_cortex-a53 ipk:aarch64_cortex-a53_neon-vfpv4 linux:aarch64 linux:x86_64 rpm:x86_64"
check "apk version_code 20000" test "$(j '.artifacts[] | select(.kind == "apk") | .version_code')" = 20000
check "only the apk has version_code" test "$(j '[.artifacts[] | select(has("version_code"))] | length')" = 1
check "url = base + name" test "$(j '.artifacts[] | select(.kind == "rpm") | .url')" = \
    "https://example.com/releases/download/v0.2.0/tailbond-server-0.2.0-1.fc44.x86_64.rpm"
x="tailbond-linux-x86_64.tar.gz"
check "sha256 and size" test "$(j ".artifacts[] | select(.name == \"$x\") | .sha256 + \" \" + (.size | tostring)")" = \
    "$(sha256sum "$D/$x" | cut -d' ' -f1) $(stat -c %s "$D/$x")"
if "$HERE/make-manifest.sh" --version 0.3.0 --channel stable --dir "$D" --out "$W/x" --base-url https://x >/dev/null 2>&1; then
    bad "artifacts of another version are refused"
else
    ok "artifacts of another version are refused"
fi
B="$W/beta"
mkdir -p "$B"
for f in tailbond-1.2.3-beta.4.apk tailbond-server-1.2.3-0.beta.4.fc44.x86_64.rpm tailbond_1.2.3-beta.4-r7_aarch64_cortex-a53.ipk; do
    echo "$f" >"$B/$f"
done
mb="$("$HERE/make-manifest.sh" --version 1.2.3-beta.4 --channel beta --dir "$B" --out "$W/channels" --base-url https://x --security 2>/dev/null)"
check "beta: version_code below the release (1020204)" test "$(jq -r '.artifacts[] | select(.kind == "apk") | .version_code' "$mb")" = 1020204
check "beta: security flag, three artifacts" test "$(jq -r '[.security, (.artifacts | length)] | map(tostring) | join(" ")' "$mb")" = "true 3"

echo "== sign-manifest.sh"
sig="$("$HERE/sign-manifest.sh" "$m" "$TB_UPDATE_KEY_FILE")"
check "writes stable.json.sig" test "$sig" = "$m.sig"
check "key_id matches the key" test "$(jq -r .key_id "$sig")" = "$TB_UPDATE_KEY_ID"
# Verify the way a client does: from the raw base64 key only. An ed25519
# SubjectPublicKeyInfo is this 12-byte prefix plus the 32 raw bytes.
{ printf '\x30\x2a\x30\x05\x06\x03\x2b\x65\x70\x03\x21\x00'; printf '%s' "$TB_UPDATE_PUBKEY" | base64 -d; } >"$W/pub.der"
jq -r .sig "$sig" | base64 -d >"$W/sig.bin"
check "signature is 64 bytes" test "$(wc -c <"$W/sig.bin")" = 64
verify() { openssl pkeyutl -verify -pubin -keyform DER -inkey "$W/pub.der" -rawin -in "$1" -sigfile "$W/sig.bin" >/dev/null 2>&1; }
check "openssl verifies the signature with the base64 public key" verify "$m"
cp "$m" "$W/tampered.json"
sed -i 's/"security": false/"security": true/' "$W/tampered.json"
if verify "$W/tampered.json"; then
    bad "a changed manifest fails verification"
else
    ok "a changed manifest fails verification"
fi
sig2="$(TB_UPDATE_KEY="$(cat "$TB_UPDATE_KEY_FILE")" "$HERE/sign-manifest.sh" "$mb")"
check "key from TB_UPDATE_KEY (env) works too" test "$(jq -r .key_id "$sig2")" = "$TB_UPDATE_KEY_ID"
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$W/ec.pem" 2>/dev/null
if "$HERE/sign-manifest.sh" "$m" "$W/ec.pem" >/dev/null 2>&1; then bad "a non-ed25519 key is refused"; else ok "a non-ed25519 key is refused"; fi

echo "== publish-channels.sh (dry run)"
P="$HERE/publish-channels.sh"
gt() { "$P" --semver-gt "$1" "$2"; }
order() { gt 1.2.3 1.2.3-rc.1 && gt 1.2.3-rc.1 1.2.3-beta.10 && gt 1.2.3-beta.10 1.2.3-beta.9 && gt 1.2.3-beta.9 1.2.2; }
check "semver order: 1.2.3 > 1.2.3-rc.1 > 1.2.3-beta.10 > 1.2.3-beta.9 > 1.2.2" order
if gt 1.2.3-beta.1 1.2.3 || gt 1.2.3 1.2.3; then bad "semver: beta and equal are not newer"; else ok "semver: beta and equal are not newer"; fi
has() { grep -q -- "$1" <<<"$out"; }
hasnt() { ! grep -q -- "$1" <<<"$out"; }
out="$(GITHUB_REPOSITORY=o/r TB_UPDATE_KEY="$(cat "$TB_UPDATE_KEY_FILE")" "$P" v0.2.0 "$D" --dry-run 2>&1)"
check "plain tag: release with assets + signed manifests" \
    has '^+ gh release create v0.2.0 --repo o/r --generate-notes --verify-tag .*tailbond-0.2.0.apk.*stable.json.*stable.json.sig'
check "plain tag: not a pre-release" hasnt 'create v0.2.0.*--prerelease'
check "plain tag: stable and beta to the channels release" \
    has '^+ gh release upload channels --repo o/r --clobber .*/stable.json .*/stable.json.sig .*/beta.json .*/beta.json.sig$'
out="$(GITHUB_REPOSITORY=o/r TB_UPDATE_KEY="$(cat "$TB_UPDATE_KEY_FILE")" "$P" v1.2.3-beta.4 "$B" --dry-run 2>&1)"
check "beta tag: GitHub pre-release" has '^+ gh release create v1.2.3-beta.4 .*--prerelease'
check "beta tag: beta.json to the channels release" has '^+ gh release upload channels .*/beta.json .*/beta.json.sig$'
check "beta tag: no stable.json" hasnt '/stable.json'
out="$(env -u TB_UPDATE_KEY GITHUB_REPOSITORY=o/r "$P" v0.2.0 "$D" --dry-run 2>&1)"
check "no key: a warning" has '^::warning::TB_UPDATE_KEY'
check "no key: unsigned manifests on the version release" has 'create v0.2.0 .*stable.json'
check "no key: no signatures, channels untouched" hasnt 'json.sig\|release upload channels'

echo
[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit "$fail"
