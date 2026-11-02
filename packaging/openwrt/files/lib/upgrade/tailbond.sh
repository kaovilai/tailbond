# SPDX-License-Identifier: Apache-2.0
# shellcheck shell=busybox
#
# Sourced by sysupgrade (include /lib/upgrade) before a firmware upgrade
# that keeps settings. Registers a hook that runs after sysupgrade listed the
# files to keep (add_conffiles: /etc/sysupgrade.conf, /lib/upgrade/keep.d/*,
# changed conffiles) and, for a real flash (IMAGE set, not `sysupgrade -b`):
# - makes sure /etc/tailbond/pkg/ has the running package (asks the agent
#   for it when it was installed by hand) and the reinstall script;
# - makes sure /etc/rc.local runs that script at boot;
# - lists those files explicitly (add_conffiles already expanded the
#   directories) and drops the previous ipk from the list.
# The new firmware then boots without the package but with these files, and
# the rc.local line reinstalls it (docs/glinet.md, "Firmware upgrades").
# Same mechanism as GL's own /lib/upgrade/99-migrate-downgrade.sh.

tailbond_sysupgrade_keep() {
	[ -n "${IMAGE:-}" ] || return 0
	[ -n "${1:-}" ] || return 0
	[ -x /usr/lib/tailbond/pkg-cache ] || return 0
	/usr/lib/tailbond/pkg-cache seed >/dev/null 2>&1
	/usr/lib/tailbond/pkg-cache install-reinstall >/dev/null 2>&1
	/usr/lib/tailbond/pkg-cache rc-local-add >/dev/null 2>&1
	/usr/lib/tailbond/pkg-cache keep-list "$1" >/dev/null 2>&1
	return 0
}

case " ${sysupgrade_init_conffiles:-} " in
*" tailbond_sysupgrade_keep "*) ;;
*" add_conffiles "* | *" add_overlayfiles "*)
	sysupgrade_init_conffiles="$sysupgrade_init_conffiles tailbond_sysupgrade_keep"
	;;
esac
