#!/usr/bin/env bash
# End-to-end test of the server certificate pin (a self-signed server's
# cert_sha256 in the pairing), in two network namespaces: a home server and
# a paired client. While the client is connected, the server is replaced by
# another mqvpn server at the same address with a different self-signed
# certificate that accepts the same key, as an on-path attacker relaying to
# the real server would. The client must not bring its tunnel up with it
# (mqvpn checks the pin on its reconnect, not just tailbondd before
# starting mqvpn), must not reach it with the pinned mqvpn CLI either, and
# must connect again once the real server is back. A client pinned to the
# second server's certificate connects to it (the pin works with any
# self-signed certificate). An empty pin (`--cert-sha256 "" --insecure`, or
# an empty CertSHA256 in a config file with Insecure = true) is refused.
#
#   scripts/e2e-cert-pin.sh            (needs root: re-execs with sudo)
#   KEEP=1 scripts/e2e-cert-pin.sh     leave namespaces and agents running
#   MQVPN=/path/to/mqvpn TAILBONDD=/path/to/tailbondd scripts/e2e-cert-pin.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/e2e-cert-pin/tailbondd}"
NS_S="tb-pin-srv"
NS_C="tb-pin-cli"
API=127.0.0.1:8077

if [ "$(id -u)" -ne 0 ]; then
  if [ "$TAILBONDD" = "$ROOT/out/e2e-cert-pin/tailbondd" ]; then
    # build as the invoking user (their Go module cache), then re-exec as root
    mkdir -p "$(dirname "$TAILBONDD")"
    (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -o "$TAILBONDD" ./cmd/tailbondd)
  fi
  exec sudo --preserve-env=KEEP,VERBOSE MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" "$0" "$@"
fi
for c in curl jq openssl ping; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }
[ -x "$TAILBONDD" ] || { echo "tailbondd not found at $TAILBONDD (run as a normal user first: it builds, then uses sudo)"; exit 2; }

WORK="$(mktemp -d /tmp/tailbond-e2e-pin.XXXXXX)"
PASS=0 FAIL=0
MITM_PID=""
say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
sapi() { ip netns exec "$NS_S" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
capi() { ip netns exec "$NS_C" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
wait_for() { # wait_for <seconds> <description> <command...>
  local t=$1 what=$2; shift 2
  for _ in $(seq 1 "$t"); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "  timed out after ${t}s waiting for: $what"
  return 1
}

cleanup() {
  local rc=$?
  if [ -n "${KEEP:-}" ]; then echo "KEEP: namespaces and agents left running; work dir $WORK"; return; fi
  touch "$WORK/stop"
  for ns in "$NS_S" "$NS_C"; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill 2>/dev/null || true
  done
  sleep 2
  for ns in "$NS_S" "$NS_C"; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    ip netns del "$ns" 2>/dev/null || true
  done
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- server log (tail)"; tail -n 30 "$WORK/server.log" 2>/dev/null || true
    echo "--- impostor log (tail)"; tail -n 30 "$WORK/mitm.log" 2>/dev/null || true
    echo "--- client log (tail)"; tail -n 60 "$WORK/client.log" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

say "network"
for ns in "$NS_S" "$NS_C"; do ip netns del "$ns" 2>/dev/null || true; ip netns add "$ns"; ip -n "$ns" link set lo up; done
ip link add tbpin-c type veth peer name tbpin-s
ip link set tbpin-c netns "$NS_C"; ip link set tbpin-s netns "$NS_S"
ip -n "$NS_C" addr add 10.212.1.2/24 dev tbpin-c; ip -n "$NS_S" addr add 10.212.1.1/24 dev tbpin-s
ip -n "$NS_C" link set tbpin-c name eth0; ip -n "$NS_C" link set eth0 up; ip -n "$NS_S" link set tbpin-s up
ip -n "$NS_C" route add default via 10.212.1.1
ok "server 10.212.1.1, client 10.212.1.2"

# run_agent NS NAME CONFIG: restart on any non-zero exit, like systemd's
# Restart=on-failure
run_agent() {
  ip netns exec "$1" bash -c "while [ ! -e '$WORK/stop-$2' ]; do '$TAILBONDD' -config '$3' >>'$WORK/$2.log' 2>&1 && break; sleep 1; done" &
}
mkdir -p "$WORK/srv" "$WORK/cli" "$WORK/hooks"
cat >"$WORK/srv/tailbond.json" <<EOF
{"role":"server","mqvpn_bin":"$MQVPN","state_dir":"$WORK/srv/state","hooks_dir":"$WORK/hooks","listen":":8077",
 "disable_tailscale":true,"landing_port":-1,
 "server":{"port":443,"public_addr":"10.212.1.1","subnet":"10.77.0.0/24","use_tailscale_cert":false},
 "update":{"policy":"off"}}
EOF

say "home server (self-signed certificate) and a paired client"
run_agent "$NS_S" server "$WORK/srv/tailbond.json"
wait_for 20 "server up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "server up" || { bad "server did not come up"; exit 1; }
URI="$(sapi -X POST -d '{"name":"slate7"}' "http://$API/api/clients" | jq -r .uri)"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
PAYLOAD="$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d)"
PIN_A="$(jq -r .cert_sha256 <<<"$PAYLOAD")"
KEY="$(jq -r .key <<<"$PAYLOAD")"
REAL_PIN="$(openssl x509 -in "$WORK/srv/state/server.crt" -outform DER 2>/dev/null | openssl dgst -sha256 -r | cut -d' ' -f1 || true)"
jq -e '.insecure == true and (.cert_sha256 | length) == 64' <<<"$PAYLOAD" >/dev/null && [ "$PIN_A" = "$REAL_PIN" ] \
  && ok "pairing: self-signed server, pinned to its certificate" || bad "pairing payload: $PAYLOAD (server cert sha256 ${REAL_PIN:-?})"

cat >"$WORK/cli/tailbond.json" <<EOF
{"role":"client","mqvpn_bin":"$MQVPN","state_dir":"$WORK/cli/state","hooks_dir":"$WORK/hooks","listen":"$API",
 "disable_tailscale":true,"client":{"pairing_uri":"$URI","paths":[{"iface":"eth0","enabled":true,"priority":"primary"}],"no_dns":true,"manage_routes":false,"captive_portal_detect":false},
 "update":{"policy":"off"}}
EOF
run_agent "$NS_C" client "$WORK/cli/tailbond.json"
wait_for 30 "client connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "client connected" || { bad "client never connected"; exit 1; }
# -I tb0: the server's namespace (the client's default route) has
# 10.77.0.1 too, so only a ping through the tunnel interface proves a tunnel
wait_for 10 "ping through the tunnel" ip netns exec "$NS_C" ping -I tb0 -c1 -W1 10.77.0.1 \
  && ok "ping 10.77.0.1 through tb0" || bad "no ping through the tunnel"
if grep -q '^CertSHA256 = ' "$WORK/cli/state/client.conf" && grep -q '^Insecure = false' "$WORK/cli/state/client.conf"; then
  ok "client.conf: CertSHA256 pinned, Insecure off"
else
  bad "client.conf: $(grep -E '^(CertSHA256|Insecure)' "$WORK/cli/state/client.conf" | tr '\n' ' ')"
fi

say "impostor: another server at 10.212.1.1:443, different self-signed certificate, same key"
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 30 -subj /CN=10.212.1.1 \
  -addext subjectAltName=IP:10.212.1.1 -keyout "$WORK/mitm.key" -out "$WORK/mitm.crt" >/dev/null 2>&1
PIN_B="$(openssl x509 -in "$WORK/mitm.crt" -outform DER | openssl dgst -sha256 -r | cut -d' ' -f1)"
[ "$PIN_B" != "$PIN_A" ] && ok "impostor certificate differs (sha256 ${PIN_B:0:16}...)" || bad "same certificate"
# the real server goes away (its tailbondd and mqvpn), the impostor takes its address
touch "$WORK/stop-server"
ip netns pids "$NS_S" | xargs -r kill 2>/dev/null || true
wait_for 15 "server stopped" sh -c "! ip netns pids $NS_S | grep -q ."
ip netns exec "$NS_S" "$MQVPN" --mode server --listen 0.0.0.0:443 --subnet 10.77.0.0/24 --tun-name tbm0 \
  --cert "$WORK/mitm.crt" --key "$WORK/mitm.key" --user "slate7:$KEY" --log-level debug >"$WORK/mitm.log" 2>&1 &
MITM_PID=$!
wait_for 10 "impostor tunnel" sh -c "ip -n $NS_S -4 addr show tbm0 | grep -q 'inet 10.77.0.1'" \
  && ok "impostor up on 10.212.1.1:443 (tunnel 10.77.0.1)" || { bad "impostor did not start: $(tail -5 "$WORK/mitm.log")"; exit 1; }

# The client's mqvpn reconnects by itself (Reconnect = true); tailbondd
# restarts it when it exits. Neither may bring the tunnel up.
MITM_UP=0
for _ in $(seq 1 40); do
  if ip netns exec "$NS_C" ping -I tb0 -c1 -W1 10.77.0.1 >/dev/null 2>&1; then MITM_UP=1; break; fi
  sleep 0.5
done
[ "$MITM_UP" = 0 ] && ok "no tunnel through the impostor (40 tries over ~60 s)" \
  || bad "the client's tunnel came up through the impostor (it accepted a certificate other than the pinned one)"
STATE="$(capi "http://$API/api/status" | jq -r .state || true)"
[ "$STATE" != connected ] && ok "client status: $STATE" || bad "client status: connected (to the impostor)"
# mqvpn's own check, on the reconnect of the tunnel it had up (tailbondd's
# pre-start check, "server certificate does not match the pinned SHA-256",
# only runs when it restarts mqvpn)
grep -q 'leaf SHA-256 does not match the pinned certificate' "$WORK/client.log" \
  && ok "mqvpn rejected the impostor's certificate on its reconnect" || bad "no rejection by mqvpn in the client log"
# the key goes over the tunnel's HTTP/3 request, after the handshake: an
# impostor the client hung up on never sees it
if grep -q 'client authenticated' "$WORK/mitm.log"; then
  bad "the impostor authenticated the client: $(grep 'client authenticated' "$WORK/mitm.log" | tail -1)"
else
  ok "the impostor never authenticated the client"
fi

say "the mqvpn CLI with the pin, straight at the impostor"
set +e
ip netns exec "$NS_C" timeout 20 "$MQVPN" --mode client --server 10.212.1.1:443 --cert-sha256 "$PIN_A" --insecure \
  --auth-key "$KEY" --tun-name tbx0 --no-reconnect --no-manage-routes >"$WORK/cli-wrong.log" 2>&1
RC=$?
set -e
if [ "$RC" -ne 0 ] && [ "$RC" -ne 124 ] && grep -q 'does not match the pinned' "$WORK/cli-wrong.log" && ! grep -q 'ESTABLISHED' "$WORK/cli-wrong.log"; then
  ok "rejected (exit $RC, pin mismatch; --insecure did not override the pin)"
else
  bad "mqvpn --cert-sha256 <real server> against the impostor: exit $RC: $(tail -4 "$WORK/cli-wrong.log" | tr '\n' ' ')"
fi
# An empty pin (an unset $PIN) is an error, never "no pin": with --insecure,
# or Insecure = true in the file, it would otherwise accept any certificate.
# Both must stop mqvpn (or at worst fail the handshake), never reach the
# impostor's tunnel.
cat >"$WORK/empty-pin.conf" <<EOF
[Server]
Address = 10.212.1.1:443
Insecure = true
CertSHA256 =

[Auth]
Key = $KEY

[Interface]
TunName = tbx0
ManageRoutes = false
Reconnect = false
EOF
for how in cli file; do
  if [ "$how" = cli ]; then
    set -- --mode client --server 10.212.1.1:443 --cert-sha256 "" --insecure \
      --auth-key "$KEY" --tun-name tbx0 --no-reconnect --no-manage-routes
  else
    set -- --config "$WORK/empty-pin.conf"
  fi
  set +e
  ip netns exec "$NS_C" timeout 20 "$MQVPN" "$@" >"$WORK/cli-empty-$how.log" 2>&1
  RC=$?
  set -e
  what='--cert-sha256 "" --insecure'; [ "$how" = file ] && what='CertSHA256 = (empty), Insecure = true'
  if [ "$RC" -ne 0 ] && [ "$RC" -ne 124 ] && ! grep -q 'ESTABLISHED' "$WORK/cli-empty-$how.log" \
    && grep -qE 'must be 64 hex digits|does not match the pinned' "$WORK/cli-empty-$how.log"; then
    ok "$what: refused (exit $RC: $(grep -m1 -oE 'must be 64 hex digits|does not match the pinned' "$WORK/cli-empty-$how.log"))"
  else
    bad "$what against the impostor: exit $RC: $(tail -4 "$WORK/cli-empty-$how.log" | tr '\n' ' ')"
  fi
done
set --
set +e
ip netns exec "$NS_C" timeout 20 "$MQVPN" --mode client --server 10.212.1.1:443 --cert-sha256 "${PIN_B^^}" \
  --auth-key "$KEY" --tun-name tbx0 --no-reconnect --no-manage-routes >"$WORK/cli-right.log" 2>&1 &
XPID=$!
set -e
if wait_for 15 "pinned CLI tunnel" grep -q 'ESTABLISHED' "$WORK/cli-right.log"; then
  ok "pinned to the impostor's own certificate (uppercase hex): connects"
else
  bad "mqvpn --cert-sha256 <its own certificate> did not connect: $(tail -4 "$WORK/cli-right.log" | tr '\n' ' ')"
fi
kill "$XPID" 2>/dev/null || true; wait "$XPID" 2>/dev/null || true

say "the real server is back"
kill "$MITM_PID" 2>/dev/null || true; wait "$MITM_PID" 2>/dev/null || true
rm -f "$WORK/stop-server"
run_agent "$NS_S" server "$WORK/srv/tailbond.json"
wait_for 20 "server up again" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "server up again (same certificate)" || bad "server did not come back"
if wait_for 90 "client reconnected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"' && ip netns exec $NS_C ping -I tb0 -c1 -W1 10.77.0.1"; then
  ok "client reconnected to the real server, ping ok"
else
  bad "client did not reconnect: $(capi "http://$API/api/status" | jq -c '{state, error}' || true)"
fi

echo
echo "e2e-cert-pin: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
