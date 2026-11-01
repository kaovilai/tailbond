#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Publish a tagged release (CI, .github/workflows/ci.yml job "release"):
#
#   scripts/release/publish-channels.sh TAG ASSETS_DIR [--dry-run]
#
# 1. Update manifests for the channels this tag feeds: a plain tag
#    (v1.2.3) feeds stable and beta, a pre-release tag (v1.2.3-beta.1) only
#    beta and is published as a GitHub pre-release. A channel never goes
#    back: if its current manifest has a newer version, it's left alone.
#    The tag message containing "[security]" marks the release security.
# 2. Sign them with TB_UPDATE_KEY (sign-manifest.sh). Without the secret
#    the manifests are attached unsigned to the version release only and the
#    channels are not touched (agents would reject them; a warning says so).
# 3. gh release create TAG with every file of ASSETS_DIR plus the manifests,
#    then upload <channel>.json(.sig) with --clobber to the rolling release
#    "channels" (created on first use), which update.source points at:
#    https://github.com/OWNER/REPO/releases/download/channels
#
# Env: GITHUB_REPOSITORY, GH_TOKEN (contents: write), TB_UPDATE_KEY
# (optional). The agent protocol comes from agent/internal/update
# (const Protocol; requires_server_protocol from const MinServerProtocol
# when defined, else the same number: servers update first).
set -euo pipefail

# semver_gt A B: A > B (MAJOR.MINOR.PATCH[-pre], pre-release below release)
semver_gt() {
    local a="$1" b="$2" ac bc ap bp i x y
    ac="${a%%-*}" bc="${b%%-*}"
    ap=""; [[ "$a" == *-* ]] && ap="${a#*-}"
    bp=""; [[ "$b" == *-* ]] && bp="${b#*-}"
    IFS=. read -r -a x <<<"$ac"
    IFS=. read -r -a y <<<"$bc"
    for i in 0 1 2; do
        [ "$((10#${x[i]:-0}))" -gt "$((10#${y[i]:-0}))" ] && return 0
        [ "$((10#${x[i]:-0}))" -lt "$((10#${y[i]:-0}))" ] && return 1
    done
    [ "$ap" = "$bp" ] && return 1
    [ -z "$ap" ] && return 0
    [ -z "$bp" ] && return 1
    IFS=. read -r -a x <<<"$ap"
    IFS=. read -r -a y <<<"$bp"
    for ((i = 0; i < ${#x[@]} || i < ${#y[@]}; i++)); do
        [ "$i" -ge "${#x[@]}" ] && return 1
        [ "$i" -ge "${#y[@]}" ] && return 0
        [ "${x[i]}" = "${y[i]}" ] && continue
        if [[ "${x[i]}" =~ ^[0-9]+$ && "${y[i]}" =~ ^[0-9]+$ ]]; then
            [ "$((10#${x[i]}))" -gt "$((10#${y[i]}))" ]; return
        fi
        [[ "${x[i]}" =~ ^[0-9]+$ ]] && return 1
        [[ "${y[i]}" =~ ^[0-9]+$ ]] && return 0
        [[ "${x[i]}" > "${y[i]}" ]]; return
    done
    return 1
}
# (tests) publish-channels.sh --semver-gt A B
if [ "${1:-}" = --semver-gt ]; then semver_gt "$2" "$3"; exit; fi

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
TAG="${1:?usage: publish-channels.sh TAG ASSETS_DIR [--dry-run]}"
ASSETS="${2:?usage: publish-channels.sh TAG ASSETS_DIR [--dry-run]}"
DRY=0
[ "${3:-}" = --dry-run ] && DRY=1
REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}"
CH_TAG=channels
warn() { echo "::warning::$*"; }
run() { if [ "$DRY" = 1 ]; then echo "+ $*"; else "$@"; fi; }

VERSION="${TAG#v}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || { echo "tag $TAG is not vMAJOR.MINOR.PATCH[-pre]" >&2; exit 1; }
if [[ "$VERSION" == *-* ]]; then CHANNELS="beta"; PRE=1; else CHANNELS="stable beta"; PRE=0; fi

src="$ROOT/agent/internal/update"
PROTO="$(sed -n 's/^const Protocol = \([0-9][0-9]*\).*/\1/p' "$src"/*.go 2>/dev/null | head -1)"
REQ="$(sed -n 's/^const MinServerProtocol = \([0-9][0-9]*\).*/\1/p' "$src"/*.go 2>/dev/null | head -1)"
PROTO="${PROTO:-1}"
REQ="${REQ:-$PROTO}"
SECURITY=()
if git -C "$ROOT" tag -l --format='%(contents)' "$TAG" 2>/dev/null | grep -qiF '[security]'; then SECURITY=(--security); fi


MAN="$(mktemp -d)"
trap 'rm -rf "$MAN"' EXIT
signed=0
if [ -n "${TB_UPDATE_KEY:-}" ]; then signed=1; else
    warn "TB_UPDATE_KEY secret is not set: manifests are attached unsigned to $TAG and the '$CH_TAG' release is not updated (docs/updates.md, Signing keys)"
fi
publish=()
for ch in $CHANNELS; do
    "$HERE/make-manifest.sh" --version "$VERSION" --channel "$ch" --dir "$ASSETS" --out "$MAN" \
        --base-url "https://github.com/$REPO/releases/download/$TAG" \
        --notes-url "https://github.com/$REPO/releases/tag/$TAG" \
        --protocol "$PROTO" --requires-server-protocol "$REQ" "${SECURITY[@]}" >/dev/null
    if [ "$signed" = 1 ]; then "$HERE/sign-manifest.sh" "$MAN/$ch.json" >/dev/null; fi
    # Never move a channel back (a hotfix tag for an older line).
    cur=""
    if [ "$DRY" = 0 ]; then
        cur="$(curl -fsSL "https://github.com/$REPO/releases/download/$CH_TAG/$ch.json" 2>/dev/null | jq -r '.version // empty' 2>/dev/null || true)"
    fi
    if [ -n "$cur" ] && ! semver_gt "$VERSION" "$cur"; then
        echo "$ch: stays at $cur (not older than $VERSION)"
    else
        publish+=("$ch")
        echo "$ch: ${cur:-none} -> $VERSION"
    fi
done
files=("$MAN"/*.json)
[ "$signed" = 1 ] && files+=("$MAN"/*.json.sig)

prerelease=()
[ "$PRE" = 1 ] && prerelease=(--prerelease)
mapfile -t assets < <(find "$ASSETS" -maxdepth 1 -type f | sort)
run gh release create "$TAG" --repo "$REPO" --generate-notes --verify-tag "${prerelease[@]}" "${assets[@]}" "${files[@]}"

if [ "$signed" = 1 ] && [ "${#publish[@]}" -gt 0 ]; then
    if [ "$DRY" = 1 ] || ! gh release view "$CH_TAG" --repo "$REPO" >/dev/null 2>&1; then
        run gh release create "$CH_TAG" --repo "$REPO" --prerelease --latest=false --title "Update channels" \
            --notes "Signed update manifests (stable.json, beta.json and their .sig) that tailbond installs check. Rewritten by CI on every release tag; see docs/updates.md." \
            || gh release view "$CH_TAG" --repo "$REPO" >/dev/null
    fi
    up=()
    for ch in "${publish[@]}"; do up+=("$MAN/$ch.json" "$MAN/$ch.json.sig"); done
    run gh release upload "$CH_TAG" --repo "$REPO" --clobber "${up[@]}"
fi
