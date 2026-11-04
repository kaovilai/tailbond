#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Build the tailbond server image (deploy/docker/Dockerfile) from the Linux
# bundles, as CI does (job "oci-image"):
#
#   packaging/linux/build-bundle.sh            # out/linux/tailbond-linux-<arch>.tar.gz
#   deploy/docker/build-image.sh               # this machine's arch -> localhost/tailbond:dev
#   deploy/docker/build-image.sh amd64 arm64   # a manifest list (the other arch needs qemu-user-static)
#
# Env:
#   IMAGE      default localhost/tailbond:dev
#   PODMAN     default podman (buildah's "bud" works too: PODMAN=buildah)
#   VERSION    image label (default: the bundle's VERSION file)
#   REVISION   image label (default: git rev-parse HEAD)
#   TB_BUNDLE_DIR  default out/linux
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE="${IMAGE:-localhost/tailbond:dev}"
read -ra PODMAN <<<"${PODMAN:-podman}" # "sudo podman" works too
BUNDLES="${TB_BUNDLE_DIR:-$ROOT/out/linux}"
die() { echo "build-image: $*" >&2; exit 1; }

archs=("$@")
if [ ${#archs[@]} -eq 0 ]; then
  case "$(uname -m)" in
    x86_64) archs=(amd64) ;;
    aarch64 | arm64) archs=(arm64) ;;
    *) die "unsupported machine $(uname -m)" ;;
  esac
fi

ctx="$(mktemp -d)"
trap 'rm -rf "$ctx"' EXIT
cp "$ROOT/deploy/docker/Dockerfile" "$ROOT/deploy/docker/entrypoint.sh" "$ROOT/deploy/docker/healthcheck.sh" "$ctx/"
chmod 0755 "$ctx/entrypoint.sh" "$ctx/healthcheck.sh"
platforms=()
for a in "${archs[@]}"; do
  case "$a" in
    amd64) ba=x86_64 ;;
    arm64) ba=aarch64 ;;
    *) die "unknown arch $a (amd64 or arm64)" ;;
  esac
  tgz="$BUNDLES/tailbond-linux-$ba.tar.gz"
  [ -f "$tgz" ] || die "$tgz missing: packaging/linux/build-bundle.sh"
  d="$ctx/bundle/$a"
  mkdir -p "$d"
  tar -C "$d" --strip-components=1 -xzf "$tgz"
  # the image runs the server only: no installer, systemd unit or firewalld files
  rm -rf "$d/install.sh" "$d/uninstall.sh" "$d/tailbond.service.in" "$d/firewalld" "$d/README.md"
  [ -x "$d/bin/tailbondd" ] && [ -x "$d/bin/mqvpn" ] || die "$tgz has no bin/tailbondd + bin/mqvpn"
  platforms+=("linux/$a")
done
version="${VERSION:-$(cat "$ctx/bundle/${archs[0]}/VERSION")}"
version="${version#v}"
revision="${REVISION:-$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || echo unknown)}"

# --format docker keeps HEALTHCHECK (the OCI image config has no field for it)
args=(build --format docker --build-arg "VERSION=$version" --build-arg "REVISION=$revision")
if [ ${#archs[@]} -eq 1 ]; then
  "${PODMAN[@]}" "${args[@]}" --platform "${platforms[0]}" -t "$IMAGE" "$ctx"
else
  "${PODMAN[@]}" manifest rm "$IMAGE" >/dev/null 2>&1 || true
  "${PODMAN[@]}" "${args[@]}" --platform "$(IFS=,; echo "${platforms[*]}")" --manifest "$IMAGE" "$ctx"
fi
echo "built $IMAGE (${platforms[*]}, version $version)"
