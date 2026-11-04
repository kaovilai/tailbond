#!/usr/bin/env bash
# tailbond server container entrypoint (deploy/docker/Dockerfile, docs/deploy.md).
#
# Environment:
#   TAILBOND_PUBLIC_ADDR   address phones/routers dial (public IP or DNS name). Required.
#   TAILBOND_PORT          tunnel UDP port inside the container (default 443);
#                          publish the same port number on the host
#   TAILBOND_SUBNET        IPv4 tunnel subnet (default 10.77.0.0/24)
#   TAILBOND_UI_PASSWORD   dashboard password; when set the dashboard listens on
#                          0.0.0.0:8077, otherwise only on 127.0.0.1 inside the
#                          container (use `podman exec` + the CLI, or set it)
#   TAILBOND_MODE          speed | streaming | redundant | failover (default speed)
#   TAILBOND_UPDATE_POLICY notify (default: the dashboard says when a release
#                          is out) | off. The image never updates itself: a new
#                          release is a new image tag.
#
# Tailscale isn't run in this container: devices pair with the QR code / link
# from the dashboard, and the server is an internet exit. For tailnet access
# run Tailscale next to it (deploy/kubernetes/tailscale).
set -euo pipefail

cfg=/data/tailbond.json
bundle=/opt/tailbond
: "${TAILBOND_PUBLIC_ADDR:?set TAILBOND_PUBLIC_ADDR to the address clients dial}"
port="${TAILBOND_PORT:-443}"
subnet="${TAILBOND_SUBNET:-10.77.0.0/24}"
mode="${TAILBOND_MODE:-speed}"
upol="${TAILBOND_UPDATE_POLICY:-notify}"
case "$upol" in
  notify | off) ;;
  *) echo "tailbond: TAILBOND_UPDATE_POLICY must be notify or off (the image can't update itself; pull a new tag)" >&2; exit 1 ;;
esac
listen="127.0.0.1:8077"
[ -n "${TAILBOND_UI_PASSWORD:-}" ] && listen="0.0.0.0:8077"

if [ ! -c /dev/net/tun ]; then
  echo "tailbond: /dev/net/tun is missing; run with --device /dev/net/tun --cap-add NET_ADMIN" >&2
  exit 1
fi

mkdir -p /data/state
if [ ! -s "$cfg" ]; then
  "$bundle/bin/tailbondd" -config "$cfg" -init
fi

apk=""
[ -f "$bundle/share/tailbond.apk" ] && apk="$bundle/share/tailbond.apk"

# Re-apply the environment on every start; users/pairings in the file are kept.
tmp="$(mktemp)"
jq --arg addr "$TAILBOND_PUBLIC_ADDR" --argjson port "$port" --arg subnet "$subnet" \
   --arg listen "$listen" --arg pw "${TAILBOND_UI_PASSWORD:-}" --arg mode "$mode" \
   --arg bin "$bundle/bin/mqvpn" --arg hooks "$bundle/hooks" --arg upol "$upol" --arg apk "$apk" '
  .role = "server" | .mqvpn_bin = $bin | .hooks_dir = $hooks | .state_dir = "/data/state"
  | .listen = $listen | .ui_password = $pw | .mode = $mode | .disable_tailscale = true
  | .android_apk = $apk | .update = ((.update // {}) | .policy = $upol)
  | .server.public_addr = $addr | .server.port = $port | .server.subnet = $subnet
  | .server.use_tailscale_cert = false | .server.no_advertise_route = true
' "$cfg" >"$tmp"
cat "$tmp" >"$cfg"
rm -f "$tmp"
chmod 600 "$cfg"

exec "$bundle/bin/tailbondd" -config "$cfg"
