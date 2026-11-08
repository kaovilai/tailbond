#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The apk feed of a release (docs/glinet.md, "OpenWrt 25.12 and apk"):
# OpenWrt 25.12+ routers add the packages.adb URL and apk fetches it and
# the packages it lists, checking the index's signature against the keys
# in /etc/apk/keys/ (the package installs the release's public key).
#
#   scripts/release/apk-feed.sh DIR [SECRET_KEY]
#
# DIR holds the release's apk packages, named tailbond_<version>_<arch>.apk
# (out/apk, as packaging/openwrt/build-apk.sh writes them). Writes, per
# architecture in DIR:
#   DIR/feed/<arch>/tailbond-<version>.apk  the packages, as apk fetches
#                                           them from the index
#   DIR/feed/<arch>/packages.adb            the index (apk mkndx), signed
#                                           with SECRET_KEY when given
# and DIR/tailbond-apk.pub, the public key for the documented first step
# (cp into /etc/apk/keys/). Unlike opkg's Packages.gz, an apk feed cannot
# live behind a redirect (apk does not follow them, openwrt/openwrt#17180):
# GitHub Releases URLs redirect, so publish DIR/feed/ on a server that
# answers directly (GitHub Pages, a VPS).
#
# With SECRET_KEY (an EC P-256 private key, scripts/release/
# gen-apk-key.sh): every package must ship that key's public half in
# /etc/apk/keys/ (its content hash is checked), or routers that installed
# one could not check the feed. Fails otherwise.
#
# Env: APK_TOOLS (default scripts/release/apk-tools.sh), PODMAN, APK_IMAGE
# (see apk-tools.sh).
set -euo pipefail

die() { echo "::error::apk-feed: $*" >&2; exit 1; }
[ $# -ge 1 ] && [ $# -le 2 ] || { echo "usage: $0 DIR [SECRET_KEY]" >&2; exit 2; }
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DIR="$1"
KEY="${2:-}"
APK_TOOLS="${APK_TOOLS:-$ROOT/scripts/release/apk-tools.sh}"
[ -d "$DIR" ] || die "$DIR is not a directory"
DIR="$(cd "$DIR" && pwd)"
[ -z "$KEY" ] || [ -s "$KEY" ] || die "secret key $KEY is missing or empty"
command -v python3 >/dev/null || die "python3 not found"

# the public key half of SECRET_KEY, checked against the packages
pub=""
if [ -n "$KEY" ]; then
    command -v openssl >/dev/null || die "openssl not found"
    pub="$(mktemp "${TMPDIR:-/tmp}/apk-feed-pub.XXXXXX")"
    trap 'rm -f "$pub"' EXIT
    openssl ec -in "$KEY" -pubout -out "$pub" 2>/dev/null || die "$KEY is not an EC private key"
fi

# the apk files by arch, their name/version/arch read from the package
# metadata (an arch like aarch64_cortex-a53 has an underscore, so the file
# name alone doesn't split)
archs=()
declare -A names=()
declare -A vers=()
for f in "$DIR"/tailbond_*.apk; do
    [ -f "$f" ] || continue
    b="${f##*/}"
    meta="$("$APK_TOOLS" "$DIR" adbdump --format json "$b" |
        python3 -c '
import json, sys
i = json.load(sys.stdin).get("info", {})
print(i.get("name", ""), i.get("version", ""), i.get("arch", ""))
')" || die "$b: not an apk package"
    read -r name ver arch <<<"$meta"
    [ "$name" = tailbond ] && [ -n "$ver" ] && [ -n "$arch" ] ||
        die "$b: apk metadata is $meta (want the tailbond package)"
    if [ -z "${names[$arch]+x}" ]; then
        archs+=("$arch")
        names["$arch"]=""
    fi
    names["$arch"]+="$b "
    vers["$b"]="$ver"
done
[ "${#archs[@]}" -gt 0 ] || die "no tailbond_*.apk in $DIR"

for arch in "${archs[@]}"; do
    out="$DIR/feed/$arch"
    rm -rf "$out"
    mkdir -p "$out"
    # the packages, under the names the index points at: <name>-<version>.apk
    for b in ${names["$arch"]}; do
        cp "$DIR/$b" "$out/tailbond-${vers["$b"]}.apk"
    done
    sign=()
    if [ -n "$KEY" ]; then
        # apk-tools mounts only $out: the secret key travels with it
        cp "$KEY" "$out/.signing-key"
        sign=(--sign .signing-key)
    fi
    (cd "$out" && "$APK_TOOLS" "$out" mkndx --allow-untrusted "${sign[@]}" \
        --output packages.adb tailbond-*.apk) >/dev/null ||
        die "mkndx failed for $arch"
    rm -f "$out/.signing-key"
    # the packages ship the feed's public key: check the content hash apk
    # recorded for it (adbdump), as the router sees it after installing
    if [ -n "$pub" ]; then
        want="$(sha256sum "$pub" | cut -d' ' -f1)"
        for b in ${names["$arch"]}; do
            got="$("$APK_TOOLS" "$DIR" adbdump --format json "$b" |
                python3 -c '
import json, sys
d = json.load(sys.stdin)
for p in d.get("paths", []):
    if p.get("name") == "etc/apk/keys":
        for f in p.get("files", []):
            print(f.get("hash", ""))
' | grep -x "$want" | head -1)"
            [ "$got" = "$want" ] ||
                die "$b does not ship the feed public key (hash $want): commit the key that matches secret TB_APK_SIGNING_KEY (scripts/release/gen-apk-key.sh) and rebuild, or routers that install it cannot check the feed"
        done
    fi
    echo "apk feed: $out ($({ cd "$out" && ls tailbond-*.apk | wc -l; }) packages$( [ -n "$KEY" ] && echo ', signed'))"
done
[ -z "$pub" ] || cp "$pub" "$DIR/tailbond-apk.pub"
