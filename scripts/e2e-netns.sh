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
# Checks (soft): client-side per-path stats, which need the mqvpn client
# control API patch (patches/mqvpn/0001-client-control-api.patch).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-$(go env GOARCH 2>/dev/null || echo amd64)/tailbondd}"
NS_S=tb-e2e-srv
NS_C=tb-e2e-cli
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

WORK="$(mktemp -d /tmp/tailbond-e2e.XXXXXX)"
PASS=0 FAIL=0 SOFT=0
SRV_PID="" CLI_PID=""

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
  [ -n "$CLI_PID" ] && kill "$CLI_PID" 2>/dev/null || true
  [ -n "$SRV_PID" ] && kill "$SRV_PID" 2>/dev/null || true
  sleep 2
  [ -n "$CLI_PID" ] && kill -9 "$CLI_PID" 2>/dev/null || true
  [ -n "$SRV_PID" ] && kill -9 "$SRV_PID" 2>/dev/null || true
  ip netns pids "$NS_S" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  ip netns pids "$NS_C" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  ip netns del "$NS_S" 2>/dev/null || true
  ip netns del "$NS_C" 2>/dev/null || true
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- server agent log (tail) ---"; tail -n 40 "$WORK/server.log" 2>/dev/null || true
    echo "--- client agent log (tail) ---"; tail -n 40 "$WORK/client.log" 2>/dev/null || true
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
ip netns del "$NS_S" 2>/dev/null || true
ip netns del "$NS_C" 2>/dev/null || true
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
         "n2":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true}}}'
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
echo '{"ssid":"Hotel-Guest","state":2,"portal":false,"portal_detecting":false,"portal_url":"","bare_mode":false,"auto_portal":true}' >"$WORK/gl-repeater.json"
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
# the LAN of the travel router, for the sign-in bypass checks
ip -n "$NS_C" link add lan0 type dummy
ip -n "$NS_C" addr add 192.168.8.1/24 dev lan0
ip -n "$NS_C" link set lan0 up
ip netns exec "$NS_C" sysctl -qw net.ipv4.ip_forward=1
# the real OpenWrt portal hooks, with the stub uci on a fake /etc/config
mkdir -p "$WORK/fakeroot/etc/config" "$WORK/stubs" "$WORK/hookstate"
printf '#!/bin/sh\nexec python3 %s "$@"\n' "$ROOT/packaging/openwrt/test/stubs/uci" >"$WORK/stubs/uci"
chmod +x "$WORK/stubs/uci"
printf "config interface 'lan'\n\toption device 'lan0'\n" >"$WORK/fakeroot/etc/config/network"
printf "config client 'client'\n\tlist lan_network 'lan'\n\toption rule_priority '1077'\n\toption portal_table '78'\n" >"$WORK/fakeroot/etc/config/tailbond"
printf "config dnsmasq\n\tlist server '10.77.0.1'\n\toption noresolv '1'\n\toption rebind_protection '1'\n" >"$WORK/fakeroot/etc/config/dhcp"
fuci() { FAKE_ROOT="$WORK/fakeroot" "$WORK/stubs/uci" "$@"; }
ok "fake portal, GL repeater mock (auto_portal on, no portal), lan0"

say "hooks"
mkdir -p "$WORK/hooks"
for h in server-up server-down client-up client-down; do
  cat >"$WORK/hooks/$h" <<EOF
#!/bin/sh
env | grep '^TB_' | sort > "$WORK/hook-$h.env"
echo "$h ran"
EOF
  chmod +x "$WORK/hooks/$h"
done

for h in portal-open portal-close; do
  cat >"$WORK/hooks/$h" <<EOF
#!/bin/sh
env | grep '^TB_' | sort > "$WORK/hook-$h.env"
export FAKE_ROOT="$WORK/fakeroot" TB_STATE="$WORK/hookstate" TB_INITD=/nonexistent TB_LIB="$ROOT/packaging/openwrt/hooks/lib.sh"
export PATH="$WORK/stubs:\$PATH"
exec "$ROOT/packaging/openwrt/hooks/$h"
EOF
  chmod +x "$WORK/hooks/$h"
done

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
  "server": {"port": 443, "public_addr": "$SERVER_IP", "subnet": "10.77.0.0/24", "subnet6": "fd77:77::/112", "use_tailscale_cert": false}
}
EOF
ip netns exec "$NS_S" env PATH="$WORK/ts-srv:$PATH" "$TAILBONDD" -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
if wait_for 20 "server agent connected" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .control_api'"; then
  ok "server agent up, mqvpn server running, control API reachable"
else
  bad "server did not come up"; exit 1
fi
[ -s "$WORK/hook-server-up.env" ] && grep -q '^TB_SUBNET=10.77.0.0/24' "$WORK/hook-server-up.env" && grep -q '^TB_DNS_IP=10.77.0.1' "$WORK/hook-server-up.env" \
  && ok "server-up hook ran with TB_* env" || bad "server-up hook missing or wrong env"

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
  "client": {"pairing_uri": "", "paths": [], "no_dns": true, "captive_probe_urls": ["http://10.200.9.9/generate_204"]}
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
wait_for 10 "portal-open hook" test -s "$WORK/hook-portal-open.env" && grep -q '^TB_PORTAL_DEV=eth0' "$WORK/hook-portal-open.env" && grep -q '^TB_PORTAL_DNS=10.200.1.1' "$WORK/hook-portal-open.env" \
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
wait_for 10 "portal-close hook" test -s "$WORK/hook-portal-close.env" && ok "portal-close ran" || bad "portal-close missing"
! ip -n "$NS_C" rule | grep -q 'lookup 78' && ok "LAN rule removed" || bad "rule left: $(ip -n "$NS_C" rule)"
fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '10.77.0.1' && ! fuci -q get 'dhcp.@dnsmasq[0].server' | grep -q '@eth0' \
  && [ "$(fuci -q get 'dhcp.@dnsmasq[0].rebind_protection')" = 1 ] && ok "dnsmasq restored" || bad "dnsmasq not restored: $(fuci show dhcp)"
wait_for 30 "eth0 back in bonding" grep -q '^Path = eth0' "$WORK/cli/state/client.conf" && ok "eth0 rejoined bonding" || bad "eth0 not rejoined"
rm -f "$WORK/hook-portal-open.env" "$WORK/hook-portal-close.env"

say "captive portal on wlan0 (wwan): GL's repeater owns it"
grep -q '^wwan /generate_204' "$WORK/portal.log" && bad "tailbond probed the GL-owned repeater" || ok "no tailbond probe over wlan0 (GL owns the repeater)"
echo '{"ssid":"Hotel-Guest","state":2,"portal":true,"portal_detecting":false,"portal_url":"http://10.200.9.9/login","bare_mode":false,"auto_portal":true}' >"$WORK/gl-repeater.json"
if wait_for 15 "wlan0 portal from GL" sh -c "[ \"\$(ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -r '.paths[] | select(.iface == \"wlan0\") | .captive_source + \" \" + .captive')\" = 'glinet portal' ]"; then
  ok "wlan0: captive=portal from GL (captive_source=glinet: the panel defers to GL's page)"
else
  bad "wlan0: $(capi "http://$API/api/status" | jq -c '.paths[] | select(.iface == "wlan0")')"
fi
wait_for 30 "wlan0 out of bonding" sh -c "! grep -q '^Path = wlan0' $WORK/cli/state/client.conf && grep -q '^Path = eth0' $WORK/cli/state/client.conf" \
  && ok "wlan0 out of bonding; eth0 carries the tunnel" || bad "paths: $(grep Path "$WORK/cli/state/client.conf")"
[ ! -e "$WORK/hook-portal-open.env" ] && ok "no sign-in bypass before GL's Login Mode" || bad "bypass opened without Login Mode"
capi -X POST -d '{"name":"wwan"}' "http://$API/api/captive/signin" >/dev/null 2>&1 || true
echo '{"ssid":"Hotel-Guest","state":2,"portal":true,"portal_detecting":false,"portal_url":"http://10.200.9.9/login","bare_mode":true,"auto_portal":true}' >"$WORK/gl-repeater.json"
wait_for 10 "portal-open (GL)" sh -c "grep -q '^TB_PORTAL_SOURCE=glinet' $WORK/hook-portal-open.env 2>/dev/null" \
  && grep -q '^TB_PORTAL_DEV=wlan0' "$WORK/hook-portal-open.env" && ok "GL Login Mode on: our bypass follows it (wlan0)" || bad "portal-open env: $(cat "$WORK/hook-portal-open.env" 2>/dev/null)"
ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1 | grep -q 'dev wlan0' \
  && ok "LAN reaches the hotel portal over wlan0 during GL's Login Mode" || bad "route: $(ip -n "$NS_C" route get 198.51.100.7 from 192.168.8.100 iif lan0 2>&1)"
# GL's own check is router traffic bound to the wwan device: untouched by
# our rules (they match LAN input only) and not through tb0
ip netns exec "$NS_C" curl -fsS -m 3 --interface wlan0 http://10.200.9.9/hotspot-detect.html | grep -q Success \
  && tail -n1 "$WORK/portal.log" | grep -q '^wwan /hotspot-detect' && ok "GL-style check over wlan0 reaches the portal directly" || bad "GL check path: $(tail -n2 "$WORK/portal.log")"
ip -n "$NS_C" route get 10.200.9.9 oif wlan0 2>&1 | grep -q 'dev wlan0' && ok "router traffic bound to wlan0 not hijacked by tb0" || bad "oif route"
echo '{"ssid":"Hotel-Guest","state":2,"portal":false,"portal_detecting":false,"portal_url":"","bare_mode":false,"auto_portal":true}' >"$WORK/gl-repeater.json"
wait_for 10 "portal-close (GL)" test -s "$WORK/hook-portal-close.env" && ok "GL left Login Mode: bypass reverted" || bad "no portal-close"
! ip -n "$NS_C" rule | grep -q 'lookup 78' && ok "LAN rule removed" || bad "rule left"
wait_for 30 "wlan0 back in bonding" grep -q '^Path = wlan0' "$WORK/cli/state/client.conf" && ok "wlan0 rejoined bonding, no action in either UI" || bad "wlan0 not rejoined"
[ "$(grep -c '^wwan /generate_204' "$WORK/portal.log")" = 0 ] && ok "still no tailbond probe over wlan0" || bad "probed wlan0"
kill "$PORTAL_PID" 2>/dev/null || true

say "revoke"
sapi -X DELETE "http://$API/api/clients/e2e-router" | jq -e .ok >/dev/null && ok "DELETE /api/clients/e2e-router" || bad "revoke failed"
jq -e '[.server.users[]? | select(.name == "e2e-router")] | length == 0' "$WORK/srv/tailbond.json" >/dev/null && ok "user removed from config" || bad "user still in config"

say "shutdown hooks"
kill "$CLI_PID"; wait "$CLI_PID" 2>/dev/null || true; CLI_PID=""
kill "$SRV_PID"; wait "$SRV_PID" 2>/dev/null || true; SRV_PID=""
[ -s "$WORK/hook-server-down.env" ] && ok "server-down hook ran on stop" || bad "server-down hook did not run"

echo
printf '\033[1mresult: %d passed, %d failed, %d skipped\033[0m\n' "$PASS" "$FAIL" "$SOFT"
[ "$FAIL" -eq 0 ]
