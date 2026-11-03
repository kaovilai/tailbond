#!/usr/bin/env bash
# End-to-end test of backup and restore (docs/architecture.md, "Backup and
# restore"), in two network namespaces: a home server and a paired client;
# an encrypted backup of the server, then the server is wiped (the client
# can't connect), restored from the backup (preview, rollback copy), and the
# client reconnects with its old pairing. Updates: packaging/linux/
# test-update.sh (scripts/e2e-update-vps.sh) and scripts/test-openwrt-ipk.sh.
#
#   scripts/e2e-backup.sh            (needs root: re-execs with sudo)
#   KEEP=1 scripts/e2e-backup.sh     leave namespaces and agents running
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
NS_S="tb-up-srv"
NS_C="tb-up-cli"
API=127.0.0.1:8077

BUILD="$ROOT/out/e2e-backup"
if [ "$(id -u)" -ne 0 ]; then
  # build as the invoking user (their Go module cache), then re-exec as root
  mkdir -p "$BUILD"
  (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=0.0.1" -o "$BUILD/tailbondd-0.0.1" ./cmd/tailbondd)
  exec sudo --preserve-env=KEEP,VERBOSE MQVPN="$MQVPN" "$0" "$@"
fi
[ -x "$BUILD/tailbondd-0.0.1" ] || { echo "run as a normal user first (it builds, then uses sudo)"; exit 2; }
for c in curl jq python3 nft; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }

WORK="$(mktemp -d /tmp/tailbond-e2e-backup.XXXXXX)"
PASS=0 FAIL=0
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
    echo "--- server log (tail)"; tail -n 40 "$WORK/server.log" 2>/dev/null || true
    echo "--- client log (tail)"; tail -n 40 "$WORK/client.log" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

say "tailbondd 0.0.1"
mkdir -p "$WORK/bin"
cp "$BUILD/tailbondd-0.0.1" "$WORK/bin/tailbondd"
cp "$MQVPN" "$WORK/bin/mqvpn"
ok "built"

say "network"
for ns in "$NS_S" "$NS_C"; do ip netns del "$ns" 2>/dev/null || true; ip netns add "$ns"; ip -n "$ns" link set lo up; done
ip link add tbup-c type veth peer name tbup-s
ip link set tbup-c netns "$NS_C"; ip link set tbup-s netns "$NS_S"
ip -n "$NS_C" addr add 10.210.1.2/24 dev tbup-c; ip -n "$NS_S" addr add 10.210.1.1/24 dev tbup-s
ip -n "$NS_C" link set tbup-c name eth0; ip -n "$NS_C" link set eth0 up; ip -n "$NS_S" link set tbup-s up
ip -n "$NS_C" route add default via 10.210.1.1
ok "server 10.210.1.1, client 10.210.1.2"

# run_agent NS NAME CONFIG: restart on any non-zero exit (75 = restart after
# a restore), like systemd's Restart=on-failure
run_agent() {
  ip netns exec "$1" bash -c "while [ ! -e '$WORK/stop' ]; do '$WORK/bin/tailbondd' -config '$3' >>'$WORK/$2.log' 2>&1 && break; sleep 1; done" &
}
mkdir -p "$WORK/srv" "$WORK/cli"
srv_config() {
  cat >"$WORK/srv/tailbond.json" <<EOF
{"role":"server","mqvpn_bin":"$WORK/bin/mqvpn","state_dir":"$WORK/srv/state","hooks_dir":"$WORK/hooks","listen":":8077",
 "disable_tailscale":true,"landing_port":-1,
 "server":{"port":443,"public_addr":"10.210.1.1","subnet":"10.77.0.0/24","use_tailscale_cert":false},
 "update":{"policy":"off"}}
EOF
}
srv_config
run_agent "$NS_S" server "$WORK/srv/tailbond.json"
wait_for 20 "server up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "server 0.0.1 up" || { bad "server"; exit 1; }
URI="$(sapi -X POST -d '{"name":"slate7"}' "http://$API/api/clients" | jq -r .uri)"
cat >"$WORK/cli/tailbond.json" <<EOF
{"role":"client","mqvpn_bin":"$WORK/bin/mqvpn","state_dir":"$WORK/cli/state","hooks_dir":"$WORK/hooks","listen":"$API",
 "disable_tailscale":true,"client":{"pairing_uri":"$URI","paths":[{"iface":"eth0","enabled":true,"priority":"primary"}],"no_dns":true,"manage_routes":false,"captive_portal_detect":false},
 "update":{"policy":"off"}}
EOF
run_agent "$NS_C" client "$WORK/cli/tailbond.json"
wait_for 30 "client connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "client 0.0.1 connected" || { bad "client"; exit 1; }
jq -e '.version == "0.0.1" and .update.policy == "off"' <(sapi "http://$API/api/status") >/dev/null && ok "status: version 0.0.1, updates off" || bad "status: $(sapi "http://$API/api/status" | jq -c '{version, update}')"

say "backup the server, wipe it, restore it"
PASSPHRASE="e2e backup passphrase"
ip netns exec "$NS_S" curl -fsS -m 30 -H 'Content-Type: application/json' -d "{\"passphrase\":\"$PASSPHRASE\"}" -o "$WORK/srv.tbbackup" "http://$API/api/backup"
head -1 "$WORK/srv.tbbackup" | grep -q TBBACKUP1 && ! grep -q slate7 "$WORK/srv.tbbackup" && ok "encrypted backup ($(stat -c %s "$WORK/srv.tbbackup") bytes)" || bad "backup file"
ip netns exec "$NS_S" "$WORK/bin/tailbondd" backup --plain -config "$WORK/srv/tailbond.json" | jq -e '.clients == ["slate7"]' >/dev/null && ok "tailbondd backup --plain lists the clients, no keys" || bad "--plain"
touch "$WORK/stop"; ip netns pids "$NS_S" | xargs -r kill 2>/dev/null || true
wait_for 15 "server stopped" sh -c "! ip netns pids $NS_S | grep -q ."
rm -f "$WORK/stop"
rm -rf "$WORK/srv/state"; srv_config # a new machine: same address, no clients, new certificate
run_agent "$NS_S" server "$WORK/srv/tailbond.json"
wait_for 20 "fresh server up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "wiped server up (0.0.1, no clients)" || bad "fresh server"
wait_for 40 "client can't connect" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state != \"connected\"'" && ok "the client can't connect to the wiped server" || bad "client still connected"
B64="$(base64 -w0 "$WORK/srv.tbbackup")"
jq -n --arg f "$B64" --arg p "$PASSPHRASE" '{file:$f, passphrase:$p}' >"$WORK/restore.json"
ip netns exec "$NS_S" curl -fsS -m 30 -H 'Content-Type: application/json' --data @"$WORK/restore.json" "http://$API/api/restore/preview" >"$WORK/preview.json" || true
jq -e '.role == "server" and ([.changes[] | select(.what == "Devices (clients)" and .to == "slate7")] | length == 1)' "$WORK/preview.json" >/dev/null \
  && ok "preview: slate7 comes back" || bad "preview: $(cat "$WORK/preview.json")"
jq '.confirm = true' "$WORK/restore.json" >"$WORK/restore2.json"
ip netns exec "$NS_S" curl -fsS -m 30 -H 'Content-Type: application/json' --data @"$WORK/restore2.json" "http://$API/api/restore" | jq -e '.ok and .restarting' >/dev/null \
  && ok "restored (rollback copy kept)" || bad "restore failed"
ls "$WORK/srv/state/backups/"before-restore-*.tbbackup >/dev/null 2>&1 && ok "before-restore copy in state/backups" || bad "no rollback copy"
# The client's own status can still say "connected" from before the
# server's restart (its QUIC idle timeout hasn't fired yet): the restarted
# server listing slate7 connected is what proves the reconnect.
if wait_for 90 "client reconnected" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/clients | jq -e '.[] | select(.name == \"slate7\" and .connected)'"; then
  ok "server lists slate7 connected"
else
  bad "server clients: $(sapi "http://$API/api/clients")"
fi
# the client's status follows within its 1 s poll
wait_for 10 "client status connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "the client reconnected with its old pairing (same key, same pinned certificate)" \
  || bad "client after restore: $(capi "http://$API/api/status" | jq -c '{state, error}')"

echo
printf '\033[1mresult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
