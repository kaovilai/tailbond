#!/usr/bin/env bash
# Remove tailbond installed by install.sh. Keeps /etc/tailbond (config,
# registered devices) and /var/lib/tailbond unless --purge.
#   sudo ./uninstall.sh [--purge] [--dry-run]
exec "$(dirname "$0")/install.sh" --uninstall "$@"
