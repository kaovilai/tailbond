# SPDX-License-Identifier: Apache-2.0
# shellcheck shell=bash
#
# Sourced by the build scripts (agent/build.sh, packaging/*/build-*.sh):
#
#   tb_norm_version V     V without a leading "v" (v0.2.0 -> 0.2.0)
#   tb_ldflags VERSION    go build -ldflags for tailbondd: -s -w, main.version,
#                         and from the environment (empty = not set):
#                           TB_UPDATE_PUBKEY  trusted update keys (base64 raw
#                                             ed25519, comma separated) -> update.BuildKeys
#                           TB_UPDATE_SOURCE  default update source URL -> update.DefaultSource
#
# Without TB_UPDATE_PUBKEY the build has no update key and never updates
# itself ("this build has no update key"); see docs/updates.md.

tb_norm_version() {
    local v="${1#v}"
    printf '%s\n' "${v:-dev}"
}

tb_ldflags() {
    local pkg=github.com/tailbond/tailbond/agent/internal/update f
    f="-s -w -X main.version=$(tb_norm_version "$1")"
    case "${TB_UPDATE_PUBKEY:-}${TB_UPDATE_SOURCE:-}" in *[[:space:]\'\"]*)
        echo "tb_ldflags: TB_UPDATE_PUBKEY/TB_UPDATE_SOURCE must not contain spaces or quotes" >&2
        return 1 ;;
    esac
    [ -n "${TB_UPDATE_PUBKEY:-}" ] && f="$f -X $pkg.BuildKeys=$TB_UPDATE_PUBKEY"
    [ -n "${TB_UPDATE_SOURCE:-}" ] && f="$f -X $pkg.DefaultSource=$TB_UPDATE_SOURCE"
    printf '%s\n' "$f"
}
