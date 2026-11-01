#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Assemble the OpenWrt / GL.iNet package from prebuilt static binaries.
# No OpenWrt SDK: an .ipk is an ar archive of debian-binary,
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
# Env: TB_VERSION (default: git describe), TB_ARCHS (space separated).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/openwrt"
OUT="$ROOT/out/ipk"
MQVPN_BIN="${MQVPN_BIN:-$ROOT/out/aarch64/mqvpn}"
ARCHS="${TB_ARCHS:-aarch64_cortex-a53 aarch64_cortex-a53_neon-vfpv4}"

die() { echo "build-ipk: $*" >&2; exit 1; }

for c in ar tar gzip file; do
    command -v "$c" >/dev/null || die "'$c' not found"
done

# Version: 0.1.0 plus git distance, valid for opkg's version compare.
if [ -z "${TB_VERSION:-}" ]; then
    n="$(git -C "$ROOT" rev-list --count HEAD 2>/dev/null || echo 0)"
    TB_VERSION="0.1.0-r$n"
fi

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
    (cd "$ROOT/agent" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
        go build -trimpath -ldflags "-s -w" -o "$tailbondd" ./cmd/tailbondd)
else
    die "no agent/ in this tree: set TAILBONDD_BIN to a linux/arm64 tailbondd binary"
fi
is_static_arm64 "$tailbondd" || die "$tailbondd is not a static aarch64 binary: $(file -b "$tailbondd")"

# ── data ──
data="$stage/data"
install -d -m 0755 "$data"/usr/bin "$data"/usr/lib/tailbond/hooks \
    "$data"/etc/init.d "$data"/etc/config "$data"/etc/uci-defaults \
    "$data"/lib/upgrade/keep.d \
    "$data"/usr/share/oui/menu.d "$data"/www/views \
    "$data"/usr/share/luci/menu.d "$data"/usr/share/rpcd/acl.d \
    "$data"/www/luci-static/resources/view
install -m 0755 "$MQVPN_BIN" "$data/usr/bin/mqvpn"
install -m 0755 "$tailbondd" "$data/usr/bin/tailbondd"
install -m 0755 "$PKG/files/etc/init.d/tailbond" "$data/etc/init.d/tailbond"
install -m 0600 "$PKG/files/etc/config/tailbond" "$data/etc/config/tailbond"
install -m 0755 "$PKG/files/etc/uci-defaults/90-tailbond" "$data/etc/uci-defaults/90-tailbond"
install -m 0755 "$PKG/files/usr/lib/tailbond/render-config" "$data/usr/lib/tailbond/render-config"
install -m 0644 "$PKG/hooks/lib.sh" "$data/usr/lib/tailbond/hook-lib.sh"
for h in client-up client-down server-up server-down; do
    install -m 0755 "$PKG/hooks/$h" "$data/usr/lib/tailbond/hooks/$h"
done
# sysupgrade keeps the agent config and state (server users, pairing)
echo /etc/tailbond/ >"$data/lib/upgrade/keep.d/tailbond"
chmod 0644 "$data/lib/upgrade/keep.d/tailbond"
# GL.iNet 4.x admin panel entry (Applications -> tailbond)
install -m 0644 "$PKG/gl-ui/usr/share/oui/menu.d/tailbond.json" "$data/usr/share/oui/menu.d/"
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
    tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="$mtime" \
        -C "$2" -cf - . | gzip -9n >"$1"
}

size="$(du -sk --apparent-size "$data" | cut -f1)"
mkdir -p "$OUT"
for arch in $ARCHS; do
    ctl="$stage/control-$arch"
    install -d -m 0755 "$ctl"
    sed -e "s/@VERSION@/$TB_VERSION/" -e "s/@ARCH@/$arch/" -e "s/@SIZE@/$((size * 1024))/" \
        "$PKG/control/control.in" >"$ctl/control"
    install -m 0644 "$PKG/control/conffiles" "$ctl/conffiles"
    install -m 0755 "$PKG/control/postinst" "$ctl/postinst"
    install -m 0755 "$PKG/control/prerm" "$ctl/prerm"
    chmod 0644 "$ctl/control"

    w="$stage/ipk-$arch"
    mkdir -p "$w"
    echo "2.0" >"$w/debian-binary"
    tgz "$w/control.tar.gz" "$ctl"
    tgz "$w/data.tar.gz" "$data"
    ipk="$OUT/tailbond_${TB_VERSION}_${arch}.ipk"
    rm -f "$ipk"
    (cd "$w" && ar -rcD "$ipk" debian-binary control.tar.gz data.tar.gz)
    echo "Built: $ipk ($(du -h "$ipk" | cut -f1))"
done
