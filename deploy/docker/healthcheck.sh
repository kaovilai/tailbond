#!/bin/sh
# tailbond server image health check (HEALTHCHECK, Kubernetes probes): the
# agent answers on its API (GET /api/hello needs no login) and mqvpn runs.
wget -q -T 3 -O - http://127.0.0.1:8077/api/hello 2>/dev/null | grep -q '"tailbond":true' || exit 1
pidof mqvpn >/dev/null || exit 1
