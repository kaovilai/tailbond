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
# Env: APK_IMAGE (default docker.io/library/alpine:3.23@sha256:..., the
# manifest-list digest pinning that tag: a moved tag would change the code
# that sees a signing key, and the apk format it writes; APK_IMAGE is also
# the override to try a newer apk-tools), PODMAN (CI: "sudo podman").
set -euo pipefail

die() { echo "apk-tools: $*" >&2; exit 2; }
[ $# -ge 2 ] || die "usage: $0 DIR APK_ARGS..."
DIR="$1"
shift
[ -d "$DIR" ] || die "$DIR is not a directory"
DIR="$(cd "$DIR" && pwd)"

IMAGE="${APK_IMAGE:-docker.io/library/alpine:3.23@sha256:f3d55900449227ae9e237ede521e7b665d53a253b42065ba90b9db9275e198ad}"
read -ra podman <<<"${PODMAN:-podman}"
command -v "${podman[0]}" >/dev/null || die "${podman[0]} not found"

# --network none: mkpkg/mkndx/adbdump need no network, and the container
# sometimes handles a signing key
"${podman[@]}" run --rm --log-level error --network none -v "$DIR:/w:z" -w /w "$IMAGE" apk "$@"
