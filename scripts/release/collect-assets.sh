#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Gather a release's assets from the CI artifacts (job "release") under
# names that stay the same in every release, so
# https://github.com/<repo>/releases/latest/download/<name> always works
# (README, Download):
#
#   tailbond-android.apk                                 Android (release key)
#   tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk    GL.iNet stock firmware
#   tailbond-openwrt-aarch64_cortex-a53.ipk              OpenWrt 23.05+
#   tailbond-server.x86_64.rpm                           Fedora
#   tailbond-linux-x86_64.tar.gz, tailbond-linux-aarch64.tar.gz   VPS, Linux client
#   tailbond-macos-arm64.tar.gz, tailbond.rb             Mac (bundle, Homebrew)
#   tailbondd-linux-amd64.tar.gz, tailbondd-linux-arm64.tar.gz    the agent alone
#   SHA256SUMS                                           all of the above
#
# plus the versioned packages package tools read the version from:
# tailbond_<version>-r1_<arch>.ipk (the router's package cache and its
# reinstall after a firmware upgrade, the opkg feed) and
# tailbond-server-<version>.rpm.
#
# The opkg feed (docs/glinet.md, "Install and update from GL's Plug-ins
# page"), when TB_OPKG_SIGNING_KEY holds the usign secret key: Packages,
# Packages.gz, Packages.sig and tailbond-opkg.pub
# (scripts/release/opkg-feed.sh). Without the key the release has no feed
# (a warning); with a key the packages don't carry, it fails.
#
#   scripts/release/collect-assets.sh DIST OUT
#
# DIST: the downloaded artifacts (one directory per artifact name). OUT:
# created, and must not exist or be empty: everything in it is published.
# Fails when a file is missing.
set -euo pipefail

[ $# -eq 2 ] || { echo "usage: $0 DIST OUT" >&2; exit 2; }
DIST="$1"
OUT="$2"
die() { echo "::error::collect-assets: $*" >&2; exit 1; }
one() { # PATTERN: exactly one file
    local m=()
    # shellcheck disable=SC2206 # a glob on purpose
    m=($1)
    [ ${#m[@]} -eq 1 ] && [ -f "${m[0]}" ] || die "want one file for $1, found: ${m[*]}"
    echo "${m[0]}"
}

# Everything in OUT is published, so start from nothing: a reused directory
# (the repo's own assets/, say) would ship stray files or break SHA256SUMS.
if [ -e "$OUT" ] && [ -n "$(ls -A "$OUT" 2>/dev/null)" ]; then
    die "$OUT already exists and isn't empty: give a new directory"
fi
mkdir -p "$OUT"

# Android: never publish an APK that can't update the installed app
if ! compgen -G "$DIST/android-release-apk/tailbond-*.apk" >/dev/null; then
    die "no release APK: the Android signing secrets are missing (TB_RELEASE_KEYSTORE_B64 ...). Run scripts/release/gen-android-keystore.sh once (or --restore the key from its backup), then re-run this tag. docs/android.md, Release key."
fi
cp "$(one "$DIST/android-release-apk/tailbond-*.apk")" "$OUT/tailbond-android.apk"

# GL.iNet / OpenWrt: the versioned ipk and its stable copy
n=0
for f in "$DIST"/glinet-ipk/tailbond_*.ipk; do
    [ -f "$f" ] || continue
    b="${f##*/}"
    arch="$(sed -n 's/^tailbond_.*-r[0-9]*_\(.*\)\.ipk$/\1/p' <<<"$b")"
    [ -n "$arch" ] || die "$b: not tailbond_<version>-r<N>_<arch>.ipk"
    case "$arch" in
        *_neon-vfpv4) stable="tailbond-glinet-$arch.ipk" ;;
        *) stable="tailbond-openwrt-$arch.ipk" ;;
    esac
    cp "$f" "$OUT/$b"
    cp "$f" "$OUT/$stable"
    n=$((n + 1))
done
[ "$n" -gt 0 ] || die "no ipk in $DIST/glinet-ipk"

# Fedora: the versioned RPM (dnf shows its name) and a stable copy
rpm="$(one "$DIST/fedora-rpm/tailbond-server-[0-9]*.x86_64.rpm")"
cp "$rpm" "$OUT/"
cp "$rpm" "$OUT/tailbond-server.x86_64.rpm"

for f in tailbond-linux-x86_64.tar.gz tailbond-linux-aarch64.tar.gz; do
    cp "$(one "$DIST/linux-bundle/$f")" "$OUT/"
done
# macOS: the bundle (install.sh, one-click updates) and the Homebrew
# formula with this version's checksum (docs/macos.md)
cp "$(one "$DIST/macos-bundle/tailbond-macos-arm64.tar.gz")" "$(one "$DIST/macos-bundle/tailbond.rb")" "$OUT/"

# the agent alone, per architecture
for d in "$DIST"/tailbondd/linux-*; do
    [ -f "$d/tailbondd" ] || continue
    tar -czf "$OUT/tailbondd-$(basename "$d").tar.gz" -C "$d" tailbondd
done

for f in tailbond-android.apk tailbond-glinet-aarch64_cortex-a53_neon-vfpv4.ipk tailbond-openwrt-aarch64_cortex-a53.ipk \
    tailbond-server.x86_64.rpm tailbond-linux-x86_64.tar.gz tailbond-linux-aarch64.tar.gz \
    tailbond-macos-arm64.tar.gz tailbond.rb tailbondd-linux-amd64.tar.gz tailbondd-linux-arm64.tar.gz; do
    [ -s "$OUT/$f" ] || die "$f missing"
done
# the opkg feed: signed, or none
if [ -n "${TB_OPKG_SIGNING_KEY:-}" ]; then
    kd="$(mktemp -d "${TMPDIR:-/tmp}/tb-opkg-key.XXXXXX")"
    (umask 077; printf '%s\n' "$TB_OPKG_SIGNING_KEY" >"$kd/opkg.sec")
    rc=0
    "$(dirname "$0")/opkg-feed.sh" "$OUT" "$kd/opkg.sec" || rc=$?
    rm -rf "$kd"
    [ "$rc" = 0 ] || die "the opkg feed failed (above)"
    for f in Packages Packages.gz Packages.sig tailbond-opkg.pub; do
        [ -s "$OUT/$f" ] || die "$f missing"
    done
else
    echo "::warning::collect-assets: secret TB_OPKG_SIGNING_KEY is not set, so this release has no opkg feed (Packages, Packages.sig): GL's Plug-ins page can't install or update it. Run scripts/release/gen-opkg-key.sh once and commit the public key it writes (docs/glinet.md)."
fi

(cd "$OUT" && rm -f SHA256SUMS && sha256sum -- * >SHA256SUMS.tmp && mv SHA256SUMS.tmp SHA256SUMS)
cat "$OUT/SHA256SUMS"
