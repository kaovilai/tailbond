#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Smoke test for patches/mqvpn/0001-client-control-api.patch: a server and a
# two-path client in network namespaces; the client's control API must answer
# get_status with both paths and their bound interface names. Needs root.
#
#   sudo scripts/smoke-client-control.sh [path/to/mqvpn]   (default out/native/mqvpn)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${1:-$ROOT/out/native/mqvpn}"
NS_S=tb-smoke-srv
NS_C=tb-smoke-cli
CTRL_PORT=9077
WORK="$(mktemp -d)"

die() { echo "smoke: $*" >&2; exit 1; }
[ "$(id -u)" = 0 ] || die "run as root (network namespaces)"
[ -x "$MQVPN" ] || die "$MQVPN not built (scripts/build-mqvpn-native.sh)"
command -v openssl >/dev/null || die "openssl not found"

cleanup() {
    [ -n "${CPID:-}" ] && kill "$CPID" 2>/dev/null || true
    [ -n "${SPID:-}" ] && kill "$SPID" 2>/dev/null || true
    wait 2>/dev/null || true
    ip netns del "$NS_C" 2>/dev/null || true
    ip netns del "$NS_S" 2>/dev/null || true
    rm -rf "$WORK"
}
trap cleanup EXIT

# One JSON request -> response (Debian nc needs -q1; nmap ncat half-closes).
query() {
    local ns="$1" port="$2" req="$3"
    if nc -h 2>&1 | grep -q -- '-q'; then
        echo "$req" | ip netns exec "$ns" nc -q1 127.0.0.1 "$port"
    else
        echo "$req" | ip netns exec "$ns" nc 127.0.0.1 "$port"
    fi
}

ip netns add "$NS_S"
ip netns add "$NS_C"
ip -n "$NS_S" link set lo up
ip -n "$NS_C" link set lo up
i=0
for l in a b; do
    i=$((i + 1))
    ip link add "veth-${l}0" netns "$NS_C" type veth peer name "veth-${l}1" netns "$NS_S"
    ip -n "$NS_C" addr add "10.10${i}.0.2/24" dev "veth-${l}0"
    ip -n "$NS_S" addr add "10.10${i}.0.1/24" dev "veth-${l}1"
    ip -n "$NS_C" link set "veth-${l}0" up
    ip -n "$NS_S" link set "veth-${l}1" up
done
# Server reachable from path b too (as a second WAN would see it).
ip -n "$NS_C" route add 10.101.0.1/32 dev veth-b0 metric 200 2>/dev/null || true

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 1 \
    -subj /CN=tailbond-smoke -keyout "$WORK/k.pem" -out "$WORK/c.pem" 2>/dev/null
PSK="$("$MQVPN" --genkey 2>/dev/null | tail -1)"

ip netns exec "$NS_S" "$MQVPN" --mode server --listen 0.0.0.0:4433 \
    --subnet 10.77.0.0/24 --tun-name tb0 --cert "$WORK/c.pem" --key "$WORK/k.pem" \
    --auth-key "$PSK" --control-port 9090 >"$WORK/server.log" 2>&1 &
SPID=$!
sleep 1

ip netns exec "$NS_C" "$MQVPN" --mode client --server 10.101.0.1:4433 --insecure \
    --auth-key "$PSK" --tun-name tb0 --path veth-a0 --path veth-b0 \
    --no-manage-routes --control-port "$CTRL_PORT" >"$WORK/client.log" 2>&1 &
CPID=$!

status=""
for _ in $(seq 1 20); do
    sleep 1
    status="$(query "$NS_C" "$CTRL_PORT" '{"cmd":"get_status"}' 2>/dev/null || true)"
    case "$status" in
        *'"state":"established"'*'"iface":"veth-a0"'*'"iface":"veth-b0"'*'"status":"active"'*) break ;;
    esac
done

echo "client get_status: $status"
echo "client get_stats:  $(query "$NS_C" "$CTRL_PORT" '{"cmd":"get_stats"}' || true)"
echo "client list_users: $(query "$NS_C" "$CTRL_PORT" '{"cmd":"list_users"}' || true)"
echo "server get_status: $(query "$NS_S" 9090 '{"cmd":"get_status"}' || true)"

fail=0
for want in '"state":"established"' '"iface":"veth-a0"' '"iface":"veth-b0"' \
    '"min_rtt_ms"' '"pkt_lost"' '"n_clients":1'; do
    case "$status" in *"$want"*) ;; *) echo "MISSING: $want"; fail=1 ;; esac
done
case "$(query "$NS_C" "$CTRL_PORT" '{"cmd":"list_users"}')" in
    *'not supported in client mode'*) ;;
    *) echo "list_users was not refused"; fail=1 ;;
esac
if [ "$fail" != 0 ]; then
    echo "--- client log"; tail -30 "$WORK/client.log"
    echo "--- server log"; tail -30 "$WORK/server.log"
    die "FAILED"
fi
echo "smoke: OK"
