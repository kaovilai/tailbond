#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# scripts/release/collect-assets.sh against fake CI artifacts: every stable
# name the docs link to (releases/latest/download/<name>) is published and
# in SHA256SUMS, plus the versioned ipk/rpm; a missing file fails it.
# The opkg feed: scripts/release/gen-opkg-key.sh (--no-upload) makes a key,
# its backup restores it; with TB_OPKG_SIGNING_KEY the release gets a
# Packages index of the versioned ipks (ipkg-make-index fields), its .gz
# and a usign signature that verifies against the key the packages ship;
# without the key no feed and a warning; a key the packages don't ship
# fails it. (The router side: scripts/test-openwrt-ipk.sh.)
#
#   scripts/release/test-collect-assets.sh      (needs gpg, git, cc)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
fail=0
ok() { echo "  ok: $*"; }
bad() { echo "  FAIL: $*"; fail=1; }

USIGN="$("$ROOT/scripts/release/usign.sh")"
export USIGN

echo "== gen-opkg-key.sh --no-upload"
printf 'test passphrase\n' >"$T/pass"
export TB_BACKUP_PASSPHRASE_FILE="$T/pass"
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --backup "$T/backup" --keys-dir "$T/keys" >"$T/log" 2>&1 || { cat "$T/log"; bad "gen-opkg-key failed"; }
PUB="$(find "$T/keys" -type f | head -n1)"
FP="${PUB##*/}"
[ -n "$PUB" ] && [ "$(find "$T/keys" -type f | wc -l)" = 1 ] && [ "$("$USIGN" -F -p "$PUB")" = "$FP" ] \
    && ok "public key written as keys/<fingerprint> ($FP)" || bad "public key file: $(ls "$T/keys")"
BK="$(find "$T/backup" -name 'tailbond-opkg-key-*.tar.gpg' | head -n1)"
mkdir -p "$T/sec"
gpg --quiet --batch --pinentry-mode loopback --passphrase-file "$T/pass" --decrypt "$BK" | tar -xf - -C "$T/sec" \
    && [ "$("$USIGN" -F -s "$T/sec/opkg.sec")" = "$FP" ] && ok "the encrypted backup holds the matching secret key" || bad "backup"
if "$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --backup "$T/backup" --keys-dir "$T/keys" >"$T/log" 2>&1; then
    bad "a second key replaced the first without --force"
else
    grep -q -- '--force' "$T/log" && [ "$(find "$T/keys" -type f)" = "$PUB" ] && ok "an existing key is kept without --force" || bad "refusal: $(cat "$T/log")"
fi
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --restore "$BK" --keys-dir "$T/keys2" >"$T/log" 2>&1 && cmp -s "$T/keys2/$FP" "$PUB" \
    && ok "--restore writes the same public key" || { cat "$T/log"; bad "restore"; }
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --restore "$BK" --keys-dir "$T/keys" >"$T/log" 2>&1 \
    && ok "--restore over the same key is fine" || { cat "$T/log"; bad "restore over same key"; }
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --force --backup "$T/backup" --keys-dir "$T/keys2" >"$T/log" 2>&1 \
    && [ "$(find "$T/keys2" -type f | wc -l)" = 1 ] && [ ! -e "$T/keys2/$FP" ] && ok "--force replaces the key" || { cat "$T/log"; bad "--force"; }
if (cd "$ROOT" && "$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --backup "$ROOT/out/bk-test" --keys-dir "$T/keys3") >"$T/log" 2>&1; then
    bad "a backup inside the git checkout was accepted"
else
    grep -q 'inside a git checkout' "$T/log" && ok "a backup inside a git checkout is refused" || bad "in-git message: $(cat "$T/log")"
fi
"$USIGN" -G -s "$T/other.sec" -p "$T/other.pub"

# mkipk OUT ARCH [PUBKEY]: an ipk like build-ipk.sh's (gzipped tar of
# debian-binary, data.tar.gz, control.tar.gz), shipping PUBKEY as a conffile
mkipk() {
    local w
    w="$(mktemp -d "$T/ipk.XXXXXX")"
    mkdir -p "$w/c" "$w/d/usr/bin" "$w/x"
    printf 'Package: tailbond\nVersion: 1.2.3-r1\nDepends: kmod-tun, ip-full\nLicense: Apache-2.0\nSection: net\nArchitecture: %s\nInstalled-Size: 4096\nDescription: Test package.\n Second line.\n' "$2" >"$w/c/control"
    echo /etc/config/tailbond >"$w/c/conffiles"
    echo "$2" >"$w/d/usr/bin/tailbondd"
    if [ -n "${3:-}" ]; then
        mkdir -p "$w/d/etc/opkg/keys"
        cp "$3" "$w/d/etc/opkg/keys/$("$USIGN" -F -p "$3")"
        echo "/etc/opkg/keys/$("$USIGN" -F -p "$3")" >>"$w/c/conffiles"
    fi
    echo 2.0 >"$w/x/debian-binary"
    tar -C "$w/c" -czf "$w/x/control.tar.gz" .
    tar -C "$w/d" -czf "$w/x/data.tar.gz" .
    tar -C "$w/x" -czf "$1" ./debian-binary ./data.tar.gz ./control.tar.gz
}
arch_of() { tar -xzOf "$1" ./control.tar.gz | tar -xzOf - ./control | sed -n 's/^Architecture: //p'; }

D="$T/dist"
mkdir -p "$D"/{android-release-apk,glinet-ipk,fedora-rpm,linux-bundle,macos-bundle,tailbondd/linux-amd64,tailbondd/linux-arm64}
echo apk >"$D/android-release-apk/tailbond-1.2.3.apk"
mkipk "$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk" aarch64_cortex-a53_neon-vfpv4 "$PUB"
mkipk "$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53.ipk" aarch64_cortex-a53 "$PUB"
echo rpm >"$D/fedora-rpm/tailbond-server-1.2.3-1.fc44.x86_64.rpm"
echo x86 >"$D/linux-bundle/tailbond-linux-x86_64.tar.gz"
echo arm >"$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
echo mac >"$D/macos-bundle/tailbond-macos-arm64.tar.gz"
echo rb >"$D/macos-bundle/tailbond.rb"
echo bin >"$D/tailbondd/linux-amd64/tailbondd"
echo bin >"$D/tailbondd/linux-arm64/tailbondd"

echo "== collect-assets.sh with the feed key"
# as GitHub hands a secret over: the file's text
TB_OPKG_SIGNING_KEY="$(cat "$T/sec/opkg.sec")" "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/a" >"$T/log" 2>&1 || { cat "$T/log"; bad "collect-assets failed"; }
for f in tailbond-android.apk tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk tailbond-openwrt-aarch64_cortex-a53.ipk \
    tailbond-server.x86_64.rpm tailbond-linux-x86_64.tar.gz tailbond-linux-aarch64.tar.gz tailbond-macos-arm64.tar.gz \
    tailbond.rb tailbondd-linux-amd64.tar.gz tailbondd-linux-arm64.tar.gz \
    tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk tailbond_1.2.3-r1_aarch64_cortex-a53.ipk tailbond-server-1.2.3-1.fc44.x86_64.rpm \
    Packages Packages.gz Packages.sig tailbond-opkg.pub; do
    if [ -s "$T/a/$f" ] && grep -q "  $f\$" "$T/a/SHA256SUMS"; then ok "$f (in SHA256SUMS)"; else bad "$f"; fi
done
[ "$(arch_of "$T/a/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk")" = aarch64_cortex-a53_neon-vfpv4 ] && [ "$(arch_of "$T/a/tailbond-openwrt-aarch64_cortex-a53.ipk")" = aarch64_cortex-a53 ] \
    && ok "each stable ipk is its architecture's" || bad "ipk architectures mixed up"
(cd "$T/a" && sha256sum -c --quiet SHA256SUMS) && ok "SHA256SUMS checks" || bad "SHA256SUMS"
[ "$(grep -c . "$T/a/SHA256SUMS")" = 17 ] && ok "17 files, no SHA256SUMS line for itself" || bad "SHA256SUMS lines: $(cat "$T/a/SHA256SUMS")"

echo "== the opkg feed"
P="$T/a/Packages"
[ "$(grep -c '^Package: tailbond$' "$P")" = 2 ] && ok "Packages lists two packages" || bad "Packages entries: $(cat "$P")"
[ "$(sed -n 's/^Filename: //p' "$P" | sort | tr '\n' ' ')" = "tailbond_1.2.3-r1_aarch64_cortex-a53.ipk tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk " ] \
    && ok "Filename: the versioned ipks only (not the stable copies)" || bad "Filenames: $(grep Filename "$P")"
feed_ok=1
for f in $(sed -n 's/^Filename: //p' "$P"); do
    entry="$(awk -v f="$f" 'BEGIN { RS = "" } $0 ~ "\nFilename: " f "\n" { print }' "$P")"
    [ "$(sed -n 's/^Size: //p' <<<"$entry")" = "$(wc -c <"$T/a/$f" | tr -d ' ')" ] || { feed_ok=0; bad "$f: Size"; }
    [ "$(sed -n 's/^SHA256sum: //p' <<<"$entry")" = "$(sha256sum "$T/a/$f" | cut -d' ' -f1)" ] || { feed_ok=0; bad "$f: SHA256sum"; }
    [ "$(sed -n 's/^Architecture: //p' <<<"$entry")" = "$(arch_of "$T/a/$f")" ] || { feed_ok=0; bad "$f: Architecture"; }
    # ipkg-make-index: the control file, Filename/Size/SHA256sum before Description
    [ "$(grep -o '^[A-Za-z0-9-]*:' <<<"$entry" | tr -d '\n')" = "Package:Version:Depends:License:Section:Architecture:Installed-Size:Filename:Size:SHA256sum:Description:" ] \
        && grep -q '^ Second line[.]$' <<<"$entry" || { feed_ok=0; bad "$f: fields $(grep -o '^[A-Za-z0-9-]*:' <<<"$entry" | tr -d '\n')"; }
done
[ "$feed_ok" = 1 ] && ok "each entry: control fields + Filename, Size, SHA256sum of the published file"
[ "$(tail -c 2 "$P" | od -An -c | tr -d ' ')" = '\n\n' ] && ok "entries end with a blank line" || bad "Packages ending"
gzip -dc "$T/a/Packages.gz" | cmp -s - "$P" && ok "Packages.gz is Packages" || bad "Packages.gz"
cmp -s "$T/a/tailbond-opkg.pub" "$PUB" && ok "tailbond-opkg.pub is the key the packages ship" || bad "tailbond-opkg.pub"
"$USIGN" -V -q -m "$P" -x "$T/a/Packages.sig" -p "$PUB" && ok "Packages.sig verifies (usign -V) with that key" || bad "signature"
mkdir -p "$T/kd" && cp "$PUB" "$T/kd/$FP"
"$USIGN" -V -q -m "$P" -x "$T/a/Packages.sig" -P "$T/kd" && ok "... and from a key directory, as opkg-key verify does" || bad "signature via -P"
sed 's/^Version: 1.2.3-r1$/Version: 9.9.9-r1/' "$P" >"$T/tampered"
! "$USIGN" -V -q -m "$T/tampered" -x "$T/a/Packages.sig" -p "$PUB" 2>/dev/null && ok "a changed Packages fails the check" || bad "tampered Packages verified"
! "$USIGN" -V -q -m "$P" -x "$T/a/Packages.sig" -p "$T/other.pub" 2>/dev/null && ok "another key fails the check" || bad "wrong key verified"
# the docs' stable names are the ones published
for f in $(grep -rhoE 'releases/latest/download/[A-Za-z0-9._-]+' "$ROOT/README.md" "$ROOT/docs" "$ROOT/deploy" "$ROOT/agent/internal" "$ROOT/packaging" | sed 's|.*/||; s|[.]$||' | sort -u); do
    case "$f" in
        *-) continue ;; # tailbond-linux-$(uname -m).tar.gz: both bundles checked above
    esac
    [ -e "$T/a/$f" ] && ok "linked name $f is published" || bad "docs link releases/latest/download/$f, which no release has"
done

echo "== no feed key: a release without the feed, and a warning"
"$ROOT/scripts/release/collect-assets.sh" "$D" "$T/nk" >"$T/log" 2>&1 || { cat "$T/log"; bad "no key: collect-assets failed"; }
grep -q '^::warning::.*TB_OPKG_SIGNING_KEY is not set' "$T/log" && ok "warns that there's no feed" || bad "no warning: $(cat "$T/log")"
[ ! -e "$T/nk/Packages" ] && [ ! -e "$T/nk/Packages.gz" ] && [ ! -e "$T/nk/Packages.sig" ] && ! grep -q Packages "$T/nk/SHA256SUMS" \
    && [ "$(grep -c . "$T/nk/SHA256SUMS")" = 13 ] && ok "no Packages* published, the 13 other files are" || bad "keyless release: $(ls "$T/nk")"

echo "== a key the packages don't ship fails"
if TB_OPKG_SIGNING_KEY="$(cat "$T/other.sec")" "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/wk" >"$T/log" 2>&1; then
    bad "signed with a key the packages don't carry, but it passed"
else
    grep -q "doesn't ship the feed's public key" "$T/log" && ok "the packages don't carry the signing key: fails, names the fix" || bad "wrong key message: $(cat "$T/log")"
fi
mkdir -p "$T/nokey"
mkipk "$T/nokey/tailbond_1.2.3-r1_aarch64_cortex-a53.ipk" aarch64_cortex-a53
if "$ROOT/scripts/release/opkg-feed.sh" "$T/nokey" "$T/sec/opkg.sec" >"$T/log" 2>&1; then
    bad "a package without any key, but it signed"
else
    grep -q "doesn't ship the feed's public key" "$T/log" && ok "a package without the key fails" || bad "no-key package: $(cat "$T/log")"
fi
"$ROOT/scripts/release/opkg-feed.sh" "$T/nokey" >"$T/log" 2>&1 && [ -s "$T/nokey/Packages" ] && [ ! -e "$T/nokey/Packages.sig" ] \
    && ok "no key given: an unsigned index (scripts/test-openwrt-ipk.sh signs its own)" || { cat "$T/log"; bad "unsigned index"; }

echo "== missing files"
# a missing file fails it
rm "$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
if "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/b" >"$T/log" 2>&1; then bad "no aarch64 bundle, but it passed"; else ok "a missing bundle fails it"; fi
echo arm >"$D/linux-bundle/tailbond-linux-aarch64.tar.gz"
rm "$D/android-release-apk/tailbond-1.2.3.apk"
mkdir -p "$T/used/brand" && echo x >"$T/used/icon.svg"
if "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/used" >"$T/log" 2>&1; then bad "a non-empty OUT (like the repo's assets/), but it passed"; else grep -q "isn't empty" "$T/log" && ok "a non-empty OUT is refused" || bad "non-empty OUT message: $(cat "$T/log")"; fi
if "$ROOT/scripts/release/collect-assets.sh" "$D" "$T/c" >"$T/log" 2>&1; then bad "no APK, but it passed"; else grep -q 'signing secrets' "$T/log" && ok "no release APK fails with the signing hint" || bad "APK message: $(cat "$T/log")"; fi

[ "$fail" = 0 ] && echo "RESULT: PASS" || { echo "RESULT: FAIL"; exit 1; }
