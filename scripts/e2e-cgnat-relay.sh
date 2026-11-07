#!/usr/bin/env bash
# End-to-end test of the CGNAT relay (docs/behind-cgnat.md, option 4).
#
# The home server sits behind a carrier NAT that accepts nothing inbound, so
# no device can reach its tunnel port. It opens WireGuard *out* to a VPS, and
# the VPS forwards udp/443 into it; devices connect to the VPS's address.
#
#   rv-cli    a device, 203.0.113.20, default via the VPS
#   rv-vps    the relay, pub 203.0.113.10, wg0 10.200.0.1, WG on 51820
#   rv-cgnat  the carrier: 192.0.2.2 toward the VPS, 100.64.0.1 toward home,
#             masquerades home's traffic and drops inbound it did not ask for
#   rv-home   the home server, 100.64.0.2 only, wg0 10.200.0.2
#
# Checks: the home server is unreachable directly; WireGuard comes up on 51820
# (not 443, which the DNAT owns); the relayed port carries the tunnel; the
# server's health says "reached through your relay"; and traffic through the
# tunnel leaves from home's line, not the VPS's.
#
#   scripts/e2e-cgnat-relay.sh
set -euo pipefail
[ "$(id -u)" = 0 ] || exec sudo -E "$0" "$@"

REPO="$(cd "$(dirname "$0")/.." && pwd)"
TAILBONDD="${TAILBONDD:-$REPO/out/agent/linux-amd64/tailbondd}"
[ -x "$TAILBONDD" ] || { echo "no tailbondd at $TAILBONDD" >&2; exit 1; }
command -v wg >/dev/null || { echo "needs wireguard-tools (wg)" >&2; exit 1; }
[ -e /sys/module/wireguard ] || modprobe wireguard 2>/dev/null || true

W="$(mktemp -d /tmp/tb-relay.XXXXXX)"
NS_CLI=rv-cli NS_VPS=rv-vps NS_CG=rv-cgnat NS_HOME=rv-home
PASS=0 FAIL=0
ok() { printf '  PASS %s\n' "$*"; PASS=$((PASS + 1)); }
bad() { printf '  FAIL %s\n' "$*"; FAIL=$((FAIL + 1)); }
say() { printf '==> %s\n' "$*"; }
# tail a log for CI output with pairing URIs (credentials) redacted: the
# agent does not log them today, but these dumps must never become the leak.
# Redact to the end of the token: a charset allowlist would stop at the
# first character it didn't list and print the rest of the credential.
dumplog() { tail -n "${2:-10}" "$1" | sed 's#tailbond://[^[:space:]]*#tailbond://[redacted]#g'; }
cleanup() {
	# deleting a netns does not stop the processes in it: kill this run's
	# agents and listeners first, or they linger and hold the namespace alive.
	# Only namespaces this run created ($MADE): if a name already existed the
	# add failed and the namespace, and everything in it, is someone else's.
	for n in $MADE; do
		ip netns pids "$n" 2>/dev/null | xargs -r kill 2>/dev/null || true
	done
	sleep 1
	for n in $MADE; do
		ip netns pids "$n" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
		ip netns del "$n" 2>/dev/null || true
	done
	rm -rf "$W"
}
trap cleanup EXIT

say "namespaces and links"
MADE="" # the namespaces this run created; cleanup only touches these
for n in $NS_CLI $NS_VPS $NS_CG $NS_HOME; do
	# an && list would shield the failure from set -e and silently run inside
	# whatever already owns the name, so refuse instead
	ip netns add "$n" || { echo "namespace $n already exists; remove it first: ip netns del $n" >&2; exit 1; }
	MADE="$MADE $n"
done
link() { # A B IFACE_A IFACE_B
	ip link add "$3" netns "$1" type veth peer name "$4" netns "$2"
	ip -n "$1" link set "$3" up; ip -n "$2" link set "$4" up
	ip -n "$1" link set lo up; ip -n "$2" link set lo up
}
link $NS_CLI $NS_VPS cli0 vps1
link $NS_VPS $NS_CG vps2 cg1
link $NS_CG $NS_HOME cg2 home0
ip -n $NS_CLI addr add 203.0.113.20/24 dev cli0
ip -n $NS_VPS addr add 203.0.113.10/24 dev vps1
ip -n $NS_VPS addr add 192.0.2.1/24 dev vps2
ip -n $NS_CG addr add 192.0.2.2/24 dev cg1
ip -n $NS_CG addr add 100.64.0.1/24 dev cg2
ip -n $NS_HOME addr add 100.64.0.2/24 dev home0
ip -n $NS_CLI route add default via 203.0.113.10
ip -n $NS_VPS route add 100.64.0.0/24 via 192.0.2.2
ip -n $NS_CG route add default via 192.0.2.1
ip -n $NS_HOME route add default via 100.64.0.1

say "the carrier blocks inbound (that is the CGNAT the relay exists for)"
ip netns exec $NS_CG sysctl -qw net.ipv4.ip_forward=1
ip netns exec $NS_VPS sysctl -qw net.ipv4.ip_forward=1
ip netns exec $NS_CG iptables -t nat -A POSTROUTING -o cg1 -s 100.64.0.0/24 -j MASQUERADE
ip netns exec $NS_CG iptables -A FORWARD -i cg2 -o cg1 -j ACCEPT
ip netns exec $NS_CG iptables -A FORWARD -i cg1 -o cg2 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
# something to probe: the check only means anything against a live listener
cat >"$W/listen.py" <<PY
import socket, sys
s = socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("0.0.0.0", int(sys.argv[1]))); s.listen(8); s.settimeout(30)
while True:
    try: c, _ = s.accept()
    except socket.timeout: break
    c.send(b"ok"); c.close()
PY
ip netns exec $NS_HOME python3 "$W/listen.py" 8077 &
LISTEN=$!
sleep 1
# positive control: with no inbound filter, the carrier forwards to home
ip netns exec $NS_CLI timeout 3 bash -c ': >/dev/tcp/100.64.0.2/8077' 2>/dev/null \
	&& ok "the path to home works before the carrier filters it" || bad "no path to home at all"
ip netns exec $NS_CG iptables -A FORWARD -i cg1 -o cg2 -j DROP
ip netns exec $NS_CG iptables -A FORWARD -i cg2 -o cg2 -j DROP
ip netns exec $NS_CLI timeout 3 nc -zu -w1 100.64.0.2 443 >/dev/null 2>&1
! ip netns exec $NS_CLI timeout 3 bash -c ': >/dev/tcp/100.64.0.2/8077' 2>/dev/null \
	&& ok "home is unreachable from the internet" || bad "home answered directly"
kill "$LISTEN" 2>/dev/null || true
wait "$LISTEN" 2>/dev/null || true
ip netns exec $NS_CLI ping -c1 -W1 203.0.113.10 >/dev/null && ok "the VPS is reachable" || bad "no route to the VPS"

say "WireGuard: home dials out to the VPS on 51820 (never 443, the DNAT owns it)"
wg genkey >"$W/vps.key"; wg pubkey <"$W/vps.key" >"$W/vps.pub"
wg genkey >"$W/home.key"; wg pubkey <"$W/home.key" >"$W/home.pub"
ip -n $NS_VPS link add wg0 type wireguard
ip -n $NS_VPS link set wg0 netns $NS_VPS 2>/dev/null || true
ip netns exec $NS_VPS wg set wg0 private-key "$W/vps.key" listen-port 51820 peer "$(cat "$W/home.pub")" allowed-ips 10.200.0.2/32
ip -n $NS_VPS addr add 10.200.0.1/24 dev wg0 && ip -n $NS_VPS link set wg0 up
ip -n $NS_HOME link add wg0 type wireguard
ip netns exec $NS_HOME wg set wg0 private-key "$W/home.key" peer "$(cat "$W/vps.pub")" endpoint 192.0.2.1:51820 allowed-ips 10.200.0.1/32 persistent-keepalive 5
ip -n $NS_HOME addr add 10.200.0.2/24 dev wg0 && ip -n $NS_HOME link set wg0 up
for _ in $(seq 1 10); do
	[ "$(ip netns exec $NS_VPS wg show wg0 latest-handshakes | awk '{print $2}')" != 0 ] && break
	sleep 1
done
hs=$(ip netns exec $NS_VPS wg show wg0 latest-handshakes | awk '{print $2}')
[ "${hs:-0}" != 0 ] && ok "handshake over UDP 51820 (through the carrier NAT)" || bad "no WireGuard handshake"

say "the VPS forwards udp/443 into home over wg0"
ip netns exec $NS_VPS iptables -t nat -A PREROUTING -i vps1 -p udp --dport 443 -j DNAT --to-destination 10.200.0.2:443
ip netns exec $NS_VPS iptables -t nat -A POSTROUTING -o wg0 -p udp -d 10.200.0.2 --dport 443 -j MASQUERADE
ip netns exec $NS_VPS iptables -A FORWARD -i vps1 -o wg0 -p udp -d 10.200.0.2 --dport 443 -j ACCEPT
ip netns exec $NS_VPS iptables -A FORWARD -i wg0 -o vps1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
ip netns exec $NS_VPS ss -lun | grep -q ':51820' && ok "WireGuard listens on 51820" || bad "no listener on 51820"
ip netns exec $NS_VPS ss -lun | grep -q ':443 ' && bad "something listens on 443 and would eat the DNAT" || ok "443 is free for the relay"

say "home server, reached through the relay"
mkdir -p "$W/srv/state" "$W/hooks"
cat >"$W/srv/tailbond.json" <<JSON
{"role": "server", "mqvpn_bin": "${MQVPN:-/usr/bin/mqvpn}", "state_dir": "$W/srv/state", "hooks_dir": "$W/hooks",
 "listen": ":8077", "mode": "speed",
 "server": {"port": 443, "public_addr": "203.0.113.10:443", "relayed": true, "subnet": "10.77.0.0/24", "use_tailscale_cert": false}}
JSON
ip netns exec $NS_HOME "$TAILBONDD" -config "$W/srv/tailbond.json" >"$W/srv.log" 2>&1 &
for _ in $(seq 1 20); do
	ip netns exec $NS_HOME curl -fsS -m 2 http://127.0.0.1:8077/api/hello >/dev/null 2>&1 && break
	sleep 1
done
ip netns exec $NS_HOME curl -fsS -m 3 http://127.0.0.1:8077/api/hello | grep -q '"role":"server"' \
	&& ok "the agent answers on its own machine" || { bad "agent did not come up"; dumplog "$W/srv.log" 5; }

say "a device pairs and connects to the VPS address (never to home)"
sapi() { ip netns exec $NS_HOME curl -fsS -m 5 "$@"; }
URI="$(sapi -X POST -H 'Content-Type: application/json' -d '{"name":"phone"}' http://127.0.0.1:8077/api/clients | jq -r .uri)"
[ "${#URI}" -gt 40 ] && ok "pairing issued" || bad "no pairing (got ${#URI} chars, expected a tailbond:// URI)"
mkdir -p "$W/cli/state"
cat >"$W/cli/tailbond.json" <<JSON
{"role": "client", "mqvpn_bin": "${MQVPN:-/usr/bin/mqvpn}", "state_dir": "$W/cli/state", "hooks_dir": "$W/hooks",
 "listen": ":8078", "mode": "speed",
 "client": {"pairing_uri": "$URI", "paths": [{"iface": "cli0", "enabled": true, "priority": "primary"}],
            "no_dns": true, "manage_routes": false}}
JSON
ip netns exec $NS_CLI "$TAILBONDD" -config "$W/cli/tailbond.json" >"$W/cli.log" 2>&1 &
# 75 s: mqvpn's reconnect backoff reaches 24 s by attempt 4, and a retry
# that lands just past a 40 s window reads as "did not connect" while the
# client is on its way back
for _ in $(seq 1 75); do
	ip netns exec $NS_CLI curl -fsS -m 2 http://127.0.0.1:8078/api/status 2>/dev/null | jq -e '.state == "connected"' >/dev/null 2>&1 && break
	sleep 1
done
ip netns exec $NS_CLI curl -fsS -m 3 http://127.0.0.1:8078/api/status | jq -e '.state == "connected"' >/dev/null \
	&& ok "the client is connected through the relay" \
	|| { bad "client did not connect"; dumplog "$W/cli.log" 12; echo "--- server log:"; dumplog "$W/srv.log" 15; }

say "Setup health knows it is a relay, not a broken port"
sleep 2
ip netns exec $NS_HOME curl -fsS -m 3 http://127.0.0.1:8077/api/health >"$W/health.json" 2>/dev/null || true
grep -q 'reached through your relay' "$W/health.json" \
	&& ok "health: UDP 443 reached through your relay" || bad "health did not confirm the relay: $(jq -r '.checks[]?|select(.id=="port")|.title' "$W/health.json" 2>/dev/null)"
grep -q 'not confirmed' "$W/health.json" && bad "health still says not confirmed" || ok "no stale outside-port warning"

say "the far end sees home's line, not the VPS"
cat >"$W/echo.py" <<PY
import socket, sys
s = socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("192.0.2.1", 8080)); s.listen(1); s.settimeout(20)
c, a = s.accept(); open(sys.argv[1], "w").write(a[0]); c.send(b"ok"); c.close()
PY
ip netns exec $NS_VPS python3 "$W/echo.py" "$W/peer" &
sleep 1
ip netns exec $NS_HOME timeout 20 curl -fsS -m 15 http://192.0.2.1:8080/ >/dev/null 2>&1 || true
wait $! 2>/dev/null || true
peer="$(cat "$W/peer" 2>/dev/null || echo none)"
[ "$peer" = 192.0.2.2 ] && ok "home egresses through its own line ($peer = the carrier)" \
	|| bad "home egressed as $peer (expected the carrier 192.0.2.2)"

echo
echo "RESULT: $PASS passed, $FAIL failed"
[ "$FAIL" = 0 ]
