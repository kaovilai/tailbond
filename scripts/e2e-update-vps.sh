#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# End-to-end update test of the static (VPS) install: builds tailbondd as
# 0.0.1, 0.0.2, 0.0.4 and 0.0.5, a broken 0.0.3 (exits at once), packs each
# into the generic bundle (the template's mqvpn, this tree's install.sh and
# hooks), then runs packaging/linux/test-update.sh in a container (podman):
# fake GitHub release → the agent sees it → Update now → rollback of a
# release that doesn't start → install.sh --update alone → SHA256SUMS refusal.
#
#   scripts/e2e-update-vps.sh [template-bundle.tar.gz] [image]
#   default: out/linux/tailbond-linux-x86_64.tar.gz (packaging/linux/build-bundle.sh), almalinux:10
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE="${1:-$ROOT/out/linux/tailbond-linux-x86_64.tar.gz}"
IMAGE="${2:-almalinux:10}"
OUT="$ROOT/out/e2e-update-vps"
[ -f "$TEMPLATE" ] || { echo "no bundle at $TEMPLATE: run packaging/linux/build-bundle.sh" >&2; exit 2; }
ENGINE=podman
ARCH=x86_64

rm -rf "$OUT" && mkdir -p "$OUT/src"
tar -C "$OUT/src" -xzf "$TEMPLATE"
src="$OUT/src/tailbond-linux-$ARCH"
cp "$ROOT/packaging/linux/install.sh" "$ROOT/packaging/linux/tailbond.service.in" "$src/"
cp "$ROOT/packaging/linux/hooks/"* "$src/hooks/"

pack() { # VERSION TAILBONDD
    local d="$OUT/p/tailbond-linux-$ARCH"
    rm -rf "$OUT/p" && mkdir -p "$OUT/p" && cp -a "$src" "$d"
    install -m 0755 "$2" "$d/bin/tailbondd"
    echo "$1" >"$d/VERSION"
    tar -C "$OUT/p" -czf "$OUT/tailbond-linux-$ARCH-$1.tar.gz" "tailbond-linux-$ARCH"
}
for v in 0.0.1 0.0.2 0.0.4 0.0.5; do
    (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$v" -o "$OUT/tailbondd-$v" ./cmd/tailbondd)
    pack "$v" "$OUT/tailbondd-$v"
done
# 0.0.3: passes install.sh's "does it run" check, then never starts
printf '#!/bin/sh\n[ "$1" = -version ] && { echo 0.0.3; exit 0; }\necho "broken release" >&2\nexit 1\n' >"$OUT/tailbondd-0.0.3"
pack 0.0.3 "$OUT/tailbondd-0.0.3"
rm -rf "$OUT/p" "$OUT/src"
ls -1 "$OUT"/*.tar.gz

"$ENGINE" run --rm --cap-add NET_ADMIN --device /dev/net/tun \
    -v "$ROOT:/src:ro,Z" -v "$OUT:/bundles:ro,Z" "$IMAGE" \
    bash /src/packaging/linux/test-update.sh /bundles
