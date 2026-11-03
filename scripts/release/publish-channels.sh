#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Publish a tagged release (CI, .github/workflows/ci.yml job "release"),
# in two steps around actions/attest (GitHub artifact attestation, Sigstore
# keyless: no signing key anywhere):
#
#   scripts/release/publish-channels.sh prepare TAG ASSETS_DIR MANIFEST_DIR [--dry-run]
#   (CI: actions/attest with subject-path ASSETS_DIR/* and MANIFEST_DIR/*.json)
#   scripts/release/publish-channels.sh publish TAG ASSETS_DIR MANIFEST_DIR BUNDLE [--dry-run]
#
# prepare: update manifests for the channels this tag feeds into
#    MANIFEST_DIR: a plain tag (v1.2.3) feeds stable and beta, a pre-release
#    tag (v1.2.3-beta.1) only beta and is published as a GitHub pre-release.
#    A channel never goes back: if its current manifest has a newer version,
#    it's left alone (MANIFEST_DIR/publish lists the channels to update).
#    The tag message containing "[security]" marks the release security.
# publish: BUNDLE is the attestation (actions/attest output bundle-path),
#    copied next to each manifest as <channel>.json.sigstore.json. gh release
#    create TAG with every file of ASSETS_DIR plus the manifests and their
#    bundles, then upload <channel>.json(.sigstore.json) with --clobber to the
#    rolling release "channels" (created on first use), which update.source
#    points at: https://github.com/OWNER/REPO/releases/download/channels
#    Agents verify the attestation (bundle file, else the GitHub API) against
#    this repository and workflow (docs/updates.md).
#
# Env: GITHUB_REPOSITORY, GH_TOKEN (contents: write). The agent protocol
# comes from agent/internal/update (const Protocol; requires_server_protocol
# from const MinServerProtocol when defined, else the same number: servers
# update first).
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
usage="usage: publish-channels.sh prepare TAG ASSETS_DIR MANIFEST_DIR | publish TAG ASSETS_DIR MANIFEST_DIR BUNDLE [--dry-run]"
STEP="${1:?$usage}"
TAG="${2:?$usage}"
ASSETS="${3:?$usage}"
MAN="${4:?$usage}"
BUNDLE=""
case "$STEP" in
prepare) shift 4 ;;
publish) BUNDLE="${5:?$usage}"; shift 5 ;;
*) echo "$usage" >&2; exit 2 ;;
esac
DRY=0
[ "${1:-}" = --dry-run ] && DRY=1
REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is not set}"
CH_TAG=channels
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

if [ "$STEP" = prepare ]; then
mkdir -p "$MAN"
rm -f "$MAN"/stable.json* "$MAN"/beta.json* "$MAN/publish"
: >"$MAN/publish"
for ch in $CHANNELS; do
    "$HERE/make-manifest.sh" --version "$VERSION" --channel "$ch" --dir "$ASSETS" --out "$MAN" \
        --base-url "https://github.com/$REPO/releases/download/$TAG" \
        --notes-url "https://github.com/$REPO/releases/tag/$TAG" \
        --protocol "$PROTO" --requires-server-protocol "$REQ" "${SECURITY[@]}" >/dev/null
    # Never move a channel back (a hotfix tag for an older line).
    cur=""
    if [ "$DRY" = 0 ]; then
        cur="$(curl -fsSL "https://github.com/$REPO/releases/download/$CH_TAG/$ch.json" 2>/dev/null | jq -r '.version // empty' 2>/dev/null || true)"
    fi
    if [ -n "$cur" ] && ! semver_gt "$VERSION" "$cur"; then
        echo "$ch: stays at $cur (not older than $VERSION)"
    else
        echo "$ch" >>"$MAN/publish"
        echo "$ch: ${cur:-none} -> $VERSION"
    fi
done
exit 0
fi

# publish
[ -s "$BUNDLE" ] || { echo "no attestation bundle at $BUNDLE: run actions/attest between prepare and publish" >&2; exit 1; }
jq -e '.mediaType | startswith("application/vnd.dev.sigstore.bundle")' "$BUNDLE" >/dev/null ||
    { echo "$BUNDLE isn't a Sigstore bundle" >&2; exit 1; }
files=()
for ch in stable beta; do
    [ -f "$MAN/$ch.json" ] || continue
    cp "$BUNDLE" "$MAN/$ch.json.sigstore.json"
    files+=("$MAN/$ch.json" "$MAN/$ch.json.sigstore.json")
done
mapfile -t publish <"$MAN/publish"

prerelease=()
[ "$PRE" = 1 ] && prerelease=(--prerelease)
mapfile -t assets < <(find "$ASSETS" -maxdepth 1 -type f | sort)
run gh release create "$TAG" --repo "$REPO" --generate-notes --verify-tag "${prerelease[@]}" "${assets[@]}" "${files[@]}"

if [ "${#publish[@]}" -gt 0 ]; then
    if [ "$DRY" = 1 ] || ! gh release view "$CH_TAG" --repo "$REPO" >/dev/null 2>&1; then
        run gh release create "$CH_TAG" --repo "$REPO" --prerelease --latest=false --title "Update channels" \
            --notes "Update manifests (stable.json, beta.json) and their GitHub artifact attestations (.sigstore.json) that tailbond installs check. Rewritten by CI on every release tag; see docs/updates.md." \
            || gh release view "$CH_TAG" --repo "$REPO" >/dev/null
    fi
    up=()
    for ch in "${publish[@]}"; do up+=("$MAN/$ch.json" "$MAN/$ch.json.sigstore.json"); done
    run gh release upload "$CH_TAG" --repo "$REPO" --clobber "${up[@]}"
fi
