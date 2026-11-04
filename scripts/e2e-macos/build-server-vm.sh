#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# The tailbond server for scripts/e2e-macos.sh: a tiny aarch64 Linux VM
# (kernel + initramfs) that runs tailbondd in the server role with the
# static mqvpn, plus a token web server only the tunnel reaches. A server in
# the Mac's own network stack would answer the hub's tunnel traffic locally,
# so the macOS runner boots this under QEMU instead. The same image boots as
# a bonding Linux client (kernel argument tb.role=client, vm-init.sh) for
# scripts/e2e-macos-server.sh, where the Mac is the server.
#
#   MQVPN=out/aarch64/mqvpn TAILBONDD=out/agent/linux-arm64/tailbondd \
#       scripts/e2e-macos/build-server-vm.sh        # -> out/e2e-vm/{vmlinuz,initramfs.gz}
#
# Kernel, modules (tun, virtio-net, virtio-rng) and busybox come from Alpine
# (linux-virt, busybox-static) in an arm64 container: run it on an arm64
# Linux host (CI: ubuntu-24.04-arm) with podman.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MQVPN="${MQVPN:-$ROOT/out/aarch64/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-arm64/tailbondd}"
OUT="${OUT:-$ROOT/out/e2e-vm}"
IMAGE="${TB_ALPINE_IMAGE:-docker.io/library/alpine:3.20}"
PODMAN="${PODMAN:-podman}"

die() { echo "build-server-vm: $*" >&2; exit 1; }
[ -x "$MQVPN" ] || die "no static aarch64 mqvpn at $MQVPN (scripts/build-mqvpn-aarch64-musl.sh)"
[ -x "$TAILBONDD" ] || die "no linux/arm64 tailbondd at $TAILBONDD (agent/build.sh arm64)"
file "$MQVPN" | grep -q 'aarch64\|ARM aarch64' || die "$MQVPN is not an aarch64 binary"
for c in "$PODMAN" cpio gzip; do command -v "$c" >/dev/null || die "'$c' not found"; done

rm -rf "$OUT"
mkdir -p "$OUT/root"
# Kernel, the modules it needs for the tunnel and the NIC (dependencies
# first, decompressed: busybox insmod takes plain .ko), static busybox.
"$PODMAN" run --rm --arch arm64 -v "$OUT:/out" "$IMAGE" sh -euc '
  apk add --no-cache --no-scripts linux-virt busybox-static busybox-extras kmod xz zstd >/dev/null
  kver="$(ls /lib/modules | head -n1)"
  cp /boot/vmlinuz-virt /out/vmlinuz
  mkdir -p /out/root/bin /out/root/lib/modules
  cp /bin/busybox.static /out/root/bin/busybox
  # httpd (the token server) is in busybox-extras, linked against musl
  mkdir -p /out/root/lib
  cp /bin/busybox-extras /out/root/bin/busybox-extras
  cp /lib/ld-musl-aarch64.so.1 /out/root/lib/
  n=0
  modprobe -a -S "$kver" --show-depends tun virtio_net virtio_pci virtio_rng |
  while read -r what path _; do
    [ "$what" = insmod ] || continue
    n=$((n + 1))
    name="$(printf %02d "$n")-$(basename "$path" | sed "s/\.ko.*//").ko"
    case "$path" in
      *.gz) gzip -dc "$path" ;;
      *.xz) xz -dc "$path" ;;
      *.zst) zstd -qdc "$path" ;;
      *) cat "$path" ;;
    esac >"/out/root/lib/modules/$name"
  done
  ls /out/root/lib/modules
  echo "kernel $kver"
'
mkdir -p "$OUT/root/usr/bin" "$OUT/root/www"
install -m 0755 "$MQVPN" "$OUT/root/usr/bin/mqvpn"
install -m 0755 "$TAILBONDD" "$OUT/root/usr/bin/tailbondd"
install -m 0755 "$ROOT/scripts/e2e-macos/vm-init.sh" "$OUT/root/init"
# no Tailscale in the VM: the server role checks it and carries on
# a fake tailscale CLI: the server's tailnet for GET /api/peers. "build"
# sits inside the simulated NetBird range on the Mac (100.92.0.0/16), so the
# hub must reach it by a /32 (scripts/e2e-macos.sh, other VPNs).
cat >"$OUT/root/usr/bin/tailscale" <<'TS'
#!/bin/sh
case "$1" in
status) cat <<'J'
{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.1.1"],"Online":true},
 "Peer":{"n1":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true},
         "n2":{"HostName":"build","DNSName":"build.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.92.5.5"],"Online":true}}}
J
;;
debug) echo '{"AdvertiseRoutes":[]}' ;;
cert|whois) echo "not in this test" >&2; exit 1 ;;
*) exit 0 ;;
esac
TS
chmod 0755 "$OUT/root/usr/bin/tailscale"
(cd "$OUT/root" && find . | LC_ALL=C sort | cpio -o -H newc --quiet -R 0:0 | gzip -9) >"$OUT/initramfs.gz"
rm -rf "$OUT/root"
ls -l "$OUT"
