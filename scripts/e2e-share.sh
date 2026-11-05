#!/usr/bin/env bash
# End-to-end test of Pair & Share (docs/pair-and-share.md) in network
# namespaces: a hub (tailbondd client role, eth0 to the server) bonds its own
# uplink plus the mobile data a sharer on its LAN lends it.
#
#   MQVPN=/path/to/mqvpn TAILBONDD=/path/to/tailbondd scripts/e2e-share.sh
#
#   tb-sh-srv  server role; eth0 10.200.1.1 (hub's uplink), cell 10.200.3.1
#   tb-sh-hub  client role (the hub); eth0 10.200.1.2, lan0 192.168.8.1
#   tb-sh-phn  client role sharing wwan0 (its "cellular", 10.200.3.2 to the
#              server) on its Wi-Fi wlan0 192.168.8.23 (the hub's LAN)
#   tb-sh-lap  a Linux laptop ("thinkpad"), client role, on the hub's other
#              LAN (wlan0 192.168.9.42; its default route: the hub, which
#              routes it out eth0 with NAT, as a travel router does) with
#              the hotel's Ethernet eth1 10.200.4.2 (a second default route,
#              metric 200) that it shares
#
# Checks: the sharer advertises over mDNS and registers with the server; the
# hub finds it, opens a session and adds a relay path (mqvpn Path
# shNNNNNN@127.0.0.1:port, patches/mqvpn/0002); the server sees the hub with
# 2 active paths, one from the sharer's "cellular" address; traffic through
# the tunnel crosses the share (byte counters on both ends); the sharer
# stops: the hub keeps working over eth0; client isolation (the hub learns of
# the sharer from the server but can't reach it on the LAN) shows the
# "This Wi-Fi blocks devices…" note.
#
# A computer sharing (docs/pair-and-share.md "Computers can share too"):
# the laptop's dashboard API offers eth1 and wlan0, refuses the tunnel, the
# loopback and a missing interface; once eth1 is shared the server lists it
# (kind "linux", "Ethernet from thinkpad"), the hub bonds it under that
# name, and the relay's upstream socket is pinned to eth1: it leaves from
# 10.200.4.2 although the default route is wlan0 (unpinned, it would go
# back through the hub, NATed to 10.200.1.2), and the server counts its
# datagrams on its hotel link; the laptop's own tunnel over eth1 stays up.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MQVPN="${MQVPN:-$ROOT/third_party/mqvpn/build/mqvpn}"
TAILBONDD="${TAILBONDD:-$ROOT/out/agent/linux-$(go env GOARCH 2>/dev/null || echo amd64)/tailbondd}"
NS_S=tb-sh-srv NS_H=tb-sh-hub NS_P=tb-sh-phn NS_L=tb-sh-lap
API=127.0.0.1:8077

if [ "$(id -u)" -ne 0 ]; then
  if [ ! -x "$TAILBONDD" ]; then
    echo "building tailbondd -> $TAILBONDD"
    (cd "$ROOT/agent" && CGO_ENABLED=0 go build -trimpath -o "$TAILBONDD" ./cmd/tailbondd)
  fi
  exec sudo --preserve-env=KEEP,VERBOSE MQVPN="$MQVPN" TAILBONDD="$TAILBONDD" "$0" "$@"
fi
[ -x "$MQVPN" ] || { echo "mqvpn not found at $MQVPN"; exit 2; }
[ -x "$TAILBONDD" ] || { echo "tailbondd not found at $TAILBONDD"; exit 2; }
for c in curl jq nft ping; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done

WORK="$(mktemp -d /tmp/tailbond-e2e-share.XXXXXX)"
PASS=0 FAIL=0
SRV_PID="" HUB_PID="" PHN_PID="" LAP_PID=""
say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
sapi() { ip netns exec "$NS_S" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
hapi() { ip netns exec "$NS_H" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
papi() { ip netns exec "$NS_P" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
lapi() { ip netns exec "$NS_L" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
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
  if [ -n "${KEEP:-}" ]; then echo "KEEP set: namespaces and agents left running; work dir $WORK"; return; fi
  for p in "$HUB_PID" "$PHN_PID" "$LAP_PID" "$SRV_PID"; do [ -n "$p" ] && kill "$p" 2>/dev/null || true; done
  sleep 1
  for ns in "$NS_S" "$NS_H" "$NS_P" "$NS_L"; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    ip netns del "$ns" 2>/dev/null || true
  done
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    for l in server hub phone laptop; do echo "--- $l agent log (tail) ---"; tail -n 50 "$WORK/$l.log" 2>/dev/null || true; done
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

say "network: server, hub (eth0, lan0, lan1), phone (wlan0 on lan0, wwan0 its own uplink), laptop (wlan0 on lan1, eth1 the hotel's)"
for ns in "$NS_S" "$NS_H" "$NS_P" "$NS_L"; do ip netns del "$ns" 2>/dev/null || true; ip netns add "$ns"; ip -n "$ns" link set lo up; done
ip link add tbsh-a type veth peer name tbsh-as   # hub eth0 <-> server
ip link add tbsh-l type veth peer name tbsh-lp   # hub lan0 <-> phone wlan0
ip link add tbsh-c type veth peer name tbsh-cs   # phone wwan0 <-> server ("cellular")
ip link set tbsh-a netns "$NS_H"; ip -n "$NS_H" link set tbsh-a name eth0
ip link set tbsh-l netns "$NS_H"; ip -n "$NS_H" link set tbsh-l name lan0
ip link set tbsh-lp netns "$NS_P"; ip -n "$NS_P" link set tbsh-lp name wlan0
ip link set tbsh-c netns "$NS_P"; ip -n "$NS_P" link set tbsh-c name wwan0
ip link set tbsh-as netns "$NS_S"; ip -n "$NS_S" link set tbsh-as name eth0
ip link set tbsh-cs netns "$NS_S"; ip -n "$NS_S" link set tbsh-cs name cell
ip link add tbsh-w type veth peer name tbsh-wl   # hub lan1 <-> laptop wlan0
ip link add tbsh-e type veth peer name tbsh-es   # laptop eth1 <-> server ("hotel Ethernet")
ip link set tbsh-w netns "$NS_H"; ip -n "$NS_H" link set tbsh-w name lan1
ip link set tbsh-wl netns "$NS_L"; ip -n "$NS_L" link set tbsh-wl name wlan0
ip link set tbsh-e netns "$NS_L"; ip -n "$NS_L" link set tbsh-e name eth1
ip link set tbsh-es netns "$NS_S"; ip -n "$NS_S" link set tbsh-es name hotel
ip -n "$NS_S" addr add 10.200.1.1/24 dev eth0
ip -n "$NS_S" addr add 10.200.3.1/24 dev cell
ip -n "$NS_H" addr add 10.200.1.2/24 dev eth0
ip -n "$NS_H" addr add 192.168.8.1/24 dev lan0
ip -n "$NS_P" addr add 192.168.8.23/24 dev wlan0
ip -n "$NS_P" addr add 10.200.3.2/24 dev wwan0
ip -n "$NS_S" addr add 10.200.4.1/24 dev hotel
ip -n "$NS_H" addr add 192.168.9.1/24 dev lan1
ip -n "$NS_L" addr add 192.168.9.42/24 dev wlan0
ip -n "$NS_L" addr add 10.200.4.2/24 dev eth1
for x in "$NS_S eth0" "$NS_S cell" "$NS_S hotel" "$NS_H eth0" "$NS_H lan0" "$NS_H lan1" "$NS_P wlan0" "$NS_P wwan0" "$NS_L wlan0" "$NS_L eth1"; do
  set -- $x; ip -n "$1" link set "$2" up multicast on
done
ip -n "$NS_H" route add default via 10.200.1.1 dev eth0
# the phone reaches the server only over its "cellular" wwan0; the server
# has one public address (10.200.1.1) and answers from it on every link
ip -n "$NS_P" route add default via 10.200.3.1 dev wwan0
ip -n "$NS_S" route replace 10.200.3.0/24 dev cell src 10.200.1.1
ip -n "$NS_S" route replace 10.200.4.0/24 dev hotel src 10.200.1.1
# the laptop's default route is the hub's LAN (the hub routes it out eth0
# with NAT, like a travel router); the hotel Ethernet is a second default,
# as DHCP on a second adapter leaves it. Only a socket pinned to eth1
# reaches the server from 10.200.4.2.
ip netns exec "$NS_H" sysctl -qw net.ipv4.ip_forward=1
ip netns exec "$NS_H" nft -f - <<'EOF'
table ip tbnat {
  chain post { type nat hook postrouting priority 100; oifname "eth0" ip saddr 192.168.9.0/24 masquerade; }
}
EOF
ip -n "$NS_L" route add default via 192.168.9.1 dev wlan0 metric 100
ip -n "$NS_L" route add default via 10.200.4.1 dev eth1 metric 200
ip netns exec "$NS_H" ping -c1 -W2 10.200.1.1 >/dev/null && ip netns exec "$NS_P" ping -c1 -W2 10.200.1.1 >/dev/null \
  && ip netns exec "$NS_H" ping -c1 -W2 192.168.8.23 >/dev/null && ok "underlay: hub and phone reach the server; the phone is on the hub's LAN"

mkdir -p "$WORK/srv" "$WORK/hub" "$WORK/phn" "$WORK/lap" "$WORK/hooks" "$WORK/ts"
printf '#!/bin/sh\n[ "$1" = status ] && echo '"'"'{"BackendState":"Stopped"}'"'"'\nexit 1\n' >"$WORK/ts/tailscale"
chmod +x "$WORK/ts/tailscale"

say "server"
cat >"$WORK/srv/tailbond.json" <<EOF
{"role": "server", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/srv/state", "hooks_dir": "$WORK/hooks", "listen": ":8077", "mode": "speed",
 "server": {"port": 443, "public_addr": "10.200.1.1", "subnet": "10.77.0.0/24", "use_tailscale_cert": false}}
EOF
ip netns exec "$NS_S" env PATH="$WORK/ts:$PATH" "$TAILBONDD" -config "$WORK/srv/tailbond.json" >"$WORK/server.log" 2>&1 &
SRV_PID=$!
wait_for 20 "server up" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "server up" || { bad "server did not come up"; exit 1; }
pair() { sapi -X POST -d "{\"name\":\"$1\"}" "http://$API/api/clients" | jq -r .uri; }
HUB_URI="$(pair slate7)"; PHN_URI="$(pair pixel)"; LAP_URI="$(pair thinkpad)"
share_key_of() { local b="${1#tailbond://pair?d=}"; while [ $(( ${#b} % 4 )) -ne 0 ]; do b="$b="; done; printf '%s' "$b" | tr '_-' '/+' | base64 -d | jq -r '.share_key // ""'; }
PK="$(share_key_of "$PHN_URI")"; HK="$(share_key_of "$HUB_URI")"
[ "${#PK}" -gt 20 ] && [ "${#HK}" -gt 20 ] && [ "$PK" != "$HK" ] \
  && ok "each pairing carries its own share key" || bad "share keys missing or shared: phone ${#PK} hub ${#HK} chars"

say "phone: client role over wwan0, sharing wwan0 on its Wi-Fi"
cat >"$WORK/phn/tailbond.json" <<EOF
{"role": "client", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/phn/state", "hooks_dir": "$WORK/hooks", "listen": "$API", "mode": "speed",
 "client": {"pairing_uri": "$PHN_URI", "paths": [{"iface": "wwan0", "enabled": true, "priority": "primary"}], "no_dns": true, "manage_routes": false,
            "captive_portal_detect": false, "use_shares": false,
            "share": {"enabled": true, "iface": "wwan0", "label": "Pixel's 5G"}}}
EOF
ip netns exec "$NS_P" env PATH="$WORK/ts:$PATH" "$TAILBONDD" -config "$WORK/phn/tailbond.json" >"$WORK/phone.log" 2>&1 &
PHN_PID=$!
wait_for 30 "phone sharing" sh -c "ip netns exec $NS_P curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\" and .sharing.state == \"sharing\"'" \
  && ok "phone connected and sharing: $(papi "http://$API/api/status" | jq -c '.sharing | {label, port, state}')" \
  || bad "phone not sharing: $(papi "http://$API/api/status" | jq -c '{state, sharing}')"
wait_for 90 "phone registered with the server" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/shares | jq -e '.sharers | length == 1'" \
  && ok "server lists the sharer: $(sapi "http://$API/api/shares" | jq -c '.sharers[0] | {label, client, lan, port}')" \
  || bad "sharer not registered: $(sapi "http://$API/api/shares" | jq -c .)"

say "hub: client role over eth0, bonding the phone's share"
cat >"$WORK/hub/tailbond.json" <<EOF
{"role": "client", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/hub/state", "hooks_dir": "$WORK/hooks", "listen": "$API", "mode": "speed",
 "client": {"pairing_uri": "$HUB_URI", "paths": [{"iface": "eth0", "enabled": true, "priority": "primary"}], "no_dns": true, "manage_routes": false,
            "captive_portal_detect": false}}
EOF
start_hub() {
  ip netns exec "$NS_H" env PATH="$WORK/ts:$PATH" "$TAILBONDD" -config "$WORK/hub/tailbond.json" >>"$WORK/hub.log" 2>&1 &
  HUB_PID=$!
}
start_hub
wait_for 30 "hub connected" sh -c "ip netns exec $NS_H curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "hub connected" || bad "hub not connected"
if wait_for 60 "share ready" sh -c "ip netns exec $NS_H curl -fsS -m 2 http://$API/api/status | jq -e '[.shares[]? | select(.state == \"ready\")] | length == 1'"; then
  ok "hub found the share: $(hapi "http://$API/api/status" | jq -c '.shares[0] | {label, via, state, path}')"
else
  bad "no ready share: $(hapi "http://$API/api/status" | jq -c '{shares, share_note}')"
fi
SHP="$(hapi "http://$API/api/status" | jq -r '.shares[0].path // empty')"
two_paths() { sapi "http://$API/api/status" | jq -e '.clients[] | select(.name == "slate7" and .connected) | [.paths[] | select(.state == "active")] | length >= 2' >/dev/null; }
if wait_for 60 "server sees 2 paths for slate7" two_paths; then
  ok "server: slate7 bonds 2 active paths: $(sapi "http://$API/api/status" | jq -c '[.clients[] | select(.name == "slate7") | .paths[] | {state, srtt_ms}]')"
else
  bad "server paths for slate7: $(sapi "http://$API/api/status" | jq -c '[.clients[] | select(.name == "slate7") | .paths[]]')"
fi
hapi "http://$API/api/status" | jq -e --arg p "$SHP" '.paths[] | select(.iface == $p and .kind == "shared-cellular" and .label == "Pixel'"'"'s 5G (shared)")' >/dev/null \
  && ok "hub path $SHP: kind shared-cellular, label \"Pixel's 5G (shared)\", priority $(hapi "http://$API/api/status" | jq -r --arg p "$SHP" '.paths[] | select(.iface == $p) | .priority')" \
  || bad "hub paths: $(hapi "http://$API/api/status" | jq -c '[.paths[] | {iface, kind, label, state}]')"
grep -q "Path = $SHP@127.0.0.1:" "$WORK/hub/state/mqvpn.conf" 2>/dev/null || grep -rqs "Path = $SHP@127.0.0.1:" "$WORK/hub/state" \
  && ok "mqvpn runs the share as a relay path ($SHP@127.0.0.1:…)" || bad "no relay Path line in the hub's mqvpn config"

say "traffic through the tunnel crosses the share"
ip netns exec "$NS_H" ping -c 20 -i 0.1 -W 2 10.77.0.1 >/dev/null 2>&1 && ok "ping 10.77.0.1 through the tunnel" || bad "ping through the tunnel failed"
ip netns exec "$NS_H" ping -c 200 -i 0.01 -s 1000 -W 2 10.77.0.1 >/dev/null 2>&1 || true
sleep 2
UP="$(papi "http://$API/api/status" | jq '[.sharing.hubs[]? | select(.hub == "slate7") | .up_bytes + .down_bytes] | add // 0')"
[ "${UP:-0}" -gt 0 ] && ok "the phone relayed $UP bytes for slate7" || bad "no bytes through the share: $(papi "http://$API/api/status" | jq -c .sharing)"
HB="$(hapi "http://$API/api/status" | jq '.shares[0] | .up_bytes + .down_bytes')"
[ "${HB:-0}" -gt 0 ] && ok "hub counts $HB bytes on the share" || bad "hub share counters zero"

say "the phone stops sharing (agent killed): the hub keeps working over eth0"
kill "$PHN_PID"; wait "$PHN_PID" 2>/dev/null || true; PHN_PID=""
ip netns pids "$NS_P" | xargs -r kill 2>/dev/null || true
T0=$(date +%s)
clean() { [ "$(ip netns exec "$NS_H" ping -c 10 -i 0.2 -W 1 10.77.0.1 2>/dev/null | sed -n 's/.* \([0-9.]*\)% packet loss.*/\1/p')" = 0 ]; }
if wait_for 20 "traffic clean again without the share" clean; then
  ok "traffic continues without the share (clean again $(( $(date +%s) - T0 )) s after it went)"
else
  bad "tunnel still losing traffic 20 s after the share went: $(hapi "http://$API/api/status" | jq -c '{state, shares, paths: [.paths[] | {iface, state}]}')"
fi
wait_for 40 "share no longer ready" sh -c "ip netns exec $NS_H curl -fsS -m 2 http://$API/api/status | jq -e '[.shares[]? | select(.state == \"ready\")] | length == 0'" \
  && ok "hub marks the share $(hapi "http://$API/api/status" | jq -r '.shares[0].state // "gone"')" || bad "share still ready after the phone left"

say "a Linux laptop lends its hotel Ethernet (eth1) to the hub: a real tailbondd sharer"
cat >"$WORK/lap/tailbond.json" <<EOF
{"role": "client", "mqvpn_bin": "$MQVPN", "state_dir": "$WORK/lap/state", "hooks_dir": "$WORK/hooks", "listen": "$API", "mode": "speed",
 "client": {"pairing_uri": "$LAP_URI", "paths": [{"iface": "eth1", "enabled": true, "priority": "primary"}], "no_dns": true, "manage_routes": false,
            "captive_portal_detect": false, "use_shares": false}}
EOF
ip netns exec "$NS_L" env PATH="$WORK/ts:$PATH" "$TAILBONDD" -config "$WORK/lap/tailbond.json" >"$WORK/laptop.log" 2>&1 &
LAP_PID=$!
wait_for 30 "laptop connected over eth1" sh -c "ip netns exec $NS_L curl -fsS -m 2 http://$API/api/status | jq -e '.state == \"connected\"'" \
  && ok "laptop's own tunnel up over eth1" || bad "laptop not connected: $(lapi "http://$API/api/status" | jq -c '{state, error}')"
cands() {
  lapi "http://$API/api/status" | jq -e '.sharing.state == "off"
    and ([.sharing.ifaces[] | select(.name == "eth1" and .kind == "ethernet" and .share_label == "Ethernet from thinkpad")] | length == 1)
    and ([.sharing.ifaces[] | select(.name == "wlan0" and .kind == "wifi")] | length == 1)
    and ([.sharing.ifaces[].name] | (index("tb0") == null and index("lo") == null))' >/dev/null
}
wait_for 20 "share choices" cands \
  && ok "laptop offers to share: $(lapi "http://$API/api/status" | jq -c '[.sharing.ifaces[] | {name, kind, share_label}]')" \
  || bad "share choices: $(lapi "http://$API/api/status" | jq -c .sharing)"
for x in tb0 lo nope0; do
  code="$(ip netns exec "$NS_L" curl -sS -m 5 -o "$WORK/refused.json" -w '%{http_code}' -H 'Content-Type: application/json' \
    -d "{\"share\":{\"enabled\":true,\"iface\":\"$x\"}}" "http://$API/api/share")"
  [ "$code" = 400 ] && ok "refuses to share $x: $(jq -r .error "$WORK/refused.json")" || bad "sharing $x: HTTP $code $(cat "$WORK/refused.json")"
done
lapi -X POST -d '{"share": {"enabled": true, "iface": "eth1", "allowed_hubs": ["slate7"]}}' "http://$API/api/share" >/dev/null \
  && ok "POST /api/share: eth1, for slate7" || bad "could not turn sharing on"
wait_for 20 "laptop sharing" sh -c "ip netns exec $NS_L curl -fsS -m 2 http://$API/api/status | jq -e '.sharing.state == \"sharing\" and .sharing.label == \"Ethernet from thinkpad\"'" \
  && ok "laptop sharing: $(lapi "http://$API/api/status" | jq -c '.sharing | {iface, label, port, state, allowed_hubs}')" \
  || bad "laptop not sharing: $(lapi "http://$API/api/status" | jq -c .sharing)"
registered() {
  sapi "http://$API/api/shares" | jq -e '.sharers[] | select(.client == "thinkpad" and .kind == "linux" and .label == "Ethernet from thinkpad" and (.lan | index("192.168.9.42")))' >/dev/null
}
wait_for 90 "laptop registered" registered \
  && ok "server lists it: $(sapi "http://$API/api/shares" | jq -c '.sharers[] | select(.client == "thinkpad") | {label, kind, lan, port}')" \
  || bad "laptop not registered: $(sapi "http://$API/api/shares" | jq -c .sharers)"
lap_ready() { hapi "http://$API/api/status" | jq -e '.shares[]? | select(.label == "Ethernet from thinkpad" and .state == "ready")' >/dev/null; }
wait_for 60 "hub bonds the laptop's share" lap_ready \
  && ok "hub found it: $(hapi "http://$API/api/status" | jq -c '.shares[] | select(.label == "Ethernet from thinkpad") | {label, via, state, path}')" \
  || bad "hub shares: $(hapi "http://$API/api/status" | jq -c '{shares, share_note}')"
LSHP="$(hapi "http://$API/api/status" | jq -r '.shares[]? | select(.label == "Ethernet from thinkpad") | .path')"
lap_path() { hapi "http://$API/api/status" | jq -e --arg p "$LSHP" '.paths[] | select(.iface == $p and .kind == "shared-cellular" and .label == "Ethernet from thinkpad" and .state == "active")' >/dev/null; }
wait_for 60 "the laptop's share is an active path" lap_path \
  && ok "hub path $LSHP: \"Ethernet from thinkpad\", active" || bad "hub paths: $(hapi "http://$API/api/status" | jq -c '[.paths[] | {iface, kind, label, state}]')"
wait_for 60 "server sees 2 paths for slate7" two_paths \
  && ok "server: slate7 bonds eth0 + the laptop's Ethernet" || bad "server paths for slate7: $(sapi "http://$API/api/status" | jq -c '[.clients[] | select(.name == "slate7") | .paths[]]')"
# the relay's upstream socket (tailbondd's own, connected to the server;
# the laptop's tunnel is mqvpn's): bound to eth1 (ss shows addr%dev), from
# eth1's address, not wlan0's
SOCKS="$(ip netns exec "$NS_L" ss -Hunp dst 10.200.1.1)"
RELAY="$(awk '/"tailbondd"/ {for (i = 1; i <= NF; i++) if ($i ~ /^[0-9.]+(%[^:]+)?:[0-9]+$/ && $i !~ /^10\.200\.1\.1:/) {print $i; exit}}' <<<"$SOCKS")"
case "$RELAY" in
  10.200.4.2%eth1:*) ok "the relay's upstream is bound to eth1: $RELAY -> 10.200.1.1:443 (default route: wlan0)" ;;
  *) bad "relay socket: '${RELAY}' ($(tr -s ' ' <<<"$SOCKS" | tr '\n' ';'))" ;;
esac
RPORT="${RELAY##*:}"
ip netns exec "$NS_S" nft -f - <<EOF
table inet tbhotel {
  chain in { type filter hook input priority 0; iifname "hotel" ip saddr 10.200.4.2 udp sport ${RPORT:-1} udp dport 443 counter; }
}
EOF
ip netns exec "$NS_H" ping -c 100 -i 0.01 -s 1000 -W 2 10.77.0.1 >/dev/null 2>&1 || true
ip netns exec "$NS_H" ping -c 10 -i 0.1 -W 2 10.77.0.1 >/dev/null 2>&1 && ok "ping through the tunnel" || bad "ping through the tunnel failed"
sleep 2
SEEN="$(ip netns exec "$NS_S" nft list chain inet tbhotel in | sed -n 's/.*counter packets \([0-9]*\) bytes.*/\1/p' | head -n1)"
[ "${SEEN:-0}" -gt 0 ] && ok "the server got $SEEN datagrams from 10.200.4.2:$RPORT on its hotel link (the relay path)" \
  || bad "no datagrams from the relay on the hotel link"
LB="$(lapi "http://$API/api/status" | jq '[.sharing.hubs[]? | select(.hub == "slate7") | .up_bytes + .down_bytes] | add // 0')"
[ "${LB:-0}" -gt 0 ] && ok "the laptop relayed $LB bytes for slate7: $(lapi "http://$API/api/status" | jq -c '.sharing | {used_bytes, hubs}')" \
  || bad "no bytes through the laptop: $(lapi "http://$API/api/status" | jq -c .sharing)"
lapi "http://$API/api/status" | jq -e '.state == "connected" and ([.paths[] | select(.iface == "eth1" and .state == "active")] | length == 1)' >/dev/null \
  && ok "the laptop's own tunnel still bonds eth1 while sharing it" || bad "laptop: $(lapi "http://$API/api/status" | jq -c '{state, paths: [.paths[] | {iface, state}]}')"
lapi -X POST -d '{"share_enabled": false}' "http://$API/api/share" >/dev/null
# gone: the hub answers with a status and lists no ready share by that name
# (a hub that doesn't answer isn't proof)
lap_gone() { hapi "http://$API/api/status" | jq -e 'if type == "object" and has("state") then [(.shares // [])[] | select(.label == "Ethernet from thinkpad" and .state == "ready")] | length == 0 else false end' >/dev/null; }
wait_for 30 "the hub drops the laptop's share" lap_gone \
  && ok "sharing off (share_enabled false): $(lapi "http://$API/api/status" | jq -c '.sharing | {state, iface}'); the hub stopped using it" \
  || bad "hub still uses it: $(hapi "http://$API/api/status" | jq -c .shares)"
kill "$LAP_PID"; wait "$LAP_PID" 2>/dev/null || true; LAP_PID=""
ip netns pids "$NS_L" | xargs -r kill 2>/dev/null || true
sapi -X DELETE "http://$API/api/clients/thinkpad" >/dev/null || true

say "client isolation: the hub hears of the phone from the server but can't reach it"
kill "$HUB_PID"; wait "$HUB_PID" 2>/dev/null || true; HUB_PID=""
ip netns pids "$NS_H" | xargs -r kill 2>/dev/null || true
sleep 1
# the hotel Wi-Fi: devices can't talk to each other (mDNS included)
ip netns exec "$NS_H" nft -f - <<'EOF'
table inet isolation {
  chain in  { type filter hook input priority 0; iifname "lan0" ip saddr 192.168.8.23 drop; }
  chain out { type filter hook output priority 0; oifname "lan0" ip daddr { 192.168.8.23, 224.0.0.251 } drop; }
}
EOF
ip netns exec "$NS_P" env PATH="$WORK/ts:$PATH" "$TAILBONDD" -config "$WORK/phn/tailbond.json" >>"$WORK/phone.log" 2>&1 &
PHN_PID=$!
wait_for 60 "phone re-registered" sh -c "ip netns exec $NS_S curl -fsS -m 2 http://$API/api/shares | jq -e '.sharers | length == 1'" || true
rm -rf "$WORK/hub/state"
start_hub
if wait_for 90 "isolation note" sh -c "ip netns exec $NS_H curl -fsS -m 2 http://$API/api/status | jq -e '.share_note | test(\"blocks devices\")'"; then
  ok "hub says: $(hapi "http://$API/api/status" | jq -r .share_note)"
else
  bad "no isolation note: $(hapi "http://$API/api/status" | jq -c '{shares, share_note}')"
fi
hapi "http://$API/api/status" | jq -e '.state == "connected"' >/dev/null && ok "hub still connected over eth0" || bad "hub not connected"

say "the phone is removed on the server: the hub stops using its share"
sapi -X DELETE "http://$API/api/clients/pixel" >/dev/null && ok "pixel removed" || bad "could not remove pixel"
wait_for 40 "hub dropped the removed phone's share" sh -c "ip netns exec $NS_H curl -fsS -m 2 http://$API/api/status | jq -e '[.shares[]?] | length == 0'" \
  && ok "hub no longer lists the removed phone's share" || bad "hub still lists it: $(hapi "http://$API/api/status" | jq -c .shares)"
sapi "http://$API/api/shares" | jq -e '.sharers | length == 0' >/dev/null \
  && ok "server forgot the removed phone's registration" || bad "registry still lists it: $(sapi "http://$API/api/shares" | jq -c .sharers)"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
