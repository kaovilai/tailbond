# SPDX-License-Identifier: Apache-2.0
# shellcheck shell=bash
#
# Sourced by scripts/build-mqvpn-*.sh: patched mqvpn sources in a work tree.
# Callers set ROOT (repo root) and define die().

# mqvpn_src_prepare WORK: copy the pinned third_party/mqvpn (with its xquic,
# BoringSSL and lwIP submodules) to WORK/src and apply patches/mqvpn/*.patch.
# Build dirs inside WORK/src survive, so rebuilds are incremental. The
# submodule itself is never modified.
mqvpn_src_prepare() {
    local work="$1" src="$ROOT/third_party/mqvpn"
    [ -f "$src/build.sh" ] || die "$src missing; run: git submodule update --init --recursive"
    [ -f "$src/third_party/xquic/third_party/boringssl/CMakeLists.txt" ] \
        || die "xquic/BoringSSL submodules missing; run: git submodule update --init --recursive"
    local c
    for c in git rsync patch; do
        command -v "$c" >/dev/null || die "'$c' not found (dnf install git rsync patch)"
    done
    # GNU patch: macOS's own patch(1) lacks --no-backup-if-mismatch
    # (brew install gpatch)
    local patch=patch
    command -v gpatch >/dev/null && patch=gpatch

    # Patches target the pinned upstream commit; refuse a moved or dirty submodule.
    local pinned head
    pinned="$(git -C "$ROOT" ls-tree HEAD third_party/mqvpn | awk '{print $3}')"
    head="$(git -C "$src" rev-parse HEAD)"
    [ "$head" = "$pinned" ] \
        || die "third_party/mqvpn is at $head, not the pinned $pinned (git submodule update)"
    [ -z "$(git -C "$src" status --porcelain --untracked-files=no)" ] \
        || die "third_party/mqvpn has local changes"

    mkdir -p "$work"
    # rsync re-copies every file a previous run patched (its mtime differs),
    # and patching then gives it a fresh mtime, so make rebuilds it.
    echo "=== Syncing $src -> $work/src ==="
    rsync -a --delete \
        --exclude '.git' \
        --exclude '/build/' \
        --exclude '/build-*/' \
        --exclude '/third_party/xquic/build/' \
        --exclude '/third_party/xquic/third_party/boringssl/build/' \
        "$src/" "$work/src/"

    local p
    for p in "$ROOT"/patches/mqvpn/*.patch; do
        [ -e "$p" ] || continue
        echo "=== Applying $(basename "$p") ==="
        # patch(1), not git apply: the work tree sits inside the superproject's
        # repo, where git apply would resolve paths against the repo root.
        "$patch" -d "$work/src" -p1 --forward --no-backup-if-mismatch -s <"$p" \
            || die "$(basename "$p") does not apply to the pinned mqvpn"
    done
}
