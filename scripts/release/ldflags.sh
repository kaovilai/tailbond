# SPDX-License-Identifier: Apache-2.0
# shellcheck shell=bash
#
# Sourced by the build scripts (agent/build.sh, packaging/*/build-*.sh):
#
#   tb_norm_version V     V without a leading "v" (v0.2.0 -> 0.2.0)
#   tb_ldflags VERSION    go build -ldflags for tailbondd: -s -w, main.version,
#                         and TB_UPDATE_REPO (GitHub "owner/name" whose
#                         releases the update check reads; default
#                         kaovilai/tailbond) -> update.BuildRepo

tb_norm_version() {
    local v="${1#v}"
    printf '%s\n' "${v:-dev}"
}

tb_ldflags() {
    local f
    f="-s -w -X main.version=$(tb_norm_version "$1")"
    case "${TB_UPDATE_REPO:-}" in *[!A-Za-z0-9_./-]*)
        echo "tb_ldflags: TB_UPDATE_REPO must be owner/name" >&2
        return 1 ;;
    esac
    [ -n "${TB_UPDATE_REPO:-}" ] && f="$f -X github.com/kaovilai/tailbond/agent/internal/update.BuildRepo=$TB_UPDATE_REPO"
    printf '%s\n' "$f"
}
