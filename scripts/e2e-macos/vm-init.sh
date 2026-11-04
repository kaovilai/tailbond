#!/bin/busybox sh
# shellcheck shell=busybox
# /init of the e2e server VM (scripts/e2e-macos/build-server-vm.sh): mount
# the basics, load the tunnel and NIC modules, bring up QEMU's user network
# (10.0.2.15, host 10.0.2.2), start tailbondd in the server role and a token
# web server on :8080, which QEMU doesn't forward: only the tunnel reaches it.
#
# Kernel command line: tb.time=EPOCH (the clock: TLS needs it) tb.token=STR
# tb.public=ADDR (the address in pairings; QEMU forwards it to us)
# tb.port=PORT (mqvpn's UDP port, forwarded 1:1).
/bin/busybox mkdir -p /proc /sys /dev /tmp /run /bin /sbin /usr/sbin
/bin/busybox --install -s
mount -t proc proc /proc
mount -t sysfs sys /sys
mount -t devtmpfs dev /dev
mount -t tmpfs tmp /tmp
for m in /lib/modules/*.ko; do insmod "$m" || echo "vm: insmod $m failed"; done
echo "vm: modules: $(ls /lib/modules | tr '\n' ' ')"
mkdir -p /dev/net
[ -c /dev/net/tun ] || mknod /dev/net/tun c 10 200

arg() { tr ' ' '\n' </proc/cmdline | sed -n "s/^tb\.$1=//p" | tail -n1; }
T="$(arg time)"; TOKEN="$(arg token)"; PUBLIC="$(arg public)"; PORT="$(arg port)"
[ -n "$T" ] && date -s "@$T" >/dev/null
: "${PUBLIC:=127.0.0.1}" "${PORT:=4443}"

ip link set lo up
ip link show eth0 >/dev/null 2>&1 || { echo "vm: no eth0"; ls /sys/class/net; }
ip link set eth0 up
ip addr add 10.0.2.15/24 dev eth0
ip route add default via 10.0.2.2
hostname tb-e2e-server

mkdir -p /etc/tailbond /var/lib/tailbond /usr/lib/tailbond/hooks /www
printf '%s\n' "$TOKEN" >/www/token
dd if=/dev/urandom of=/www/blob bs=1024 count=2048 2>/dev/null
cat >/etc/tailbond/tailbond.json <<JSON
{"role": "server", "mqvpn_bin": "/usr/bin/mqvpn", "state_dir": "/var/lib/tailbond", "hooks_dir": "/usr/lib/tailbond/hooks",
 "listen": "0.0.0.0:8077", "mode": "speed", "disable_tailscale": true,
 "server": {"port": $PORT, "public_addr": "$PUBLIC", "subnet": "10.77.0.0/24", "use_tailscale_cert": false}}
JSON
tailbondd -config /etc/tailbond/tailbond.json >/tmp/server.log 2>&1 &
/bin/busybox-extras httpd -p 8080 -h /www || echo "vm: httpd failed"
echo "vm: up ($(uname -r)), public $PUBLIC:$PORT; $(ip -4 addr show eth0 | grep -o 'inet [0-9./]*')"
# the server's log on the console (the e2e prints it on failure)
tail -F /tmp/server.log &
while :; do sleep 3600; done
