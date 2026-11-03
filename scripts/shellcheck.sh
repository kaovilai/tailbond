#!/usr/bin/env bash
# ShellCheck over every shell script in the repo, as CI runs it (job
# "ShellCheck"). The OpenWrt files are busybox ash (shell=busybox,
# packaging/openwrt/.shellcheckrc), which needs ShellCheck 0.10 or newer;
# CI installs a pinned release (SHELLCHECK_VERSION in .github/workflows/ci.yml).
#
#   scripts/shellcheck.sh            (SHELLCHECK=/path/to/shellcheck to pick one)
set -euo pipefail

cd "$(dirname "$0")/.."
SC="${SHELLCHECK:-shellcheck}"
command -v "$SC" >/dev/null || { echo "shellcheck not found (https://github.com/koalaman/shellcheck/releases)" >&2; exit 2; }
ver="$("$SC" --version | sed -n 's/^version: //p')"
# shell=busybox: 0.10.0+
if [ "$(printf '%s\n' 0.10.0 "$ver" | sort -V | head -n1)" != 0.10.0 ]; then
    echo "shellcheck $ver is too old: the OpenWrt scripts need 0.10.0 or newer (shell=busybox)" >&2
    exit 2
fi
echo "shellcheck $ver"

rc=0
# Note: git pathspec * also matches /, so packaging/openwrt/*.sh includes
# the .sh files under packaging/openwrt/files/ (busybox; .shellcheckrc).
mapfile -t files < <(git ls-files 'scripts/*.sh' 'scripts/**/*.sh' 'agent/*.sh' \
    'android/scripts/*.sh' 'packaging/fedora/*.sh' 'packaging/fedora/hooks/*' \
    'packaging/openwrt/*.sh' 'packaging/linux/*.sh' 'packaging/linux/hooks/*')
"$SC" -S warning "${files[@]}" || rc=1

# OpenWrt hooks and init scripts (no .sh suffix): busybox ash
mapfile -t files < <(git ls-files 'packaging/openwrt/hooks/*' 'packaging/openwrt/control/postinst' \
    'packaging/openwrt/control/prerm' 'packaging/openwrt/control/postrm' 'packaging/openwrt/files/etc/init.d/*' \
    'packaging/openwrt/files/etc/uci-defaults/*' 'packaging/openwrt/files/usr/lib/tailbond/*' \
    'packaging/openwrt/files/lib/upgrade/*')
"$SC" -S warning --shell=busybox -x "${files[@]}" || rc=1
exit "$rc"
