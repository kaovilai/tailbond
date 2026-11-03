#!/usr/bin/env bash
# Build the tailbond-server RPM for this machine's architecture into out/rpm/.
#
#   packaging/fedora/build-rpm.sh
#
# Steps: build mqvpn natively from the third_party/mqvpn submodule (skipped
# when build/mqvpn exists; JOBS caps parallelism, default 4), build a static
# tailbondd, stage everything into a tarball, then rpmbuild -bb.
# Needs: cmake gcc gcc-c++ make libevent-devel golang rpm-build systemd-rpm-macros.
#
# Env: VERSION (semver, a leading v is dropped; default: the v* tag at HEAD,
# else 0.1.0 with tailbondd reporting git describe, so a dev build never
# updates itself), TB_UPDATE_REPO / TB_UPDATE_SOURCE (compiled into
# tailbondd, scripts/release/ldflags.sh), JOBS, TAILBOND_APK,
# TAILBOND_GLINET_IPK.
# A pre-release 1.2.3-beta.1 becomes Version 1.2.3, Release 0.beta.1 (RPM
# versions can't hold "-"; 0.x sorts below the release's 1):
# tailbond-server-1.2.3-0.beta.1.fc44.x86_64.rpm.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG="$ROOT/packaging/fedora"
MQ="$ROOT/third_party/mqvpn"
OUT="$ROOT/out/rpm"
JOBS="${JOBS:-4}"
# shellcheck source=../../scripts/release/ldflags.sh
. "$ROOT/scripts/release/ldflags.sh"
AGENT_VERSION=""
if [ -z "${VERSION:-}" ]; then
  VERSION="$(git -C "$ROOT" describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null || true)"
  if [ -z "$VERSION" ]; then
    VERSION=0.1.0
    AGENT_VERSION="$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)"
  fi
fi
VERSION="$(tb_norm_version "$VERSION")"
[[ "$VERSION" =~ ^([0-9]+\.[0-9]+\.[0-9]+)(-([0-9A-Za-z.-]+))?$ ]] ||
  { echo "VERSION=$VERSION is not semver MAJOR.MINOR.PATCH[-pre]"; exit 1; }
RPM_VERSION="${BASH_REMATCH[1]}"
PRE="${BASH_REMATCH[3]}"
if [ -n "$PRE" ]; then RPM_RELEASE="0.${PRE//-/_}"; else RPM_RELEASE=1; fi

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
VERSION="${AGENT_VERSION:-$VERSION}" "$ROOT/agent/build.sh" "$GOARCH" >/dev/null
TBD="$ROOT/out/agent/linux-$GOARCH/tailbondd"

echo "==> staging"
TOP="$(mktemp -d)"
trap 'rm -rf "$TOP"' EXIT
SRC="$TOP/tailbond-server-$RPM_VERSION"
mkdir -p "$SRC"/{bin,lib,hooks,firewalld,doc,share} "$TOP"/{SOURCES,SPECS,BUILD,RPMS,SRPMS}
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
# Offered for download by the agent (/usr/share/tailbond/): the Android app
# (TAILBOND_APK, else the local debug build) and the GL router package.
apk="${TAILBOND_APK:-$ROOT/out/android/tailbond-debug.apk}"
if [ -f "$apk" ]; then install -m 0644 "$apk" "$SRC/share/tailbond.apk"
else echo "note: no Android APK at $apk; the phone page won't offer the app (set TAILBOND_APK)" >&2; fi
ipk="${TAILBOND_GLINET_IPK:-}"
if [ -z "$ipk" ]; then
  for f in "$ROOT"/out/ipk/tailbond_*_aarch64_cortex-a53_neon-vfpv4.ipk; do if [ -f "$f" ]; then ipk="$f"; fi; done
fi
if [ -n "$ipk" ]; then install -m 0644 "$ipk" "$SRC/share/tailbond-glinet.ipk"; fi
tar -C "$TOP" -czf "$TOP/SOURCES/tailbond-server-$RPM_VERSION.tar.gz" "tailbond-server-$RPM_VERSION"
cp "$PKG/tailbond-server.spec" "$TOP/SPECS/"

echo "==> rpmbuild"
rpmbuild -bb --define "_topdir $TOP" --define "tb_version $RPM_VERSION" --define "tb_release $RPM_RELEASE" "$TOP/SPECS/tailbond-server.spec" >"$TOP/rpmbuild.log" 2>&1 ||
  { tail -40 "$TOP/rpmbuild.log"; exit 1; }
mkdir -p "$OUT"
find "$TOP/RPMS" -name '*.rpm' -exec cp {} "$OUT/" \;
ls -1 "$OUT"/*.rpm
