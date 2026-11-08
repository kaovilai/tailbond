#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# scripts/release/collect-assets.sh against fake CI artifacts: every stable
# name the docs link to (releases/latest/download/<name>) is published and
# in SHA256SUMS, plus the versioned ipk/rpm; a missing file fails it.
# The opkg feed: scripts/release/gen-opkg-key.sh (--no-upload) makes a key,
# its backup restores it; with a fake gh, a secret listing that fails or
# names the secret stops it before anything changes; a rotation (--next,
# then --restore --force) in a scratch git repository, committed with the
# commands the script prints, ends with only the new key tracked, and the
# feeds on the way verify for routers with the old key, then both; with TB_OPKG_SIGNING_KEY the release gets a
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

# mkipk OUT ARCH [PUBKEY...]: an ipk like build-ipk.sh's (gzipped tar of
# debian-binary, data.tar.gz, control.tar.gz), shipping each PUBKEY as a
# conffile
mkipk() {
    local w
    w="$(mktemp -d "$T/ipk.XXXXXX")"
    mkdir -p "$w/c" "$w/d/usr/bin" "$w/x"
    printf 'Package: tailbond\nVersion: 1.2.3-r1\nDepends: kmod-tun, ip-full\nLicense: Apache-2.0\nSection: net\nArchitecture: %s\nInstalled-Size: 4096\nDescription: Test package.\n Second line.\n' "$2" >"$w/c/control"
    echo /etc/config/tailbond >"$w/c/conffiles"
    echo "$2" >"$w/d/usr/bin/tailbondd"
    local k
    for k in "${@:3}"; do
        mkdir -p "$w/d/etc/opkg/keys"
        cp "$k" "$w/d/etc/opkg/keys/$("$USIGN" -F -p "$k")"
        echo "/etc/opkg/keys/$("$USIGN" -F -p "$k")" >>"$w/c/conffiles"
    done
    echo 2.0 >"$w/x/debian-binary"
    tar -C "$w/c" -czf "$w/x/control.tar.gz" .
    tar -C "$w/d" -czf "$w/x/data.tar.gz" .
    tar -C "$w/x" -czf "$1" ./debian-binary ./data.tar.gz ./control.tar.gz
}
arch_of() { tar -xzOf "$1" ./control.tar.gz | tar -xzOf - ./control | sed -n 's/^Architecture: //p'; }

echo "== gen-opkg-key.sh and the repository secret (fake gh)"
# FAKE_GH_LIST: fail (gh can't list: auth, network), has (the secret
# exists), else an empty list. secret set writes $T/gh-set.<NAME>.
mkdir -p "$T/bin"
cat >"$T/bin/gh" <<'GH'
#!/bin/sh
case "$1 $2" in
"secret list")
    case "${FAKE_GH_LIST:-}" in
    fail) echo "HTTP 401: Bad credentials (https://api.github.com/repos/o/r/actions/secrets)" >&2; exit 1 ;;
    has) printf 'OTHER_SECRET\nTB_OPKG_SIGNING_KEY\n' ;;
    *) echo OTHER_SECRET ;;
    esac ;;
"secret set") cat >"$FAKE_GH_DIR/gh-set.$3" ;;
*) echo "fake gh: $*" >&2; exit 2 ;;
esac
GH
chmod +x "$T/bin/gh"
export FAKE_GH_DIR="$T"
ghgen() { # FAKE_GH_LIST KEYS-DIR [ARGS...]: gen-opkg-key.sh with the fake gh
    local list="$1" keys="$2"
    shift 2
    FAKE_GH_LIST="$list" GH="$T/bin/gh" "$ROOT/scripts/release/gen-opkg-key.sh" --repo o/r --backup "$T/ghbk-$list" --keys-dir "$keys" "$@" >"$T/log" 2>&1
}
if ghgen fail "$T/gk1"; then
    bad "gh secret list failed, but it went on"
else
    grep -q "can't list the secrets" "$T/log" && [ ! -e "$T/gh-set.TB_OPKG_SIGNING_KEY" ] && [ -z "$(ls -A "$T/gk1" 2>/dev/null)" ] \
        && [ -z "$(ls -A "$T/ghbk-fail" 2>/dev/null)" ] && ok "gh can't list the secrets: stops, no secret set, no key or backup written" || bad "failed listing: $(cat "$T/log")"
fi
if ghgen has "$T/gk2"; then
    bad "the secret exists, but it was replaced without --force"
else
    grep -q 'already has an opkg feed key' "$T/log" && [ ! -e "$T/gh-set.TB_OPKG_SIGNING_KEY" ] && [ -z "$(ls -A "$T/gk2" 2>/dev/null)" ] \
        && ok "the secret exists (empty key directory): refused without --force" || bad "existing secret: $(cat "$T/log")"
fi
ghgen none "$T/gk3" && [ -s "$T/gh-set.TB_OPKG_SIGNING_KEY" ] \
    && [ "$("$USIGN" -F -s "$T/gh-set.TB_OPKG_SIGNING_KEY")" = "$(ls "$T/gk3")" ] \
    && ok "no such secret: sets it to the secret key of the public key written" || { cat "$T/log"; bad "first upload"; }
rm -f "$T/gh-set.TB_OPKG_SIGNING_KEY"
ghgen fail "$T/gk4" --force && [ -s "$T/gh-set.TB_OPKG_SIGNING_KEY" ] \
    && ok "--force replaces without listing" || { cat "$T/log"; bad "--force upload"; }

echo "== rotating the key (--next, then --restore --force), committed as printed"
R="$T/repo"
git init -q "$R"
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com
RK="$R/packaging/openwrt/files/etc/opkg/keys"
echo readme >"$R/README" && git -C "$R" add README && git -C "$R" commit -q -m init
committed() { # run the "commit it:" command the script printed
    local c
    c="$(sed -n 's/^commit it: //p' "$T/log")"
    [ -n "$c" ] && eval "$c" >/dev/null
}
tracked() { git -C "$R" ls-files -- packaging/openwrt/files/etc/opkg/keys | sed 's|.*/||' | sort | tr '\n' ' '; }
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --backup "$T/rbk1" --keys-dir "$RK" >"$T/log" 2>&1 && committed || { cat "$T/log"; bad "first key in the repo"; }
OLD="$(ls "$RK")"
BK_OLD="$(ls "$T"/rbk1/*.gpg)"
[ "$(tracked)" = "$OLD " ] && [ -z "$(git -C "$R" status --porcelain)" ] && ok "first key: the printed command commits it" || bad "first commit: $(tracked) / $(git -C "$R" status --porcelain)"
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --backup "$T/rbk2" --keys-dir "$RK" >"$T/log" 2>&1 && bad "a second key without --next or --force" \
    || { grep -q -- '--next' "$T/log" && ok "a second key needs --next or --force (the refusal names --next)" || bad "refusal: $(cat "$T/log")"; }
"$ROOT/scripts/release/gen-opkg-key.sh" --next --backup "$T/rbk2" --keys-dir "$RK" >"$T/log" 2>&1 && committed || { cat "$T/log"; bad "--next"; }
NEW=""
for k in "$RK"/*; do [ "${k##*/}" = "$OLD" ] || NEW="${k##*/}"; done
BK_NEW="$(ls "$T"/rbk2/*.gpg)"
[ -n "$NEW" ] && [ "$(tracked)" = "$(printf '%s\n' "$OLD" "$NEW" | sort | tr '\n' ' ')" ] && [ -z "$(git -C "$R" status --porcelain)" ] \
    && ok "--next: both public keys tracked, the current one kept" || bad "--next tracked: $(tracked)"
# the transition release: signed with the old key, shipping both
mkdir -p "$T/sec-old" "$T/sec-new" "$T/router-old" "$T/router-both" "$T/rot1" "$T/rot2"
gpg --quiet --batch --pinentry-mode loopback --passphrase-file "$T/pass" --decrypt "$BK_OLD" | tar -xf - -C "$T/sec-old"
gpg --quiet --batch --pinentry-mode loopback --passphrase-file "$T/pass" --decrypt "$BK_NEW" | tar -xf - -C "$T/sec-new"
cp "$RK/$OLD" "$T/router-old/"
cp "$RK/$OLD" "$RK/$NEW" "$T/router-both/"
mkipk "$T/rot1/tailbond_1.2.4-r1_aarch64_cortex-a53.ipk" aarch64_cortex-a53 "$RK/$OLD" "$RK/$NEW"
"$ROOT/scripts/release/opkg-feed.sh" "$T/rot1" "$T/sec-old/opkg.sec" >"$T/log" 2>&1 \
    && "$USIGN" -V -q -m "$T/rot1/Packages" -x "$T/rot1/Packages.sig" -P "$T/router-old" \
    && ok "transition release (both keys, signed with the old): a router with only the old key accepts it" || { cat "$T/log"; bad "transition feed"; }
# the switch: the secret becomes the new key, the old public key goes
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --restore "$BK_NEW" --keys-dir "$RK" >"$T/log" 2>&1 && bad "the switch without --force" \
    || ok "the switch (--restore of the next key) needs --force"
"$ROOT/scripts/release/gen-opkg-key.sh" --no-upload --restore "$BK_NEW" --force --keys-dir "$RK" >"$T/log" 2>&1 && committed || { cat "$T/log"; bad "switch"; }
[ "$(tracked)" = "$NEW " ] && [ ! -e "$RK/$OLD" ] && [ -z "$(git -C "$R" status --porcelain)" ] \
    && ok "--restore --force: the printed command commits the old key's removal too (only the new key tracked)" || bad "after the switch: tracked $(tracked), status $(git -C "$R" status --porcelain)"
grep -q "removed public key(s): $OLD" "$T/log" && ok "it says which key it removed" || bad "removal not reported: $(cat "$T/log")"
mkipk "$T/rot2/tailbond_1.2.5-r1_aarch64_cortex-a53.ipk" aarch64_cortex-a53 "$RK/$NEW"
"$ROOT/scripts/release/opkg-feed.sh" "$T/rot2" "$T/sec-new/opkg.sec" >"$T/log" 2>&1 \
    && "$USIGN" -V -q -m "$T/rot2/Packages" -x "$T/rot2/Packages.sig" -P "$T/router-both" \
    && ok "after the switch: a router that took the transition release accepts the new feed" || { cat "$T/log"; bad "post-switch feed"; }
! "$USIGN" -V -q -m "$T/rot2/Packages" -x "$T/rot2/Packages.sig" -P "$T/router-old" 2>/dev/null \
    && ok "... one that only has the old key refuses it (it needs the key by hand)" || bad "old-key router accepted the new feed"

D="$T/dist"
mkdir -p "$D"/{android-release-apk,glinet-ipk,openwrt-apk,fedora-rpm,linux-bundle,macos-bundle,tailbondd/linux-amd64,tailbondd/linux-arm64}
echo apk >"$D/android-release-apk/tailbond-1.2.3.apk"
mkipk "$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk" aarch64_cortex-a53_neon-vfpv4 "$PUB"
mkipk "$D/glinet-ipk/tailbond_1.2.3-r1_aarch64_cortex-a53.ipk" aarch64_cortex-a53 "$PUB"
echo apk >"$D/openwrt-apk/tailbond_1.2.3-r1_aarch64_cortex-a53.apk"
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
    tailbond-openwrt-aarch64_cortex-a53.apk \
    tailbond_1.2.3-r1_aarch64_cortex-a53_neon-vfpv4.ipk tailbond_1.2.3-r1_aarch64_cortex-a53.ipk tailbond_1.2.3-r1_aarch64_cortex-a53.apk tailbond-server-1.2.3-1.fc44.x86_64.rpm \
    Packages Packages.gz Packages.sig tailbond-opkg.pub; do
    if [ -s "$T/a/$f" ] && grep -q "  $f\$" "$T/a/SHA256SUMS"; then ok "$f (in SHA256SUMS)"; else bad "$f"; fi
done
[ "$(arch_of "$T/a/tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk")" = aarch64_cortex-a53_neon-vfpv4 ] && [ "$(arch_of "$T/a/tailbond-openwrt-aarch64_cortex-a53.ipk")" = aarch64_cortex-a53 ] \
    && ok "each stable ipk is its architecture's" || bad "ipk architectures mixed up"
(cd "$T/a" && sha256sum -c --quiet SHA256SUMS) && ok "SHA256SUMS checks" || bad "SHA256SUMS"
[ "$(grep -c . "$T/a/SHA256SUMS")" = 19 ] && ok "19 files, no SHA256SUMS line for itself" || bad "SHA256SUMS lines: $(cat "$T/a/SHA256SUMS")"

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
    && [ "$(grep -c . "$T/nk/SHA256SUMS")" = 15 ] && ok "no Packages* published, the 15 other files are" || bad "keyless release: $(ls "$T/nk")"

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
