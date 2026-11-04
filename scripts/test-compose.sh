#!/usr/bin/env bash
# deploy/docker/compose.yaml with the image under test, in each engine
# present: `podman compose` and `docker compose` (CI job "oci-image").
# The settings come from deploy/docker/.env, as docs/deploy.md says. Up,
# the agent answers on 127.0.0.1:8077, the NAT table is loaded, every
# setting reached the config (TAILBOND_UPDATE_POLICY=off), (docker)
# HEALTHCHECK reports healthy, down -v.
#
#   scripts/test-compose.sh localhost/tailbond:ci      (rootful: uses sudo)
#   TB_REQUIRE_COMPOSE=1 ...                           a missing engine fails
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[ $# -eq 1 ] || { echo "usage: $0 IMAGE (in root's podman storage)" >&2; exit 2; }
IMAGE="$1"
SUDO=(sudo --preserve-env=PODMAN_COMPOSE_PROVIDER)
cd "$ROOT/deploy/docker"
[ ! -e .env ] || { echo "deploy/docker/.env exists (yours?): move it away first" >&2; exit 2; }
cat >.env <<EOF
TAILBOND_IMAGE=$IMAGE
TAILBOND_PUBLIC_ADDR=203.0.113.7
TAILBOND_PORT=4443
TAILBOND_UI_PASSWORD=compose-$RANDOM
TAILBOND_MODE=redundant
TAILBOND_UPDATE_POLICY=off
EOF
rc=0
started=()
cleanup() {
  local e
  for e in "${started[@]}"; do "${SUDO[@]}" "$e" compose down -v >/dev/null 2>&1 || true; done
  rm -f .env
}
trap cleanup EXIT

wait_for() { # wait_for <seconds> <command...>
  local t=$1; shift
  for _ in $(seq 1 "$t"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}
hello() { curl -fsS -m 2 http://127.0.0.1:8077/api/hello | jq -e '.tailbond and .role == "server"'; }

check() { # check ENGINE: compose is up
  local eng=$1 id
  id="$(sudo "$eng" ps -q --filter label=com.docker.compose.service=tailbond | head -n1)"
  if wait_for 60 hello; then echo "  PASS $eng compose: dashboard answers on 127.0.0.1:8077"; else echo "  FAIL $eng compose: no dashboard"; rc=1; fi
  # the server-up hook loads it once mqvpn is up, after the agent answers
  if wait_for 30 sh -c "sudo $eng exec $id nft list table inet tailbond | grep -q masquerade"; then
    echo "  PASS $eng compose: NAT table loaded (cap_drop ALL + NET_ADMIN, NET_BIND_SERVICE)"
  else
    echo "  FAIL $eng compose: no NAT table"; rc=1
  fi
  if sudo "$eng" exec "$id" jq -e '.update.policy == "off" and .mode == "redundant" and .server.port == 4443
      and .server.public_addr == "203.0.113.7" and .listen == "0.0.0.0:8077"' /data/tailbond.json >/dev/null; then
    echo "  PASS $eng compose: every .env setting reached tailbond.json (update policy off, mode, port, address, password)"
  else
    echo "  FAIL $eng compose: config $(sudo "$eng" exec "$id" jq -c '{update, mode, listen, server: {port: .server.port, public_addr: .server.public_addr}}' /data/tailbond.json)"; rc=1
  fi
  if [ "$eng" = docker ]; then
    if wait_for 90 sh -c "[ \"\$(sudo docker inspect -f '{{.State.Health.Status}}' $id)\" = healthy ]"; then
      echo "  PASS docker: HEALTHCHECK healthy"
    else
      echo "  FAIL docker: health $(sudo docker inspect -f '{{json .State.Health}}' "$id")"; rc=1
    fi
  fi
  [ "$rc" = 0 ] || "${SUDO[@]}" "$eng" compose logs --tail 40 || true
}

if "${SUDO[@]}" podman compose version >/dev/null 2>&1; then
  echo "==> podman compose"
  started+=(podman)
  "${SUDO[@]}" podman compose up -d
  check podman
  "${SUDO[@]}" podman compose down -v
else
  echo "  SKIP podman compose: no compose provider"; [ -z "${TB_REQUIRE_COMPOSE:-}" ] || rc=1
fi

if command -v docker >/dev/null 2>&1 && sudo docker compose version >/dev/null 2>&1; then
  echo "==> docker compose"
  sudo podman save --format docker-archive "$IMAGE" | sudo docker load
  started+=(docker)
  "${SUDO[@]}" docker compose up -d
  check docker
  "${SUDO[@]}" docker compose down -v
else
  echo "  SKIP docker compose: no docker"; [ -z "${TB_REQUIRE_COMPOSE:-}" ] || rc=1
fi
exit "$rc"
