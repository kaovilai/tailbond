#!/bin/busybox sh
# shellcheck shell=busybox
# /init of the e2e VM (scripts/e2e-macos/build-server-vm.sh): mount the
# basics, load the tunnel and NIC modules, bring up QEMU's user network.
#
# tb.role=server (default; scripts/e2e-macos.sh): eth0 10.0.2.15 (host
# 10.0.2.2), tailbondd in the server role and a token web server on :8080,
# which QEMU doesn't forward: only the tunnel reaches it.
# Kernel command line: tb.time=EPOCH (the clock: TLS needs it) tb.token=STR
# tb.public=ADDR (the address in pairings; QEMU forwards it to us)
# tb.port=PORT (mqvpn's UDP port, forwarded 1:1).
#
# tb.role=client (scripts/e2e-macos-server.sh, the Mac is the server):
# tailbondd in the client role bonding eth0 (192.168.71.15) and eth1
# (192.168.72.15), two QEMU user networks that each reach the Mac's address
# on their own; tb.pair=URI (the pairing link). Its dashboard is :8077 and
# /cgi-bin/probe on :8080 runs fixed probes from inside the VM (wget, ping,
# nslookup, routes) for the test on the Mac (both forwarded by QEMU to the
# Mac's loopback only).
/bin/busybox mkdir -p /proc /sys /dev /tmp /run /bin /sbin /usr/sbin
/bin/busybox --install -s
# the kernel starts /init with no PATH; busybox's shell has a default but
# doesn't export it, and tailbondd looks up tailscale (the fake) by PATH
export PATH=/usr/sbin:/usr/bin:/sbin:/bin
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
if [ "$(arg role)" = client ]; then
  PAIR="$(arg pair)"
  for f in /proc/sys/net/ipv4/conf/*/rp_filter; do echo 0 >"$f"; done
  ip link set eth0 up
  ip addr add 192.168.71.15/24 dev eth0
  ip route add default via 192.168.71.2 dev eth0
  ip link set eth1 up
  ip addr add 192.168.72.15/24 dev eth1
  # the second uplink: mqvpn binds its path socket to eth1, which then
  # takes this route to the server
  ip route add default via 192.168.72.2 dev eth1 metric 200
  hostname tb-e2e-client
  mkdir -p /etc/tailbond /var/lib/tailbond /usr/lib/tailbond/hooks /www/cgi-bin
  cat >/etc/tailbond/tailbond.json <<JSON
{"role": "client", "mqvpn_bin": "/usr/bin/mqvpn", "state_dir": "/var/lib/tailbond", "hooks_dir": "/usr/lib/tailbond/hooks",
 "listen": "0.0.0.0:8077", "mode": "speed", "disable_tailscale": true,
 "client": {"pairing_uri": "$PAIR", "no_dns": true, "captive_portal_detect": false,
            "paths": [{"iface": "eth0", "enabled": true, "priority": "primary", "kind": "ethernet"},
                      {"iface": "eth1", "enabled": true, "priority": "primary", "kind": "ethernet"}]}}
JSON
  cat >/www/cgi-bin/probe <<'CGI'
#!/bin/sh
# fixed probes for the e2e test: ?what=http&url=U | ping&host=H | dns&name=N&server=S | routes | log
q="$QUERY_STRING"
get() { echo "$q" | tr '&' '\n' | sed -n "s/^$1=//p" | head -n1; }
echo "Content-Type: text/plain"
echo
case "$(get what)" in
  http) wget -S -q -T 15 -O - "$(get url)" 2>&1 || echo "PROBE-FAILED" ;;
  ping) ping -c 3 -W 3 "$(get host)" 2>&1 || echo "PROBE-FAILED" ;;
  dns) nslookup "$(get name)" "$(get server)" 2>&1 || echo "PROBE-FAILED" ;;
  routes) ip route; ip -4 addr ;;
  log) tail -n 150 /tmp/client.log ;;
  *) echo "unknown probe" ;;
esac
CGI
  chmod 0755 /www/cgi-bin/probe
  tailbondd -config /etc/tailbond/tailbond.json >/tmp/client.log 2>&1 &
  /bin/busybox-extras httpd -p 8080 -h /www || echo "vm: httpd failed"
  echo "vm: client up ($(uname -r)); $(ip -4 -o addr | grep -o 'inet [0-9./]*' | tr '\n' ' ')"
  tail -F /tmp/client.log &
  while :; do sleep 3600; done
fi
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
 "listen": "0.0.0.0:8077", "mode": "speed",
 "server": {"port": $PORT, "public_addr": "$PUBLIC", "subnet": "10.77.0.0/24", "use_tailscale_cert": false}}
JSON
tailbondd -config /etc/tailbond/tailbond.json >/tmp/server.log 2>&1 &
/bin/busybox-extras httpd -p 8080 -h /www || echo "vm: httpd failed"
echo "vm: up ($(uname -r)), public $PUBLIC:$PORT; $(ip -4 addr show eth0 | grep -o 'inet [0-9./]*')"
# the server's log on the console (the e2e prints it on failure)
tail -F /tmp/server.log &
while :; do sleep 3600; done
