#!/usr/bin/env bash
# Build static tailbondd binaries: out/agent/linux-<arch>/tailbondd
#   agent/build.sh              # amd64 + arm64
#   agent/build.sh arm64        # one arch
set -euo pipefail
cd "$(dirname "$0")"
OUT="$(cd .. && pwd)/out/agent"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ARCHES=("$@")
[ ${#ARCHES[@]} -eq 0 ] && ARCHES=(amd64 arm64)
for arch in "${ARCHES[@]}"; do
  mkdir -p "$OUT/linux-$arch"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" -o "$OUT/linux-$arch/tailbondd" ./cmd/tailbondd
  echo "$OUT/linux-$arch/tailbondd"
done
