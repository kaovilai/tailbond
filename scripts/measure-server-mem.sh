#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Measure what a tailbond server costs on a small VPS: peak RSS (VmHWM) and
# CPU time of mqvpn (server) and tailbondd (server role) while 2 bonded
# clients, each with 2 paths, push traffic both ways at the same time.
#
#   scripts/measure-server-mem.sh
#   MB=300 CLIENTS=2 NETEM_B="delay 30ms 5ms loss 0.5%" scripts/measure-server-mem.sh
#   NETEM_A="rate 100mbit delay 10ms" NETEM_B="rate 50mbit delay 30ms 5ms loss 0.5%" \
#     scripts/measure-server-mem.sh          # paths shaped like fiber + LTE
#   MQVPN=out/x86_64/mqvpn TAILBONDD=out/agent/linux-amd64/tailbondd scripts/measure-server-mem.sh
#
# Network namespaces: tb-mem-srv runs tailbondd (server role, no Tailscale,
# no hooks) which runs mqvpn; tb-mem-cN run tailbondd in client role (speed
# mode: wlb + hybrid + reorder, like a real router) over eth0 (netem NETEM_A,
# default none) and wlan0 (netem NETEM_B). A fake internet host 45.45.45.45 on the server's
# loopback sinks MB megabytes from every client and sends MB to every client,
# all concurrently. Needs root (re-execs with sudo), python3, curl, jq, tc.
#
# Prints one table: idle / connected / peak RSS of each process, CPU seconds
# spent during the load, bytes moved and the derived CPU per 100 Mbit/s.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-amd64/tailbondd}"
MB="${MB:-300}"
CLIENTS="${CLIENTS:-2}"
NETEM_A="${NETEM_A:-}"
NETEM_B="${NETEM_B:-delay 30ms 5ms loss 0.5%}"
NS_S=tb-mem-srv
SERVER_IP=10.200.0.1
SINK=45.45.45.45
API=127.0.0.1:8077

if [ "$(id -u)" -ne 0 ]; then
  exec sudo -n MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" MB="$MB" CLIENTS="$CLIENTS" NETEM_A="$NETEM_A" NETEM_B="$NETEM_B" KEEP="${KEEP:-}" "$0" "$@"
fi
for c in python3 curl jq tc ip; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }
[ -x "$TAILBONDD" ] || { echo "tailbondd not found at $TAILBONDD (agent/build.sh amd64)"; exit 2; }

WORK="$(mktemp -d /tmp/tailbond-mem.XXXXXX)"
PIDS=()
cleanup() {
  [ -n "${KEEP:-}" ] && { echo "KEEP set: work dir $WORK, namespaces left"; return; }
  for p in "${PIDS[@]}"; do kill "$p" 2>/dev/null || true; done
  sleep 2
  for ns in $(ip netns list | awk '/^tb-mem-/ {print $1}'); do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    ip netns del "$ns" 2>/dev/null || true
  done
  rm -rf "$WORK"
}
trap cleanup EXIT
wait_for() { local t=$1; shift; for _ in $(seq 1 "$t"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done; return 1; }
kb() { awk -v k="$2" '$1 == k":" {print $2}' "/proc/$1/status" 2>/dev/null || echo 0; }
cpu_ticks() { awk '{print $14 + $15}' "/proc/$1/stat" 2>/dev/null || echo 0; }
HZ="$(getconf CLK_TCK)"

for ns in $(ip netns list | awk '/^tb-mem-/ {print $1}'); do ip netns del "$ns"; done
ip netns add "$NS_S"
ip -n "$NS_S" link set lo up
ip -n "$NS_S" addr add "$SERVER_IP/32" dev lo
ip -n "$NS_S" addr add "$SINK/32" dev lo
for i in $(seq 1 "$CLIENTS"); do
  ns="tb-mem-c$i"
  ip netns add "$ns"
  ip -n "$ns" link set lo up
  ip link add "tbm$i-a" type veth peer name "tbm$i-as"
  ip link add "tbm$i-b" type veth peer name "tbm$i-bs"
  ip link set "tbm$i-a" netns "$ns"; ip -n "$ns" link set "tbm$i-a" name eth0
  ip link set "tbm$i-b" netns "$ns"; ip -n "$ns" link set "tbm$i-b" name wlan0
  ip link set "tbm$i-as" netns "$NS_S"; ip link set "tbm$i-bs" netns "$NS_S"
  ip -n "$ns" addr add "10.20$i.1.2/24" dev eth0
  ip -n "$ns" addr add "10.20$i.2.2/24" dev wlan0
  ip -n "$NS_S" addr add "10.20$i.1.1/24" dev "tbm$i-as"
  ip -n "$NS_S" addr add "10.20$i.2.1/24" dev "tbm$i-bs"
  for l in eth0 wlan0; do ip -n "$ns" link set "$l" up; done
  ip -n "$NS_S" link set "tbm$i-as" up; ip -n "$NS_S" link set "tbm$i-bs" up
  ip -n "$ns" route add default via "10.20$i.1.1" dev eth0 metric 100
  ip -n "$ns" route add default via "10.20$i.2.1" dev wlan0 metric 200
  ip -n "$ns" route add "$SERVER_IP/32" via "10.20$i.2.1" dev wlan0 metric 200
  # shellcheck disable=SC2086 # NETEM_* are tc argument lists
  if [ -n "$NETEM_A" ]; then
    ip netns exec "$ns" tc qdisc add dev eth0 root netem $NETEM_A
    ip netns exec "$NS_S" tc qdisc add dev "tbm$i-as" root netem $NETEM_A
  fi
  # shellcheck disable=SC2086
  ip netns exec "$ns" tc qdisc add dev wlan0 root netem $NETEM_B
  # shellcheck disable=SC2086
  ip netns exec "$NS_S" tc qdisc add dev "tbm$i-bs" root netem $NETEM_B
done

# Traffic: a sink (reads until EOF) and a source (writes MB) on the fake
# internet host; clients connect through the tunnel.
cat >"$WORK/load.py" <<'EOF'
import socket, sys, threading, time
mode = sys.argv[1]
CH = b"\0" * 65536
if mode in ("sink", "source"):
    host, port = sys.argv[2], int(sys.argv[3])
    n = int(sys.argv[4]) if mode == "source" else 0
    s = socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); s.bind((host, port)); s.listen(16)
    def serve(c):
        try:
            if mode == "sink":
                while c.recv(1 << 20):
                    pass
            else:
                left = n
                while left > 0:
                    left -= c.send(CH[:min(len(CH), left)])
        finally:
            c.close()
    while True:
        c, _ = s.accept(); threading.Thread(target=serve, args=(c,), daemon=True).start()
else:  # send|recv HOST PORT BYTES
    host, port, n = sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
    t0 = time.time(); c = socket.create_connection((host, port), timeout=30); got = 0
    if mode == "send":
        while got < n:
            got += c.send(CH[:min(len(CH), n - got)])
        c.shutdown(socket.SHUT_WR); c.recv(1)
    else:
        while True:
            b = c.recv(1 << 20)
            if not b:
                break
            got += len(b)
    dt = time.time() - t0
    print("%s %d bytes in %.1fs = %.0f Mbit/s" % (mode, got, dt, got * 8 / dt / 1e6))
EOF
ip netns exec "$NS_S" python3 "$WORK/load.py" sink "$SINK" 5001 >/dev/null 2>&1 & PIDS+=($!)
ip netns exec "$NS_S" python3 "$WORK/load.py" source "$SINK" 5002 $((MB * 1000000)) >/dev/null 2>&1 & PIDS+=($!)

mkdir -p "$WORK/srv/hooks"
cat >"$WORK/srv/tailbond.json" <<EOF
{
  "role": "server", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/srv/state",
  "hooks_dir": "$WORK/srv/hooks", "listen": "$API", "mode": "speed",
  "disable_tailscale": true, "landing_port": -1,
  "server": {"port": 443, "public_addr": "$SERVER_IP", "subnet": "10.77.0.0/24",
             "subnet6": "fd77:77::/112", "use_tailscale_cert": false}
}
EOF
ip netns exec "$NS_S" "$TAILBONDD" -quiet -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 &
SRV=$!; PIDS+=("$SRV")
wait_for 20 ip netns exec "$NS_S" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  || { echo "server did not come up"; tail -20 "$WORK/server.log"; exit 1; }
MQ="$(pgrep -P "$SRV" -x mqvpn || pgrep -P "$SRV" | head -1)"
[ -n "$MQ" ] || { echo "no mqvpn child of tailbondd"; exit 1; }
sleep 3
IDLE_MQ="$(kb "$MQ" VmRSS)" IDLE_TB="$(kb "$SRV" VmRSS)"

for i in $(seq 1 "$CLIENTS"); do
  uri="$(ip netns exec "$NS_S" curl -fsS -m 5 -H 'Content-Type: application/json' -d "{\"name\":\"mem$i\"}" "http://$API/api/clients" | jq -r .uri)"
  mkdir -p "$WORK/c$i"
  cat >"$WORK/c$i/tailbond.json" <<EOF
{
  "role": "client", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/c$i/state",
  "hooks_dir": "$WORK/c$i/hooks", "listen": "$API", "mode": "speed", "disable_tailscale": true,
  "client": {"pairing_uri": "$uri", "no_dns": true, "captive_portal_detect": false,
             "paths": [{"iface": "eth0", "enabled": true, "priority": "primary", "data_cap_mb": 0},
                       {"iface": "wlan0", "enabled": true, "priority": "primary", "data_cap_mb": 0}]}
}
EOF
  ip netns exec "tb-mem-c$i" "$TAILBONDD" -quiet -config "$WORK/c$i/tailbond.json" >"$WORK/c$i.log" 2>&1 & PIDS+=($!)
done
for i in $(seq 1 "$CLIENTS"); do
  wait_for 40 ip netns exec "tb-mem-c$i" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
    || { echo "client $i did not connect"; tail -20 "$WORK/c$i.log"; exit 1; }
  # the fake internet host goes through the tunnel whatever routes mqvpn set
  ip -n "tb-mem-c$i" route replace "$SINK/32" dev tb0
done
wait_for 20 ip netns exec "$NS_S" sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '[.clients[] | select(.connected) | [.paths[] | select(.state == \"active\")] | length >= 2] | length == $CLIENTS and all'" \
  || echo "note: not every client shows 2 active paths yet: $(ip netns exec "$NS_S" curl -fsS http://$API/api/status | jq -c '[.clients[] | {name, n: (.paths | length)}]')"
sleep 3
CONN_MQ="$(kb "$MQ" VmRSS)" CONN_TB="$(kb "$SRV" VmRSS)"

echo "load: $CLIENTS clients x (${MB} MB up + ${MB} MB down), concurrently"
echo "      eth0 paths: ${NETEM_A:-no netem}; wlan0 paths: netem $NETEM_B"
C0_MQ="$(cpu_ticks "$MQ")" C0_TB="$(cpu_ticks "$SRV")" T0="$(date +%s.%N)"
LP=()
for i in $(seq 1 "$CLIENTS"); do
  ip netns exec "tb-mem-c$i" timeout 300 python3 "$WORK/load.py" send "$SINK" 5001 $((MB * 1000000)) >"$WORK/up$i" 2>&1 & LP+=($!)
  ip netns exec "tb-mem-c$i" timeout 300 python3 "$WORK/load.py" recv "$SINK" 5002 0 >"$WORK/down$i" 2>&1 & LP+=($!)
done
PEAK_MQ=0 PEAK_TB=0
while kill -0 "${LP[@]}" 2>/dev/null || pgrep -f "$WORK/load.py (send|recv)" >/dev/null; do
  r="$(kb "$MQ" VmRSS)"; [ "${r:-0}" -gt "$PEAK_MQ" ] && PEAK_MQ=$r
  r="$(kb "$SRV" VmRSS)"; [ "${r:-0}" -gt "$PEAK_TB" ] && PEAK_TB=$r
  sleep 0.5
done
wait "${LP[@]}" || true
T1="$(date +%s.%N)"
C1_MQ="$(cpu_ticks "$MQ")" C1_TB="$(cpu_ticks "$SRV")"
HWM_MQ="$(kb "$MQ" VmHWM)" HWM_TB="$(kb "$SRV" VmHWM)"

for i in $(seq 1 "$CLIENTS"); do echo "  client $i: $(cat "$WORK/up$i") | $(cat "$WORK/down$i")"; done
BYTES=0
for f in "$WORK"/up* "$WORK"/down*; do b="$(awk '{print $2}' "$f")"; BYTES=$((BYTES + ${b:-0})); done
SECS="$(awk -v a="$T0" -v b="$T1" 'BEGIN {printf "%.1f", b - a}')"
awk -v hz="$HZ" -v mq0="$C0_MQ" -v mq1="$C1_MQ" -v tb0="$C0_TB" -v tb1="$C1_TB" -v bytes="$BYTES" -v secs="$SECS" \
    -v imq="$IDLE_MQ" -v itb="$IDLE_TB" -v cmq="$CONN_MQ" -v ctb="$CONN_TB" -v pmq="$PEAK_MQ" -v ptb="$PEAK_TB" \
    -v hmq="$HWM_MQ" -v htb="$HWM_TB" 'BEGIN {
  mqs = (mq1 - mq0) / hz; tbs = (tb1 - tb0) / hz; mbit = bytes * 8 / 1e6
  printf "\n%-26s %10s %10s\n", "", "mqvpn", "tailbondd"
  printf "%-26s %8.1f MB %8.1f MB\n", "RSS idle (no clients)", imq / 1024, itb / 1024
  printf "%-26s %8.1f MB %8.1f MB\n", "RSS clients connected", cmq / 1024, ctb / 1024
  printf "%-26s %8.1f MB %8.1f MB\n", "RSS peak under load", pmq / 1024, ptb / 1024
  printf "%-26s %8.1f MB %8.1f MB\n", "VmHWM (since start)", hmq / 1024, htb / 1024
  printf "%-26s %9.2f s %9.2f s\n", "CPU during load", mqs, tbs
  printf "\nmoved %.0f MB through the server in %s s (%.0f Mbit/s aggregate)\n", bytes / 1e6, secs, mbit / secs
  printf "CPU per 100 Mbit/s sustained: %.1f%% of one core (mqvpn %.1f%%, tailbondd %.1f%%)\n", \
    (mqs + tbs) / (mbit / 100) * 100, mqs / (mbit / 100) * 100, tbs / (mbit / 100) * 100
  printf "CPU seconds per GB moved: %.2f\n", (mqs + tbs) / (bytes / 1e9)
}'
