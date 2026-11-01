#!/usr/bin/env bash
# End-to-end test of auto-updates and backup/restore (docs/architecture.md,
# "Updates", "Backup and restore"), in two network namespaces:
#
#   - a fake release server (python http.server in the server namespace)
#     serving a channel manifest signed with a throwaway ed25519 key
#     (openssl) and a linux tarball (tailbondd built as 0.0.2 + mqvpn);
#   - the home server (tailbondd 0.0.1) checks it, mirrors it, installs it
#     (self-update under state_dir, exit 75, restarted by a loop standing in
#     for systemd's Restart=on-failure, the base execs the new version, the
#     trial health check commits it);
#   - the client (0.0.1, public source blocked like on hotel Wi-Fi without
#     GitHub) updates through the server's mirror over the tunnel;
#   - a broken 0.0.3 (its tailbondd exits at once) is rolled back to 0.0.2
#     by the base after its trial boots, and not retried;
#   - an encrypted backup of the server, then the server is wiped (the client
#     can't connect), restored from the backup, and the client reconnects
#     with its old pairing.
#
#   scripts/e2e-updates.sh            (needs root: re-execs with sudo)
#   KEEP=1 scripts/e2e-updates.sh     leave namespaces and agents running
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
NS_S="tb-up-srv"
NS_C="tb-up-cli"
API=127.0.0.1:8077

BUILD="$ROOT/out/e2e-updates"
if [ "$(id -u)" -ne 0 ]; then
  # build as the invoking user (their Go module cache), then re-exec as root
  mkdir -p "$BUILD"
  (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=0.0.1" -o "$BUILD/tailbondd-0.0.1" ./cmd/tailbondd &&
    CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=0.0.2" -o "$BUILD/tailbondd-0.0.2" ./cmd/tailbondd)
  exec sudo --preserve-env=KEEP,VERBOSE MQVPN="$MQVPN" "$0" "$@"
fi
[ -x "$BUILD/tailbondd-0.0.1" ] && [ -x "$BUILD/tailbondd-0.0.2" ] || { echo "run as a normal user first (it builds, then uses sudo)"; exit 2; }
for c in curl jq openssl python3 nft; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }

WORK="$(mktemp -d /tmp/tailbond-e2e-updates.XXXXXX)"
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

say "tailbondd 0.0.1 (installed) and 0.0.2 (the release)"
mkdir -p "$WORK/bin" "$WORK/rel/channels" "$WORK/rel/dl" "$WORK/pkg/tailbond-linux-x86_64/bin"
cp "$BUILD/tailbondd-0.0.1" "$WORK/bin/tailbondd"
cp "$BUILD/tailbondd-0.0.2" "$WORK/pkg/tailbond-linux-x86_64/bin/tailbondd"
cp "$MQVPN" "$WORK/bin/mqvpn"
cp "$MQVPN" "$WORK/pkg/tailbond-linux-x86_64/bin/mqvpn"
ok "built"

say "release signing key (throwaway)"
openssl genpkey -algorithm ed25519 -out "$WORK/key.pem" 2>/dev/null
PUB="$(openssl pkey -in "$WORK/key.pem" -pubout -outform DER | tail -c 32 | base64 -w0)"
KEYID="$(openssl pkey -in "$WORK/key.pem" -pubout -outform DER | tail -c 32 | sha256sum | cut -c1-16)"
publish() { # publish VERSION TARBALL: writes stable.json(.sig) and the artifact
  local v=$1 tb=$2 name=tailbond-linux-x86_64.tar.gz sha size
  cp "$tb" "$WORK/rel/dl/$v-$name"
  sha="$(sha256sum "$tb" | cut -d' ' -f1)"; size="$(stat -c %s "$tb")"
  jq -n --arg v "$v" --arg url "http://10.210.1.1:8088/dl/$v-$name" --arg sha "$sha" --argjson size "$size" --arg name "$name" \
    '{format:1, channel:"stable", version:$v, protocol:1, requires_server_protocol:1, notes_url:"https://example.invalid/notes",
      artifacts:[{name:$name, kind:"linux", arch:"x86_64", url:$url, sha256:$sha, size:$size}]}' >"$WORK/rel/channels/stable.json"
  openssl pkeyutl -sign -rawin -inkey "$WORK/key.pem" -in "$WORK/rel/channels/stable.json" -out "$WORK/sig.bin"
  jq -n --arg k "$KEYID" --arg s "$(base64 -w0 "$WORK/sig.bin")" '{key_id:$k, sig:$s}' >"$WORK/rel/channels/stable.json.sig"
}
tar -C "$WORK/pkg" -czf "$WORK/v002.tar.gz" tailbond-linux-x86_64
ok "key $KEYID"

say "network"
for ns in "$NS_S" "$NS_C"; do ip netns del "$ns" 2>/dev/null || true; ip netns add "$ns"; ip -n "$ns" link set lo up; done
ip link add tbup-c type veth peer name tbup-s
ip link set tbup-c netns "$NS_C"; ip link set tbup-s netns "$NS_S"
ip -n "$NS_C" addr add 10.210.1.2/24 dev tbup-c; ip -n "$NS_S" addr add 10.210.1.1/24 dev tbup-s
ip -n "$NS_C" link set tbup-c name eth0; ip -n "$NS_C" link set eth0 up; ip -n "$NS_S" link set tbup-s up
ip -n "$NS_C" route add default via 10.210.1.1
ip netns exec "$NS_S" python3 -m http.server 8088 --bind 10.210.1.1 --directory "$WORK/rel" >/dev/null 2>&1 &
ok "server 10.210.1.1, client 10.210.1.2, release server :8088"

# run_agent NS NAME CONFIG: restart on any non-zero exit (75 = update), like
# systemd's Restart=on-failure; the base binary is $WORK/bin/tailbondd
run_agent() {
  ip netns exec "$1" bash -c "while [ ! -e '$WORK/stop' ]; do '$WORK/bin/tailbondd' -config '$3' >>'$WORK/$2.log' 2>&1 && break; sleep 1; done" &
}
mkdir -p "$WORK/srv" "$WORK/cli"
srv_config() {
  cat >"$WORK/srv/tailbond.json" <<EOF
{"role":"server","mqvpn_bin":"$WORK/bin/mqvpn","state_dir":"$WORK/srv/state","hooks_dir":"$WORK/hooks","listen":":8077",
 "disable_tailscale":true,"landing_port":-1,
 "server":{"port":443,"public_addr":"10.210.1.1","subnet":"10.77.0.0/24","use_tailscale_cert":false},
 "update":{"source":"http://10.210.1.1:8088/channels","trusted_keys":["$PUB"],"policy":"auto","window":"00:00-00:00"}}
EOF
}
srv_config
run_agent "$NS_S" server "$WORK/srv/tailbond.json"
wait_for 20 "server up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "server 0.0.1 up" || { bad "server"; exit 1; }
URI="$(sapi -X POST -d '{"name":"slate7"}' "http://$API/api/clients" | jq -r .uri)"
cat >"$WORK/cli/tailbond.json" <<EOF
{"role":"client","mqvpn_bin":"$WORK/bin/mqvpn","state_dir":"$WORK/cli/state","hooks_dir":"$WORK/hooks","listen":"$API",
 "disable_tailscale":true,"client":{"pairing_uri":"$URI","paths":[{"iface":"eth0","enabled":true,"priority":"primary"}],"no_dns":true,"manage_routes":false,"captive_portal_detect":false},
 "update":{"source":"http://10.210.1.1:8088/channels","trusted_keys":["$PUB"],"policy":"auto"}}
EOF
# hotel Wi-Fi without GitHub: the client can't reach the public source
ip netns exec "$NS_C" nft -f - <<'NFT'
table inet e2e_block {
  chain out {
    type filter hook output priority 0;
    ip daddr 10.210.1.1 tcp dport 8088 drop
  }
}
NFT
run_agent "$NS_C" client "$WORK/cli/tailbond.json"
wait_for 30 "client connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "client 0.0.1 connected" || { bad "client"; exit 1; }
jq -e '.version == "0.0.1" and .protocol == 1 and .update.policy == "auto"' <(sapi "http://$API/api/status") >/dev/null && ok "status: version 0.0.1, protocol 1" || bad "status: $(sapi "http://$API/api/status" | jq -c '{version, protocol, update}')"

say "release 0.0.2: the server mirrors and installs it"
publish 0.0.2 "$WORK/v002.tar.gz"
sapi -X POST -d '{}' "http://$API/api/update/check" >/dev/null
if wait_for 90 "server on 0.0.2" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.version == \"0.0.2\" and .update.last_result == \"ok\"'"; then
  ok "server self-updated to 0.0.2 and passed its health check (client reconnected)"
else
  bad "server: $(sapi "http://$API/api/status" | jq -c '{version, update}')"
fi
jq -e '.active == "0.0.2" and (.trial == null)' "$WORK/srv/state/update/state.json" >/dev/null && ok "state.json: 0.0.2 active, trial committed" || bad "state: $(cat "$WORK/srv/state/update/state.json")"
[ -x "$WORK/srv/state/update/versions/0.0.2/mqvpn" ] && pgrep -f "$WORK/srv/state/update/versions/0.0.2/mqvpn" >/dev/null \
  && ok "the update's own mqvpn runs" || bad "mqvpn from the update not running"
[ "$(sha256sum <"$WORK/bin/tailbondd")" != "$(sha256sum <"$WORK/srv/state/update/versions/0.0.2/tailbondd")" ] && ok "the packaged binary is untouched" || bad "base binary replaced"
sapi "http://$API/api/status" | jq -e '.update.mirror.version == "0.0.2" and .update.mirror.files == 1' >/dev/null && ok "mirror holds 0.0.2" || bad "mirror: $(sapi "http://$API/api/status" | jq -c .update.mirror)"

say "the client updates through the server's mirror (public source blocked)"
capi -X POST -d '{}' "http://$API/api/update/check" >/dev/null
if wait_for 120 "client on 0.0.2" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.version == \"0.0.2\" and .update.last_result == \"ok\" and .state == \"connected\"'"; then
  ok "client updated to 0.0.2 from the mirror and reconnected"
else
  bad "client: $(capi "http://$API/api/status" | jq -c '{version, state, update}')"
fi
capi "http://$API/api/status" | jq -e '.update.source == "mirror"' >/dev/null && ok "client's source: mirror" || bad "source: $(capi "http://$API/api/status" | jq -r .update.source)"
wait_for 20 "server sees the client's version" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.clients[] | select(.name == \"slate7\" and .version == \"0.0.2\" and .kind == \"router\")'" \
  && ok "server lists slate7 on 0.0.2, nothing outdated" || bad "clients: $(sapi "http://$API/api/status" | jq -c '[.clients[] | {name, version, kind}], .update.outdated')"

say "a broken release (0.0.3) is rolled back"
sapi -X POST -d '{"policy":"notify"}' "http://$API/api/update/policy" >/dev/null # the server keeps 0.0.2
rm -rf "$WORK/pkg" && mkdir -p "$WORK/pkg/tailbond-linux-x86_64/bin"
printf '#!/bin/sh\necho "broken release" >&2\nexit 1\n' >"$WORK/pkg/tailbond-linux-x86_64/bin/tailbondd"
chmod +x "$WORK/pkg/tailbond-linux-x86_64/bin/tailbondd"
tar -C "$WORK/pkg" -czf "$WORK/v003.tar.gz" tailbond-linux-x86_64
publish 0.0.3 "$WORK/v003.tar.gz"
sapi -X POST -d '{}' "http://$API/api/update/check" >/dev/null
wait_for 30 "mirror has 0.0.3" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.update.mirror.version == \"0.0.3\"'" || bad "mirror didn't pick up 0.0.3"
capi -X POST -d '{}' "http://$API/api/update/check" >/dev/null
if wait_for 120 "client rolled back" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.version == \"0.0.2\" and .update.last_result == \"rolled_back\"'"; then
  ok "client tried 0.0.3, it didn't start, the base rolled back to 0.0.2"
else
  bad "client: $(capi "http://$API/api/status" | jq -c '{version, update}' 2>&1)"
fi
grep -q "broken release" "$WORK/client.log" && ok "0.0.3 was really started" || bad "0.0.3 never ran"
jq -e '.active == "0.0.2" and (.failed | index("0.0.3") != null)' "$WORK/cli/state/update/state.json" >/dev/null && ok "0.0.3 marked failed" || bad "state: $(cat "$WORK/cli/state/update/state.json")"
capi -X POST -d '{}' "http://$API/api/update/check" >/dev/null; sleep 5
capi "http://$API/api/status" | jq -e '.version == "0.0.2" and (.update.state != "installing")' >/dev/null && ok "not retried" || bad "retried 0.0.3"
wait_for 30 "client connected again" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" && ok "client connected on 0.0.2" || bad "client not connected"

say "backup the server, wipe it, restore it"
PASSPHRASE="e2e backup passphrase"
ip netns exec "$NS_S" curl -fsS -m 30 -H 'Content-Type: application/json' -d "{\"passphrase\":\"$PASSPHRASE\"}" -o "$WORK/srv.tbbackup" "http://$API/api/backup"
head -1 "$WORK/srv.tbbackup" | grep -q TBBACKUP1 && ! grep -q slate7 "$WORK/srv.tbbackup" && ok "encrypted backup ($(stat -c %s "$WORK/srv.tbbackup") bytes)" || bad "backup file"
ip netns exec "$NS_S" "$WORK/bin/tailbondd" backup --plain -config "$WORK/srv/tailbond.json" | jq -e '.clients == ["slate7"]' >/dev/null && ok "tailbondd backup --plain lists the clients, no keys" || bad "--plain"
touch "$WORK/stop"; ip netns pids "$NS_S" | xargs -r kill 2>/dev/null || true
wait_for 15 "server stopped" sh -c "! ip netns pids $NS_S | grep -q ."
rm -f "$WORK/stop"
rm -rf "$WORK/srv/state"; srv_config # a new machine: same address, no clients, new certificate
ip netns exec "$NS_S" python3 -m http.server 8088 --bind 10.210.1.1 --directory "$WORK/rel" >/dev/null 2>&1 &
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
if wait_for 90 "client reconnected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'"; then
  ok "the client reconnected with its old pairing (same key, same pinned certificate)"
else
  bad "client after restore: $(capi "http://$API/api/status" | jq -c '{state, error}')"
fi
sapi "http://$API/api/clients" | jq -e '.[] | select(.name == "slate7" and .connected)' >/dev/null && ok "server lists slate7 connected" || bad "server clients: $(sapi "http://$API/api/clients")"

echo
printf '\033[1mresult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
