#!/usr/bin/env bash
# Build static tailbondd binaries: out/agent/linux-<arch>/tailbondd
#   agent/build.sh              # amd64 + arm64
#   agent/build.sh arm64        # one arch
# Env: VERSION (default git describe; a leading v is dropped),
#      TB_UPDATE_REPO / TB_UPDATE_SOURCE (scripts/release/ldflags.sh).
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck source=../scripts/release/ldflags.sh
. ../scripts/release/ldflags.sh
OUT="$(cd .. && pwd)/out/agent"
VERSION="$(tb_norm_version "${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}")"
LDFLAGS="$(tb_ldflags "$VERSION")"
ARCHES=("$@")
[ ${#ARCHES[@]} -eq 0 ] && ARCHES=(amd64 arm64)
for arch in "${ARCHES[@]}"; do
  mkdir -p "$OUT/linux-$arch"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "$LDFLAGS" -o "$OUT/linux-$arch/tailbondd" ./cmd/tailbondd
  echo "$OUT/linux-$arch/tailbondd"
done
