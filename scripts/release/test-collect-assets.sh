#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# scripts/release/collect-assets.sh against fake CI artifacts: every stable
# name the docs link to (releases/latest/download/<name>) is published and
# in SHA256SUMS, plus the versioned ipk/rpm; a missing file fails it.
#
#   scripts/release/test-collect-assets.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0
ok() { echo "  ok: $*"; }
bad() { echo "  FAIL: $*"; fail=1; }

D="$T/dist"
mkdir -p "$D"/{android-release-apk,glinet-ipk,fedora-rpm,linux-bundle,macos-bundle,tailbondd/linux-amd64,tailbondd/linux-arm64}
echo apk >"$D/android-release-apk/tailbond-1.2.3.apk"
echo neon >"$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk"
echo a53 >"$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53.ipk"
echo rpm >"$D/fedora-rpm/tailbond-server-1.2.3-1.fc44.x86_64.rpm"
echo x86 >"$D/linux-bundle/tailbond-linux-x86_64.tar.gz"
echo arm >"$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
echo mac >"$D/macos-bundle/tailbond-macos-arm64.tar.gz"
echo rb >"$D/macos-bundle/tailbond.rb"
echo bin >"$D/tailbondd/linux-amd64/tailbondd"
echo bin >"$D/tailbondd/linux-arm64/tailbondd"

"$ROOT/scripts/release/collect-assets.sh" "$D" "$T/a" >"$T/log" 2>&1 || { cat "$T/log"; bad "collect-assets failed"; }
for f in tailbond-android.apk tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk tailbond-openwrt-aarch64_cortex-a53.ipk \
    tailbond-server.x86_64.rpm tailbond-linux-x86_64.tar.gz tailbond-linux-aarch64.tar.gz tailbond-macos-arm64.tar.gz \
    tailbond.rb tailbondd-linux-amd64.tar.gz tailbondd-linux-arm64.tar.gz \
    tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk tailbond_1.2.3-r1_aarch64_cortex-a53.ipk tailbond-server-1.2.3-1.fc44.x86_64.rpm; do
    if [ -s "$T/a/$f" ] && grep -q "  $f\$" "$T/a/SHA256SUMS"; then ok "$f (in SHA256SUMS)"; else bad "$f"; fi
done
[ "$(cat "$T/a/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk")" = neon ] && [ "$(cat "$T/a/tailbond-openwrt-aarch64_cortex-a53.ipk")" = a53 ] \
    && ok "each stable ipk is its architecture's" || bad "ipk architectures mixed up"
(cd "$T/a" && sha256sum -c --quiet SHA256SUMS) && ok "SHA256SUMS checks" || bad "SHA256SUMS"
[ "$(grep -c . "$T/a/SHA256SUMS")" = 13 ] && ok "13 files, no SHA256SUMS line for itself" || bad "SHA256SUMS lines: $(cat "$T/a/SHA256SUMS")"
# the docs' stable names are the ones published
for f in $(grep -rhoE 'releases/latest/download/[A-Za-z0-9._-]+' "$ROOT/README.md" "$ROOT/docs" "$ROOT/deploy" "$ROOT/agent/internal" "$ROOT/packaging" | sed 's|.*/||; s|[.]$||' | sort -u); do
    case "$f" in
        *-) continue ;; # tailbond-linux-$(uname -m).tar.gz: both bundles checked above
    esac
    [ -e "$T/a/$f" ] && ok "linked name $f is published" || bad "docs link releases/latest/download/$f, which no release has"
done

# a missing file fails it
rm "$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
if "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/b" >"$T/log" 2>&1; then bad "no aarch64 bundle, but it passed"; else ok "a missing bundle fails it"; fi
echo arm >"$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
rm "$D/android-release-apk/tailbond-1.2.3.apk"
if "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/c" >"$T/log" 2>&1; then bad "no APK, but it passed"; else grep -q 'signing secrets' "$T/log" && ok "no release APK fails with the signing hint" || bad "APK message: $(cat "$T/log")"; fi

[ "$fail" = 0 ] && echo "RESULT: PASS" || { echo "RESULT: FAIL"; exit 1; }
