#!/usr/bin/env bash
# Build static tailbondd binaries: out/agent/<os>-<arch>/tailbondd
#   agent/build.sh                # linux amd64 + arm64
#   agent/build.sh arm64          # one Linux arch
#   agent/build.sh darwin/arm64   # macOS (docs/macos.md)
# Env: VERSION (default git describe; a leading v is dropped),
#      TB_UPDATE_REPO (scripts/release/ldflags.sh).
set -euo pipefail
cd "$(dirname "$0")"
# shellcheck source=../scripts/release/ldflags.sh
. ../scripts/release/ldflags.sh
OUT="$(cd .. && pwd)/out/agent"
VERSION="$(tb_norm_version "${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}")"
LDFLAGS="$(tb_ldflags "$VERSION")"
ARCHES=("$@")
[ ${#ARCHES[@]} -eq 0 ] && ARCHES=(amd64 arm64)
for target in "${ARCHES[@]}"; do
  os=linux arch="$target"
  case "$target" in */*) os="${target%/*}" arch="${target#*/}" ;; esac
  mkdir -p "$OUT/$os-$arch"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "$LDFLAGS" -o "$OUT/$os-$arch/tailbondd" ./cmd/tailbondd
  echo "$OUT/$os-$arch/tailbondd"
done
