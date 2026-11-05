#!/usr/bin/env bash
# Screenshots of the agent's phone-facing pages (headless Chromium via
# Playwright) → out/ux/web/:
#   web-00*  the phone page ("Get tailbond", http://homebox/) on a phone,
#            on a computer, and after the app link fell back to it;
#   web-01..03  the dashboard's "Add this phone" flow at phone width;
#   web-04   the dashboard's Devices tab with the phone QR card;
#   web-05   "Get tailbond" per platform tab (Android, Mac, iPhone, Linux) at
#            phone and desktop widths; web-06 a Mac picking Android (QR);
#   web-07   the Mac's / laptop's own dashboard opened by "Open tailbond and
#            ask homebox" (/?ask=…: Setup, the server's name filled in);
#   web-08   the Mac's "Use a phone's bonded connection" card: a phone found,
#            none, in use, lost (phone_hub states patched into its status).
#   web-09   the Mac's "Share this Mac's connection" card (to set up, sharing
#            with a hub, unplugged) and a hub listing a computer's share.
#
#   scripts/ux/web-screens.sh [outdir]
#
# Runs a throwaway server-role tailbondd (a stand-in mqvpn that only reports
# "ready", a fake `tailscale` CLI that says this machine is homebox on
# tail1234.ts.net with the Tailscale IP 127.0.0.1) with the dashboard on
# 127.0.0.1:8077 and the phone page on 127.0.0.1:18080 (published as port
# 80, like the Flint's redirect). The Playwright npm package goes into
# ~/.cache/tailbond-ux on first use; Chromium runs in the matching
# mcr.microsoft.com/playwright image (podman), so the host needs no browser
# libraries.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${1:-$REPO/out/ux/web}"
PW="${TAILBOND_UX_NODE:-$HOME/.cache/tailbond-ux}"
TMP="$(mktemp -d)"
# stop_pids: kill only PIDs that were set. `kill 0` (the old default) signals
# the whole process group, which would take the caller down with the test.
stop_pids() {
    local pid
    for pid in "$@"; do [ -n "$pid" ] && kill "$pid" 2>/dev/null || true; done
}
trap 'stop_pids "${AGENT_PID:-}" "${CLIENT_PID:-}"; rm -rf "$TMP"' EXIT

if [[ ! -d "$PW/node_modules/playwright" ]]; then
    mkdir -p "$PW"
    (cd "$PW" && npm init -y >/dev/null && npm i playwright >/dev/null)
fi
PW_VERSION="$(node -p "require('$PW/node_modules/playwright/package.json').version")"

(cd "$REPO/agent" && CGO_ENABLED=0 go build -ldflags "-X main.version=0.1.0-r42" -o "$TMP/tailbondd" ./cmd/tailbondd)
mkdir -p "$TMP/state" "$TMP/hooks" "$TMP/bin" "$OUT"
head -c 27800000 /dev/zero > "$TMP/tailbond.apk"  # offered as GET /tailbond.apk (size shown on the page)
head -c 9400000 /dev/zero > "$TMP/tailbond-macos-arm64.tar.gz"  # GET /tailbond-macos-arm64.tar.gz
printf '#!/bin/sh\necho "mqvpn server ready"\nexec sleep 3600\n' > "$TMP/mqvpn" && chmod +x "$TMP/mqvpn"
cat > "$TMP/bin/tailscale" <<'SH'
#!/bin/sh
# fake tailscale CLI: this machine is homebox (Tailscale IP 127.0.0.1)
case "$1" in
status) echo '{"BackendState":"Running","MagicDNSSuffix":"tail1234.ts.net","CurrentTailnet":{"Name":"alex@example.com"},
 "Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","OS":"linux","TailscaleIPs":["127.0.0.1"],"Online":true},
 "Peer":{"n1":{"HostName":"nas","DNSName":"nas.tail1234.ts.net.","OS":"linux","TailscaleIPs":["100.101.2.2"],"Online":true,"CurAddr":"192.168.1.20:41641"}}}' ;;
debug) echo '{"AdvertiseRoutes":["10.77.0.0/24"]}' ;;
*) exit 1 ;;
esac
SH
chmod +x "$TMP/bin/tailscale"
cat > "$TMP/tailbond.json" <<JSON
{"role":"server","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/state","hooks_dir":"$TMP/hooks",
 "listen":"127.0.0.1:8077","android_apk":"$TMP/tailbond.apk","macos_bundle":"$TMP/tailbond-macos-arm64.tar.gz","landing_port":18080,"landing_public_port":80,
 "server":{"port":443,"public_addr":"home.example.net:443","subnet":"10.77.0.0/24","subnet6":"fd77:77::/112","use_tailscale_cert":false}}
JSON
PATH="$TMP/bin:$PATH" "$TMP/tailbondd" -config "$TMP/tailbond.json" -quiet &
AGENT_PID=$!
for _ in $(seq 50); do curl -fsS -o /dev/null http://127.0.0.1:18080/ && break; sleep 0.2; done
# a client-role agent (the Mac or laptop being added), unpaired, for the
# phone page's "Open tailbond and ask homebox" step: /?ask= on 127.0.0.1:18077
mkdir -p "$TMP/cstate"
cat > "$TMP/client.json" <<JSON
{"role":"client","mqvpn_bin":"$TMP/mqvpn","state_dir":"$TMP/cstate","hooks_dir":"$TMP/hooks","listen":"127.0.0.1:18077",
 "client":{"pairing_uri":"","paths":[],"captive_portal_detect":false}}
JSON
PATH="$TMP/bin:$PATH" "$TMP/tailbondd" -config "$TMP/client.json" -quiet &
CLIENT_PID=$!
for _ in $(seq 50); do curl -fsS -o /dev/null http://127.0.0.1:18077/ && break; sleep 0.2; done

podman run --rm --network=host --userns=keep-id \
    -v "$PW/node_modules:/pw/node_modules:ro,z" -v "$REPO/scripts/ux:/scripts:ro,z" -v "$OUT:/out:z" \
    -e NODE_PATH=/pw/node_modules \
    "mcr.microsoft.com/playwright:v$PW_VERSION-noble" \
    node /scripts/web-screens.mjs http://127.0.0.1:8077/ /out http://127.0.0.1:18080/ http://127.0.0.1:18077/
ls -1 "$OUT"
