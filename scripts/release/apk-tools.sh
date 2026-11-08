#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# apk-tools v3 (mkpkg, mkndx, adbdump) from a pinned Alpine image, run in
# podman: no apk-tools build or install on the host. OpenWrt 25.12 installs
# apk-tools 3 packages and indexes (the "apkv3" format, .apk files and
# packages.adb), so this is the toolchain that builds them
# (packaging/openwrt/build-apk.sh, scripts/release/apk-feed.sh).
#
#   scripts/release/apk-tools.sh DIR APK_ARGS...
#
# Runs `apk APK_ARGS...` with DIR mounted at /w and /w as the working
# directory (relative paths in APK_ARGS resolve there), passing apk's exit
# status through. Alpine's apk-tools follows the upstream v3 format; the
# pinned image's version writes what OpenWrt 25.12's apk (3.0.5) installs,
# which scripts/test-openwrt-apk.sh verifies against the real rootfs.
#
# Env: APK_IMAGE (default docker.io/library/alpine:3.23, whose apk-tools is
# 3.0.x; bump the pin with the format), PODMAN (CI: "sudo podman").
set -euo pipefail

die() { echo "apk-tools: $*" >&2; exit 2; }
[ $# -ge 2 ] || die "usage: $0 DIR APK_ARGS..."
DIR="$1"
shift
[ -d "$DIR" ] || die "$DIR is not a directory"
DIR="$(cd "$DIR" && pwd)"

IMAGE="${APK_IMAGE:-docker.io/library/alpine:3.23}"
read -ra podman <<<"${PODMAN:-podman}"
command -v "${podman[0]}" >/dev/null || die "${podman[0]} not found"

"${podman[@]}" run --rm --log-level error -v "$DIR:/w:z" -w /w "$IMAGE" apk "$@"
