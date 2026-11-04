#!/usr/bin/env bash
# End-to-end test: tailbondd (server role) + tailbondd (client role) driving
# mqvpn in two network namespaces joined by TWO veth paths, one with netem
# delay/loss. Needs root (re-execs itself with sudo), iproute-tc and curl.
#
#   scripts/e2e-netns.sh                 # build what is missing, run, clean up
#   KEEP=1 scripts/e2e-netns.sh          # leave namespaces + agents running
#   MQVPN=/path/to/mqvpn TAILBONDD=/path/to/tailbondd scripts/e2e-netns.sh
#
# Checks (hard): ping through the tunnel; the server agent's /api/status lists
# the client with 2 active paths; traffic continues after one path goes down;
# mode switch reconnects; hooks fire; revocation works; GET /api/peers is
# served on the tunnel IP to paired clients only (401 without / 200 with the
# token derived from the pairing key); pairing over Tailscale: a fake
# `tailscale` CLI on PATH in each namespace (status/whois return canned
# JSON) plus 100.x addresses, so the client agent discovers the server
# (GET /api/hello), asks to pair (POST /api/join -> /api/pair-requests), the
# admin approves on the server, and the client connects by itself.
# Captive portals: a fake portal in the server namespace intercepts one
# path at a time. eth0 ("wan") is checked by tailbond's own probe; wlan0
# ("wwan", the repeater) is owned by a mocked GL `ubus call repeater status`
# (portal -> Login Mode -> passed). The real OpenWrt portal-open/close hooks
# run against the stub uci; a dummy lan0 stands in for the LAN to check the
# sign-in bypass with `ip route get ... iif lan0`.
# Client routing uses the real OpenWrt client-up/client-down hooks (stub uci,
# real ip rules; manage_routes off, as on a router). Three LAN namespaces
# (laptop, tv, phone) sit behind the client's lan0 bridge, the router's
# dnsmasq (with nftset) serves them, and a fake internet in the server
# namespace (web server + DNS) reports the source address it sees.
# Bypass rules: a website rule and a device rule exit the WAN directly
# (source 10.200.x.2), other traffic and an "always tailbond" device use the
# tunnel; with the tunnel down the kill switch blocks the rest while bypassed
# traffic still flows. Failover: a second server namespace ("vps") is added
# over the fake tailnet with "add as backup" (it moves off 10.77.0.0/24 by
# itself); the home server loses power (agent + mqvpn killed) and the client
# switches to the vps, then fails back once home has answered for the hold
# time (timers shortened in the client config).
# Checks (soft): client-side per-path stats, which need the mqvpn client
# control API patch (patches/mqvpn/0001-client-control-api.patch).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-$(go env GOARCH 2>/dev/null || echo amd64)/tailbondd}"
NS_S=tb-e2e-srv
NS_C=tb-e2e-cli
NS_V=tb-e2e-vps
NS_LAN="tb-e2e-laptop tb-e2e-tv tb-e2e-phone"
SERVER_IP=10.200.1.1
API=127.0.0.1:8077

if [ "$(id -u)" -ne 0 ]; then
  if [ ! -x "$TAILBONDD" ]; then
    echo "building tailbondd -> $TAILBONDD"
    (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -o "$TAILBONDD" ./cmd/tailbondd)
  fi
  exec sudo --preserve-env=KEEP,VERBOSE MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" "$0" "$@"
fi

[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN (cd third_party/mqvpn && ./build.sh)"; exit 2; }
[ -x "$TAILBONDD" ] || { echo "tailbondd not found at $TAILBONDD"; exit 2; }
command -v tc >/dev/null || { echo "tc missing: dnf install iproute-tc"; exit 2; }
command -v curl >/dev/null || { echo "curl missing"; exit 2; }
command -v jq >/dev/null || { echo "jq missing"; exit 2; }
command -v openssl >/dev/null || { echo "openssl missing"; exit 2; }
command -v nft >/dev/null || { echo "nft missing: dnf install nftables"; exit 2; }
command -v dnsmasq >/dev/null || { echo "dnsmasq missing: dnf install dnsmasq (or apt-get install dnsmasq-base)"; exit 2; }
command -v nslookup >/dev/null || { echo "nslookup missing: dnf install bind-utils"; exit 2; }

WORK="$(mktemp -d /tmp/tailbond-e2e.XXXXXX)"
PASS=0 FAIL=0 SOFT=0
SRV_PID="" CLI_PID="" VPS_PID=""

say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
soft() { printf '  \033[33mSKIP\033[0m %s\n' "$*"; SOFT=$((SOFT + 1)); }

sapi() { ip netns exec "$NS_S" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
capi() { ip netns exec "$NS_C" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }

cleanup() {
  local rc=$?
  if [ -n "${KEEP:-}" ]; then
    echo "KEEP set: namespaces $NS_S/$NS_C and agents left running; work dir $WORK"
    return
  fi
  for p in "$CLI_PID" "$SRV_PID" "$VPS_PID"; do [ -n "$p" ] && kill "$p" 2>/dev/null || true; done
  sleep 2
  for ns in "$NS_S" "$NS_C" "$NS_V" $NS_LAN; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    ip netns del "$ns" 2>/dev/null || true
    rm -rf "/etc/netns/$ns"
  done
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- server agent log (tail) ---"; tail -n 40 "$WORK/server.log" 2>/dev/null || true
    echo "--- client agent log (tail) ---"; tail -n 60 "$WORK/client.log" 2>/dev/null || true
    echo "--- vps agent log (tail) ---"; tail -n 20 "$WORK/vps.log" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

wait_for() { # wait_for <seconds> <description> <command...>
  local t=$1 what=$2; shift 2
  for _ in $(seq 1 "$t"); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "  timed out after ${t}s waiting for: $what"
  return 1
}

say "network: $NS_S <-> $NS_C over two veth paths"
for ns in "$NS_S" "$NS_C" "$NS_V" $NS_LAN; do ip netns del "$ns" 2>/dev/null || true; done
ip netns add "$NS_S"
ip netns add "$NS_C"
for ns in "$NS_S" "$NS_C"; do ip -n "$ns" link set lo up; done
# Path A: "eth0" (clean). Path B: "wlan0" (netem delay + loss), as a phone
# hotspot or flaky Wi-Fi would be. Names drive the kind heuristic.
ip link add tbe2e-a type veth peer name tbe2e-as
ip link add tbe2e-b type veth peer name tbe2e-bs
ip link set tbe2e-a netns "$NS_C"; ip -n "$NS_C" link set tbe2e-a name eth0
ip link set tbe2e-b netns "$NS_C"; ip -n "$NS_C" link set tbe2e-b name wlan0
ip link set tbe2e-as netns "$NS_S"; ip -n "$NS_S" link set tbe2e-as name pa
ip link set tbe2e-bs netns "$NS_S"; ip -n "$NS_S" link set tbe2e-bs name pb
ip -n "$NS_C" addr add 10.200.1.2/24 dev eth0
ip -n "$NS_C" addr add 10.200.2.2/24 dev wlan0
ip -n "$NS_S" addr add 10.200.1.1/24 dev pa
ip -n "$NS_S" addr add 10.200.2.1/24 dev pb
for l in eth0 wlan0; do ip -n "$NS_C" link set "$l" up; done
for l in pa pb; do ip -n "$NS_S" link set "$l" up; done
# The server address must be reachable over either path.
ip -n "$NS_C" route add default via 10.200.1.1 dev eth0 metric 100
ip -n "$NS_C" route add default via 10.200.2.1 dev wlan0 metric 200
ip -n "$NS_C" route add "$SERVER_IP/32" via 10.200.2.1 dev wlan0 metric 200
ip netns exec "$NS_S" sysctl -qw net.ipv4.ip_forward=1
ip netns exec "$NS_C" tc qdisc add dev wlan0 root netem delay 40ms 5ms loss 1%
ip netns exec "$NS_S" tc qdisc add dev pb root netem delay 40ms 5ms loss 1%
ip netns exec "$NS_C" ping -c3 -i0.2 -w5 10.200.1.1 >/dev/null && ip netns exec "$NS_C" ping -c3 -i0.2 -w5 10.200.2.1 >/dev/null
ok "underlay reachable on both paths (wlan0: netem 40ms ±5ms, 1% loss)"

say "fake tailnet: homebox 100.101.1.1 (server), slate7 100.101.9.9 (client)"
ip -n "$NS_S" addr add 100.101.1.1/32 dev lo
ip -n "$NS_C" addr add 100.101.9.9/32 dev lo
ip -n "$NS_C" route add 100.101.1.1/32 via 10.200.1.1 dev eth0 src 100.101.9.9
ip -n "$NS_S" route add 100.101.9.9/32 via 10.200.1.2 dev pa
mkdir -p "$WORK/ts-srv" "$WORK/ts-cli"
fake_ts() { # fake_ts DIR STATUS_JSON
  cat >"$1/tailscale" <<EOF
#!/bin/sh
# fake tailscale CLI for the e2e test
case "\$1" in
status) cat <<'J'
$2
J
;;
whois)
  [ "\$3" = 100.101.9.9 ] || { echo "no such peer" >&2; exit 1; }
  echo '{"Node":{"ID":9,"StableID":"nE2ESLATE","Name":"slate7.tail1234.ts.net.","ComputedName":"slate7","Hostinfo":{"Hostname":"GL-BE3600","OS":"linux"}},"UserProfile":{"LoginName":"alex@example.com","DisplayName":"Alex"}}' ;;
debug) echo '{"AdvertiseRoutes":[]}' ;;
cert) echo "certs disabled" >&2; exit 1 ;;
*) exit 0 ;;
esac
EOF
  chmod +x "$1/tailscale"
}
fake_ts "$WORK/ts-srv" '{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.1.1"],"Online":true},
 "Peer":{"n1":{"HostName":"slate7","DNSName":"slate7.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.9.9"],"Online":true},
         "n2":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true},
         "n3":{"HostName":"build","DNSName":"build.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.92.5.5"],"Online":true}}}'
fake_ts "$WORK/ts-cli" '{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"slate7","DNSName":"slate7.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.9.9"],"Online":true},
 "Peer":{"n1":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.1.1"],"Online":true},
         "n2":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true}}}'
ok "fake tailscale CLIs in place"

say "fake captive portal (10.200.9.9) and a mocked GL repeater"
ip -n "$NS_S" addr add 10.200.9.9/32 dev lo
echo '{"wan": false, "wwan": false}' >"$WORK/portal-state.json"
cat >"$WORK/portal.py" <<'EOF'
import http.server, json, sys
STATE, LOG = sys.argv[1], sys.argv[2]
SIDE = {"10.200.1.2": "wan", "10.200.2.2": "wwan"}
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        side = SIDE.get(self.client_address[0], "?")
        with open(LOG, "a") as f: f.write("%s %s\n" % (side, self.path))
        st = json.load(open(STATE))
        if self.path.startswith("/login"):
            st[side] = False
            json.dump(st, open(STATE, "w"))
            body = b"signed in"
            self.send_response(200); self.send_header("Content-Length", str(len(body))); self.end_headers(); self.wfile.write(body); return
        if self.path.startswith("/hotspot-detect"):  # GL's own check URL
            body = b"<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>"
            self.send_response(200); self.send_header("Content-Length", str(len(body))); self.end_headers(); self.wfile.write(body); return
        if st.get(side):
            self.send_response(302); self.send_header("Location", "http://10.200.9.9/login"); self.send_header("Content-Length", "0"); self.end_headers(); return
        self.send_response(204); self.end_headers()
http.server.HTTPServer(("10.200.9.9", 80), H).serve_forever()
EOF
ip netns exec "$NS_S" python3 "$WORK/portal.py" "$WORK/portal-state.json" "$WORK/portal.log" >/dev/null 2>&1 &
PORTAL_PID=$!
# The GL repeater mock, replaced in one rename: the fake ubus reading a
# half-written file would look like "no GL" and make the agent probe wlan0
# itself and run portal hooks out of turn.
gl_repeater() { printf '%s\n' "$1" >"$WORK/gl-repeater.json.tmp" && mv -f "$WORK/gl-repeater.json.tmp" "$WORK/gl-repeater.json"; }
gl_repeater '{"ssid":"Hotel-Guest","state":2,"portal":false,"portal_detecting":false,"portal_url":"","bare_mode":false,"auto_portal":true}'
cat >"$WORK/ts-cli/ubus" <<EOF
#!/bin/sh
# fake ubus: OpenWrt netifd (eth0 = wan, wlan0 = wwan) and GL's repeater
case "\$*" in
"call network.interface dump") cat <<'J'
{"interface":[
 {"interface":"lan","up":true,"proto":"static","l3_device":"lan0","device":"lan0"},
 {"interface":"wan","up":true,"proto":"dhcp","l3_device":"eth0","device":"eth0","dns-server":["10.200.1.1"]},
 {"interface":"wwan","up":true,"proto":"dhcp","l3_device":"wlan0","device":"wlan0","dns-server":["10.200.2.1"]}]}
J
;;
"call repeater status") cat "$WORK/gl-repeater.json" ;;
*) exit 1 ;;
esac
EOF
chmod +x "$WORK/ts-cli/ubus"
# the LAN of the travel router: a bridge with three devices behind it
ip -n "$NS_C" link add lan0 type bridge
ip -n "$NS_C" addr add 192.168.8.1/24 dev lan0
ip -n "$NS_C" link set lan0 up
ip netns exec "$NS_C" sysctl -qw net.ipv4.ip_forward=1
i=0
for dev in laptop=02:00:00:00:00:0a=192.168.8.10 tv=02:00:00:00:00:0b=192.168.8.20 phone=02:00:00:00:00:0c=192.168.8.30; do
  IFS='=' read -r name mac lip <<<"$dev"
  ns="tb-e2e-$name"
  ip netns add "$ns"; ip -n "$ns" link set lo up
  ip link add "tbe2e-l$i" type veth peer name "tbe2e-l${i}p"
  ip link set "tbe2e-l$i" netns "$NS_C"; ip -n "$NS_C" link set "tbe2e-l$i" master lan0 up
  ip link set "tbe2e-l${i}p" netns "$ns"; ip -n "$ns" link set "tbe2e-l${i}p" name eth0
  ip -n "$ns" link set eth0 address "$mac"; ip -n "$ns" addr add "$lip/24" dev eth0; ip -n "$ns" link set eth0 up
  ip -n "$ns" route add default via 192.168.8.1
  mkdir -p "/etc/netns/$ns"; echo "nameserver 192.168.8.1" >"/etc/netns/$ns/resolv.conf"
  i=$((i + 1))
done
printf '%s\n' "1793000000 02:00:00:00:00:0a 192.168.8.10 work-laptop *" "1793000000 02:00:00:00:00:0b 192.168.8.20 living-room-tv *" \
  "1793000000 02:00:00:00:00:0c 192.168.8.30 pixel-8 *" >"$WORK/dhcp.leases"
# what fw4's zones would do: masquerade LAN traffic leaving by any WAN or tb0
ip netns exec "$NS_C" nft -f - <<'NFT'
table ip e2e_nat {
  chain post { type nat hook postrouting priority 100; ip saddr 192.168.8.0/24 oifname != "lan0" masquerade; }
}
NFT
# the real OpenWrt portal hooks, with the stub uci on a fake /etc/config
mkdir -p "$WORK/fakeroot/etc/config" "$WORK/stubs" "$WORK/hookstate"
printf '#!/bin/sh\nexec python3 %s "$@"\n' "$ROOT/packaging/openwrt/test/stubs/uci" >"$WORK/stubs/uci"
chmod +x "$WORK/stubs/uci"
printf "config interface 'lan'\n\toption device 'lan0'\n" >"$WORK/fakeroot/etc/config/network"
printf "config client 'client'\n\tlist lan_network 'lan'\n\toption rule_priority '1077'\n\toption portal_table '78'\n" >"$WORK/fakeroot/etc/config/tailbond"
printf "config dnsmasq\n\tlist server '10.77.0.1'\n\toption noresolv '1'\n\toption rebind_protection '1'\n" >"$WORK/fakeroot/etc/config/dhcp"
fuci() { FAKE_ROOT="$WORK/fakeroot" "$WORK/stubs/uci" "$@"; }
# the client hooks caught up: the last one was client-up, and it finished
hooks_settled() { grep -E ' client-(up|down) ' "$WORK/hook-history.log" | tail -1 | grep -q ' client-up ' && [ -e "$WORK/hook-client-up.done" ]; }
ok "fake portal, GL repeater mock (auto_portal on, no portal), lan0"

say "hooks"
mkdir -p "$WORK/hooks"
for h in server-up server-down; do
  cat >"$WORK/hooks/$h" <<EOF
#!/bin/sh
env | grep '^TB_' | sort > "$WORK/hook-$h.env"
echo "$h ran"
EOF
  chmod +x "$WORK/hooks/$h"
done

# The real OpenWrt client and bypass hooks too (stub uci, real ip/nft). The
# router's dnsmasq is a test one (with nftset) that follows the tunnel DNS
# the way UCI dnsmasq would.
cat >"$WORK/dnsmasq-restart" <<EOF
#!/bin/sh
[ -f "$WORK/dnsmasq-cli.pid" ] && kill "\$(cat "$WORK/dnsmasq-cli.pid")" 2>/dev/null && sleep 0.3
exec dnsmasq --pid-file="$WORK/dnsmasq-cli.pid" --conf-dir="$WORK/dnsmasq.d" --listen-address=192.168.8.1,127.0.0.1 \\
  --bind-interfaces --no-resolv --no-hosts --cache-size=0 --server="\$(cat "$WORK/tunnel-dns" 2>/dev/null || echo 10.77.0.1)" --user=root
EOF
chmod +x "$WORK/dnsmasq-restart"
mkdir -p "$WORK/dnsmasq.d"
for h in client-up client-down bypass-apply portal-open portal-close; do
  cat >"$WORK/hooks/$h" <<EOF
#!/bin/sh
env | grep '^TB_' | sort > "$WORK/hook-$h.env"
export FAKE_ROOT="$WORK/fakeroot" TB_STATE="$WORK/hookstate" TB_INITD=/nonexistent TB_LIB="$ROOT/packaging/openwrt/hooks/lib.sh"
export PATH="$WORK/stubs:\$PATH"
export TB_BYPASS_FIREWALL=nft TB_DNSMASQ_DIR="$WORK/dnsmasq.d" TB_DNSMASQ_RESTART="$WORK/dnsmasq-restart"
echo "\$(date +%s) $h \${TB_SERVER_NAME:-} \${TB_SUBNET:-} \${TB_DNS_IP:-} switch=\${TB_SWITCH:-0} \${TB_BYPASS_ACTION:-}" >> "$WORK/hook-history.log"
if [ "$h" = client-up ] && [ -n "\${TB_DNS_IP:-}" ] && [ "\$(cat "$WORK/tunnel-dns" 2>/dev/null)" != "\$TB_DNS_IP" ]; then
  echo "\$TB_DNS_IP" > "$WORK/tunnel-dns"; "$WORK/dnsmasq-restart" >/dev/null 2>&1 &
fi
rm -f "$WORK/hook-$h.done"
"$ROOT/packaging/openwrt/hooks/$h"; rc=\$?
touch "$WORK/hook-$h.done"  # the checks wait for the hook to finish, not just start
exit \$rc
EOF
  chmod +x "$WORK/hooks/$h"
done

say "fake internet in $NS_S: DNS on 10.200.1.1/10.200.2.1 (bank.example, other.example), web server :8099 echoing the source"
ip -n "$NS_S" addr add 198.51.100.10/32 dev lo
ip -n "$NS_S" addr add 198.51.100.20/32 dev lo
ip netns exec "$NS_S" dnsmasq --pid-file="$WORK/dnsmasq-inet.pid" --listen-address=10.200.1.1,10.200.2.1 --bind-interfaces \
  --no-resolv --no-hosts --address=/bank.example/198.51.100.20 --address=/other.example/198.51.100.10 --user=root
cat >"$WORK/echo.py" <<'EOF'
import http.server
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        body = self.client_address[0].encode()
        self.send_response(200); self.send_header("Content-Length", str(len(body))); self.end_headers(); self.wfile.write(body)
http.server.ThreadingHTTPServer(("0.0.0.0", 8099), H).serve_forever()
EOF
ip netns exec "$NS_S" python3 "$WORK/echo.py" >/dev/null 2>&1 &
# the home server's own resolver (its tunnel DNS forwards there)
mkdir -p "/etc/netns/$NS_S"
echo "nameserver 10.200.1.1" >"/etc/netns/$NS_S/resolv.conf"
ip netns exec "$NS_C" "$WORK/dnsmasq-restart" >/dev/null 2>&1 &
ok "fake internet + the router's dnsmasq up"

say "start tailbondd (server role)"
mkdir -p "$WORK/srv" "$WORK/cli"
cat >"$WORK/srv/tailbond.json" <<EOF
{
  "role": "server",
  "mqvpn_bin": "$MQVPN",
  "state_dir": "$WORK/srv/state",
  "hooks_dir": "$WORK/hooks",
  "listen": ":8077",
  "mode": "speed",
  "android_apk": "$WORK/srv/tailbond.apk",
  "server": {"port": 443, "public_addr": "$SERVER_IP", "subnet": "10.77.0.0/24", "subnet6": "fd77:77::/112", "use_tailscale_cert": false}
}
EOF
printf 'PK\003\004 e2e apk' >"$WORK/srv/tailbond.apk"
ip netns exec "$NS_S" env PATH="$WORK/ts-srv:$PATH" "$TAILBONDD" -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
if wait_for 20 "server agent connected" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .control_api'"; then
  ok "server agent up, mqvpn server running, control API reachable"
else
  bad "server did not come up"; exit 1
fi
[ -s "$WORK/hook-server-up.env" ] && grep -q '^TB_SUBNET=10.77.0.0/24' "$WORK/hook-server-up.env" && grep -q '^TB_DNS_IP=10.77.0.1' "$WORK/hook-server-up.env" \
  && ok "server-up hook ran with TB_* env" || bad "server-up hook missing or wrong env"
grep -q '^TB_WEB_PORTS=8077 80$' "$WORK/hook-server-up.env" && ok "server-up hook told which tcp ports to open to the tailnet (8077 80)" \
  || bad "TB_WEB_PORTS: $(grep TB_WEB_PORTS "$WORK/hook-server-up.env")"

say "pairing link (POST /api/clients)"
PAIR="$(sapi -X POST -d '{"name":"e2e-legacy"}' "http://$API/api/clients")"
URI="$(jq -r .uri <<<"$PAIR")"
[[ "$URI" == tailbond://pair\?d=* ]] && jq -e '.qr_svg | startswith("<svg")' <<<"$PAIR" >/dev/null \
  && ok "POST /api/clients issued a pairing URI + QR SVG" || bad "bad pairing response: $PAIR"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
PAYLOAD="$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d)"
jq -e '.v == 1 and .server == "10.200.1.1:443" and .insecure == true and (.cert_sha256 | length) == 64 and .dns == ["10.77.0.1"]' <<<"$PAYLOAD" >/dev/null \
  && ok "payload: v1, server, pinned self-signed cert, dns" || bad "payload unexpected: $PAYLOAD"
sapi "http://$API/api/clients" | jq -e '.[] | select(.name == "e2e-legacy" and .registered)' >/dev/null \
  && ok "client registered (persisted to config)" || bad "client not listed"
jq -e '.server.users[] | select(.name == "e2e-legacy")' "$WORK/srv/tailbond.json" >/dev/null && ok "user persisted in tailbond.json" || bad "user not persisted"

say "start tailbondd (client role), find the server over Tailscale, ask to pair"
cat >"$WORK/cli/tailbond.json" <<EOF
{
  "role": "client",
  "mqvpn_bin": "$MQVPN",
  "state_dir": "$WORK/cli/state",
  "hooks_dir": "$WORK/hooks",
  "listen": "$API",
  "client": {"pairing_uri": "", "paths": [], "no_dns": true, "manage_routes": false, "captive_probe_urls": ["http://10.200.9.9/generate_204"],
             "coexist": "off",
             "leases_file": "$WORK/dhcp.leases",
             "failover": {"switch_after_sec": 6, "failback_after_sec": 20, "probe_every_sec": 2}}
}
EOF
ip netns exec "$NS_C" env PATH="$WORK/ts-cli:$PATH" "$TAILBONDD" -config "$WORK/cli/tailbond.json" >"$WORK/client.log" 2>&1 &
CLI_PID=$!
wait_for 10 "client agent API" capi "http://$API/api/status" || { bad "client agent did not start"; exit 1; }
capi "http://$API/api/paths" | jq -e '[.[] | select(.name == "eth0" and .kind == "ethernet"), select(.name == "wlan0" and .kind == "wifi")] | length == 2' >/dev/null \
  && ok "interface discovery: eth0=ethernet, wlan0=wifi" || bad "discovery wrong: $(capi "http://$API/api/paths")"
HELLO="$(ip netns exec "$NS_C" curl -fsS -m 3 http://100.101.1.1:8077/api/hello || true)"
jq -e '.tailbond == true and .role == "server" and .name == "homebox"' <<<"$HELLO" >/dev/null \
  && ok "GET /api/hello on the server's Tailscale IP: $HELLO" || bad "hello: $HELLO"
say "phone page on the server's Tailscale IP, port 80 (http://homebox/)"
if wait_for 10 "phone page" ip netns exec "$NS_C" curl -fsS -m 3 -A "Mozilla/5.0 (Linux; Android 14; Pixel 8) Mobile" -o "$WORK/landing.html" http://100.101.1.1/; then
  grep -q 'Get tailbond' "$WORK/landing.html" && grep -q 'href="/tailbond.apk"' "$WORK/landing.html" \
    && grep -q 'intent://ask?server=homebox.tail1234.ts.net' "$WORK/landing.html" \
    && ok "http://100.101.1.1/ serves the phone page (app download + ask link)" || bad "phone page content: $(head -c 300 "$WORK/landing.html")"
else
  bad "nothing on 100.101.1.1:80"
fi
[ "$(ip netns exec "$NS_C" curl -s -m 3 http://100.101.1.1/tailbond.apk)" = "$(cat "$WORK/srv/tailbond.apk")" ] \
  && ok "the APK downloads from the phone page" || bad "APK not served on port 80"
# the Mac section (/#mac: same HTML, the fragment picks the section) and its
# download: 200 when the server has the bundle, 404 when it can't get it
# (this test server has no internet to fetch it from GitHub)
ip netns exec "$NS_C" curl -fsS -m 3 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15" -o "$WORK/landing-mac.html" "http://100.101.1.1/#mac" \
  && grep -q 'id="mac"' "$WORK/landing-mac.html" && grep -q 'class="plat first" id="mac"' "$WORK/landing-mac.html" \
  && grep -q 'localhost:8077/?ask=homebox.tail1234.ts.net' "$WORK/landing-mac.html" \
  && ok "/#mac: the Mac section (shown first for a Mac), with the ask link" || bad "Mac section: $(grep -c 'id=\"mac\"' "$WORK/landing-mac.html" 2>/dev/null)"
MCODE="$(ip netns exec "$NS_C" curl -s -m 20 -o /dev/null -w '%{http_code}' http://100.101.1.1/tailbond-macos-arm64.tar.gz)"
case "$MCODE" in 200|404) ok "macOS download endpoint answers ($MCODE)" ;; *) bad "macOS download: HTTP $MCODE" ;; esac
LCODE="$(ip netns exec "$NS_C" curl -s -m 3 -o /dev/null -w '%{http_code} %{redirect_url}' http://100.101.1.1/api/status)"
[ "$LCODE" = "302 http://100.101.1.1/" ] && ok "port 80 serves no API (/api/status → redirect to the page)" || bad "port 80 /api/status: $LCODE"
ip netns exec "$NS_S" curl -fsS -m 3 http://127.0.0.1:8077/api/settings | jq -e '.landing_url == "http://homebox.tail1234.ts.net/" and .landing_typed == "homebox/" and (.landing_qr_svg | startswith("<svg"))' >/dev/null \
  && ok "settings point phones at http://homebox.tail1234.ts.net/" || bad "settings landing: $(ip netns exec "$NS_S" curl -fsS -m 3 http://127.0.0.1:8077/api/settings | jq -c '{landing_url,landing_typed}')"
DISC="$(capi "http://$API/api/discover?refresh=1" || true)"
jq -e '.tailscale_running and ([.servers[] | select(.name == "homebox" and .ip == "100.101.1.1")] | length == 1) and .checked == 2' <<<"$DISC" >/dev/null \
  && ok "GET /api/discover: found homebox among 2 online peers" || bad "discover: $DISC"
capi -X POST -d '{"server":"homebox","name":"e2e-router"}' "http://$API/api/join" | jq -e '.state == "asking"' >/dev/null \
  && ok "POST /api/join homebox" || bad "join failed"
if wait_for 10 "join waiting" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/join | jq -e '.state == \"waiting\"'"; then
  ok "client waiting for approval"
else
  bad "join state: $(capi "http://$API/api/join")"
fi
REQS="$(sapi "http://$API/api/pair-requests")"
REQ_ID="$(jq -r '.[] | select(.status == "pending") | .id' <<<"$REQS" | head -1)"
jq -e '.[0] | .name == "e2e-router" and .device == "slate7" and .user == "alex@example.com" and .ip == "100.101.9.9" and .kind == "router" and (has("token") | not)' <<<"$REQS" >/dev/null \
  && ok "server lists the pending request (whois: slate7, alex@example.com)" || bad "pair-requests: $REQS"
wait_for 5 "status lists the request" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.pair_requests | length == 1'" \
  && ok "status shows 1 pending request" || bad "status pair_requests: $(sapi "http://$API/api/status" | jq -c .pair_requests)"
[ "$(ip netns exec "$NS_C" curl -s -m 3 -o /dev/null -w '%{http_code}' "http://100.101.1.1:8077/api/pair-requests/$REQ_ID")" = 404 ] \
  && ok "polling without the request token: 404" || bad "poll without token not rejected"
sapi -X POST -d '{}' "http://$API/api/pair-requests/$REQ_ID/approve" | jq -e '.client == "e2e-router"' >/dev/null \
  && ok "admin approved the request" || bad "approve failed"
if wait_for 15 "client imported the pairing" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.paired and .join.state == \"approved\"'"; then
  ok "client picked up the pairing by itself"
else
  bad "client not paired: $(capi "http://$API/api/status" | jq -c '{paired, join}')"
fi
jq -e '.server.users[] | select(.name == "e2e-router" and .ts_node == "nE2ESLATE")' "$WORK/srv/tailbond.json" >/dev/null \
  && ok "server user e2e-router bound to the Tailscale node" || bad "ts_node not stored"
URI="$(jq -r .client.pairing_uri "$WORK/cli/tailbond.json")"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
PAYLOAD="$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d)"
jq -e '.user == "e2e-router" and .server == "10.200.1.1:443"' <<<"$PAYLOAD" >/dev/null && ok "approved payload: e2e-router @ 10.200.1.1:443" || bad "payload: $PAYLOAD"

if wait_for 30 "client connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'"; then
  ok "client agent reports connected (pin check passed)"
else
  bad "client never connected"; exit 1
fi
grep -q '^Path = eth0' "$WORK/cli/state/client.conf" && grep -q '^Path = wlan0' "$WORK/cli/state/client.conf" \
  && ok "client.conf binds both paths" || bad "client.conf paths: $(grep Path "$WORK/cli/state/client.conf")"
# Speed reorders: mqvpn takes its 8-byte header off after the cap, so the
# cap is 1288 and the TUN still gets IPv6's minimum
TBMTU="$(ip -n "$NS_C" -o link show tb0 2>/dev/null | sed -n 's/.* mtu \([0-9]*\) .*/\1/p')"
[ "$TBMTU" = 1280 ] && ok "tb0 MTU 1280 in Speed (cap $(sed -n 's/^MTU = //p' "$WORK/cli/state/client.conf"))" \
  || bad "tb0 MTU is ${TBMTU:-unknown}, not 1280 (client.conf: $(grep '^MTU' "$WORK/cli/state/client.conf"))"

say "traffic through the tunnel"
if ip netns exec "$NS_C" ping -c 5 -i 0.3 -W 2 10.77.0.1 >"$WORK/ping1" 2>&1; then
  ok "ping 10.77.0.1 through tb0: $(grep -o '[0-9]*% packet loss' "$WORK/ping1")"
else
  bad "ping through tunnel failed"; cat "$WORK/ping1"
fi
TUNIP="$(capi "http://$API/api/status" | jq -r .tunnel_ip)"
[[ "$TUNIP" == 10.77.0.* ]] && ok "client tunnel_ip $TUNIP" || bad "client tunnel_ip '$TUNIP'"
[ -s "$WORK/hook-client-up.env" ] && grep -q "^TB_TUNNEL_IP=$TUNIP" "$WORK/hook-client-up.env" && grep -q '^TB_SERVER_IP=10.200.1.1' "$WORK/hook-client-up.env" \
  && ok "client-up hook ran with TB_TUNNEL_IP/TB_SERVER_IP" || bad "client-up hook env: $(cat "$WORK/hook-client-up.env" 2>/dev/null)"

say "machine list over the tunnel (GET /api/peers on 10.77.0.1:8077)"
PKEY="$(jq -r .key <<<"$PAYLOAD")"
# docs/architecture.md "Client API token": hex HMAC-SHA256(key = pairing key, "tailbond-api")
TOKEN="$(printf '%s' tailbond-api | openssl dgst -sha256 -hmac "$PKEY" -r | cut -d' ' -f1)"
PEERS_URL="http://10.77.0.1:8077/api/peers"
code() { ip netns exec "$NS_C" curl -s -m 3 -o "$WORK/peers.out" -w '%{http_code}' "$@"; }
# the agent binds the tunnel address once tb0 has it (retried every 5 s)
if wait_for 20 "agent listening on the tunnel IP" sh -c "[ \"\$(ip netns exec $NS_C curl -s -m 2 -o /dev/null -w '%{http_code}' $PEERS_URL)\" = 401 ]"; then
  ok "no token: 401 on $PEERS_URL"
else
  bad "tunnel listener: $(code "$PEERS_URL")"
fi
[ "$(code -H "Authorization: Bearer $(printf '%064d' 0)" "$PEERS_URL")" = 401 ] && ok "wrong token: 401" || bad "wrong token not rejected"
if [ "$(code -H "Authorization: Bearer $TOKEN" "$PEERS_URL")" = 200 ] &&
   jq -e '.client == "e2e-router" and .server == "homebox" and ([.peers[].hostname] | index("nas") != null) and .tailscale_running' "$WORK/peers.out" >/dev/null; then
  ok "derived token: 200, peers list (homebox's tailnet: nas, slate7)"
else
  bad "derived token: $(cat "$WORK/peers.out")"
fi
[ "$(code -H "Authorization: Bearer $TOKEN" http://10.77.0.1:8077/api/status)" = 404 ] && ok "admin API not served on the tunnel IP" || bad "tunnel IP serves /api/status"
capi "http://$API/api/peers" | jq -e '.server == "homebox" and (.peers | length) >= 1' >/dev/null \
  && ok "client agent's GET /api/peers: the server's list through the tunnel" || bad "client /api/peers: $(capi "http://$API/api/peers")"

two_paths() {
  ip netns exec "$NS_S" curl -fsS -m 2 "http://$API/api/status" |
    jq -e '.clients[] | select(.name == "e2e-router" and .connected) | [.paths[] | select(.state == "active")] | length >= 2'
}
if wait_for 20 "2 active paths on server" two_paths; then
  ok "server /api/status: e2e-router connected with 2 active paths"
else
  bad "server does not show 2 active paths: $(sapi "http://$API/api/status" | jq -c '.clients')"
fi
# generate some traffic, then check rate/RTT normalization
ip netns exec "$NS_C" ping -c 20 -i 0.05 -s 1200 -W 2 10.77.0.1 >/dev/null 2>&1 || true
SST="$(sapi "http://$API/api/status")"
jq -e '.totals.rx_bytes > 0 and .totals.tx_bytes > 0 and ([(.clients[] | select(.name == "e2e-router") | .paths)[] | select(.srtt_ms > 0)] | length) >= 1' <<<"$SST" >/dev/null \
  && ok "server totals and per-path RTT populated" || bad "server stats empty: $(jq -c '{totals, p: (.clients[] | select(.name == "e2e-router") | .paths)}' <<<"$SST")"
jq -e '[(.clients[] | select(.name == "e2e-router") | .paths)[] | select(.srtt_ms >= 30)] | length >= 1' <<<"$SST" >/dev/null \
  && ok "netem path visible (srtt >= 30ms on one path)" || soft "netem RTT not visible yet: $(jq -c '[(.clients[] | select(.name == "e2e-router") | .paths)[] | .srtt_ms]' <<<"$SST")"

say "client-side stats (needs the client control API patch)"
CST="$(capi "http://$API/api/status")"
if jq -e .control_api <<<"$CST" >/dev/null; then
  jq -e '[.paths[] | select(.state == "active" and (.iface == "eth0" or .iface == "wlan0"))] | length == 2' <<<"$CST" >/dev/null \
    && ok "client /api/status: eth0 + wlan0 active with live stats" || bad "client paths: $(jq -c .paths <<<"$CST")"
else
  soft "mqvpn client control API absent (unpatched mqvpn); client uses fallback stats"
  jq -e '[.paths[] | select(.iface == "eth0" or .iface == "wlan0")] | length == 2' <<<"$CST" >/dev/null \
    && ok "client fallback status lists eth0 + wlan0" || bad "client fallback paths: $(jq -c .paths <<<"$CST")"
fi

say "pull path A (eth0) down while pinging"
ip netns exec "$NS_C" ping -c 40 -i 0.25 -W 2 10.77.0.1 >"$WORK/ping2" 2>&1 &
PING_PID=$!
sleep 2
ip -n "$NS_C" link set eth0 down
wait "$PING_PID" || true
RECV="$(grep -o '[0-9]* received' "$WORK/ping2" | awk '{print $1}')"
echo "  ping during failover: $(grep -o '[0-9]* received.*loss' "$WORK/ping2")"
if [ "${RECV:-0}" -ge 30 ]; then ok "traffic continued over wlan0 ($RECV/40 replies)"; else bad "traffic stalled after path loss ($RECV/40)"; fi
if ip netns exec "$NS_C" ping -c 5 -i 0.3 -W 3 10.77.0.1 >/dev/null 2>&1; then ok "tunnel still up on the remaining path"; else bad "tunnel down after path loss"; fi
one_path() {
  ip netns exec "$NS_S" curl -fsS -m 2 "http://$API/api/status" |
    jq -e '.clients[] | select(.name == "e2e-router") | [.paths[] | select(.state == "active")] | length == 1'
}
wait_for 20 "server sees 1 active path" one_path && ok "server status dropped to 1 active path" || soft "server still lists 2 active paths (path timeout not reached)"

say "bring path A back"
ip -n "$NS_C" link set eth0 up
ip -n "$NS_C" route replace default via 10.200.1.1 dev eth0 metric 100
wait_for 30 "2 active paths again" two_paths && ok "path A re-joined" || soft "path A did not re-join within 30s"

say "mode switch"
capi -X POST -d '{"mode":"redundant"}' "http://$API/api/mode" | jq -e .ok >/dev/null || bad "mode POST failed"
grep -q '^Scheduler = minrtt' "$WORK/cli/state/client.conf" 2>/dev/null || sleep 3
if wait_for 30 "reconnect after mode switch" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .mode == \"redundant\"' && ip netns exec $NS_C ping -c1 -W2 10.77.0.1"; then
  grep -q '^Redundant = true' "$WORK/cli/state/client.conf" && ok "redundant mode: config regenerated, reconnected, ping ok" || bad "client.conf not regenerated for redundant"
else
  bad "no reconnect after mode switch"
fi
jq -e '.mode == "redundant"' "$WORK/cli/tailbond.json" >/dev/null && ok "mode persisted" || bad "mode not persisted"

say "logs + SSE"
sapi "http://$API/api/logs?limit=5" | jq -e '.entries | length > 0' >/dev/null && ok "/api/logs returns ring buffer" || bad "/api/logs empty"
ip netns exec "$NS_S" curl -sN -m 3 "http://$API/api/events" >"$WORK/sse" 2>&1 || true
grep -m1 '^data: {' "$WORK/sse" | sed 's/^data: //' | jq -e .role >/dev/null \
  && ok "/api/events streams status" || bad "/api/events produced no data: $(head -c 300 "$WORK/sse")"

say "DNS forwarder on the tunnel IP"
if command -v dig >/dev/null; then
  if ip netns exec "$NS_C" dig +time=2 +tries=1 @10.77.0.1 localhost A >/dev/null 2>&1; then
    ok "forwarder answers on 10.77.0.1:53"
  else soft "forwarder did not answer (no upstream resolver reachable from the server namespace?)"; fi
else
  soft "dig not installed; skipping DNS check"
fi

say "captive portal on eth0 (wan): tailbond's own probe"
path_field() { capi "http://$API/api/status" | jq -r --arg i "$1" --arg f "$2" '.paths[] | select(.iface == $i) | .[$f] // empty'; }
jq '.wan = true' "$WORK/portal-state.json" >"$WORK/ps.tmp" && mv "$WORK/ps.tmp" "$WORK/portal-state.json"
capi -X POST -d '{}' "http://$API/api/captive/check" >/dev/null
if wait_for 20 "eth0 seen behind a portal" sh -c "[ \"\$(ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -r '.paths[] | select(.iface == \"eth0\") | .captive')\" = portal ]"; then
  ok "eth0: captive=portal, portal_url=$(path_field eth0 portal_url)"
else
  bad "eth0 captive: $(capi "http://$API/api/status" | jq -c '[.paths[] | {iface, state, captive}]')"
fi
[ "$(path_field eth0 captive_source)" = tailbond ] && ok "eth0 checked by tailbond's probe (GL doesn't cover wan)" || bad "eth0 source $(path_field eth0 captive_source)"
if wait_for 30 "eth0 out of bonding" sh -c "! grep -q '^Path = eth0' $WORK/cli/state/client.conf && grep -q '^Path = wlan0' $WORK/cli/state/client.conf"; then
  ok "eth0 kept out of mqvpn's paths while captive"
else
  bad "client.conf paths: $(grep Path "$WORK/cli/state/client.conf")"
fi
wait_for 30 "tunnel back on wlan0" ip netns exec "$NS_C" ping -c1 -W2 10.77.0.1 && ok "tunnel stays up on wlan0" || bad "tunnel down while eth0 captive"
capi -X POST -d '{"name":"wan"}' "http://$API/api/captive/signin" | jq -e '.portal_url == "http://10.200.9.9/login"' >/dev/null \
  && ok "sign-in mode on (portal_url http://10.200.9.9/login)" || bad "signin failed"
wait_for 10 "portal-open hook" test -e "$WORK/hook-portal-open.done" && grep -q '^TB_PORTAL_DEV=eth0' "$WORK/hook-portal-open.env" && grep -q '^TB_PORTAL_DNS=10.200.1.1' "$WORK/hook-portal-open.env" \
  && ok "portal-open ran (eth0, WAN DNS 10.200.1.1)" || bad "portal-open env: $(cat "$WORK/hook-portal-open.env" 2>/dev/null)"
ip -n "$NS_C" rule | grep -q 'iif lan0 lookup 78' && ok "rule: LAN -> table 78" || bad "no LAN rule: $(ip -n "$NS_C" rule)"
ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1 | grep -q 'dev eth0' \
  && ok "LAN traffic goes straight out eth0 during sign-in" || bad "route: $(ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1)"
fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.200.1.1@eth0' && ! fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.77.0.1' \
  && [ "$(fuci -q get 'dhcp.@dnsmasq[0].rebind_protection')" = 0 ] \
  && ok "dnsmasq: WAN DNS bound to eth0, no tunnel DNS, rebind protection off" || bad "dnsmasq: $(fuci show dhcp)"
ip netns exec "$NS_C" curl -fsS -m 3 --interface eth0 http://10.200.9.9/login >/dev/null && ok "signed in (as a LAN laptop would)" || bad "login request failed"
if wait_for 20 "eth0 open again" sh -c "[ \"\$(ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -r '.paths[] | select(.iface == \"eth0\") | .captive')\" = open ]"; then
  ok "probe says open; sign-in mode ended by itself"
else
  bad "eth0 still: $(path_field eth0 captive)"
fi
wait_for 10 "portal-close hook" test -e "$WORK/hook-portal-close.done" && ok "portal-close ran" || bad "portal-close missing"
! ip -n "$NS_C" rule | grep -q 'lookup 78' && ok "LAN rule removed" || bad "rule left: $(ip -n "$NS_C" rule)"
# eth0 rejoining bonding restarts mqvpn: client-down (drops the tunnel DNS)
# then client-up (adds it back) may run right after portal-close, so check
# once the client hooks have caught up.
dns_restored() {
  hooks_settled && fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.77.0.1' && ! fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '@eth0' \
    && [ "$(fuci -q get 'dhcp.@dnsmasq[0].rebind_protection')" = 1 ]
}
wait_for 30 "dnsmasq restored" dns_restored && ok "dnsmasq restored" || bad "dnsmasq not restored: $(fuci show dhcp; tail -n 4 "$WORK/hook-history.log")"
wait_for 30 "eth0 back in bonding" grep -q '^Path = eth0' "$WORK/cli/state/client.conf" && ok "eth0 rejoined bonding" || bad "eth0 not rejoined"
rm -f "$WORK/hook-portal-open.env" "$WORK/hook-portal-close.env" "$WORK/hook-portal-open.done" "$WORK/hook-portal-close.done"

say "captive portal on wlan0 (wwan): GL's repeater owns it"
grep -q '^wwan /generate_204' "$WORK/portal.log" && bad "tailbond probed the GL-owned repeater" || ok "no tailbond probe over wlan0 (GL owns the repeater)"
gl_repeater '{"ssid":"Hotel-Guest","state":2,"portal":true,"portal_detecting":false,"portal_url":"http://10.200.9.9/login","bare_mode":false,"auto_portal":true}'
if wait_for 15 "wlan0 portal from GL" sh -c "[ \"\$(ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -r '.paths[] | select(.iface == \"wlan0\") | .captive_source + \" \" + .captive')\" = 'glinet portal' ]"; then
  ok "wlan0: captive=portal from GL (captive_source=glinet: the panel defers to GL's page)"
else
  bad "wlan0: $(capi "http://$API/api/status" | jq -c '.paths[] | select(.iface == "wlan0")')"
fi
wait_for 30 "wlan0 out of bonding" sh -c "! grep -q '^Path = wlan0' $WORK/cli/state/client.conf && grep -q '^Path = eth0' $WORK/cli/state/client.conf" \
  && ok "wlan0 out of bonding; eth0 carries the tunnel" || bad "paths: $(grep Path "$WORK/cli/state/client.conf")"
[ ! -e "$WORK/hook-portal-open.env" ] && ok "no sign-in bypass before GL's Login Mode" || bad "bypass opened without Login Mode"
capi -X POST -d '{"name":"wwan"}' "http://$API/api/captive/signin" >/dev/null 2>&1 || true
gl_repeater '{"ssid":"Hotel-Guest","state":2,"portal":true,"portal_detecting":false,"portal_url":"http://10.200.9.9/login","bare_mode":true,"auto_portal":true}'
wait_for 10 "portal-open (GL)" sh -c "test -e $WORK/hook-portal-open.done && grep -q '^TB_PORTAL_SOURCE=glinet' $WORK/hook-portal-open.env 2>/dev/null" \
  && grep -q '^TB_PORTAL_DEV=wlan0' "$WORK/hook-portal-open.env" && ok "GL Login Mode on: our bypass follows it (wlan0)" || bad "portal-open env: $(cat "$WORK/hook-portal-open.env" 2>/dev/null)"
ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1 | grep -q 'dev wlan0' \
  && ok "LAN reaches the hotel portal over wlan0 during GL's Login Mode" || bad "route: $(ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1)"
# GL's own check is router traffic bound to the wwan device: untouched by
# our rules (they match LAN input only) and not through tb0
ip netns exec "$NS_C" curl -fsS -m 3 --interface wlan0 http://10.200.9.9/hotspot-detect.html | grep -q Success \
  && tail -n1 "$WORK/portal.log" | grep -q '^wwan /hotspot-detect' && ok "GL-style check over wlan0 reaches the portal directly" || bad "GL check path: $(tail -n2 "$WORK/portal.log")"
ip -n "$NS_C" route get 10.200.9.9 oif wlan0 2>&1 | grep -q 'dev wlan0' && ok "router traffic bound to wlan0 not hijacked by tb0" || bad "oif route"
gl_repeater '{"ssid":"Hotel-Guest","state":2,"portal":false,"portal_detecting":false,"portal_url":"","bare_mode":false,"auto_portal":true}'
wait_for 10 "portal-close (GL)" test -e "$WORK/hook-portal-close.done" && ok "GL left Login Mode: bypass reverted" || bad "no portal-close"
wait_for 10 "LAN rule removed" sh -c "! ip -n $NS_C rule | grep -q 'lookup 78'" && ok "LAN rule removed" || bad "rule left"
wait_for 30 "wlan0 back in bonding" grep -q '^Path = wlan0' "$WORK/cli/state/client.conf" && ok "wlan0 rejoined bonding, no action in either UI" || bad "wlan0 not rejoined"
[ "$(grep -c '^wwan /generate_204' "$WORK/portal.log")" = 0 ] && ok "still no tailbond probe over wlan0" || bad "probed wlan0"
kill "$PORTAL_PID" 2>/dev/null || true

lan_curl() { # lan_curl <device> <url>: the source address the fake internet saw, or FAIL
  # names are looked up at the router's dnsmasq explicitly: the host's
  # nss-resolve (systemd-resolved) would answer for curl inside a netns
  local host url="$2" ip
  host="${url#http://}"; host="${host%%[:/]*}"
  if [[ ! "$host" =~ ^[0-9.]+$ ]]; then
    ip="$(ip netns exec "tb-e2e-$1" dig +short +time=2 +tries=1 @192.168.8.1 "$host" A 2>/dev/null | grep -E '^[0-9.]+$' | tail -1)"
    [ -n "$ip" ] || { echo FAIL; return; }
    url="${url/$host/$ip}"
  fi
  ip netns exec "tb-e2e-$1" curl -fsS -m 4 "$url" 2>/dev/null || echo FAIL
}
is_direct() { [ "$1" = 10.200.1.2 ] || [ "$1" = 10.200.2.2 ]; }
is_tunnel() { [ "$1" != FAIL ] && ! is_direct "$1"; }
# await_src <secs> <device> <url> direct|tunnel|FAIL|<ip>: retry until the
# fake internet sees what's expected (hooks run queued, a few seconds after
# the status changes); prints the last answer
await_src() {
  local end=$(( $(date +%s) + $1 )) got
  while :; do
    got="$(lan_curl "$2" "$3")"
    case "$4" in direct) is_direct "$got" && break ;; tunnel) is_tunnel "$got" && break ;; *) [ "$got" = "$4" ] && break ;; esac
    [ "$(date +%s)" -ge "$end" ] && break
    sleep 1
  done
  echo "$got"
}
cstatus() { capi "http://$API/api/status"; }
connected() { cstatus | jq -e '.state == "connected"' >/dev/null; }

say "bypass rules: three LAN devices behind the router"
wait_for 30 "client connected" connected || bad "client not connected before the bypass checks"
wait_for 30 "client hooks settled" hooks_settled || bad "client-up never caught up: $(tail -n 3 "$WORK/hook-history.log")"
SRC="$(await_src 20 phone http://other.example:8099/ tunnel)"
is_tunnel "$SRC" && ok "phone → other.example through the tunnel (the internet sees $SRC)" || {
  bad "phone → other.example: $SRC"
  echo "--- debug: LAN state"
  ip netns exec tb-e2e-phone ping -c1 -W1 192.168.8.1 | tail -1
  ip netns exec tb-e2e-phone ping -c1 -W1 198.51.100.10 | tail -1
  ip netns exec tb-e2e-phone dig +time=2 +tries=1 @192.168.8.1 other.example +short
  lan_curl phone http://198.51.100.10:8099/
  ip -n "$NS_C" neigh; ip -n "$NS_C" rule; ip -n "$NS_C" route show table 77
  ip netns exec "$NS_C" nft list ruleset | head -40
  ip netns exec "$NS_C" ss -lunp | grep ':53 ' || true
}
SRC="$(await_src 10 phone http://bank.example:8099/ tunnel)"
is_tunnel "$SRC" && ok "phone → bank.example through the tunnel before any rule ($SRC)" || bad "phone → bank.example: $SRC"
capi "http://$API/api/lan-clients" | jq -e 'length == 3 and ([.[].name] | index("living-room-tv") != null) and (.[0] | .online and .name == "pixel-8")' >/dev/null \
  && ok "GET /api/lan-clients: 3 devices with names from the DHCP leases, the phone online (neighbour table)" || bad "lan-clients: $(capi "http://$API/api/lan-clients")"
rm -f "$WORK/hook-bypass-apply.done"
BANK="$(capi -X POST -d '{"kind":"domain","match":"https://bank.example/login","target":"direct","note":"Bank"}' "http://$API/api/bypass" | jq -r .id)"
capi -X POST -d '{"kind":"device","match":"02:00:00:00:00:0b","ip":"192.168.8.20","target":"direct","note":"TV"}' "http://$API/api/bypass" >/dev/null
capi -X POST -d '{"kind":"device","match":"02:00:00:00:00:0a","target":"tunnel","note":"Work laptop"}' "http://$API/api/bypass" >/dev/null
[ "$(capi "http://$API/api/bypass" | jq '.rules | length')" = 3 ] && ok "3 rules: bank.example direct, TV direct, work laptop always tailbond" || bad "rules: $(capi "http://$API/api/bypass")"
wait_for 15 "bypass-apply" sh -c "ip netns exec $NS_C nft list set inet tailbond lan_ifs 2>/dev/null | grep -q lan0 && ip netns exec $NS_C nft list table inet tailbond | grep -q 'ether saddr 02:00:00:00:00:0a'" \
  && ok "bypass-apply loaded table inet tailbond (LAN = lan0)" || bad "nft: $(ip netns exec "$NS_C" nft list table inet tailbond 2>&1 | head -20)"
ip -n "$NS_C" rule | grep -q '^1070:.*fwmark 0x2000000/0x7000000 lookup main suppress_prefixlength 0' && ip -n "$NS_C" rule | grep -q '^1071:.*fwmark 0x2000000/0x7000000 lookup 77' \
  && ip -n "$NS_C" rule | grep -q '^1072:.*fwmark 0x1000000/0x7000000 lookup main' && ok "ip rules 1070-1072 ahead of the tunnel's 1076/1077" || bad "rules: $(ip -n "$NS_C" rule)"
grep -q '^nftset=/bank.example/4#inet#tailbond#bp_' "$WORK/dnsmasq.d/tailbond-bypass.conf" && grep -q '^server=/bank.example/10.200.1.1$' "$WORK/dnsmasq.d/tailbond-bypass.conf" \
  && ok "dnsmasq drop-in: nftset for bank.example, resolved through the WAN's DNS" || bad "dnsmasq conf: $(cat "$WORK/dnsmasq.d/tailbond-bypass.conf" 2>&1)"
capi "http://$API/api/bypass" | jq -e '.support.domains == "nftset" and .support.firewall == "nft"' >/dev/null && ok "support: websites via nftset" || bad "support: $(capi "http://$API/api/bypass" | jq -c .support)"
SRC="$(await_src 20 phone http://bank.example:8099/ 10.200.1.2)"
[ "$SRC" = 10.200.1.2 ] && ok "phone → bank.example leaves the WAN directly (the internet sees 10.200.1.2)" || bad "phone → bank.example: $SRC"
ip netns exec "$NS_C" nft list table inet tailbond | grep -A3 "set bp_${BANK}_4" | grep -q 198.51.100.20 && ok "the website's set holds 198.51.100.20 (filled by dnsmasq)" \
  || bad "set: $(ip netns exec "$NS_C" nft list table inet tailbond | grep -A3 "bp_${BANK}_4")"
SRC="$(await_src 10 phone http://other.example:8099/ tunnel)"
is_tunnel "$SRC" && ok "phone → other.example still through the tunnel ($SRC)" || bad "phone → other.example: $SRC"
SRC="$(await_src 10 tv http://other.example:8099/ direct)"
is_direct "$SRC" && ok "TV (device rule) → other.example directly ($SRC)" || bad "TV → other.example: $SRC"
SRC="$(await_src 10 laptop http://bank.example:8099/ tunnel)"
is_tunnel "$SRC" && ok "work laptop (always tailbond) → bank.example through the tunnel ($SRC): device rule beats the website rule" || bad "laptop → bank.example: $SRC"
wait_for 15 "counters" sh -c "ip netns exec $NS_C curl -fsS -m 3 http://$API/api/bypass | jq -e --arg b $BANK '.counters[\$b].connections >= 1'" \
  && ok "counters: bank.example rule matched $(capi "http://$API/api/bypass" | jq --arg b "$BANK" '.counters[$b].connections') connection(s)" \
  || { bad "counters: $(capi "http://$API/api/bypass" | jq -c .counters)"; ip netns exec "$NS_C" nft list counters table inet tailbond; grep ' bypass-apply ' "$WORK/hook-history.log" | tail -5; }

say "kill switch: tunnel down, bypassed traffic still flows"
ip netns exec "$NS_C" nft -f - <<'NFT'
table inet e2e_block {
  chain out { type filter hook output priority 0; udp dport 443 drop; }
}
NFT
ip netns exec "$NS_C" pkill -f "$WORK/cli/state/client.conf" || true
if wait_for 20 "kill switch armed" sh -c "ip -n $NS_C route show table 77 | grep -q blackhole"; then
  ok "tunnel down: table 77 is the kill switch's blackhole"
else
  bad "table 77: $(ip -n "$NS_C" route show table 77)"
fi
wait_for 20 "client-down ran" sh -c "grep -E ' client-(up|down) ' $WORK/hook-history.log | tail -1 | grep -q ' client-down ' && [ -e $WORK/hook-client-down.done ]" || bad "client-down didn't run"
SRC="$(lan_curl phone http://198.51.100.10:8099/)"
[ "$SRC" = FAIL ] && ok "phone → the internet: blocked (kill switch)" || bad "phone leaked: $SRC"
SRC="$(await_src 15 phone http://bank.example:8099/ direct)"
is_direct "$SRC" && ok "phone → bank.example still works directly, its DNS through the WAN ($SRC)" || bad "phone → bank.example with the tunnel down: $SRC"
SRC="$(await_src 10 tv http://198.51.100.10:8099/ direct)"
is_direct "$SRC" && ok "TV still online directly ($SRC)" || {
  bad "TV with the tunnel down: $SRC"
  echo "--- debug: kill switch + bypass"
  ip -n "$NS_C" rule; ip -n "$NS_C" route; ip -n "$NS_C" route show table 77
  ip -n "$NS_C" route get 198.51.100.10 from 192.168.8.20 iif lan0 mark 0x01000000
  ip netns exec "$NS_C" nft list chain inet tailbond prerouting | head -12
  ip netns exec "$NS_C" nft list counters table inet tailbond | grep -B1 packets | head -12
  ip netns exec tb-e2e-tv ping -c1 -W1 198.51.100.10 | tail -1
  ip netns exec "$NS_C" sysctl net.ipv4.conf.all.src_valid_mark || true
}
SRC="$(lan_curl laptop http://bank.example:8099/)"
[ "$SRC" = FAIL ] && ok "work laptop (always tailbond) blocked too" || bad "laptop leaked: $SRC"
ip netns exec "$NS_C" nft delete table inet e2e_block
if wait_for 60 "tunnel back" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"' && ip netns exec $NS_C ping -c1 -W2 10.77.0.1"; then
  ok "tunnel back after the block is lifted"
else
  bad "tunnel did not come back"
fi
wait_for 15 "phone through the tunnel again" sh -c "[ \"\$(ip netns exec tb-e2e-phone curl -fsS -m 3 http://198.51.100.10:8099/)\" != 10.200.1.2 ] && ip netns exec tb-e2e-phone curl -fsS -m 3 http://198.51.100.10:8099/" \
  && ok "phone back on the tunnel" || bad "phone after the block: $(lan_curl phone http://198.51.100.10:8099/)"

say "second server: a VPS, asked over the fake tailnet and added as a backup"
ip netns add "$NS_V"
ip -n "$NS_V" link set lo up
ip link add tbe2e-v type veth peer name tbe2e-vs
ip link set tbe2e-v netns "$NS_S"; ip -n "$NS_S" link set tbe2e-v name pv
ip link set tbe2e-vs netns "$NS_V"; ip -n "$NS_V" link set tbe2e-vs name eth0
ip -n "$NS_S" addr add 10.200.5.254/24 dev pv; ip -n "$NS_S" link set pv up
ip -n "$NS_V" addr add 10.200.5.1/24 dev eth0; ip -n "$NS_V" link set eth0 up
ip -n "$NS_V" route add default via 10.200.5.254
ip -n "$NS_V" addr add 100.101.5.5/32 dev lo
ip -n "$NS_S" route add 100.101.5.5/32 via 10.200.5.1
ip -n "$NS_C" route add 100.101.5.5/32 via 10.200.1.1 dev eth0 src 100.101.9.9
ip netns exec "$NS_V" sysctl -qw net.ipv4.ip_forward=1
# what the installer's firewall setup does: tunnel clients exit masqueraded
ip netns exec "$NS_V" nft -f - <<'NFT'
table ip e2e_nat {
  chain post { type nat hook postrouting priority 100; ip saddr 10.64.0.0/10 oifname "eth0" masquerade; }
}
NFT
mkdir -p "/etc/netns/$NS_V" "$WORK/ts-vps" "$WORK/vps"
echo "nameserver 10.200.1.1" >"/etc/netns/$NS_V/resolv.conf"
fake_ts "$WORK/ts-vps" '{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"vps","DNSName":"vps.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.5.5"],"Online":true},
 "Peer":{"n1":{"HostName":"slate7","DNSName":"slate7.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.9.9"],"Online":true}}}'
# a fresh server on the default subnet: the same as homebox's
cat >"$WORK/vps/tailbond.json" <<EOF
{
  "role": "server", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/vps/state", "hooks_dir": "$WORK/hooks",
  "listen": ":8077", "mode": "speed", "landing_port": -1,
  "server": {"port": 443, "public_addr": "10.200.5.1", "subnet": "10.77.0.0/24", "subnet6": "fd77:78::/112", "use_tailscale_cert": false}
}
EOF
ip netns exec "$NS_V" env PATH="$WORK/ts-vps:$PATH" "$TAILBONDD" -config "$WORK/vps/tailbond.json" >"$WORK/vps.log" 2>&1 &
VPS_PID=$!
vapi() { ip netns exec "$NS_V" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
wait_for 20 "vps up" sh -c "ip netns exec $NS_V curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .control_api'" \
  && ok "vps agent up (subnet 10.77.0.0/24, no clients yet)" || { bad "vps did not come up"; exit 1; }
capi -X POST -d '{"server":"100.101.5.5","name":"e2e-router","add":true}' "http://$API/api/join" | jq -e '.state == "asking" and .add' >/dev/null \
  && ok "POST /api/join {add: true} to the vps" || bad "join vps"
wait_for 10 "vps request" sh -c "ip netns exec $NS_V curl -fsS -m 2 http://$API/api/pair-requests | jq -e 'length == 1'" || bad "no request on the vps"
VREQ="$(vapi "http://$API/api/pair-requests" | jq -r '.[0].id')"
vapi -X POST -d '{}' "http://$API/api/pair-requests/$VREQ/approve" | jq -e '.client == "e2e-router"' >/dev/null && ok "vps admin approved" || bad "vps approve"
if wait_for 15 "vps added" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/servers | jq -e '.servers | length == 2'"; then
  ok "client lists 2 servers"
else
  bad "servers: $(capi "http://$API/api/servers") join: $(capi "http://$API/api/join")"
fi
VSUB="$(jq -r .server.subnet "$WORK/vps/tailbond.json")"
[ "$VSUB" = 10.78.0.0/24 ] && ok "the vps moved off homebox's 10.77.0.0/24 to $VSUB when asked (avoid_subnets)" || bad "vps subnet $VSUB"
wait_for 20 "vps tunnel on the new subnet" sh -c "ip netns exec $NS_V ip -4 addr show tb0 | grep -q 'inet 10.78.0.1'" && ok "vps mqvpn restarted on 10.78.0.1" || bad "vps tb0: $(ip -n "$NS_V" -4 addr show tb0 2>&1)"
SRV="$(capi "http://$API/api/servers")"
jq -e '.active_server == "homebox" and .servers[0].name == "homebox" and .servers[0].role == "primary" and .servers[1].name == "vps" and .servers[1].role == "backup" and .servers[1].subnet == "10.78.0.0/24"' <<<"$SRV" >/dev/null \
  && ok "homebox primary + active, vps backup on 10.78.0.0/24" || bad "servers: $SRV"
wait_for 15 "vps standby" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/servers | jq -e '.servers[1].state == \"standby\"'" \
  && ok "vps answers the stateless-reset probe: standby" || bad "vps state: $(capi "http://$API/api/servers" | jq -c '.servers[1]')"

say "homebox loses power (agent and mqvpn killed)"
kill -9 "$SRV_PID" 2>/dev/null || true
ip netns exec "$NS_S" pkill -9 -f "$WORK/srv/state/server.conf" || true
SRV_PID=""
T0=$(date +%s)
if wait_for 30 "switched to the vps" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.active_server == \"vps\" and .state == \"connected\"'"; then
  dt=$(( $(date +%s) - T0 ))
  [ "$dt" -le 20 ] && ok "switched to the vps and connected ${dt}s after home went dark (switch_after 6 s from the last answered packet)" || bad "switch took ${dt}s"
else
  bad "no switch: $(cstatus | jq -c '{active_server, state, failover, servers}')"
fi
ST="$(cstatus)"
wait_for 10 "homebox probed" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.servers[0].state == \"unreachable\"'" || true
wait_for 10 "the vps's tunnel address" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.tunnel_ip == \"10.78.0.2\"'" || true
ST="$(cstatus)"
jq -e '.failover.reason == "primary_down" and .servers[0].state == "unreachable" and .servers[1].state == "active" and .tunnel_ip == "10.78.0.2"' <<<"$ST" >/dev/null \
  && ok "status: reason primary_down, homebox unreachable, vps active, tunnel 10.78.0.2" || bad "status: $(jq -c '{failover, servers, tunnel_ip}' <<<"$ST")"
ip netns exec "$NS_C" ping -c 3 -i 0.3 -W 2 10.78.0.1 >/dev/null 2>&1 && ok "ping the vps through the tunnel" || bad "no ping to 10.78.0.1"
grep -q ' client-down homebox 10.77.0.0/24 10.77.0.1 switch=1' "$WORK/hook-history.log" && grep -q ' client-up vps 10.78.0.0/24 10.78.0.1 switch=0' "$WORK/hook-history.log" \
  && ok "hooks: client-down with homebox's env (TB_SWITCH=1), client-up with the vps's" || bad "hook history: $(tail -n 6 "$WORK/hook-history.log")"
fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.78.0.1' && ! fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.77.0.1' \
  && ok "dnsmasq (UCI) forwards to the vps's DNS 10.78.0.1" || bad "dnsmasq servers: $(fuci -q get 'dhcp.@dnsmasq[0].server')"
wait_for 20 "client hooks settled" hooks_settled || true
SRC="$(await_src 15 phone http://198.51.100.10:8099/ 10.200.5.1)"
[ "$SRC" = 10.200.5.1 ] && ok "phone → the internet through the vps (the internet sees 10.200.5.1)" || bad "phone via vps: $SRC"
SRC="$(await_src 20 phone http://other.example:8099/ 10.200.5.1)"
[ "$SRC" = 10.200.5.1 ] && ok "phone resolves other.example through the vps's DNS" || bad "phone other.example via vps: $SRC"
SRC="$(await_src 10 tv http://198.51.100.10:8099/ direct)"
is_direct "$SRC" && ok "bypass rules survive the switch: TV still direct ($SRC)" || bad "TV after the switch: $SRC"

say "homebox is back: fail back after the hold (20 s here, 120 s by default)"
ip netns exec "$NS_S" env PATH="$WORK/ts-srv:$PATH" "$TAILBONDD" -config "$WORK/srv/tailbond.json" >>"$WORK/server.log" 2>&1 &
SRV_PID=$!
wait_for 20 "home up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" || bad "home did not restart"
T1=$(date +%s)
wait_for 10 "home standby" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/servers | jq -e '.servers[0].state == \"standby\"'" \
  && ok "homebox answers again: standby, failback counting down ($(capi "http://$API/api/servers" | jq '.failover.failback_in_sec')s)" || bad "home state: $(capi "http://$API/api/servers" | jq -c '.servers[0]')"
sleep 8
[ "$(cstatus | jq -r .active_server)" = vps ] && ok "still on the vps 10 s later (hysteresis)" || bad "failed back too early"
if wait_for 60 "failed back" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.active_server == \"homebox\" and .state == \"connected\"'"; then
  dt=$(( $(date +%s) - T1 ))
  [ "$dt" -ge 18 ] && ok "failed back to homebox ${dt}s after it came back" || bad "failback after only ${dt}s"
else
  bad "no failback: $(cstatus | jq -c '{active_server, state, failover, servers}')"
fi
wait_for 10 "homebox's tunnel address" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.tunnel_ip | startswith(\"10.77.0.\")'" || true
cstatus | jq -e '.failover.reason == "" and .servers[1].state != "active" and (.tunnel_ip | startswith("10.77.0."))' >/dev/null \
  && ok "back on homebox (10.77.0.x), vps on standby" || bad "status: $(cstatus | jq -c '{failover, servers, tunnel_ip}')"
wait_for 10 "ping home" ip netns exec "$NS_C" ping -c1 -W2 10.77.0.1 && ok "ping homebox through the tunnel" || bad "no ping to 10.77.0.1"
wait_for 10 "phone through home" sh -c "ip netns exec tb-e2e-phone curl -fsS -m 3 http://198.51.100.10:8099/ | grep -qv '^10.200.5.1$'" \
  && ok "phone's traffic back through homebox" || bad "phone after failback: $(lan_curl phone http://198.51.100.10:8099/)"
[ "$(grep -c ' client-up ' "$WORK/hook-history.log")" -ge 3 ] && ok "client-up ran for homebox, vps, homebox" || bad "hook history: $(grep client- "$WORK/hook-history.log")"
capi -X DELETE "http://$API/api/bypass/$BANK" | jq -e .ok >/dev/null && wait_for 10 "drop-in updated" sh -c "! grep -q bank.example $WORK/dnsmasq.d/tailbond-bypass.conf" \
  && ok "deleting the website rule removes it from dnsmasq" || bad "delete rule"

say "Linux client next to NetBird: wt0 owns 100.92.0.0/16; only /32s for the tailnet, split DNS on tb0, NetBird's endpoints direct"
# the router-sim becomes a plain Linux laptop: no router hooks, Tailscale
# off here, DNS on, coexistence on. Kill switch off: mqvpn's own iptables
# kill switch doesn't come up in this runner's namespaces; the allowances
# tailbond adds with it are unit tested (internal/coexist)
kill "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true
mkdir -p "$WORK/hooks-none" "$WORK/cx-bin"
ip -n "$NS_C" link add wt0 type dummy
ip -n "$NS_C" addr add 100.92.1.2/16 dev wt0
ip -n "$NS_C" link set wt0 up
cat >"$WORK/cx-bin/netbird" <<'NB'
#!/bin/sh
# netbird CLI stand-in: `netbird status --json` of a logged-in NetBird
[ "$1 $2" = "status --json" ] || exit 0
cat <<'J'
{"peers":{"total":1,"connected":1,"details":[{"fqdn":"laptop.netbird.cloud","netbirdIp":"100.92.4.9","status":"Connected",
  "iceCandidateEndpoint":{"local":"10.200.1.2:51820","remote":"198.51.100.50:51820"},"relayAddress":""}]},
 "management":{"url":"https://192.0.2.10:443","connected":true},"signal":{"url":"https://192.0.2.11:443","connected":true},
 "relays":{"total":1,"available":1,"details":[{"uri":"rels://198.51.100.77:443","available":true}]},
 "netbirdIp":"100.92.1.2/16","networks":[],"dnsServers":[{"servers":["100.92.255.254:53"],"domains":["corp.example"],"enabled":true}]}
J
NB
cat >"$WORK/cx-bin/resolvectl" <<RV
#!/bin/sh
# resolvectl stand-in: the netns shares the host's systemd-resolved; record instead
echo "\$*" >>"$WORK/resolvectl.log"
RV
chmod +x "$WORK/cx-bin/netbird" "$WORK/cx-bin/resolvectl"
: >"$WORK/resolvectl.log"
jq --arg h "$WORK/hooks-none" '.hooks_dir = $h | .disable_tailscale = true | .client.no_dns = false | .client.kill_switch = false | del(.client.coexist)' \
  "$WORK/cli/tailbond.json" >"$WORK/cli/tb.tmp" && mv "$WORK/cli/tb.tmp" "$WORK/cli/tailbond.json"
start_laptop() {
  ip netns exec "$NS_C" env PATH="$WORK/cx-bin:$PATH" "$TAILBONDD" -config "$WORK/cli/tailbond.json" >>"$WORK/client.log" 2>&1 &
  CLI_PID=$!
}
start_laptop
cx() { capi "http://$API/api/coexist"; }
wait_for 60 "client connected again" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "the Linux client connected next to wt0" || bad "not connected: $(cstatus | jq -c '{state, error}')"
routes_on() { ip -n "$NS_C" route show dev "$1" 2>/dev/null; }
if wait_for 30 "tailnet /32s on tb0" sh -c "ip -n $NS_C route show dev tb0 | grep -q '^100.92.5.5 '"; then
  ok "tailnet machines by /32 on tb0: $(routes_on tb0 | awk '{print $1}' | tr '\n' ' ')"
else
  bad "no tailnet /32s: tb0 has $(routes_on tb0 | tr '\n' ';'); coexist $(cx | jq -c .)"
fi
routes_on tb0 | grep -Eq '^100\.(64\.0\.0/10|92\.0\.0/16) ' && bad "a covering route on tb0: $(routes_on tb0 | tr '\n' ';')" \
  || ok "nothing on tb0 covers wt0's 100.92.0.0/16"
[ "$(ip -n "$NS_C" route get 100.92.5.5 | grep -o 'dev [^ ]*')" = "dev tb0" ] && [ "$(ip -n "$NS_C" route get 100.92.9.9 | grep -o 'dev [^ ]*')" = "dev wt0" ] \
  && ok "100.92.5.5 (tailnet) via tb0, the rest of 100.92/16 via wt0" || bad "route get: $(ip -n "$NS_C" route get 100.92.5.5); $(ip -n "$NS_C" route get 100.92.9.9)"
grep -q 'domain tb0 ~ts.net' "$WORK/resolvectl.log" && grep -q 'default-route tb0 false' "$WORK/resolvectl.log" && ! grep -qE 'domain (eth0|wlan0|wt0)' "$WORK/resolvectl.log" \
  && ok "split DNS on tb0 only: $(tr '\n' ';' <"$WORK/resolvectl.log")" || bad "resolvectl: $(tr '\n' ';' <"$WORK/resolvectl.log")"
grep -Eq '^DNS *=' "$WORK/cli/state/client.conf" \
  && bad "mqvpn got DNS servers (global DNS): $(grep -E '^DNS' "$WORK/cli/state/client.conf")" || ok "mqvpn runs without DNS servers (no global DNS takeover)"
via() { ip -n "$NS_C" route get "$1" | grep -o 'dev [^ ]*' | head -n1; }
if wait_for 20 "NetBird endpoints direct" sh -c "ip -n $NS_C route show 198.51.100.50 | grep -q 'via 10.200.1.1'"; then
  ok "Direct: NetBird's endpoints via the uplink: 198.51.100.50 $(via 198.51.100.50), 198.51.100.77 $(via 198.51.100.77), 192.0.2.10 $(via 192.0.2.10)"
else
  bad "NetBird endpoints: $(ip -n "$NS_C" route show | grep -E '198.51|192.0.2' | tr '\n' ';')"
fi
ip netns exec "$NS_C" iptables -S OUTPUT 2>/dev/null | grep -q tailbond-coexist \
  && bad "kill-switch allowances added with the kill switch off" || ok "kill switch off: no iptables allowances"
cx | jq -e '.foreign | any(.iface == "wt0" and .kind == "netbird")' >/dev/null && cx | jq -e '.work_vpn == "direct" and .split_dns' >/dev/null \
  && ok "API: NetBird on wt0 detected, Work VPN traffic Direct, split DNS" || bad "API: $(cx | jq -c .)"

say "Bonded: NetBird's traffic through tailbond; the MTU rule"
ip -n "$NS_C" link set wt0 mtu 1280
capi -X POST -d '{"work_vpn": "bonded"}' "http://$API/api/coexist" >/dev/null
# 1360, less the 8-byte reorder header when the mode negotiates reordering
if wait_for 60 "tb0 MTU 1360" sh -c "ip -n $NS_C link show tb0 | grep -Eq 'mtu 13(52|60) '"; then
  ok "tailbond's MTU raised for wt0 1280 + 80: $(ip -n "$NS_C" link show tb0 | grep -o 'mtu [0-9]*') (want_mtu $(cx | jq .want_mtu))"
else
  bad "tb0 MTU: $(ip -n "$NS_C" link show tb0 | grep -o 'mtu [0-9]*'); $(cx | jq -c '{work_vpn_effective, want_mtu, bonded_blocked, warnings}')"
fi
wait_for 30 "endpoints no longer direct" sh -c "! ip -n $NS_C route show 198.51.100.50 | grep -q 'via 10.200.1.1'" \
  && ok "Bonded: NetBird's endpoints follow tailbond's tunnel ($(via 198.51.100.50))" || bad "still direct: $(ip -n "$NS_C" route show 198.51.100.50)"
ip -n "$NS_C" link set wt0 mtu 1420
if wait_for 45 "Bonded can't fit" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/coexist | jq -e '.work_vpn_effective == \"direct\" and ((.bonded_blocked // \"\") | length) > 0'"; then
  ok "wt0 at 1420 needs 1500 > the uplink allows: back to Direct, warned: $(cx | jq -r '.warnings[0]' | cut -c1-110)"
else
  bad "no fallback: $(cx | jq -c '{work_vpn_effective, want_mtu, bonded_blocked, warnings}')"
fi
capi -X POST -d '{"work_vpn": "direct"}' "http://$API/api/coexist" >/dev/null

say "stop and crash: routes, DNS and kill-switch allowances go"
kill "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true
: >"$WORK/resolvectl.log"
start_laptop
wait_for 60 "connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" || true
wait_for 30 "direct routes back" sh -c "ip -n $NS_C route show 198.51.100.50 | grep -q 'via 10.200.1.1'" || true
kill -9 "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true
pkill -9 -f "mqvpn.*$WORK/cli" 2>/dev/null || true
sleep 1
ip -n "$NS_C" route show 198.51.100.50 | grep -q 'via 10.200.1.1' && ok "after kill -9 the direct routes are still there (nobody to take them away yet)" \
  || bad "direct routes gone before the restart?"
jq '.client.disconnected = true' "$WORK/cli/tailbond.json" >"$WORK/cli/tb.tmp" && mv "$WORK/cli/tb.tmp" "$WORK/cli/tailbond.json"
N0="$(grep -c 'took away the routes and resolver files' "$WORK/client.log")"
start_laptop
# a new line (the log is appended: earlier stops logged it too)
if wait_for 20 "restored at start" sh -c "[ \$(grep -c 'took away the routes and resolver files' '$WORK/client.log') -gt $N0 ]"; then
  ! ip -n "$NS_C" route show 198.51.100.50 | grep -q . && grep -q 'revert tb0' "$WORK/resolvectl.log" \
    && ok "restart after kill -9 took them away (direct routes; resolvectl revert tb0)" \
    || bad "left after restart: $(ip -n "$NS_C" route show | grep -E '198.51|192.0.2' | tr '\n' ';') $(ip netns exec "$NS_C" iptables -S OUTPUT | grep tailbond)"
else
  bad "no restore at start: $(tail -n 5 "$WORK/client.log" | tr '\n' ' ')"
fi
kill "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true
# the killed mqvpn's own kill-switch rules (not tailbond's): gone before the
# router simulation carries on
ip netns exec "$NS_C" iptables -F OUTPUT
ip -n "$NS_C" link del wt0
jq '.client.disconnected = false | .client.coexist = "off" | .client.kill_switch = false | .client.no_dns = true | .disable_tailscale = false' "$WORK/cli/tailbond.json" >"$WORK/cli/tb.tmp" \
  && mv "$WORK/cli/tb.tmp" "$WORK/cli/tailbond.json"
jq --arg h "$WORK/hooks" '.hooks_dir = $h' "$WORK/cli/tailbond.json" >"$WORK/cli/tb.tmp" && mv "$WORK/cli/tb.tmp" "$WORK/cli/tailbond.json"
ip netns exec "$NS_C" env PATH="$WORK/ts-cli:$PATH" "$TAILBONDD" -config "$WORK/cli/tailbond.json" >>"$WORK/client.log" 2>&1 &
CLI_PID=$!
wait_for 60 "router-sim connected again" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "back to the router simulation" || bad "router-sim not connected again"

say "revoke"
sapi -X DELETE "http://$API/api/clients/e2e-router" | jq -e .ok >/dev/null && ok "DELETE /api/clients/e2e-router" || bad "revoke failed"
jq -e '[.server.users[]? | select(.name == "e2e-router")] | length == 0' "$WORK/srv/tailbond.json" >/dev/null && ok "user removed from config" || bad "user still in config"

say "shutdown hooks"
kill "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true; CLI_PID=""
kill "$SRV_PID"; wait "$SRV_PID" 2>/dev/null || true; SRV_PID=""
kill "$VPS_PID"; wait "$VPS_PID" 2>/dev/null || true; VPS_PID=""
grep -q ' bypass-apply .* clear' "$WORK/hook-history.log" && ! ip netns exec "$NS_C" nft list table inet tailbond >/dev/null 2>&1 \
  && ok "client stop clears the bypass rules" || bad "bypass rules left after stop"
[ -s "$WORK/hook-server-down.env" ] && ok "server-down hook ran on stop" || bad "server-down hook did not run"

echo
printf '\033[1mresult: %d passed, %d failed, %d skipped\033[0m\n' "$PASS" "$FAIL" "$SOFT"
[ "$FAIL" -eq 0 ]
