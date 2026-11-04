#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
#
# Shared helpers for the OpenWrt hooks in /usr/lib/tailbond/hooks
# (contract: docs/architecture.md "Platform hooks"). Sourced, not run.
#
# Everything here goes through UCI so it works the same on firewall3
# (iptables) and firewall4 (nftables): both read /etc/config/firewall.
# Every helper is idempotent: running a hook twice changes nothing the
# second time, and reloads only happen when UCI actually changed.
#
# Test seams (env): TB_INITD (default /etc/init.d), TB_STATE (default
# /var/run/tailbond), TB_SYSNET (default /sys/class/net).

TB_INITD="${TB_INITD:-/etc/init.d}"
TB_STATE="${TB_STATE:-/var/run/tailbond}"
TB_SYSNET="${TB_SYSNET:-/sys/class/net}"
TB_TUN="${TB_TUN:-tb0}"
mkdir -p "$TB_STATE"

log() { logger -t "tailbond-hook" "$*" 2>/dev/null; echo "$*"; }

tbcfg() { uci -q get "tailbond.$1"; }
tbcfg_or() {
	local v
	v="$(tbcfg "$1")"
	echo "${v:-$2}"
}
is_true() { case "$1" in 1 | on | true | yes | enabled) return 0 ;; esac; return 1; }

# uset CONFIG.SECTION[.OPTION] VALUE: set only if different (keeps
# `uci changes` empty on a re-run, so nothing reloads needlessly).
uset() {
	[ "$(uci -q get "$1")" = "$2" ] || uci set "$1=$2"
}
# ulist CONFIG.SECTION.OPTION VALUE: add a list item once.
ulist() {
	local v
	for v in $(uci -q get "$1"); do
		[ "$v" = "$2" ] && return 0
	done
	uci add_list "$1=$2"
}
# udel_list CONFIG.SECTION.OPTION VALUE
udel_list() {
	local v
	for v in $(uci -q get "$1"); do
		[ "$v" = "$2" ] && { uci del_list "$1=$2"; return 0; }
	done
	return 0
}
# udel CONFIG.SECTION: delete if present.
udel() {
	uci -q get "$1" >/dev/null && uci delete "$1"
	return 0
}

# commit_reload CONFIG SERVICE...: commit CONFIG; if it had pending changes,
# reload the given services. Returns 0 if a reload happened.
commit_reload() {
	local conf="$1" svc
	shift
	[ -n "$(uci -q changes "$conf")" ] || return 1
	uci commit "$conf"
	for svc in "$@"; do
		case "$svc" in
		network) ubus call network reload 2>/dev/null || "$TB_INITD/network" reload ;;
		*) [ -x "$TB_INITD/$svc" ] && "$TB_INITD/$svc" reload ;;
		esac
	done
	return 0
}

# No netifd interface for the tunnel: bringing up a 'proto none' interface
# on $TB_TUN flushes the addresses mqvpn put there, which kills the tunnel.
# The firewall zone matches the device instead (as GL's tailscale0 zone
# does). Older installs created network.tailbond; drop it.
ensure_network() { udel network.tailbond; }
remove_network() { udel network.tailbond; }

# firewall zone 'tailbond' for the tunnel. $1 = input policy.
ensure_zone() {
	uset firewall.tailbond zone
	uset firewall.tailbond.name tailbond
	udel firewall.tailbond.network
	uset firewall.tailbond.device "$TB_TUN"
	uset firewall.tailbond.input "${1:-REJECT}"
	uset firewall.tailbond.output ACCEPT
	uset firewall.tailbond.forward REJECT
	uset firewall.tailbond.mtu_fix 1
}

# ensure_forwarding NAME SRC DEST
ensure_forwarding() {
	uset "firewall.$1" forwarding
	uset "firewall.$1.src" "$2"
	uset "firewall.$1.dest" "$3"
}

# Name of the firewall zone that covers NETWORK (by network or device list).
zone_of_network() {
	local i=0 name nets
	while uci -q get "firewall.@zone[$i]" >/dev/null; do
		name="$(uci -q get "firewall.@zone[$i].name")"
		nets="$(uci -q get "firewall.@zone[$i].network") $(uci -q get "firewall.@zone[$i].device")"
		for n in $nets; do
			[ "$n" = "$1" ] && { echo "$name"; return 0; }
		done
		i=$((i + 1))
	done
	return 1
}

# Linux device of an OpenWrt network (lan -> br-lan).
net_device() {
	local dev
	dev="$(ifstatus "$1" 2>/dev/null | jsonfilter -e '@.l3_device' 2>/dev/null)"
	[ -n "$dev" ] || dev="$(uci -q get "network.$1.device")"
	[ -n "$dev" ] || dev="$(uci -q get "network.$1.ifname")"
	echo "$dev"
}

# Delete every copy of an ip rule (rules can stack up across crashes).
rule_del_all() {
	local fam="$1"
	shift
	while ip "$fam" rule del "$@" 2>/dev/null; do :; done
}
rule_ensure() {
	local fam="$1"
	shift
	rule_del_all "$fam" "$@"
	ip "$fam" rule add "$@"
}

tun_exists() { [ -e "$TB_SYSNET/$TB_TUN" ]; }

# Comma or space separated -> one per line.
split_list() { echo "$*" | tr ',' ' '; }
