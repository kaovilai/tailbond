# SPDX-License-Identifier: Apache-2.0
# shellcheck shell=bash
#
# Sourced by the build scripts (agent/build.sh, packaging/*/build-*.sh):
#
#   tb_norm_version V     V without a leading "v" (v0.2.0 -> 0.2.0)
#   tb_ldflags VERSION    go build -ldflags for tailbondd: -s -w, main.version,
#                         and from the environment (empty = not set):
#                           TB_UPDATE_REPO      GitHub "owner/name" whose release workflow
#                                               attests the manifests -> update.BuildRepo
#                           TB_UPDATE_WORKFLOW  that workflow's path -> update.BuildWorkflow
#                                               (default .github/workflows/ci.yml)
#                           TB_UPDATE_SOURCE    default update source URL -> update.DefaultSource
#
# No key: authenticity comes from GitHub artifact attestations (Sigstore
# keyless). Without TB_UPDATE_REPO a build only takes updates from its
# server's mirror, or with update.repo / update.verify sha256-only set
# (docs/updates.md).

tb_norm_version() {
    local v="${1#v}"
    printf '%s\n' "${v:-dev}"
}

tb_ldflags() {
    local pkg=github.com/tailbond/tailbond/agent/internal/update f
    f="-s -w -X main.version=$(tb_norm_version "$1")"
    case "${TB_UPDATE_REPO:-}${TB_UPDATE_WORKFLOW:-}${TB_UPDATE_SOURCE:-}" in *[[:space:]\'\"]*)
        echo "tb_ldflags: TB_UPDATE_REPO/TB_UPDATE_WORKFLOW/TB_UPDATE_SOURCE must not contain spaces or quotes" >&2
        return 1 ;;
    esac
    [ -n "${TB_UPDATE_REPO:-}" ] && f="$f -X $pkg.BuildRepo=$TB_UPDATE_REPO"
    [ -n "${TB_UPDATE_WORKFLOW:-}" ] && f="$f -X $pkg.BuildWorkflow=$TB_UPDATE_WORKFLOW"
    [ -n "${TB_UPDATE_SOURCE:-}" ] && f="$f -X $pkg.DefaultSource=$TB_UPDATE_SOURCE"
    printf '%s\n' "$f"
}
