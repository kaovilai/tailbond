#!/usr/bin/env bash
# Server side of the Android emulator end-to-end test (scripts/e2e-android.sh;
# CI job "android-e2e"): tailbondd in the server role with mqvpn on this host,
# for an emulator that reaches the host at 10.0.2.2.
#
#   MQVPN=/path/to/mqvpn TAILBONDD=/path/to/tailbondd scripts/e2e-android-server.sh [stop]
#
# - Tailscale is off; a fake `tailscale` CLI on the agent's PATH answers the
#   machine list (GET /api/peers): homebox (self), nas, laptop.
# - A pairing for the phone "pixel-e2e" (POST /api/clients); its
#   tailbond://pair URI goes to $WORK/pair-uri, the payload's server is
#   10.0.2.2:$PORT.
# - A token web server on 10.77.0.1:8099 (the tunnel address) that only
#   answers through the tunnel: TCP 8099 arriving on lo (the host itself, and
#   so the emulator's NAT, which connects from the host) is rejected.
#
# Runs as root (mqvpn's TUN). Leaves everything running; "stop" ends it.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${TB_E2E_DIR:-/tmp/tailbond-e2e-android}"
PORT="${TB_E2E_PORT:-443}"
TOKEN_PORT=8099
API=127.0.0.1:8077

if [ "$(id -u)" -ne 0 ]; then
  exec sudo --preserve-env=TB_E2E_DIR,TB_E2E_PORT MQVPN="${MQVPN:-}" TAILBONDD="${TAILBONDD:-}" "$0" "$@"
fi

stop() {
  for f in "$WORK/server.pid" "$WORK/www.pid"; do
    [ -s "$f" ] && kill "$(cat "$f")" 2>/dev/null || true
  done
  iptables -D INPUT -p tcp --dport "$TOKEN_PORT" -i lo -j REJECT 2>/dev/null || true
}
if [ "${1:-}" = stop ]; then stop; exit 0; fi

MQVPN="${MQVPN:-$ROOT/out/native/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-amd64/tailbondd}"
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }
[ -x "$TAILBONDD" ] || { echo "tailbondd not found at $TAILBONDD"; exit 2; }
for c in curl jq python3 iptables; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done

say() { printf '==> %s\n' "$*"; }
api() { curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
wait_for() { # wait_for SECONDS cmd...
  local t=$1; shift
  for _ in $(seq "$t"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}

stop
rm -rf "$WORK"
mkdir -p "$WORK/ts" "$WORK/srv" "$WORK/hooks" "$WORK/www"
chmod 755 "$WORK"

say "fake tailscale CLI (homebox, nas, laptop)"
cat >"$WORK/ts/tailscale" <<'EOF'
#!/bin/sh
# fake tailscale CLI for the Android e2e test
case "$1" in
status) cat <<'J'
{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.1.1"],"Online":true},
 "Peer":{"n1":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true},
         "n2":{"HostName":"laptop","DNSName":"laptop.tail1234.ts.net.","OS":"macOS","TailscaleIPs":["100.101.3.3"],"Online":false}}}
J
;;
debug) echo '{"AdvertiseRoutes":[]}' ;;
cert) echo "certs disabled" >&2; exit 1 ;;
whois) echo "no such peer" >&2; exit 1 ;;
*) exit 0 ;;
esac
EOF
chmod +x "$WORK/ts/tailscale"

say "tailbondd (server role) on UDP $PORT, phones pair to 10.0.2.2:$PORT"
cat >"$WORK/srv/tailbond.json" <<EOF
{
  "role": "server",
  "mqvpn_bin": "$MQVPN",
  "state_dir": "$WORK/srv/state",
  "hooks_dir": "$WORK/hooks",
  "listen": ":8077",
  "mode": "speed",
  "server": {"port": $PORT, "public_addr": "10.0.2.2", "subnet": "10.77.0.0/24", "subnet6": "fd77:77::/112", "use_tailscale_cert": false}
}
EOF
env PATH="$WORK/ts:$PATH" setsid "$TAILBONDD" -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 </dev/null &
echo $! >"$WORK/server.pid"
if ! wait_for 30 sh -c "curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .control_api'"; then
  echo "server did not come up"; tail -n 50 "$WORK/server.log"; exit 1
fi
wait_for 10 sh -c "ip -4 addr show dev tb0 | grep -qE 'inet 10\.77\.0\.1[ /]'" || { echo "tb0 has no 10.77.0.1"; ip addr; exit 1; }
echo "server up: $(api "http://$API/api/status" | jq -c '{state, server, tunnel_ip}')"

say "pairing for pixel-e2e"
PAIR="$(api -X POST -d '{"name":"pixel-e2e"}' "http://$API/api/clients")"
URI="$(jq -r .uri <<<"$PAIR")"
[[ "$URI" == tailbond://pair\?d=* ]] || { echo "bad pairing response: $PAIR"; exit 1; }
printf '%s\n' "$URI" >"$WORK/pair-uri"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
printf '%s' "$B64" | tr '_-' '/+' | base64 -d | jq '{v, server, sni, insecure, mode, dns, subnet, server_name}'

say "token web server on 10.77.0.1:$TOKEN_PORT, through the tunnel only"
head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$WORK/www/token.txt"
iptables -I INPUT -p tcp --dport "$TOKEN_PORT" -i lo -j REJECT
setsid python3 -m http.server "$TOKEN_PORT" --bind 10.77.0.1 --directory "$WORK/www" >"$WORK/www.log" 2>&1 </dev/null &
echo $! >"$WORK/www.pid"
wait_for 10 sh -c "ss -ltn | grep -q '10.77.0.1:$TOKEN_PORT'" || { echo "token server did not start"; cat "$WORK/www.log"; exit 1; }
if curl -fsS -m 3 "http://10.77.0.1:$TOKEN_PORT/token.txt" >/dev/null 2>&1; then
  echo "the token server answers outside the tunnel"; exit 1
fi
echo "token server up, refused from the host itself"
chmod -R a+rX "$WORK"
