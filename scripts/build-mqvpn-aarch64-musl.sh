#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Static aarch64 (musl) mqvpn for OpenWrt / GL.iNet routers -> out/aarch64/mqvpn.
# Kept for packaging/openwrt/build-ipk.sh and CI; the work is done by
# scripts/build-mqvpn-musl.sh (same env: TB_WORK_DIR, TB_JOBS,
# TB_ALPINE_IMAGE, TB_CLEAN).
set -euo pipefail
exec "$(dirname "$0")/build-mqvpn-musl.sh" aarch64 "$@"
