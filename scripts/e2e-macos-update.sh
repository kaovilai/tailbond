#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# End-to-end update test of the macOS install (the dashboard's "Update now"
# on a Mac): builds tailbondd as 0.0.1, 0.0.2, 0.0.4 and 0.0.5, a broken
# 0.0.3 (exits at once), packs each into the release bundle (the template's
# mqvpn and tailbond.app, this tree's install.sh and hooks), then runs
# packaging/macos/test-update.sh as root on this Mac: fake GitHub release ->
# the agent sees it -> Update now (download, SHA-256, the new bundle's
# install.sh --update detached from the LaunchDaemon) -> rollback of a
# release that doesn't start -> install.sh --update alone -> SHA256SUMS
# refusal -> uninstall. The Linux/VPS twin is scripts/e2e-update-vps.sh.
#
#   scripts/e2e-macos-update.sh [bundle.tar.gz]
#   default: out/macos/tailbond-macos-arm64.tar.gz (packaging/macos/build-bundle.sh)
#   CI: job macos. Needs macOS (arm64), go, python3, passwordless sudo.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE="${1:-$ROOT/out/macos/tailbond-macos-arm64.tar.gz}"
OUT="$ROOT/out/e2e-macos-update"
[ "$(uname -s)" = Darwin ] || { echo "e2e-macos-update: run it on a Mac (Linux: scripts/e2e-update-vps.sh)" >&2; exit 2; }
[ -f "$TEMPLATE" ] || { echo "no bundle at $TEMPLATE: run packaging/macos/build-bundle.sh" >&2; exit 2; }

rm -rf "$OUT" && mkdir -p "$OUT/src"
tar -C "$OUT/src" -xzf "$TEMPLATE"
src="$OUT/src/tailbond-macos-arm64"
cp "$ROOT/packaging/macos/install.sh" "$ROOT/packaging/macos/uninstall.sh" \
   "$ROOT/packaging/macos/com.tailbond.tailbondd.plist" "$src/"
cp "$ROOT/packaging/macos/hooks/"* "$src/hooks/"

pack() { # VERSION TAILBONDD
    local d="$OUT/p/tailbond-macos-arm64"
    rm -rf "$OUT/p" && mkdir -p "$OUT/p" && cp -a "$src" "$d"
    install -m 0755 "$2" "$d/bin/tailbondd"
    echo "$1" >"$d/VERSION"
    tar -C "$OUT/p" --uid 0 --gid 0 -czf "$OUT/tailbond-macos-arm64-$1.tar.gz" "tailbond-macos-arm64"
}
for v in 0.0.1 0.0.2 0.0.4 0.0.5; do
    (cd "$ROOT/agent" && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath \
        -ldflags "-s -w -X main.version=$v" -o "$OUT/tailbondd-$v" ./cmd/tailbondd)
    pack "$v" "$OUT/tailbondd-$v"
done
# 0.0.3: passes install.sh's "does it run" check (-version works), then never
# starts. It marks every crash in /tmp/tb-macos-update-broken.log, which
# packaging/macos/test-update.sh reads: launchd may truncate the daemon log
# on each restart, the marker file survives.
printf '#!/bin/sh\n[ "$1" = -version ] && { echo 0.0.3; exit 0; }\necho "broken release 0.0.3 ($$)" >>/tmp/tb-macos-update-broken.log\nexit 1\n' >"$OUT/tailbondd-0.0.3"
pack 0.0.3 "$OUT/tailbondd-0.0.3"
rm -rf "$OUT/p" "$OUT/src"
ls -1 "$OUT"/*.tar.gz

sudo /bin/bash "$ROOT/packaging/macos/test-update.sh" "$OUT"
