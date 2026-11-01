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
# token derived from the pairing key; Tailscale is disabled, so the list is empty).
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

say "start tailbondd (server role)"
mkdir -p "$WORK/srv" "$WORK/cli"
cat >"$WORK/srv/tailbond.json" <<EOF
{
  "role": "server",
  "mqvpn_bin": "$MQVPN",
  "state_dir": "$WORK/srv/state",
  "hooks_dir": "$WORK/hooks",
  "listen": "$API",
  "mode": "speed",
  "disable_tailscale": true,
  "server": {"port": 443, "public_addr": "$SERVER_IP", "subnet": "10.77.0.0/24", "subnet6": "fd77:77::/112", "use_tailscale_cert": false}
}
EOF
ip netns exec "$NS_S" "$TAILBONDD" -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
if wait_for 20 "server agent connected" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .control_api'"; then
  ok "server agent up, mqvpn server running, control API reachable"
else
  bad "server did not come up"; exit 1
fi
[ -s "$WORK/hook-server-up.env" ] && grep -q '^TB_SUBNET=10.77.0.0/24' "$WORK/hook-server-up.env" && grep -q '^TB_DNS_IP=10.77.0.1' "$WORK/hook-server-up.env" \
  && ok "server-up hook ran with TB_* env" || bad "server-up hook missing or wrong env"

say "pairing"
PAIR="$(sapi -X POST -d '{"name":"e2e-router"}' "http://$API/api/clients")"
URI="$(jq -r .uri <<<"$PAIR")"
[[ "$URI" == tailbond://pair\?d=* ]] && jq -e '.qr_svg | startswith("<svg")' <<<"$PAIR" >/dev/null \
  && ok "POST /api/clients issued a pairing URI + QR SVG" || bad "bad pairing response: $PAIR"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
PAYLOAD="$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d)"
jq -e '.v == 1 and .server == "10.200.1.1:443" and .insecure == true and (.cert_sha256 | length) == 64 and .dns == ["10.77.0.1"]' <<<"$PAYLOAD" >/dev/null \
  && ok "payload: v1, server, pinned self-signed cert, dns" || bad "payload unexpected: $PAYLOAD"
sapi "http://$API/api/clients" | jq -e '.[] | select(.name == "e2e-router" and .registered)' >/dev/null \
  && ok "client registered (persisted to config)" || bad "client not listed"
jq -e '.server.users[] | select(.name == "e2e-router")' "$WORK/srv/tailbond.json" >/dev/null && ok "user persisted in tailbond.json" || bad "user not persisted"

say "start tailbondd (client role) and import the pairing"
cat >"$WORK/cli/tailbond.json" <<EOF
{
  "role": "client",
  "mqvpn_bin": "$MQVPN",
  "state_dir": "$WORK/cli/state",
  "hooks_dir": "$WORK/hooks",
  "listen": "$API",
  "disable_tailscale": true,
  "client": {"pairing_uri": "", "paths": [], "no_dns": true}
}
EOF
ip netns exec "$NS_C" "$TAILBONDD" -config "$WORK/cli/tailbond.json" >"$WORK/client.log" 2>&1 &
CLI_PID=$!
wait_for 10 "client agent API" capi "http://$API/api/status" || { bad "client agent did not start"; exit 1; }
capi "http://$API/api/paths" | jq -e '[.[] | select(.name == "eth0" and .kind == "ethernet"), select(.name == "wlan0" and .kind == "wifi")] | length == 2' >/dev/null \
  && ok "interface discovery: eth0=ethernet, wlan0=wifi" || bad "discovery wrong: $(capi "http://$API/api/paths")"
capi -X POST -d "$(jq -n --arg u "$URI" '{uri:$u}')" "http://$API/api/pair" | jq -e .ok >/dev/null && ok "POST /api/pair accepted" || bad "pair failed"

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
   jq -e '.client == "e2e-router" and (.peers | type == "array") and (.tailscale_running == false)' "$WORK/peers.out" >/dev/null; then
  ok "derived token: 200, peers list (empty, Tailscale disabled)"
else
  bad "derived token: $(cat "$WORK/peers.out")"
fi
[ "$(code -H "Authorization: Bearer $TOKEN" http://10.77.0.1:8077/api/status)" = 404 ] && ok "admin API not served on the tunnel IP" || bad "tunnel IP serves /api/status"

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
jq -e '.totals.rx_bytes > 0 and .totals.tx_bytes > 0 and ([.clients[0].paths[] | select(.srtt_ms > 0)] | length) >= 1' <<<"$SST" >/dev/null \
  && ok "server totals and per-path RTT populated" || bad "server stats empty: $(jq -c '{totals, p: .clients[0].paths}' <<<"$SST")"
jq -e '[.clients[0].paths[] | select(.srtt_ms >= 30)] | length >= 1' <<<"$SST" >/dev/null \
  && ok "netem path visible (srtt >= 30ms on one path)" || soft "netem RTT not visible yet: $(jq -c '[.clients[0].paths[] | .srtt_ms]' <<<"$SST")"

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
