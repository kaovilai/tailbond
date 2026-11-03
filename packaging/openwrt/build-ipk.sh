#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Assemble the OpenWrt / GL.iNet package from prebuilt static binaries.
# No OpenWrt SDK: an OpenWrt .ipk is a gzipped tar of debian-binary,
# control.tar.gz and data.tar.gz, and both binaries are static.
#
# Output (out/ipk/), same contents, two architecture names:
#   tailbond_<ver>_aarch64_cortex-a53.ipk             OpenWrt 23.05/24.10 aarch64
#   tailbond_<ver>_aarch64_cortex-a53_neon-vfpv4.ipk  GL.iNet 4.x stock firmware
#     on IPQ53xx (Flint 3 GL-BE9300, Slate 7 GL-BE3600), whose opkg only
#     accepts that name (see docs/glinet.md)
#
# Inputs:
#   out/aarch64/mqvpn         scripts/build-mqvpn-aarch64-musl.sh (or MQVPN_BIN)
#   tailbondd                 built here from agent/ when it exists
#                             (CGO_ENABLED=0 GOOS=linux GOARCH=arm64), otherwise
#                             TAILBONDD_BIN=/path/to/linux-arm64/tailbondd
#   out/android/tailbond-debug.apk  optional (or TAILBOND_APK): the Android
#                             app a home server offers on its phone page
# Env: TB_VERSION (default: <tag>-r1 on a v* tag, else 0.1.0-r<commit count>;
#      a leading v is dropped), TB_ARCHS (space separated),
#      TB_UPDATE_REPO / TB_UPDATE_SOURCE (release repository + update source
#      compiled into tailbondd, scripts/release/ldflags.sh).
# tailbondd is built with -tags noattest: no GitHub attestation verifier
# (sigstore-go adds ~8 MB); the router takes updates from its server's
# mirror, which verified them (docs/updates.md).
#
# Versions: TB_VERSION is <semver>-r<N>. The file name keeps it as is
# (tailbond_1.2.3-beta.1-r1_<arch>.ipk); the control Version turns the
# pre-release dash into "~" (1.2.3~beta.1-r1) so opkg orders a beta below
# its release, as dpkg does.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=../../scripts/release/ldflags.sh
. "$ROOT/scripts/release/ldflags.sh"
PKG="$ROOT/packaging/openwrt"
OUT="$ROOT/out/ipk"
MQVPN_BIN="${MQVPN_BIN:-$ROOT/out/aarch64/mqvpn}"
ARCHS="${TB_ARCHS:-aarch64_cortex-a53 aarch64_cortex-a53_neon-vfpv4}"

die() { echo "build-ipk: $*" >&2; exit 1; }

for c in tar gzip file; do
    command -v "$c" >/dev/null || die "'$c' not found"
done

# Version: on a release tag <tag>-r1; else 0.1.0 plus git distance, valid
# for opkg's version compare. tailbondd's own version (main.version) is
# TB_VERSION, except for those untagged builds: git describe, which the
# agent never treats as a release (no self-updates from a dev build).
AGENT_VERSION=""
if [ -z "${TB_VERSION:-}" ]; then
    tag="$(git -C "$ROOT" describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null || true)"
    if [ -n "$tag" ]; then
        TB_VERSION="${tag#v}-r1"
    else
        n="$(git -C "$ROOT" rev-list --count HEAD 2>/dev/null || echo 0)"
        TB_VERSION="0.1.0-r$n"
        AGENT_VERSION="$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)"
    fi
fi
TB_VERSION="$(tb_norm_version "$TB_VERSION")"
case "$TB_VERSION" in
    *-r[0-9]*) ;;
    *) die "TB_VERSION=$TB_VERSION: needs an -rN revision (e.g. ${TB_VERSION}-r1)" ;;
esac
# control Version: the first dash before the -rN revision is a semver
# pre-release separator -> "~"
CTL_VERSION="$TB_VERSION"
base="${TB_VERSION%-r*}"
case "$base" in *-*) CTL_VERSION="${base%%-*}~${base#*-}-${TB_VERSION##*-}" ;; esac

is_static_arm64() {
    case "$(file -b "$1")" in
        *"ARM aarch64"*"statically linked"*) return 0 ;;
        *"ARM aarch64"*"static-pie linked"*) return 0 ;;
    esac
    return 1
}

[ -f "$MQVPN_BIN" ] || die "$MQVPN_BIN missing; run scripts/build-mqvpn-aarch64-musl.sh"
is_static_arm64 "$MQVPN_BIN" || die "$MQVPN_BIN is not a static aarch64 binary: $(file -b "$MQVPN_BIN")"

stage="$(mktemp -d "${TMPDIR:-/tmp}/tailbond-ipk.XXXXXX")"
trap 'rm -rf "$stage"' EXIT

if [ -n "${TAILBONDD_BIN:-}" ]; then
    [ -f "$TAILBONDD_BIN" ] || die "TAILBONDD_BIN=$TAILBONDD_BIN does not exist"
    tailbondd="$TAILBONDD_BIN"
elif [ -d "$ROOT/agent/cmd/tailbondd" ]; then
    command -v go >/dev/null || die "go not found (needed to build agent/)"
    echo "=== Building tailbondd (linux/arm64, static)"
    tailbondd="$stage/tailbondd"
    ldflags="$(tb_ldflags "${AGENT_VERSION:-$TB_VERSION}")"
    (cd "$ROOT/agent" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
        go build -trimpath -tags noattest -ldflags "$ldflags" -o "$tailbondd" ./cmd/tailbondd)
else
    die "no agent/ in this tree: set TAILBONDD_BIN to a linux/arm64 tailbondd binary"
fi
is_static_arm64 "$tailbondd" || die "$tailbondd is not a static aarch64 binary: $(file -b "$tailbondd")"

# ── data ──
data="$stage/data"
install -d -m 0755 "$data"/usr/bin "$data"/usr/lib/tailbond/hooks \
    "$data"/etc/init.d "$data"/etc/config "$data"/etc/uci-defaults \
    "$data"/lib/upgrade/keep.d \
    "$data"/usr/share/oui/menu.d "$data"/www/views "$data"/etc/hotplug.d/iface \
    "$data"/usr/lib/oui-httpd/rpc "$data"/usr/share/gl-validator.d "$data"/etc/nginx/gl-conf.d \
    "$data"/usr/share/luci/menu.d "$data"/usr/share/rpcd/acl.d \
    "$data"/www/luci-static/resources/view
install -m 0755 "$MQVPN_BIN" "$data/usr/bin/mqvpn"
install -m 0755 "$tailbondd" "$data/usr/bin/tailbondd"
install -m 0755 "$PKG/files/etc/init.d/tailbond" "$data/etc/init.d/tailbond"
install -m 0600 "$PKG/files/etc/config/tailbond" "$data/etc/config/tailbond"
install -m 0755 "$PKG/files/etc/uci-defaults/90-tailbond" "$data/etc/uci-defaults/90-tailbond"
install -m 0755 "$PKG/files/usr/lib/tailbond/render-config" "$data/usr/lib/tailbond/render-config"
install -m 0755 "$PKG/files/etc/hotplug.d/iface/90-tailbond" "$data/etc/hotplug.d/iface/90-tailbond"
# Updates (docs/updates.md): the agent's installer with rollback, the
# package cache helper, the reinstall-after-firmware-upgrade script (copied
# to /etc/tailbond/pkg/ by postinst) and the sysupgrade hook.
for f in update-ipk pkg-cache reinstall; do
    install -m 0755 "$PKG/files/usr/lib/tailbond/$f" "$data/usr/lib/tailbond/$f"
done
install -m 0644 "$PKG/files/lib/upgrade/tailbond.sh" "$data/lib/upgrade/tailbond.sh"
install -m 0644 "$PKG/hooks/lib.sh" "$data/usr/lib/tailbond/hook-lib.sh"
for h in client-up client-down server-up server-down portal-open portal-close bypass-apply; do
    install -m 0755 "$PKG/hooks/$h" "$data/usr/lib/tailbond/hooks/$h"
done
# Pristine copies of files that change after install (edited config, the
# uci-defaults script removes itself): a home server on a GL router serves
# the package it runs as GET /tailbond-glinet.ipk, rebuilt from opkg's file
# list, and uses these instead of the live files.
install -d -m 0755 "$data/usr/share/tailbond/pristine/etc/config" "$data/usr/share/tailbond/pristine/etc/uci-defaults"
install -m 0600 "$PKG/files/etc/config/tailbond" "$data/usr/share/tailbond/pristine/etc/config/tailbond"
install -m 0755 "$PKG/files/etc/uci-defaults/90-tailbond" "$data/usr/share/tailbond/pristine/etc/uci-defaults/90-tailbond"
# The Android app, so a home server on a GL router can offer it on its
# phone page (GET /tailbond.apk): TAILBOND_APK, else the local debug build.
apk="${TAILBOND_APK:-$ROOT/out/android/tailbond-debug.apk}"
if [ -f "$apk" ]; then
    install -m 0644 "$apk" "$data/usr/share/tailbond/tailbond.apk"
else
    echo "note: no Android APK at $apk; the package won't offer the app (set TAILBOND_APK)" >&2
fi
# sysupgrade keeps the agent config and state (server users, pairing)
echo /etc/tailbond/ >"$data/lib/upgrade/keep.d/tailbond"
chmod 0644 "$data/lib/upgrade/keep.d/tailbond"
# GL.iNet 4.x admin panel (VPN -> tailbond): menu entry, the view,
# its /rpc backend object "tailbond" + argument validator, and a same-origin
# nginx fallback (docs/glinet.md, "Admin panel")
install -m 0644 "$PKG/gl-ui/usr/share/oui/menu.d/tailbond.json" "$data/usr/share/oui/menu.d/"
install -m 0644 "$PKG/gl-ui/usr/lib/oui-httpd/rpc/tailbond" "$data/usr/lib/oui-httpd/rpc/tailbond"
install -m 0644 "$PKG/gl-ui/usr/share/gl-validator.d/tailbond.lua" "$data/usr/share/gl-validator.d/"
install -m 0644 "$PKG/gl-ui/etc/nginx/gl-conf.d/tailbond.conf" "$data/etc/nginx/gl-conf.d/"
gzip -9n -c "$PKG/gl-ui/www/views/gl-sdk4-ui-tailbond.common.js" \
    >"$data/www/views/gl-sdk4-ui-tailbond.common.js.gz"
chmod 0644 "$data/www/views/gl-sdk4-ui-tailbond.common.js.gz"
# LuCI entry (Services -> tailbond)
install -m 0644 "$PKG/luci/usr/share/luci/menu.d/luci-app-tailbond.json" "$data/usr/share/luci/menu.d/"
install -m 0644 "$PKG/luci/usr/share/rpcd/acl.d/luci-app-tailbond.json" "$data/usr/share/rpcd/acl.d/"
install -m 0644 "$PKG/luci/www/luci-static/resources/view/tailbond.js" "$data/www/luci-static/resources/view/"

# Reproducible tarballs: fixed owner, mtime and order.
mtime="@$(git -C "$ROOT" log -1 --format=%ct 2>/dev/null || echo 0)"
tgz() { # tgz OUT DIR
    tar --format=ustar --sort=name --owner=0 --group=0 --numeric-owner --mtime="$mtime" \
        -C "$2" -cf - . | gzip -9n >"$1"
}

size="$(du -sk --apparent-size "$data" | cut -f1)"
mkdir -p "$OUT"
for arch in $ARCHS; do
    ctl="$stage/control-$arch"
    install -d -m 0755 "$ctl"
    sed -e "s/@VERSION@/$CTL_VERSION/" -e "s/@ARCH@/$arch/" -e "s/@SIZE@/$((size * 1024))/" \
        "$PKG/control/control.in" >"$ctl/control"
    install -m 0644 "$PKG/control/conffiles" "$ctl/conffiles"
    install -m 0755 "$PKG/control/postinst" "$ctl/postinst"
    install -m 0755 "$PKG/control/prerm" "$ctl/prerm"
    install -m 0755 "$PKG/control/postrm" "$ctl/postrm"
    chmod 0644 "$ctl/control"

    w="$stage/ipk-$arch"
    mkdir -p "$w"
    echo "2.0" >"$w/debian-binary"
    tgz "$w/control.tar.gz" "$ctl"
    tgz "$w/data.tar.gz" "$data"
    ipk="$OUT/tailbond_${TB_VERSION}_${arch}.ipk"
    rm -f "$ipk"
    # OpenWrt's opkg wants the outer archive as gzipped tar (as ipkg-build
    # makes it), not the Debian-style ar archive; ar fails as "Malformed"
    # (GL's opkg reportedly crashes on it). ustar, not pax/gnu, for the same
    # reason.
    tar --format=ustar --sort=name --owner=0 --group=0 --numeric-owner --mtime="$mtime" \
        -C "$w" -cf - ./debian-binary ./data.tar.gz ./control.tar.gz | gzip -9n >"$ipk"
    echo "Built: $ipk ($(du -h "$ipk" | cut -f1))"
done
