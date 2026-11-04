#!/usr/bin/env bash
# tailbond server container entrypoint. UNTESTED TEMPLATE (see docs/deploy.md).
#
# Environment:
#   TAILBOND_PUBLIC_ADDR   address phones/routers dial (public IP or DNS name). Required.
#   TAILBOND_PORT          tunnel UDP port inside the container (default 443)
#   TAILBOND_SUBNET        IPv4 tunnel subnet (default 10.77.0.0/24)
#   TAILBOND_UI_PASSWORD   dashboard password; when set the dashboard listens on
#                          0.0.0.0:8077, otherwise only on 127.0.0.1 inside the
#                          container (use `docker exec` + the CLI, or set it)
#   TAILBOND_MODE          speed | streaming | redundant | failover (default speed)
#
# Tailscale isn't run in the container: devices pair with the QR code / link
# from the dashboard, and the server is an internet exit (no tailnet access).
set -euo pipefail

cfg=/data/tailbond.json
bundle=/opt/tailbond
mkdir -p /data/state

: "${TAILBOND_PUBLIC_ADDR:?set TAILBOND_PUBLIC_ADDR to the address clients dial}"
port="${TAILBOND_PORT:-443}"
subnet="${TAILBOND_SUBNET:-10.77.0.0/24}"
mode="${TAILBOND_MODE:-speed}"
listen="127.0.0.1:8077"
[ -n "${TAILBOND_UI_PASSWORD:-}" ] && listen="0.0.0.0:8077"

if [ ! -s "$cfg" ]; then
  "$bundle/bin/tailbondd" -config "$cfg" -init
fi

# Re-apply the environment on every start; users/pairings in the file are kept.
tmp="$(mktemp)"
jq --arg addr "$TAILBOND_PUBLIC_ADDR" --argjson port "$port" --arg subnet "$subnet" \
   --arg listen "$listen" --arg pw "${TAILBOND_UI_PASSWORD:-}" --arg mode "$mode" \
   --arg bin "$bundle/bin/mqvpn" --arg hooks "$bundle/hooks" '
  .role = "server" | .mqvpn_bin = $bin | .hooks_dir = $hooks | .state_dir = "/data/state"
  | .listen = $listen | .ui_password = $pw | .mode = $mode | .disable_tailscale = true
  | .server.public_addr = $addr | .server.port = $port | .server.subnet = $subnet
  | .server.use_tailscale_cert = false | .server.no_advertise_route = true
' "$cfg" >"$tmp"
mv "$tmp" "$cfg"
chmod 600 "$cfg"

if [ ! -c /dev/net/tun ]; then
  echo "tailbond: /dev/net/tun is missing; run with --device /dev/net/tun --cap-add NET_ADMIN" >&2
  exit 1
fi

exec "$bundle/bin/tailbondd" -config "$cfg"
