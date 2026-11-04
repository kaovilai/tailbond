#!/usr/bin/env bash
# End-to-end test of the server image (deploy/docker, docs/deploy.md): the
# image runs in rootful podman with only NET_ADMIN + NET_BIND_SERVICE and
# /dev/net/tun (or in a kind cluster from deploy/kubernetes), and a real
# tailbond client (tailbondd + mqvpn, the bundle's binaries) in a network
# namespace bonds TWO paths to it (eth0 clean, wlan0 with netem) through the
# published UDP port.
#
#   scripts/e2e-oci.sh IMAGE BUNDLE.tar.gz              # podman (default)
#   TB_OCI_BACKEND=kind scripts/e2e-oci.sh IMAGE BUNDLE.tar.gz
#   KEEP=1 ...                                          # leave it all running
#
# Checks: the dashboard answers (login, status), a pairing link works, the
# server lists the client with 2 active paths, internet traffic from the
# client goes through the container's NAT (a fake internet the client can
# reach only through the tunnel sees the server side's address, never the
# tunnel's 10.77.0.x), the health check passes, and after the container
# (pod) is replaced on the same volume (PVC) the server keeps its clients and
# certificate: the client reconnects with its old pairing.
#
# kind: needs a cluster whose node maps host UDP 443 to NodePort 30443
# (scripts/ci/kind-config.yaml) and the image loaded; the manifests are
# deploy/kubernetes/base with the image, the address and nodePort set here.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[ $# -eq 2 ] || { echo "usage: $0 IMAGE tailbond-linux-<arch>.tar.gz" >&2; exit 2; }
IMAGE="$1"
BUNDLE="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
BACKEND="${TB_OCI_BACKEND:-podman}"
if [ "$(id -u)" -ne 0 ]; then
  exec sudo --preserve-env=KEEP,VERBOSE,TB_OCI_BACKEND,KUBECONFIG "$0" "$IMAGE" "$BUNDLE"
fi
for c in curl jq python3 tc; do command -v "$c" >/dev/null || { echo "$c missing"; exit 2; }; done
case "$BACKEND" in
  podman) command -v podman >/dev/null || { echo "podman missing"; exit 2; } ;;
  kind) command -v kubectl >/dev/null || { echo "kubectl missing"; exit 2; } ;;
  *) echo "TB_OCI_BACKEND must be podman or kind" >&2; exit 2 ;;
esac

NS_C=tb-oci-cli
NS_I=tb-oci-inet
SERVER_IP=10.200.1.1
PORT=443
INET_IP=198.51.100.1
HOST_INET_IP=198.51.100.254
CTR=tb-oci
VOL=tb-oci-data
NET=tb-oci-net # a network of its own, the container on a fixed address (docs/deploy.md)
KNS=tailbond
UIPW="e2e-oci-$RANDOM$RANDOM"
SAPI=127.0.0.1:18077 # the server's dashboard, published (podman) or port-forwarded (kind)
CAPI=127.0.0.1:8077  # the client agent, inside its namespace

WORK="$(mktemp -d /tmp/tailbond-e2e-oci.XXXXXX)"
PASS=0 FAIL=0
PF_PID="" IPT_RULE=0
say()  { printf '\033[1m==> %s\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; PASS=$((PASS + 1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL + 1)); }
capi() { ip netns exec "$NS_C" curl -fsS -m 5 -H 'Content-Type: application/json' "$@"; }
sapi() { curl -fsS -m 5 -b "$WORK/cookies" -H 'Content-Type: application/json' "$@"; }
wait_for() { # wait_for <seconds> <description> <command...>
  local t=$1 what=$2; shift 2
  for _ in $(seq 1 "$t"); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  echo "  timed out after ${t}s waiting for: $what"
  return 1
}

srv_logs() {
  case "$BACKEND" in
    podman) podman logs --tail 60 "$CTR" 2>&1 || true ;;
    kind) kubectl -n "$KNS" logs deploy/tailbond -c tailbond --tail 60 2>&1 || true ;;
  esac
}
srv_id() { # the running container / pod
  case "$BACKEND" in
    podman) podman inspect -f '{{.Id}}' "$CTR" 2>/dev/null ;;
    kind) kubectl -n "$KNS" get pod -l app.kubernetes.io/name=tailbond -o jsonpath='{.items[*].metadata.uid}' 2>/dev/null ;;
  esac
}
srv_exec() {
  case "$BACKEND" in
    podman) podman exec "$CTR" "$@" ;;
    kind) kubectl -n "$KNS" exec deploy/tailbond -c tailbond -- "$@" ;;
  esac
}

cleanup() {
  local rc=$?
  set +e # every step runs, whatever failed before
  if [ -n "${KEEP:-}" ]; then echo "KEEP: namespaces, server and client left running; work dir $WORK"; return; fi
  if [ "$rc" -ne 0 ] || [ "$FAIL" -ne 0 ] || [ -n "${VERBOSE:-}" ]; then
    echo "--- server log (tail)"; srv_logs
    echo "--- client log (tail)"; tail -n 40 "$WORK/client.log" 2>/dev/null || true
  fi
  [ -z "$PF_PID" ] || kill "$PF_PID" 2>/dev/null || true
  for ns in "$NS_C" "$NS_I"; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill 2>/dev/null || true
  done
  sleep 1
  for ns in "$NS_C" "$NS_I"; do
    ip netns pids "$ns" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    ip netns del "$ns" 2>/dev/null || true
  done
  for l in tboci-a tboci-b tboci-i; do ip link del "$l" 2>/dev/null || true; done
  [ "$IPT_RULE" != 1 ] || iptables -D FORWARD -i tboci-+ -j ACCEPT -m comment --comment tb-e2e-oci 2>/dev/null || true
  case "$BACKEND" in
    podman)
      podman rm -f "$CTR" >/dev/null 2>&1; podman volume rm -f "$VOL" >/dev/null 2>&1
      podman network rm -f "$NET" >/dev/null 2>&1
      ;;
    kind) kubectl delete ns "$KNS" --wait=false >/dev/null 2>&1 ;;
  esac
  rm -rf "$WORK"
}
trap cleanup EXIT

say "client binaries from $(basename "$BUNDLE")"
mkdir -p "$WORK/b" "$WORK/cli/hooks"
tar -C "$WORK/b" --strip-components=1 -xzf "$BUNDLE"
[ -x "$WORK/b/bin/tailbondd" ] && [ -x "$WORK/b/bin/mqvpn" ] && ok "tailbondd $("$WORK/b/bin/tailbondd" -version | awk '{print $2}') + mqvpn" || { bad "bundle"; exit 1; }

say "network: client $NS_C (eth0, wlan0) -> this host $SERVER_IP:$PORT/udp -> the server; fake internet $INET_IP"
for ns in "$NS_C" "$NS_I"; do ip netns del "$ns" 2>/dev/null || true; ip netns add "$ns"; ip -n "$ns" link set lo up; done
for l in tboci-a tboci-b tboci-i; do ip link del "$l" 2>/dev/null || true; done
ip link add tboci-a type veth peer name tboci-ac
ip link add tboci-b type veth peer name tboci-bc
ip link add tboci-i type veth peer name tboci-ii
ip link set tboci-ac netns "$NS_C"; ip -n "$NS_C" link set tboci-ac name eth0
ip link set tboci-bc netns "$NS_C"; ip -n "$NS_C" link set tboci-bc name wlan0
ip link set tboci-ii netns "$NS_I"; ip -n "$NS_I" link set tboci-ii name eth0
ip addr add "$SERVER_IP/24" dev tboci-a
ip addr add 10.200.2.1/24 dev tboci-b
ip addr add "$HOST_INET_IP/24" dev tboci-i
ip -n "$NS_C" addr add 10.200.1.2/24 dev eth0
ip -n "$NS_C" addr add 10.200.2.2/24 dev wlan0
ip -n "$NS_I" addr add "$INET_IP/24" dev eth0
for l in tboci-a tboci-b tboci-i; do ip link set "$l" up; done
for l in eth0 wlan0; do ip -n "$NS_C" link set "$l" up; done
ip -n "$NS_I" link set eth0 up
# No default route in the client: the fake internet is reachable only
# through the tunnel. The server address is reachable over both paths.
ip -n "$NS_C" route add "$SERVER_IP/32" via 10.200.2.1 dev wlan0 metric 200
ip netns exec "$NS_C" tc qdisc add dev wlan0 root netem delay 30ms 5ms loss 1%
# replies from the fake internet arrive on tb0, which has no route back
# there (curl --interface tb0 below): no reverse-path filter in the client
ip netns exec "$NS_C" sysctl -qw net.ipv4.conf.all.rp_filter=0 net.ipv4.conf.default.rp_filter=0
sysctl -qw net.ipv4.ip_forward=1
# Docker on the same host (GitHub runners) sets the FORWARD policy to DROP;
# let the test links' traffic reach the container like a host firewall would.
if iptables -S FORWARD 2>/dev/null | grep -q -- '^-P FORWARD DROP'; then
  iptables -I FORWARD -i tboci-+ -j ACCEPT -m comment --comment tb-e2e-oci && IPT_RULE=1
fi
cat >"$WORK/echo.py" <<'EOF'
import http.server
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        body = self.client_address[0].encode()
        self.send_response(200); self.send_header("Content-Length", str(len(body))); self.end_headers(); self.wfile.write(body)
http.server.ThreadingHTTPServer(("0.0.0.0", 8099), H).serve_forever()
EOF
ip netns exec "$NS_I" python3 "$WORK/echo.py" >/dev/null 2>&1 &
wait_for 10 "fake internet" curl -fsS -m 2 "http://$INET_IP:8099/" && ok "fake internet answers this host" || bad "fake internet"
ip netns exec "$NS_C" curl -fsS -m 3 "http://$INET_IP:8099/" >/dev/null 2>&1 \
  && bad "the client reaches the fake internet without the tunnel" || ok "the client can't reach the fake internet without the tunnel"

# srv_start: run the server (podman: a new container on the same volume;
# kind: apply the manifests, or replace the pod) and wait until it answers.
srv_start() {
  case "$BACKEND" in
    podman)
      podman rm -f "$CTR" >/dev/null 2>&1 || true
      podman network exists "$NET" || podman network create --subnet 10.89.77.0/24 "$NET" >/dev/null
      podman run -d --name "$CTR" --network "$NET" --ip 10.89.77.2 \
        --cap-drop all --cap-add NET_ADMIN --cap-add NET_BIND_SERVICE --device /dev/net/tun \
        --sysctl net.ipv4.ip_forward=1 --sysctl net.ipv6.conf.all.forwarding=1 \
        -p "$PORT:$PORT/udp" -p "$SAPI:8077/tcp" -v "$VOL:/data" \
        -e TAILBOND_PUBLIC_ADDR="$SERVER_IP" -e TAILBOND_PORT="$PORT" -e TAILBOND_UI_PASSWORD="$UIPW" \
        "$IMAGE" >/dev/null
      ;;
    kind)
      if [ "${1:-}" = again ]; then
        kubectl -n "$KNS" rollout restart deploy/tailbond >/dev/null
      else
        mkdir -p "$WORK/k8s"
        cat >"$WORK/k8s/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - $(realpath --relative-to="$WORK/k8s" "$ROOT/deploy/kubernetes/base")
images:
  - name: ghcr.io/kaovilai/tailbond
    newName: ${IMAGE%:*}
    newTag: ${IMAGE##*:}
configMapGenerator:
  - name: tailbond
    behavior: merge
    literals:
      - TAILBOND_PUBLIC_ADDR=$SERVER_IP
secretGenerator:
  - name: tailbond-ui
    namespace: $KNS
    options:
      disableNameSuffixHash: true
    literals:
      - TAILBOND_UI_PASSWORD=$UIPW
patches:
  - target: {kind: Service, name: tailbond}
    patch: |-
      - op: add
        path: /spec/ports/0/nodePort
        value: 30443
EOF
        kubectl apply -k "$WORK/k8s" >/dev/null
      fi
      kubectl -n "$KNS" rollout status deploy/tailbond --timeout=180s >/dev/null
      [ -z "$PF_PID" ] || kill "$PF_PID" 2>/dev/null || true
      kubectl -n "$KNS" port-forward deploy/tailbond "${SAPI##*:}:8077" >"$WORK/pf.log" 2>&1 &
      PF_PID=$!
      ;;
  esac
  wait_for 90 "server dashboard" curl -fsS -m 2 "http://$SAPI/api/hello"
}

say "server ($BACKEND): $IMAGE"
srv_start || { bad "server did not start"; exit 1; }
ok "the server's dashboard answers on $SAPI"
code="$(curl -s -m 5 -o /dev/null -w '%{http_code}' "http://$SAPI/api/status")"
[ "$code" = 401 ] && ok "dashboard API needs the password (401 without it)" || bad "GET /api/status without login: $code"
curl -fsS -m 5 "http://$SAPI/" | grep -qi '<html' && ok "dashboard page served" || bad "no dashboard page"
login() { curl -fsS -m 5 -c "$WORK/cookies" -H 'Content-Type: application/json' -d "{\"password\":\"$UIPW\"}" "http://$SAPI/api/login" >/dev/null; }
login && ok "login with TAILBOND_UI_PASSWORD" || bad "login"
wait_for 30 "server connected" sh -c "curl -fsS -m 2 -b '$WORK/cookies' http://$SAPI/api/status | jq -e '.state == \"connected\" and .role == \"server\"'" \
  && ok "server role, mqvpn up" || bad "server status: $(sapi "http://$SAPI/api/status" | jq -c '{state, role, error}')"
sapi "http://$SAPI/api/status" | jq -e '.update.policy == "notify" and .update.method == "none"' >/dev/null \
  && ok "updates: notify only, no self-update (method none)" || bad "update: $(sapi "http://$SAPI/api/status" | jq -c .update)"
# the server-up hook runs once mqvpn is up, just after the status says so
nat_loaded() { srv_exec nft list table inet tailbond 2>/dev/null | grep -q masquerade; }
wait_for 30 "nft table" nat_loaded && ok "server-up hook loaded the NAT (nft table inet tailbond)" || bad "no nft table in the container"
case "$BACKEND" in
  podman) podman healthcheck run "$CTR" && ok "image HEALTHCHECK passes (podman healthcheck run)" || bad "health check fails" ;;
  kind) srv_exec tailbond-healthcheck && ok "tailbond-healthcheck passes" || bad "health check fails" ;;
esac

say "pair a client over two paths"
URI="$(sapi -X POST -d '{"name":"e2e-oci"}' "http://$SAPI/api/clients" | jq -r .uri)"
B64="${URI#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
PAYLOAD="$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d)"
CERT="$(jq -r .cert_sha256 <<<"$PAYLOAD")"
jq -e --arg s "$SERVER_IP:$PORT" '.server == $s and (.cert_sha256 | length) == 64' <<<"$PAYLOAD" >/dev/null \
  && ok "pairing link for $SERVER_IP:$PORT (cert ${CERT:0:12}…)" || bad "payload: $PAYLOAD"
cat >"$WORK/cli/tailbond.json" <<EOF
{"role":"client","mqvpn_bin":"$WORK/b/bin/mqvpn","state_dir":"$WORK/cli/state","hooks_dir":"$WORK/cli/hooks","listen":"$CAPI",
 "disable_tailscale":true,"client":{"pairing_uri":"$URI","paths":[{"iface":"eth0","enabled":true,"priority":"primary"},{"iface":"wlan0","enabled":true,"priority":"primary"}],
 "no_dns":true,"manage_routes":false,"captive_portal_detect":false},"update":{"policy":"off"}}
EOF
ip netns exec "$NS_C" "$WORK/b/bin/tailbondd" -config "$WORK/cli/tailbond.json" >"$WORK/client.log" 2>&1 &
wait_for 45 "client connected" sh -c "ip netns exec $NS_C curl -fsS -m 2 http://$CAPI/api/status | jq -e '.state == \"connected\"'" \
  && ok "client connected" || bad "client: $(capi "http://$CAPI/api/status" | jq -c '{state, error}')"
two_paths() { sapi "http://$SAPI/api/status" | jq -e '.clients[] | select(.name == "e2e-oci" and .connected) | [.paths[] | select(.state == "active")] | length >= 2'; }
wait_for 45 "2 active paths" two_paths && ok "server lists e2e-oci with 2 active paths" \
  || bad "server paths: $(sapi "http://$SAPI/api/status" | jq -c '[.clients[] | {name, connected, paths: [.paths[]? | {state}]}]')"
ip netns exec "$NS_C" ping -c 5 -i 0.3 -W 2 10.77.0.1 >/dev/null 2>&1 && ok "ping 10.77.0.1 (the server) through tb0" || bad "no ping through the tunnel"

# through_nat: the fake internet's view of the client's traffic over tb0
through_nat() {
  local seen
  seen="$(ip netns exec "$NS_C" curl -fsS -m 5 --interface tb0 "http://$INET_IP:8099/" 2>/dev/null)" || { bad "client -> internet through the tunnel failed"; return; }
  case "$seen" in
    10.77.*) bad "the internet saw the tunnel address $seen: no NAT" ;;
    *) ok "client -> internet through the tunnel and the container's NAT (seen as $seen)" ;;
  esac
}
through_nat

say "replace the server on the same volume: state persists"
OLD_ID="$(srv_id)"
case "$BACKEND" in
  podman)
    podman rm -f "$CTR" >/dev/null
    # with the server stopped: a start whose config render fails (a bad
    # TAILBOND_PORT) exits and leaves tailbond.json byte for byte as it was
    data_state() { podman run --rm --entrypoint sh -v "$VOL:/data" "$IMAGE" -c 'sha256sum /data/tailbond.json; ls -A /data'; }
    before="$(data_state)"
    if podman run --rm --device /dev/net/tun -v "$VOL:/data" -e TAILBOND_PUBLIC_ADDR="$SERVER_IP" \
        -e TAILBOND_PORT=not-a-port "$IMAGE" >"$WORK/bad-render.log" 2>&1; then
      bad "a start with TAILBOND_PORT=not-a-port succeeded"
    elif grep -q 'it is unchanged' "$WORK/bad-render.log" && [ "$(data_state)" = "$before" ]; then
      ok "a failed config render leaves tailbond.json intact (no temp file left)"
    else
      bad "after a failed render: $(cat "$WORK/bad-render.log"); before: $before; after: $(data_state)"
    fi
    ;;
esac
srv_start again || { bad "server did not come back"; exit 1; }
NEW_ID="$(srv_id)"
[ -n "$NEW_ID" ] && [ "$NEW_ID" != "$OLD_ID" ] && ok "a new $([ "$BACKEND" = kind ] && echo pod || echo container) on the same $([ "$BACKEND" = kind ] && echo PVC || echo volume)" || bad "server not replaced ($OLD_ID -> $NEW_ID)"
login
wait_for 90 "client reconnected" sh -c "curl -fsS -m 2 -b '$WORK/cookies' http://$SAPI/api/clients | jq -e '.[] | select(.name == \"e2e-oci\" and .connected)'" \
  && ok "the new server still knows e2e-oci, and it reconnected with its old pairing" || bad "after restart: $(sapi "http://$SAPI/api/clients" | jq -c .)"
URI2="$(sapi -X POST -d '{"name":"e2e-oci-2"}' "http://$SAPI/api/clients" | jq -r .uri)"
B64="${URI2#tailbond://pair?d=}"
while [ $(( ${#B64} % 4 )) -ne 0 ]; do B64="$B64="; done
[ "$(printf '%s' "$B64" | tr '_-' '/+' | base64 -d | jq -r .cert_sha256)" = "$CERT" ] && ok "same server certificate after the restart" || bad "the certificate changed"
wait_for 45 "2 active paths again" two_paths && ok "2 active paths again" || bad "paths after restart"
wait_for 30 "tunnel traffic" ip netns exec "$NS_C" ping -c1 -W2 10.77.0.1 || true
through_nat

echo
printf '\033[1mresult: %d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
