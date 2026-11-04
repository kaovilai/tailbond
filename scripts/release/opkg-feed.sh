#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The opkg feed of a release (docs/glinet.md, "Install and update from GL's
# Plug-ins page"): routers add
#   src/gz tailbond https://github.com/<repo>/releases/latest/download
# and opkg fetches Packages.gz and Packages.sig from there, then the
# Filename of the package it installs.
#
#   scripts/release/opkg-feed.sh DIR [SECRET_KEY]
#
# Writes DIR/Packages and DIR/Packages.gz listing every versioned package in
# DIR (<Package>_<version>_<Architecture>.ipk, as its control file names it;
# the stable copies like tailbond-glinet-<arch>.ipk are skipped). Each entry
# is the package's control file plus Filename, Size and SHA256sum before
# Description, as OpenWrt's ipkg-make-index.sh writes it.
#
# With SECRET_KEY (a usign secret key): signs Packages into Packages.sig
# (opkg's check_signature verifies the uncompressed list) and writes the
# public key as DIR/tailbond-opkg.pub. Every package must ship that key as
# /etc/opkg/keys/<fingerprint> (a conffile), so a router that installed one
# trusts the feed; the signature is checked against the key the packages
# carry. Fails otherwise.
#
# Env: USIGN (else scripts/release/usign.sh builds it).
set -euo pipefail

die() { echo "::error::opkg-feed: $*" >&2; exit 1; }
[ $# -ge 1 ] && [ $# -le 2 ] || { echo "usage: $0 DIR [SECRET_KEY]" >&2; exit 2; }
DIR="$1"
KEY="${2:-}"
[ -d "$DIR" ] || die "$DIR is not a directory"
[ -z "$KEY" ] || [ -s "$KEY" ] || die "secret key $KEY is missing or empty"

T="$(mktemp -d "${TMPDIR:-/tmp}/opkg-feed.XXXXXX")"
trap 'rm -rf "$T"' EXIT

field() { sed -n "s/^$1: *//p" "$2" | head -n1; }

fp=""
if [ -n "$KEY" ]; then
    USIGN="$(USIGN="${USIGN:-}" "$(dirname "$0")/usign.sh")"
    fp="$("$USIGN" -F -s "$KEY" 2>/dev/null)" || die "$KEY is not a usign secret key"
    [[ "$fp" =~ ^[0-9a-f]{16}$ ]] || die "unexpected fingerprint '$fp' from $KEY"
fi

: >"$T/Packages"
n=0
for ipk in "$DIR"/*.ipk; do
    [ -f "$ipk" ] || continue
    b="${ipk##*/}"
    tar -xzOf "$ipk" ./control.tar.gz 2>/dev/null | tar -xzOf - ./control >"$T/control" 2>/dev/null ||
        die "$b: no ./control in its control.tar.gz"
    pkg="$(field Package "$T/control")"
    arch="$(field Architecture "$T/control")"
    [ -n "$pkg" ] && [ -n "$arch" ] || die "$b: control has no Package or Architecture"
    # only the versioned file name: its stable copy is the same package
    case "$b" in
        "${pkg}_"*"_${arch}.ipk") ;;
        *) continue ;;
    esac
    grep -q '^Description:' "$T/control" || die "$b: control has no Description"
    if [ -n "$fp" ]; then
        tar -xzOf "$ipk" ./data.tar.gz | tar -xzOf - "./etc/opkg/keys/$fp" >"$T/pub" 2>/dev/null && [ -s "$T/pub" ] ||
            die "$b doesn't ship the feed's public key /etc/opkg/keys/$fp: commit the key that matches secret TB_OPKG_SIGNING_KEY (scripts/release/gen-opkg-key.sh) and rebuild, or the routers that install it can't check the feed"
        tar -xzOf "$ipk" ./control.tar.gz | tar -xzOf - ./conffiles 2>/dev/null | grep -qx "/etc/opkg/keys/$fp" ||
            die "$b: /etc/opkg/keys/$fp isn't in its conffiles"
        if [ -s "$T/key.pub" ]; then
            cmp -s "$T/pub" "$T/key.pub" || die "$b ships a different /etc/opkg/keys/$fp than the other packages"
        else
            cp "$T/pub" "$T/key.pub"
        fi
    fi
    size="$(wc -c <"$ipk" | tr -d ' ')"
    sum="$(sha256sum "$ipk" | cut -d' ' -f1)"
    awk -v f="$b" -v s="$size" -v h="$sum" '
        /^Description:/ { print "Filename: " f; print "Size: " s; print "SHA256sum: " h }
        NF { print }' "$T/control" >>"$T/Packages"
    echo >>"$T/Packages"
    n=$((n + 1))
done
[ "$n" -gt 0 ] || die "no versioned package (<Package>_<version>_<arch>.ipk) in $DIR"

cp "$T/Packages" "$DIR/Packages"
gzip -9n -c "$T/Packages" >"$DIR/Packages.gz"
if [ -n "$fp" ]; then
    rm -f "$DIR/Packages.sig"
    "$USIGN" -S -m "$DIR/Packages" -s "$KEY" -x "$DIR/Packages.sig" || die "signing Packages failed"
    # what the router does: opkg-key verify = usign -V against /etc/opkg/keys
    "$USIGN" -V -q -m "$DIR/Packages" -x "$DIR/Packages.sig" -p "$T/key.pub" ||
        die "Packages.sig doesn't verify against the key in the packages"
    cp "$T/key.pub" "$DIR/tailbond-opkg.pub"
    echo "opkg feed: $n packages, signed by $fp"
else
    echo "opkg feed: $n packages, unsigned"
fi
