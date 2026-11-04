#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
#
# Remove tailbond from this Mac (docs/macos.md):
#
#   sudo /usr/local/lib/tailbond/uninstall.sh            # keeps config and pairing
#   sudo /usr/local/lib/tailbond/uninstall.sh --purge    # removes them too
#
# Stopping tailbondd stops mqvpn, which puts the routes and DNS servers back
# and removes its kill-switch rules. Env as install.sh (TB_PREFIX,
# TB_LAUNCHD_DIR, TB_NO_LAUNCHD).
set -euo pipefail

LABEL=com.tailbond.tailbondd
PREFIX="${TB_PREFIX:-/usr/local}"
ETC="$PREFIX/etc/tailbond" STATE="$PREFIX/var/lib/tailbond"
PLIST="${TB_LAUNCHD_DIR:-/Library/LaunchDaemons}/$LABEL.plist"
PURGE=0
case "${1:-}" in
    --purge) PURGE=1 ;;
    "") ;;
    -h|--help) sed -n '4,11p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "uninstall.sh: unknown option $1 (--purge)" >&2; exit 2 ;;
esac
[ "$(id -u)" = 0 ] || { echo "run as root: sudo $0" >&2; exit 1; }

if [ -z "${TB_NO_LAUNCHD:-}" ]; then
    launchctl bootout "system/$LABEL" 2>/dev/null || true
fi
# a system proxy a phone hub left set (tailbondd puts it back when it
# stops; this covers one that was killed): docs/phone-hub.md
if [ -x "$PREFIX/bin/tailbondd" ] && [ -f "$ETC/tailbond.json" ]; then
    "$PREFIX/bin/tailbondd" restore-proxy -config "$ETC/tailbond.json" || true
fi
# an mqvpn left behind (it restores routes and DNS on SIGTERM)
if [ -f "$STATE/mqvpn.pid" ]; then kill "$(cat "$STATE/mqvpn.pid")" 2>/dev/null || true; fi
rm -f "$PLIST"
rm -f "$PREFIX/bin/tailbondd" "$PREFIX/bin/mqvpn" "$PREFIX/bin/tailbondd.prev" "$PREFIX/bin/mqvpn.prev"
rm -rf "$PREFIX/lib/tailbond"
rm -f "$ETC/install.env"
if [ "$PURGE" = 1 ]; then
    rm -rf "$ETC" "$STATE" "$PREFIX/var/log/tailbond.log"
    echo "==> tailbond removed, with its config, pairing and state"
else
    echo "==> tailbond removed; config and pairing kept in $ETC (--purge removes them)"
fi
