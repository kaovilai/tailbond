#!/usr/bin/env bash
# shellcheck disable=SC2329 # helpers are called through check()
# SPDX-License-Identifier: Apache-2.0
#
# Self-test of the release scripts: a manifest from dummy artifacts
# (make-manifest.sh) and publish-channels.sh prepare/publish (dry run) with
# an attestation bundle (a real Sigstore bundle from agent's testdata; the
# signing itself is actions/attest in CI, keyless).
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
RC="$W/rc"
mkdir -p "$RC"
echo apk >"$RC/tailbond-1.2.3-rc.apk"
mrc="$("$HERE/make-manifest.sh" --version 1.2.3-rc --channel rc --dir "$RC" --out "$W/channels" --base-url https://x 2>/dev/null || true)"
check "pre-release without a number counts as 1 (1020201, as the app)" test "$(jq -r '.artifacts[]? | select(.kind == "apk") | .version_code' "$mrc" 2>/dev/null)" = 1020201
check "beta: security flag, three artifacts" test "$(jq -r '[.security, (.artifacts | length)] | map(tostring) | join(" ")' "$mb")" = "true 3"

echo "== publish-channels.sh (dry run)"
P="$HERE/publish-channels.sh"
gt() { "$P" --semver-gt "$1" "$2"; }
order() { gt 1.2.3 1.2.3-rc.1 && gt 1.2.3-rc.1 1.2.3-beta.10 && gt 1.2.3-beta.10 1.2.3-beta.9 && gt 1.2.3-beta.9 1.2.2; }
check "semver order: 1.2.3 > 1.2.3-rc.1 > 1.2.3-beta.10 > 1.2.3-beta.9 > 1.2.2" order
if gt 1.2.3-beta.1 1.2.3 || gt 1.2.3 1.2.3; then bad "semver: beta and equal are not newer"; else ok "semver: beta and equal are not newer"; fi
has() { grep -q -- "$1" <<<"$out"; }
hasnt() { ! grep -q -- "$1" <<<"$out"; }
BUNDLE="$HERE/../../agent/internal/update/testdata/gh-provenance.sigstore.json"
pub() { # TAG DIR
    GITHUB_REPOSITORY=o/r "$P" prepare "$1" "$2" "$W/man-$1" --dry-run >/dev/null 2>&1 &&
        GITHUB_REPOSITORY=o/r "$P" publish "$1" "$2" "$W/man-$1" "$BUNDLE" --dry-run 2>&1
}
out="$(pub v0.2.0 "$D")"
check "prepare: stable.json and beta.json" test -f "$W/man-v0.2.0/stable.json" -a -f "$W/man-v0.2.0/beta.json"
check "plain tag: release with assets + manifests + attestations" \
    has '^+ gh release create v0.2.0 --repo o/r --generate-notes --verify-tag .*tailbond-0.2.0.apk.*stable.json .*stable.json.sigstore.json'
check "plain tag: not a pre-release" hasnt 'create v0.2.0.*--prerelease'
check "plain tag: stable and beta to the channels release" \
    has '^+ gh release upload channels --repo o/r --clobber .*/stable.json .*/stable.json.sigstore.json .*/beta.json .*/beta.json.sigstore.json$'
check "the attestation next to each manifest is the bundle" cmp -s "$BUNDLE" "$W/man-v0.2.0/beta.json.sigstore.json"
check "no .sig files any more" test -z "$(find "$W/man-v0.2.0" -name '*.sig')"
out="$(pub v0.2.0 "$D")"
check "publish twice: no bundle of a bundle" test -z "$(find "$W/man-v0.2.0" -name '*.sigstore.json.sigstore.json')"
out="$(pub v1.2.3-beta.4 "$B")"
check "beta tag: GitHub pre-release" has '^+ gh release create v1.2.3-beta.4 .*--prerelease'
check "beta tag: beta.json to the channels release" has '^+ gh release upload channels .*/beta.json .*/beta.json.sigstore.json$'
check "beta tag: no stable.json" hasnt '/stable.json'
echo '{"not":"a bundle"}' >"$W/junk.json"
if GITHUB_REPOSITORY=o/r "$P" publish v0.2.0 "$D" "$W/man-v0.2.0" "$W/junk.json" --dry-run >/dev/null 2>&1; then
    bad "publish refuses a file that isn't a Sigstore bundle"
else ok "publish refuses a file that isn't a Sigstore bundle"; fi
if GITHUB_REPOSITORY=o/r "$P" publish v0.2.0 "$D" "$W/man-v0.2.0" "$W/missing.json" --dry-run >/dev/null 2>&1; then
    bad "publish refuses a missing attestation"
else ok "publish refuses a missing attestation"; fi

echo
[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit "$fail"
