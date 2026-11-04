#!/usr/bin/env bash
# deploy/docker/compose.yaml with the image under test, in each engine
# present: `podman compose` and `docker compose` (CI job "oci-image").
# Up, the agent answers on 127.0.0.1:8077, the NAT table is loaded, (docker)
# HEALTHCHECK reports healthy, down -v.
#
#   scripts/test-compose.sh localhost/tailbond:ci      (rootful: uses sudo)
#   TB_REQUIRE_COMPOSE=1 ...                           a missing engine fails
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[ $# -eq 1 ] || { echo "usage: $0 IMAGE (in root's podman storage)" >&2; exit 2; }
IMAGE="$1"
export TAILBOND_IMAGE="$IMAGE" TAILBOND_PUBLIC_ADDR=203.0.113.7 TAILBOND_PORT=4443 TAILBOND_UI_PASSWORD="compose-$RANDOM"
# shellcheck disable=SC2054 # one sudo option, commas and all
SUDO=(sudo --preserve-env=TAILBOND_IMAGE,TAILBOND_PUBLIC_ADDR,TAILBOND_UI_PASSWORD,TAILBOND_PORT,PODMAN_COMPOSE_PROVIDER)
cd "$ROOT/deploy/docker"
rc=0

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
  "${SUDO[@]}" podman compose up -d
  check podman
  "${SUDO[@]}" podman compose down -v
else
  echo "  SKIP podman compose: no compose provider"; [ -z "${TB_REQUIRE_COMPOSE:-}" ] || rc=1
fi

if command -v docker >/dev/null 2>&1 && sudo docker compose version >/dev/null 2>&1; then
  echo "==> docker compose"
  sudo podman save --format docker-archive "$IMAGE" | sudo docker load
  "${SUDO[@]}" docker compose up -d
  check docker
  "${SUDO[@]}" docker compose down -v
else
  echo "  SKIP docker compose: no docker"; [ -z "${TB_REQUIRE_COMPOSE:-}" ] || rc=1
fi
exit "$rc"
