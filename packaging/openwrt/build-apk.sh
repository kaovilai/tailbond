#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Assemble the OpenWrt 25.12+ package (apk) from the .ipk build-ipk.sh
# made: same files, same scripts, apk control metadata, OpenWrt's apk
# conffile list (/lib/apk/packages/tailbond.conffiles, which OpenWrt's
# patched apk reads: /etc/config/tailbond survives upgrades as with opkg)
# and the apk feed's public key in /etc/apk/keys/. OpenWrt 25.12 replaced
# opkg with apk (docs/glinet.md, "OpenWrt 25.12 and apk"); 24.10, 23.05
# and GL.iNet's stock firmware stay on opkg and the .ipk.
#
# Output (out/apk/, or $TB_APK_OUT), per arch in TB_APK_ARCHS:
#   tailbond_<ver>_aarch64_cortex-a53.apk   OpenWrt 25.12+ aarch64
#
# Inputs:
#   out/ipk/tailbond_<ver>_<arch>.ipk   packaging/openwrt/build-ipk.sh
#                                      (or TB_IPK: exactly this one)
# Env: TB_APK_ARCHS (default: aarch64_cortex-a53 — GL.iNet's stock
#      firmware, the _neon-vfpv4 arch, has no apk), TB_APK_KEYS_DIR
#      (whose *.pem ship as /etc/apk/keys/; default the committed
#      keys dir, scripts/release/gen-apk-key.sh writes it; tests point it
#      at a throwaway key), APK_TOOLS (default
#      scripts/release/apk-tools.sh), PODMAN, APK_IMAGE (see apk-tools.sh).
#
# Built from the ipk's data.tar.gz, so the two formats cannot drift: same
# files, modes and fixed mtime. What differs:
#   - control: apk metadata as mkpkg --info keys; the description is the
#     ipk control's first Description line (apk's is one line);
#   - /etc/opkg/keys/ is dropped and the apk feed's public key(s)
#     /etc/apk/keys/<name>.pem (scripts/release/gen-apk-key.sh; none
#     committed, none shipped) take their place; they and
#     /etc/config/tailbond are conffiles via tailbond.conffiles;
#   - scripts: preinst/postinst/prerm/postrm become apk's pre-install,
#     post-install and post-upgrade (apk runs only post-upgrade, not
#     post-install, on an upgrade), pre-deinstall and post-deinstall
#     (apk runs them on removal only, never on an upgrade). OpenWrt's apk
#     sets the same IPKG_* environment the scripts already check.
#
# Versions: the apk version is the ipk file name's (TB_VERSION, a dash
# before the -rN revision); apk orders -rN revisions like opkg does. opkg's
# "~" pre-release escape is opkg's own (apk's ~ marks a commit hash), so a
# pre-release apk keeps the dash and apk's order for it.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/openwrt"
OUT="${TB_APK_OUT:-$ROOT/out/apk}"
ARCHS="${TB_APK_ARCHS:-aarch64_cortex-a53}"
KEYS_DIR="${TB_APK_KEYS_DIR:-$PKG/files/etc/apk/keys}"
APK_TOOLS="${APK_TOOLS:-$ROOT/scripts/release/apk-tools.sh}"

die() { echo "build-apk: $*" >&2; exit 1; }
for c in tar sed sha256sum find; do
    command -v "$c" >/dev/null || die "'$c' not found"
done
case " $ARCHS " in
*" aarch64_cortex-a53 "*) ;;
*) die "TB_APK_ARCHS='$ARCHS': OpenWrt 25.12 packages are built for aarch64_cortex-a53 (plain OpenWrt on the GL.iNet hardware; the stock firmware has no apk)" ;;
esac

stage="$(mktemp -d "${TMPDIR:-/tmp}/tailbond-apk.XXXXXX")"
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/ctl" "$OUT"
install -m 0755 "$PKG/control/postinst" "$stage/ctl/postinst"
install -m 0755 "$PKG/control/prerm" "$stage/ctl/prerm"
install -m 0755 "$PKG/control/postrm" "$stage/ctl/postrm"

for arch in $ARCHS; do
    # out/ipk's newest ipk of this arch, or TB_IPK (exactly that one)
    IPK="${TB_IPK:-}"
    [ -n "$IPK" ] || IPK="$(find "$ROOT/out/ipk" -name "tailbond_*_${arch}.ipk" 2>/dev/null | sort | tail -1)"
    [ -n "$IPK" ] && [ -f "$IPK" ] || die "no $arch ipk: run packaging/openwrt/build-ipk.sh (or set TB_IPK)"
    VERSION="$(basename "$IPK" | sed -n "s/^tailbond_\(.*\)_${arch}\.ipk$/\1/p")"
    [ -n "$VERSION" ] || die "$(basename "$IPK"): not a tailbond_<version>_${arch}.ipk name"

    work="$stage/$arch"
    rm -rf "$work"
    mkdir -p "$work"
    # the same files as the ipk (fixed owner, mtime, order and modes — tar
    # as non-root would apply the umask without -p), minus opkg's keys;
    # the apk feed key takes their place
    tar -xzOf "$IPK" ./data.tar.gz | tar -xzp -C "$work" -f -
    rm -rf "$work/etc/opkg"
    # conffiles: OpenWrt's apk reads /lib/apk/packages/tailbond.conffiles
    # (and the pristine hashes in .conffiles_static; a changed file is kept
    # and the new default written next to it as .apk-new, as opkg did)
    install -d -m 0755 "$work/lib/apk/packages"
    conf="$stage/conffiles.$arch"
    static="$stage/conffiles_static.$arch"
    : >"$conf"
    : >"$static"
    add_conf() { # PATH-in-the-package
        echo "$1" >>"$conf"
        printf '%s %s\n' "$1" "$(sha256sum "$work$1" | cut -d' ' -f1)" >>"$static"
    }
    add_conf /etc/config/tailbond
    # the apk feed's public key(s) (scripts/release/gen-apk-key.sh; tests
    # inject a throwaway one via TB_APK_KEYS_DIR): a router that installed
    # the package trusts the signed feed of later releases. Conffiles: a
    # key added by hand first is the same file, not a clash. None
    # committed, none shipped.
    for k in "$KEYS_DIR"/*.pem; do
        [ -f "$k" ] || continue
        install -d -m 0755 "$work/etc/apk/keys"
        install -m 0644 "$k" "$work/etc/apk/keys/${k##*/}"
        add_conf "/etc/apk/keys/${k##*/}"
    done
    install -m 0644 "$conf" "$work/lib/apk/packages/tailbond.conffiles"
    install -m 0644 "$static" "$work/lib/apk/packages/tailbond.conffiles_static"
    # preinst, generated from the staged tree (issue #49): the same
    # directories build-ipk.sh's preinst creates
    {
        echo '#!/bin/sh'
        echo '# Generated by build-apk.sh from the staged tree (as build-ipk.sh'
        echo "# does): every directory the package's files live in."
        echo 'D="${IPKG_INSTROOT:-}"'
        printf 'mkdir -p'
        (cd "$work" && find . -type d | sed -e '/^\.\/*$/d' -e 's|^\./||' | sort) |
            sed -e 's|^| "$D/|' -e 's|$|" \\|' -e '$s| \\||'
    } >"$stage/ctl/preinst"
    chmod 0755 "$stage/ctl/preinst"

    # the ipk's control: Description's first line, and its Depends (the
    # install test's repacked ipks carry none: the container image has no
    # feed for them; mkpkg records no depends for an empty value)
    tar -xzOf "$IPK" ./control.tar.gz | tar -xzOf - ./control >"$stage/ctl/control"
    DESC="$(sed -n '/^Description:/{s/^Description: *//; p; q}' "$stage/ctl/control")"
    DEPENDS="$(sed -n 's/^Depends: *//p' "$stage/ctl/control" | tr -d ' ' | sed 's/,/ /g')"
    [ -n "$DESC" ] || die "$(basename "$IPK"): no Description in its control"

    apk="$OUT/tailbond_${VERSION}_${arch}.apk"
    rm -f "$apk"
    "$APK_TOOLS" "$stage" mkpkg -F "$arch" \
        -I "name:tailbond" -I "version:$VERSION" -I "arch:$arch" \
        -I "description:$DESC" -I "license:Apache-2.0" \
        -I "maintainer:Tiger Kaovilai <passawit.kaovilai@gmail.com>" \
        -I "url:https://github.com/kaovilai/tailbond" \
        -I "depends:$DEPENDS" \
        --script "pre-install:ctl/preinst" \
        --script "post-install:ctl/postinst" \
        --script "post-upgrade:ctl/postinst" \
        --script "pre-deinstall:ctl/prerm" \
        --script "post-deinstall:ctl/postrm" \
        -o "tailbond_${VERSION}_${arch}.apk" >&2
    mv "$stage/tailbond_${VERSION}_${arch}.apk" "$apk"
    echo "Built: $apk ($(du -h "$apk" | cut -f1))"
done
