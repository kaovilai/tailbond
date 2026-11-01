#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build the generic Linux server bundle (a VPS, any distro):
#
#   packaging/linux/build-bundle.sh
#   -> out/linux/tailbond-linux-x86_64.tar.gz   (and -aarch64 when both binaries exist)
#
# Each tarball unpacks to tailbond-linux-<arch>/ with install.sh,
# uninstall.sh, bin/{mqvpn,tailbondd} (both static), hooks/, firewalld/,
# tailbond.service.in, README.md (docs/vps.md), VERSION, mqvpn's license, and
# share/tailbond.apk when an Android build is around.
#
# Inputs:
#   out/<arch>/mqvpn                  scripts/build-mqvpn-musl.sh <arch>
#   out/agent/linux-<goarch>/tailbondd   agent/build.sh <goarch> (built here when
#                                     missing, or always with TB_BUILD_AGENT=1)
# Env:
#   TB_ARCHS      default "x86_64 aarch64" (missing mqvpn: skipped, except x86_64)
#   VERSION       default git describe
#   TAILBOND_APK  default out/android/tailbond-debug.apk (optional)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/linux"
OUT="$ROOT/out/linux"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
die() { echo "build-bundle: $*" >&2; exit 1; }

built=0
for arch in ${TB_ARCHS:-x86_64 aarch64}; do
    case "$arch" in
        x86_64) goarch=amd64; filearch="x86-64" ;;
        aarch64) goarch=arm64; filearch="ARM aarch64" ;;
        *) die "unknown arch $arch" ;;
    esac
    mq="$ROOT/out/$arch/mqvpn"
    if [ ! -f "$mq" ]; then
        [ "$arch" = x86_64 ] && die "$mq missing: scripts/build-mqvpn-musl.sh x86_64"
        echo "skip $arch: no $mq"
        continue
    fi
    tbd="$ROOT/out/agent/linux-$goarch/tailbondd"
    if { [ ! -f "$tbd" ] || [ "${TB_BUILD_AGENT:-0}" = 1 ]; } && command -v go >/dev/null 2>&1; then
        VERSION="$VERSION" "$ROOT/agent/build.sh" "$goarch" >/dev/null
    fi
    [ -f "$tbd" ] || die "$tbd missing: agent/build.sh $goarch"
    for b in "$mq" "$tbd"; do
        info="$(file -b "$b")"
        case "$info" in
            *"$filearch"*"statically linked"*|*"$filearch"*"static-pie linked"*) ;;
            *) die "$b is not a static $arch binary: $info" ;;
        esac
    done

    name="tailbond-linux-$arch"
    top="$(mktemp -d)"
    d="$top/$name"
    mkdir -p "$d/bin" "$d/hooks" "$d/firewalld"
    install -m 0755 "$mq" "$d/bin/mqvpn"
    install -m 0755 "$tbd" "$d/bin/tailbondd"
    install -m 0755 "$PKG/install.sh" "$PKG/uninstall.sh" "$d/"
    install -m 0755 "$PKG"/hooks/server-up "$PKG"/hooks/server-down "$d/hooks/"
    install -m 0644 "$PKG/tailbond.service.in" "$d/"
    # Same firewalld zone/policies as the Fedora RPM; install.sh sets the
    # service's port.
    install -m 0644 "$ROOT"/packaging/fedora/firewalld/*.xml "$d/firewalld/"
    install -m 0644 "$ROOT/docs/vps.md" "$d/README.md"
    install -m 0644 "$ROOT/third_party/mqvpn/LICENSE" "$d/LICENSE.mqvpn"
    install -m 0644 "$ROOT/third_party/mqvpn/NOTICE" "$d/NOTICE.mqvpn"
    echo "$VERSION" >"$d/VERSION"
    apk="${TAILBOND_APK:-$ROOT/out/android/tailbond-debug.apk}"
    if [ -f "$apk" ]; then mkdir -p "$d/share"; install -m 0644 "$apk" "$d/share/tailbond.apk"; fi
    mkdir -p "$OUT"
    tar -C "$top" --owner=0 --group=0 -czf "$OUT/$name.tar.gz" "$name"
    rm -rf "$top"
    echo "$OUT/$name.tar.gz ($(du -h "$OUT/$name.tar.gz" | cut -f1))"
    built=$((built + 1))
done
[ "$built" -gt 0 ] || die "nothing built"
