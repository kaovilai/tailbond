#!/usr/bin/env bash
# Build the tailbond-server RPM for this machine's architecture into out/rpm/.
#
#   packaging/fedora/build-rpm.sh
#
# Steps: build mqvpn natively from the third_party/mqvpn submodule (skipped
# when build/mqvpn exists; JOBS caps parallelism, default 4), build a static
# tailbondd, stage everything into a tarball, then rpmbuild -bb.
# Needs: cmake gcc gcc-c++ make libevent-devel golang rpm-build systemd-rpm-macros.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/fedora"
MQ="$ROOT/third_party/mqvpn"
OUT="$ROOT/out/rpm"
JOBS="${JOBS:-4}"
VERSION="${VERSION:-0.1.0}"

case "$(uname -m)" in
  x86_64) GOARCH=amd64 ;;
  aarch64) GOARCH=arm64 ;;
  *) echo "unsupported arch $(uname -m)"; exit 1 ;;
esac

if [ ! -f "$MQ/CMakeLists.txt" ] || [ ! -f "$MQ/third_party/xquic/third_party/boringssl/CMakeLists.txt" ]; then
  (cd "$ROOT" && git submodule update --init --recursive)
fi

if [ ! -x "$MQ/build/mqvpn" ] || [ ! -x "$MQ/build/mqvpn-nat-config" ]; then
  echo "==> building mqvpn (-j$JOBS)"
  # build.sh sizes make -j by nproc; a shim keeps it to $JOBS.
  shim="$(mktemp -d)"
  printf '#!/bin/sh\necho %s\n' "$JOBS" >"$shim/nproc"
  chmod +x "$shim/nproc"
  (cd "$MQ" && PATH="$shim:$PATH" ./build.sh)
  rm -rf "$shim"
fi

echo "==> building tailbondd ($GOARCH)"
VERSION="$VERSION" "$ROOT/agent/build.sh" "$GOARCH" >/dev/null
TBD="$ROOT/out/agent/linux-$GOARCH/tailbondd"

echo "==> staging"
TOP="$(mktemp -d)"
trap 'rm -rf "$TOP"' EXIT
SRC="$TOP/tailbond-server-$VERSION"
mkdir -p "$SRC"/{bin,lib,hooks,firewalld,doc} "$TOP"/{SOURCES,SPECS,BUILD,RPMS,SRPMS}
install -m 0755 "$MQ/build/mqvpn" "$SRC/bin/mqvpn"
strip "$SRC/bin/mqvpn" 2>/dev/null || true
install -m 0755 "$TBD" "$SRC/bin/tailbondd"
install -m 0755 "$MQ/scripts/mqvpn-server-nat.sh" "$SRC/lib/mqvpn-server-nat.sh"
install -m 0755 "$MQ/build/mqvpn-nat-config" "$SRC/lib/mqvpn-nat-config"
install -m 0755 "$PKG"/hooks/* "$SRC/hooks/"
install -m 0644 "$PKG"/firewalld/*.xml "$SRC/firewalld/"
install -m 0644 "$PKG/tailbond.service" "$PKG/tailbond.json" "$SRC/"
install -m 0644 "$ROOT/docs/fedora-server.md" "$ROOT/docs/architecture.md" "$SRC/doc/"
install -m 0644 "$MQ/LICENSE" "$SRC/doc/LICENSE.mqvpn"
install -m 0644 "$MQ/NOTICE" "$SRC/doc/NOTICE.mqvpn"
tar -C "$TOP" -czf "$TOP/SOURCES/tailbond-server-$VERSION.tar.gz" "tailbond-server-$VERSION"
cp "$PKG/tailbond-server.spec" "$TOP/SPECS/"

echo "==> rpmbuild"
rpmbuild -bb --define "_topdir $TOP" --define "tb_version $VERSION" "$TOP/SPECS/tailbond-server.spec" >"$TOP/rpmbuild.log" 2>&1 ||
  { tail -40 "$TOP/rpmbuild.log"; exit 1; }
mkdir -p "$OUT"
find "$TOP/RPMS" -name '*.rpm' -exec cp {} "$OUT/" \;
ls -1 "$OUT"/*.rpm
