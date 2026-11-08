#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Install test of the .apk in emulated OpenWrt 25.12 arm64 (podman; a
# native arm64 runner runs it without qemu). OpenWrt 25.12 replaced opkg
# with apk, so this is the package format there; it mirrors
# scripts/test-openwrt-ipk.sh (23.05, opkg): install, GL admin panel files
# present, the agent runs from the rendered config, the GL /rpc backend
# module ("tailbond" object, run with the router's own lua + cjson)
# answers status/api calls against it, bypass rules (the image's plain
# dnsmasq has no nftset: website rules wait, the rest loads; nft/ip checks
# need netlink, i.e. a native arm64 host), render-config keeps the panel's
# servers and rules, uninstall leaves nothing behind. Updates
# (docs/updates.md): an upgrade from an older package keeps the settings
# (/etc/config/tailbond is a conffile via /lib/apk/packages/
# tailbond.conffiles: the changed file is kept, the new default lands as
# .apk-new); update-ipk installs a newer package and rolls back one whose
# agent never answers; a simulated firmware upgrade (the sysupgrade hook's
# keep list restored onto a system without the package) reinstalls the apk
# at boot from the kept cache, and from the GitHub release (checked
# against SHA256SUMS) when the cache is gone; a half-installed package
# (apk's record kept, the files not) is repaired. The signed apk feed
# (docs/glinet.md, "OpenWrt 25.12 and apk"): an index built and signed
# here (scripts/release/apk-feed.sh, apk mkndx from a pinned Alpine image,
# a throwaway EC P-256 key), added to /etc/apk/repositories.d/
# customfeeds.list: rejected until the key is in /etc/apk/keys (the
# package ships it), then apk update + add + upgrade from it; a changed
# packages.adb and another key are rejected. Plus apk's own quirks the
# update flow works around: a file install pins the package's content
# hash in /etc/apk/world (which would block every later feed upgrade), so
# update-ipk and reinstall name the package plainly again.
#
#   scripts/test-openwrt-apk.sh [path/to/tailbond_*_aarch64_cortex-a53.apk]
#
# Env: IMAGE (default docker.io/openwrt/rootfs:armsr-armv8-25.12.6; the
#      weekly openwrt-latest workflow points it at the rolling
#      armsr-armv8-openwrt-25.12 tag), PODMAN (CI: "sudo podman"), IPK
#      (the .ipk the variant packages are rebuilt from; default out/ipk's
#      newest aarch64_cortex-a53 one — the apk and the variants are built
#      from the same tree, so the formats can't drift).
#
# The image has no feed for our Depends (kmod-tun, ip-full, ...), so the
# test strips them from its variants; lua + lua-cjson (present on GL
# firmware) come from the OpenWrt feed, so the container needs network.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APK="${1:-$(find "$ROOT/out/apk" -name 'tailbond_*_aarch64_cortex-a53.apk' 2>/dev/null | sort | tail -1)}"
IPK="${IPK:-$(find "$ROOT/out/ipk" -name 'tailbond_*_aarch64_cortex-a53.ipk' 2>/dev/null | sort | tail -1)}"
IMAGE="${IMAGE:-docker.io/openwrt/rootfs:armsr-armv8-25.12.6}"
[ -f "$APK" ] || { echo "no apk: run packaging/openwrt/build-apk.sh"; exit 2; }
[ -f "$IPK" ] || { echo "no ipk: run packaging/openwrt/build-ipk.sh"; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# repack OUT [VERSION [TAILBONDD]]: the ipk without Depends, optionally
# with another control Version and another /usr/bin/tailbondd (the update
# tests' older, newer and broken builds); apk variants are rebuilt from
# these by build-apk.sh, from the same tree as the apk under test
repack() {
    local w="$TMP/repack"
    rm -rf "$w" && mkdir -p "$w/x" "$w/c" "$w/d"
    tar -xzf "$IPK" -C "$w/x"
    tar -xzf "$w/x/control.tar.gz" -C "$w/c"
    sed -i '/^Depends:/d' "$w/c/control"
    [ -z "${2:-}" ] || sed -i "s/^Version: .*/Version: $2/" "$w/c/control"
    tar -xzf "$w/x/data.tar.gz" -C "$w/d"
    [ -z "${3:-}" ] || install -m 0755 "$3" "$w/d/usr/bin/tailbondd"
    tar --format=ustar --owner=0 --group=0 -C "$w/d" -czf "$w/x/data.tar.gz" .
    tar --format=ustar --owner=0 --group=0 -C "$w/c" -czf "$w/x/control.tar.gz" .
    tar --format=ustar --owner=0 --group=0 -C "$w/x" -czf "$1" ./debian-binary ./data.tar.gz ./control.tar.gz
}

# the throwaway apk feed key: the variant packages ship its public half
# (TB_APK_KEYS_DIR), so a router that installed one trusts the feed
mkdir -p "$TMP/keys"
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TMP/feed.sec" 2>/dev/null
openssl ec -in "$TMP/feed.sec" -pubout -out "$TMP/keys/tailbond-testfeed.pem" 2>/dev/null
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "$TMP/other.sec" 2>/dev/null

# apk_of IPK: rebuild an ipk as an apk (into $TMP/apkbuild, which keeps
# every variant)
mkdir -p "$TMP/apkbuild"
apk_of() {
    TB_IPK="$1" TB_APK_ARCHS=aarch64_cortex-a53 TB_APK_OUT="$TMP/apkbuild" \
        TB_APK_KEYS_DIR="$TMP/keys" "$ROOT/packaging/openwrt/build-apk.sh" >/dev/null
}

# the apk under test: no Depends (the image has no feed for them), the
# same tree as the release apk
V="$(basename "$IPK" | sed -n 's/^tailbond_\(.*\)_aarch64_cortex-a53\.ipk$/\1/p')"
[ -n "$V" ] || { echo "$IPK: not a tailbond_<version>_aarch64_cortex-a53.ipk name" >&2; exit 2; }
repack "$TMP/tailbond_${V}_aarch64_cortex-a53.ipk"
apk_of "$TMP/tailbond_${V}_aarch64_cortex-a53.ipk"
cp "$TMP/apkbuild/tailbond_${V}_aarch64_cortex-a53.apk" "$TMP/tailbond.apk"
# the built artifact itself, Depends and all (out/apk's, not a rebuild)
cp "$APK" "$TMP/tailbond-built.apk"
printf '%s\n' "$V" >"$TMP/built-version"
# For the update tests: an older package, a newer one, and a newer one
# whose agent exits at once (rollback).
printf '#!/bin/sh\necho "broken build" >&2\nexit 1\n' >"$TMP/broken-tailbondd"
repack "$TMP/tailbond_0.0.1-r1_aarch64_cortex-a53.ipk" 0.0.1-r1
repack "$TMP/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk" 99.0.0-r1
repack "$TMP/tailbond_99.1.0-r1_aarch64_cortex-a53.ipk" 99.1.0-r1 "$TMP/broken-tailbondd"
repack "$TMP/tailbond_99.2.0-r1_aarch64_cortex-a53.ipk" 99.2.0-r1
repack "$TMP/tailbond_99.3.0-r1_aarch64_cortex-a53.ipk" 99.3.0-r1
for v in 0.0.1 99.0.0 99.1.0 99.2.0 99.3.0; do
    apk_of "$TMP/tailbond_${v}-r1_aarch64_cortex-a53.ipk"
    cp "$TMP/apkbuild/tailbond_${v}-r1_aarch64_cortex-a53.apk" "$TMP/"
done
# The signed apk feed: the release's index (apk-feed.sh) for packages
# that ship the throwaway key; v1 to install, v2 to upgrade to, one with a
# changed packages.adb, one signed by a key the router doesn't have.
feed() { # DIR VERSION
    mkdir -p "$1"
    cp "$TMP/apkbuild/tailbond_$2_aarch64_cortex-a53.apk" "$1/"
    "$ROOT/scripts/release/apk-feed.sh" "$1" "$TMP/feed.sec" >/dev/null
}
feed "$TMP/feed1" 99.2.0-r1
feed "$TMP/feed2" 99.3.0-r1
cp -r "$TMP/feed1" "$TMP/feed-tampered"
printf 'junkjunk' >>"$TMP/feed-tampered/feed/aarch64_cortex-a53/packages.adb"
mkdir -p "$TMP/feed-otherkey/feed/aarch64_cortex-a53"
cp "$TMP/feed1/feed/aarch64_cortex-a53/tailbond-99.2.0-r1.apk" "$TMP/feed-otherkey/feed/aarch64_cortex-a53/"
cp "$TMP/other.sec" "$TMP/feed-otherkey/feed/aarch64_cortex-a53/"
"$ROOT/scripts/release/apk-tools.sh" "$TMP/feed-otherkey/feed/aarch64_cortex-a53" \
    mkndx --allow-untrusted --sign other.sec --output packages.adb tailbond-99.2.0-r1.apk >/dev/null
rm -f "$TMP/feed-otherkey/feed/aarch64_cortex-a53/other.sec"
# best effort: under rootful podman (CI) the files apk-tools wrote are
# root-owned and can't be chmod'ed — the container runs as root and reads
# them anyway
chmod -R a+rX "$TMP" 2>/dev/null || true
# A pairing link whose payload names the server.
printf '%s' '{"v":1,"name":"slate7","server_name":"homebox","sni":"homebox.tail1234.ts.net","server":"203.0.113.7:443"}' |
    base64 -w0 | tr '+/' '-_' | tr -d '=' >"$TMP/pair-d.txt"

cat >"$TMP/test.sh" <<'IN'
#!/bin/sh
set -u
fail=0
ok() { echo "  PASS $*"; }
bad() { echo "  FAIL $*"; fail=1; }
mkdir -p /var/lock /var/run /tmp/sysinfo
# the package is aarch64_cortex-a53; the image's own arch is
# aarch64_generic (armsr), so allow both, as a GL.iNet-device image would
grep -qx aarch64_cortex-a53 /etc/apk/arch || echo aarch64_cortex-a53 >>/etc/apk/arch
echo "GL.iNet GL-BE3600" >/tmp/sysinfo/model
ver() { apk list --installed 2>/dev/null | sed -n 's/^tailbond-\([^ ]*\) .*/\1/p'; }
world_pin() { grep -E '^tailbond[><=]' /etc/apk/world; }
# the feed over the network (a CI runner's network may need a retry)
feed_ok=0
for try in 1 2 3; do
	if apk update >/tmp/apk.log 2>&1 && apk add lua lua-cjson >>/tmp/apk.log 2>&1; then feed_ok=1; break; fi
	sleep $((try * 5))
done
[ "$feed_ok" = 1 ] || {
	echo "need lua + lua-cjson from the feed:"; tail -n 25 /tmp/apk.log
	cat /etc/resolv.conf; nslookup downloads.openwrt.org; ip route
	apk update 2>&1 | tail -5
	exit 2
}

# feed_update: apk update, retried while it fails without a verdict on our
# feed (a distfeed fetch that flakes is not this feed's doing); the last
# attempt's output is in /tmp/feed.log
feed_update() {
	local i=0
	while :; do
		apk update >/tmp/feed.log 2>&1 && return 0
		grep -qi 'untrusted\|signature\|invalid' /tmp/feed.log && return 1
		i=$((i + 1)); [ $i -ge 3 ] && return 1
		sleep $((i * 3))
	done
}

echo "== the built artifact, with its Depends (the release's apk, not a rebuild)"
BUILTVER="$(cat /t/built-version)"
# the feed serves kmod-tun, ip-full, jshn and jsonfilter, so the Depends
# resolve in the container too
apk add --allow-untrusted /t/tailbond-built.apk >/tmp/built.log 2>&1 && ok "apk add the built tailbond apk (with its Depends)" || { cat /tmp/built.log; bad "built apk install"; }
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
[ "$(ver)" = "$BUILTVER" ] && ok "installed at $(ver)" || bad "built version: $(ver)"
apk del tailbond >/tmp/built-del.log 2>&1 && ok "removed again (the repacked one takes over)" || { cat /tmp/built-del.log; bad "built apk del"; }

echo "== install"
apk add --allow-untrusted /t/tailbond.apk >/tmp/install.log 2>&1 && ok "apk add (file)" || { cat /tmp/install.log; bad "apk add"; }
world_pin >/dev/null && ok "a file install pins the content hash in world (apk's own doing)" || bad "no world pin (apk changed?)"
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
grep -qx tailbond /etc/apk/world && ok "world names the package plainly again (or every feed upgrade would be blocked)" || bad "world"
for f in /usr/bin/tailbondd /usr/bin/mqvpn /etc/init.d/tailbond /usr/share/oui/menu.d/tailbond.json \
	/www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/lib/oui-httpd/rpc/tailbond /usr/share/gl-validator.d/tailbond.lua \
	/usr/share/tailbond/nginx-tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/tailbond/tailbond.json; do
	[ -e "$f" ] && ok "$f" || bad "missing $f"
done
[ -f /lib/apk/packages/tailbond.conffiles ] && grep -qx /etc/config/tailbond /lib/apk/packages/tailbond.conffiles \
	&& ok "apk conffile list ships /etc/config/tailbond" || bad "conffiles"
[ -f /etc/apk/keys/tailbond-testfeed.pem ] && ok "the package installs the feed's public key in /etc/apk/keys" || bad "no feed key"
jsonfilter -i /usr/share/oui/menu.d/tailbond.json -e '@.view' | grep -qx tailbond && ok "menu.d entry parses (view=tailbond)" || bad "menu json"
gzip -dc /www/views/gl-sdk4-ui-tailbond.common.js.gz | grep -q 'name: "tailbond"' && ok "view is the panel component" || bad "view content"
[ -x /etc/tailbond/pkg/reinstall ] && [ "$(grep -c '# tailbond-reinstall$' /etc/rc.local)" = 1 ] \
	&& [ "$(grep -n '# tailbond-reinstall$' /etc/rc.local | cut -d: -f1)" -lt "$(grep -n '^exit 0' /etc/rc.local | head -1 | cut -d: -f1)" ] \
	&& ok "reinstall script kept in /etc/tailbond/pkg, one rc.local line before 'exit 0'" || { bad "reinstall hook"; cat /etc/rc.local; }
[ -f /lib/upgrade/tailbond.sh ] && [ -x /usr/lib/tailbond/update-ipk ] && ok "sysupgrade hook and update-ipk installed" || bad "update files"
[ "$(uci -q get tailbond.main.enabled)" = 1 ] && ok "enabled by default" || bad "not enabled"
[ "$(uci -q get tailbond.main.role)" = client ] && ok "role client on a Slate 7 (GL-BE3600)" || bad "role $(uci -q get tailbond.main.role)"
[ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.listen')" = 127.0.0.1:8077 ] && ok "agent listens on loopback only" || bad "listen $(jsonfilter -i /etc/tailbond/tailbond.json -e '@.listen')"
[ -z "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.paths[*]')" ] && ok "no hardcoded sample paths" || bad "sample paths rendered"

echo "== agent (procd isn't running in the container: start it like the init script would)"
/usr/bin/tailbondd -config /etc/tailbond/tailbond.json -quiet >/tmp/agent.log 2>&1 &
pid=$!
i=0; while [ $i -lt 30 ] && ! uclient-fetch -q -O /dev/null http://127.0.0.1:8077/api/status 2>/dev/null; do sleep 1; i=$((i+1)); done
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "agent serves /api/status" || { bad "agent"; cat /tmp/agent.log; }

echo "== GL /rpc backend module"
cat >/tmp/rpc.lua <<'LUA'
local cjson = require "cjson"
local m = dofile("/usr/lib/oui-httpd/rpc/tailbond")
local function show(name, r) print(name .. " " .. cjson.encode(r)) end
show("status", m.status())
show("hello", m.api({ method = "GET", path = "/api/hello" }))
show("paths", m.api({ method = "GET", path = "/api/paths" }))
show("discover", m.api({ method = "GET", path = "/api/discover" }))
show("badjoin", m.api({ method = "POST", path = "/api/join", body = { server = "" } }))
show("login", m.api({ method = "POST", path = "/api/login", body = {} }))
show("escape", m.api({ method = "GET", path = "/api/../etc/passwd" }))
show("info", m.info())
show("ksoff", m.set_kill_switch({ on = false }))
show("ksinfo", m.info())
show("ksbad", m.set_kill_switch({ on = "yes" }))
show("kson", m.set_kill_switch({ on = true }))
LUA
lua /tmp/rpc.lua >/tmp/rpc.out 2>&1
cat /tmp/rpc.out | cut -c1-200
grep '^status ' /tmp/rpc.out | grep -q '"ok":true' && grep '^status ' /tmp/rpc.out | grep -q '"role":"client"' && ok "rpc status answers from the running agent" || bad "rpc status"
grep '^hello ' /tmp/rpc.out | grep -q '"tailbond":true' && ok "rpc api GET /api/hello" || bad "rpc hello"
grep '^discover ' /tmp/rpc.out | grep -q '"ok":true' && ok "rpc api GET /api/discover" || bad "rpc discover"
grep '^badjoin ' /tmp/rpc.out | grep -q '"code":400' && ok "rpc api POST passes the agent's 400 through" || bad "rpc post"
grep '^login ' /tmp/rpc.out | grep -q 'not_allowed' && ok "login path refused" || bad "login allowed"
grep '^escape ' /tmp/rpc.out | grep -q 'not_allowed' && ok "path escape refused" || bad "escape allowed"
grep '^info ' /tmp/rpc.out | grep -q '"role":"client"' && ok "rpc info" || bad "rpc info"
grep '^ksoff ' /tmp/rpc.out | grep -q '"ok":true' && grep '^ksinfo ' /tmp/rpc.out | grep -q '"kill_switch":false' && grep '^ksbad ' /tmp/rpc.out | grep -q bad_value \
	&& [ "$(uci -q get tailbond.client.kill_switch)" = 1 ] && ok "rpc set_kill_switch (UCI client.kill_switch)" || bad "set_kill_switch"

echo "== bypass rules (this image's dnsmasq has no nftset: website rules wait, the rest applies)"
cat >/tmp/bypass.lua <<'LUA'
local cjson = require "cjson"
local m = dofile("/usr/lib/oui-httpd/rpc/tailbond")
local function show(name, r) print(name .. " " .. cjson.encode(r)) end
show("add1", m.api({ method = "POST", path = "/api/bypass", body = { kind = "domain", match = "bank.example", target = "direct", note = "Bank" } }))
show("add2", m.api({ method = "POST", path = "/api/bypass", body = { kind = "device", match = "aa:bb:cc:dd:ee:ff", target = "tunnel", note = "Work laptop" } }))
show("bad", m.api({ method = "POST", path = "/api/bypass", body = { kind = "port", match = "udp/99999", target = "direct" } }))
show("lan", m.api({ method = "GET", path = "/api/lan-clients" }))
LUA
lua /tmp/bypass.lua >/tmp/bypass.out 2>&1
grep '^add1 ' /tmp/bypass.out | grep -q '"ok":true' && grep '^add2 ' /tmp/bypass.out | grep -q '"ok":true' && ok "rpc POST /api/bypass (website + device)" || { bad "bypass add"; cat /tmp/bypass.out; }
grep '^bad ' /tmp/bypass.out | grep -q '"code":400' && ok "invalid rule refused (400)" || bad "bad rule accepted"
grep '^lan ' /tmp/bypass.out | grep -q '"ok":true' && ok "rpc GET /api/lan-clients" || bad "lan-clients"
i=0; while [ $i -lt 20 ] && [ ! -s /etc/tailbond/state/bypass-status.json ]; do sleep 1; i=$((i+1)); done
cat /etc/tailbond/state/bypass-status.json 2>/dev/null
[ -x /usr/lib/tailbond/hooks/bypass-apply ] && ok "bypass-apply hook installed" || bad "no bypass-apply"
grep -q 'nftset=/bank.example/4#inet#tailbond#' /etc/tailbond/state/bypass-nftset.conf && grep -q 'ether saddr aa:bb:cc:dd:ee:ff' /etc/tailbond/state/bypass.nft \
	&& ok "agent rendered the nft table and the dnsmasq drop-in" || bad "rendered files"
jsonfilter -i /etc/tailbond/state/bypass-status.json -e '@.domains' | grep -qx none && ok "plain dnsmasq detected: website rules wait (domains=none)" || bad "domains status"
if nft list tables >/dev/null 2>&1; then
	grep -q dnsmasq-full /etc/tailbond/state/bypass-status.json && ok "the reason names dnsmasq-full" || bad "reason"
	applied() {
		nft list table inet tailbond 2>/dev/null | grep -q 'ether saddr aa:bb:cc:dd:ee:ff' &&
			ip rule | grep -q 'fwmark 0x2000000/0x7000000 lookup 77' && ip rule | grep -q 'fwmark 0x1000000/0x7000000 lookup main'
	}
	i=0; while [ $i -lt 30 ] && ! applied; do sleep 1; i=$((i+1)); done
	applied || uclient-fetch -q -O - http://127.0.0.1:8077/api/logs | grep -o '[^"]*bypass[^"]*' | tail -n 20
	nft list table inet tailbond | grep -q 'ether saddr aa:bb:cc:dd:ee:ff' && ok "nft table inet tailbond loaded (device rule)" || bad "nft table"
	ip rule | grep -q 'fwmark 0x2000000/0x7000000 lookup 77' && ip rule | grep -q 'fwmark 0x1000000/0x7000000 lookup main' \
		&& ok "ip rules for 'always tailbond' and 'direct'" || { bad "ip rules"; ip rule; }
	[ ! -e /tmp/dnsmasq.d/tailbond-bypass.conf ] && ok "no dnsmasq drop-in without nftset support" || bad "drop-in written for plain dnsmasq"
	TB_BYPASS_ACTION=clear /usr/lib/tailbond/hooks/bypass-apply >/dev/null 2>&1
	! nft list table inet tailbond >/dev/null 2>&1 && ! ip rule | grep -q 0x7000000 && ok "clear removes the table and rules" || bad "clear"
else
	echo "  SKIP nft/ip checks: no netlink under qemu-user (CI runs them on a native arm64 runner)"
fi

echo "== render-config keeps panel-managed servers and rules; UCI bypass sections replace the rules"
lua -e '
local cjson = require "cjson"
local f = io.open("/etc/tailbond/tailbond.json"); local c = cjson.decode(f:read("*a")); f:close()
c.client.servers = { { name = "homebox", pairing_uri = "tailbond://pair?d=x" }, { name = "vps", pairing_uri = "tailbond://pair?d=y", enabled = false } }
c.client.failover = { policy = "prefer", failback_after_sec = 300 }
f = io.open("/etc/tailbond/tailbond.json", "w"); f:write(cjson.encode(c)); f:close()'
/usr/lib/tailbond/render-config --force
J=/etc/tailbond/tailbond.json
[ "$(jsonfilter -i $J -e '@.client.servers[1].name')" = vps ] && [ "$(jsonfilter -i $J -e '@.client.servers[1].enabled')" = false ] \
	&& [ "$(jsonfilter -i $J -e '@.client.failover.failback_after_sec')" = 300 ] && ok "servers + failover kept" || { bad "servers"; cat $J; }
[ "$(jsonfilter -i $J -e '@.client.bypass[0].match')" = bank.example ] && [ "$(jsonfilter -i $J -e '@.client.bypass[1].target')" = tunnel ] \
	&& ok "panel bypass rules kept" || bad "bypass kept"
uci set tailbond.client.mtu=1340 && uci commit tailbond
/usr/lib/tailbond/render-config --force
[ "$(jsonfilter -i $J -e '@.client.mtu')" = 1340 ] && ok "client.mtu reaches the config" || bad "mtu dropped: $(jsonfilter -i $J -e '@.client')"
uci delete tailbond.client.mtu && uci commit tailbond
/usr/lib/tailbond/render-config --force
[ -z "$(jsonfilter -i $J -e '@.client.mtu' 2>/dev/null)" ] && ok "no mtu set: mqvpn's default" || bad "mtu left behind"
uci set tailbond.client.mtu=1200 && uci commit tailbond
if /usr/lib/tailbond/render-config --force 2>/tmp/mtu-err.txt; then
	bad "client.mtu=1200 accepted (mqvpn would ignore it)"
else
	ok "client.mtu below 1280 is refused: $(head -1 /tmp/mtu-err.txt)"
fi
uci set tailbond.client.mtu=8993 && uci commit tailbond
if /usr/lib/tailbond/render-config --force 2>/dev/null; then
	bad "mtu 8993 accepted in Speed (mqconf would send 9001)"
else
	ok "mtu above the Speed limit is refused"
fi
uci set tailbond.client.mtu=8992 && uci commit tailbond
/usr/lib/tailbond/render-config --force &&
	[ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.mtu')" = 8992 ] &&
	ok "mtu at the Speed limit is accepted" || bad "mtu 8992"
uci delete tailbond.client.mtu && uci commit tailbond
/usr/lib/tailbond/render-config --force
[ "$(jsonfilter -i $J -e '@.client.bypass_mark_mask')" = 0x07000000 ] && ok "bypass_mark_mask from UCI" || bad "mark mask"
uci add tailbond bypass >/dev/null && uci set tailbond.@bypass[-1].kind=port && uci set tailbond.@bypass[-1].match=udp/4500 \
	&& uci set tailbond.@bypass[-1].target=wan && uci set tailbond.@bypass[-1].wan=wwan && uci commit tailbond
/usr/lib/tailbond/render-config
[ "$(jsonfilter -i $J -e '@.client.bypass[0].match')" = udp/4500 ] && [ "$(jsonfilter -i $J -e '@.client.bypass[0].id')" = u0 ] \
	&& [ -z "$(jsonfilter -i $J -e '@.client.bypass[1]')" ] && ok "UCI 'config bypass' sections replace the rules" || { bad "uci bypass"; cat $J; }
/usr/bin/tailbondd -config $J -version >/dev/null && kill -0 "$pid" 2>/dev/null && ok "agent still running" || true

kill "$pid" 2>/dev/null; sleep 1
lua -e 'local m=dofile("/usr/lib/oui-httpd/rpc/tailbond"); print(require("cjson").encode(m.status()))' | grep -q agent_down && ok "agent down reported as agent_down" || bad "agent_down"

echo "== uninstall"
apk del tailbond >/tmp/remove.log 2>&1 && ok "apk del" || { cat /tmp/remove.log; bad "apk del"; }
for f in /usr/bin/tailbondd /usr/lib/oui-httpd/rpc/tailbond /www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/share/oui/menu.d/tailbond.json \
	/usr/share/gl-validator.d/tailbond.lua /etc/nginx/gl-conf.d/tailbond.conf /usr/share/tailbond/nginx-tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/init.d/tailbond; do
	[ -e "$f" ] && bad "left behind: $f"
done
[ -e /etc/rc.d/S99tailbond ] && bad "init still enabled" || ok "files and init links removed"
[ -e /lib/apk/packages/tailbond.conffiles ] && bad "apk package metadata left behind" || ok "apk metadata removed"
! grep -q 'tailbond-reinstall' /etc/rc.local && [ ! -e /etc/tailbond/pkg ] \
	&& ok "removal drops the rc.local line and the package cache (at once: apk runs no remove-then-reinstall)" || bad "rc.local/pkg left"
# an unmodified conffile goes with the package (the feed key: a key added
# by hand first is the same file); a changed one (/etc/config/tailbond)
# stays, as opkg kept conffiles
[ ! -e /etc/apk/keys/tailbond-testfeed.pem ] && ok "the feed key (an unmodified conffile) goes with the package" || bad "key left"
[ -e /etc/config/tailbond ] && ok "/etc/config/tailbond (a changed conffile) kept after removal" || bad "config gone"

echo "== upgrade from an older package keeps the settings"
OLD=/t/tailbond_0.0.1-r1_aarch64_cortex-a53.apk V99=/t/tailbond_99.0.0-r1_aarch64_cortex-a53.apk BROKEN=/t/tailbond_99.1.0-r1_aarch64_cortex-a53.apk
apk add --allow-untrusted $OLD >/tmp/old.log 2>&1 && [ "$(ver)" = 0.0.1-r1 ] && ok "older package installed" || { bad "install old"; cat /tmp/old.log; }
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
uci set tailbond.main.keep_test=kept && uci commit tailbond
export PAIR="tailbond://pair?d=$(cat /t/pair-d.txt)"
lua -e '
local cjson = require "cjson"
local f = io.open("/etc/tailbond/tailbond.json"); local c = cjson.decode(f:read("*a")); f:close()
c.client.servers = { { name = "homebox", pairing_uri = os.getenv("PAIR") } }
f = io.open("/etc/tailbond/tailbond.json", "w"); f:write(cjson.encode(c)); f:close()'
apk add --allow-untrusted /t/tailbond.apk >/tmp/up.log 2>&1 && NEWV="$(ver)" && [ "$NEWV" != 0.0.1-r1 ] && ok "upgraded to $NEWV" || { bad "upgrade"; cat /tmp/up.log; }
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "/etc/config/tailbond kept (a conffile)" || bad "uci lost"
[ -e /etc/config/tailbond.apk-new ] && ok "the new default landed as .apk-new (as opkg's opkg-new)" || bad "no .apk-new"
[ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.servers[0].name')" = homebox ] && ok "/etc/tailbond/tailbond.json kept (servers)" || bad "tailbond.json lost"
apk add --allow-untrusted --force-reinstall /t/tailbond.apk >/dev/null 2>&1
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
[ "$(grep -c '# tailbond-reinstall$' /etc/rc.local)" = 1 ] && ok "rc.local line stays single over re-installs" || bad "rc.local line duplicated"
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "settings kept over the re-install" || bad "uci lost on reinstall"

echo "== pkg-cache: seeding keeps a package that is downloaded but not installed"
sd=$(mktemp -d)
cp "$V99" "$sd/new.apk" && cp "$V99" "$sd/old.apk" || true
TB_PKG_DIR=$sd TB_PENDING=$sd/new.apk /usr/lib/tailbond/pkg-cache keep "$sd/old.apk" >/dev/null 2>&1
[ -f "$sd/new.apk" ] && [ -f "$sd/old.apk" ] \
	&& ok "seeding spares the downloaded package" || bad "seeding deleted the downloaded package: $(ls $sd)"
[ "$(TB_PKG_DIR=$sd /usr/lib/tailbond/pkg-cache ipk-version "$V99")" = 99.0.0-r1 ] \
	&& ok "pkg-cache reads an apk's version from its name" || bad "apk version"
rm -rf "$sd"

echo "== update-ipk"
cat >/tmp/restart.sh <<'SH'
#!/bin/sh
# what procd does on /etc/init.d/tailbond restart, without procd
for p in $(pidof tailbondd); do kill "$p"; done
i=0; while pidof tailbondd >/dev/null && [ $i -lt 10 ]; do sleep 1; i=$((i+1)); done
/usr/bin/tailbondd -config /etc/tailbond/tailbond.json -quiet >>/tmp/agent.log 2>&1 </dev/null &
SH
chmod +x /tmp/restart.sh
/tmp/restart.sh
i=0; while [ $i -lt 30 ] && ! uclient-fetch -q -O /dev/null http://127.0.0.1:8077/api/status 2>/dev/null; do sleep 1; i=$((i+1)); done
R=/etc/tailbond/state/update-result.json
res() { jsonfilter -i $R -e "@.$1" 2>/dev/null; }
napkg() { ls /etc/tailbond/pkg/*.apk 2>/dev/null | wc -l; }
export TB_RESTART=/tmp/restart.sh TB_WAS_CONNECTED=0 TB_ROLE=client TB_WAIT=30
# The hand-installed router: no rollback copy in the cache (the agent
# rebuilds .ipk only, so an apk system can't seed one), and the new
# package already sitting there, because the agent downloads it before
# update-ipk starts.
rm -rf /etc/tailbond/pkg && mkdir -p /etc/tailbond/pkg
cp $V99 /etc/tailbond/pkg/ && [ -f /etc/tailbond/pkg/$(basename $V99) ] \
	&& ok "no rollback copy, the download sits in the pkg dir" || bad "could not stage the download"
/usr/lib/tailbond/update-ipk /etc/tailbond/pkg/$(basename $V99) ""
[ -f /etc/tailbond/pkg/$(basename $V99) ] && ok "the download survived the cache prune" || bad "prune deleted the package it was installing"
[ "$(res result)" = ok ] && [ "$(res version)" = 99.0.0-r1 ] && [ "$(res from)" = "$NEWV" ] && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "update to 99.0.0-r1 from a hand-installed router: result ok, from $NEWV" || { bad "update-ipk happy path"; cat $R /tmp/tailbond-update.log; }
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "agent answers after the update" || bad "agent down after update"
! world_pin >/dev/null && ok "update-ipk left world naming the package plainly" || { bad "world pin left"; world_pin; }
[ "$(napkg)" = 1 ] && [ "$(cat /etc/tailbond/pkg/current)" = tailbond_99.0.0-r1_aarch64_cortex-a53.apk ] \
	&& ok "cache: the new package (no rollback copy to keep on apk)" || { bad "cache after update"; ls -la /etc/tailbond/pkg; }
/usr/lib/tailbond/update-ipk $V99 "" || true
[ "$(res result)" = failed ] && res detail | grep -q 'already the installed' && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "the same version again is refused, not reported as done: $(res detail)" || { bad "same-version update"; cat $R /tmp/tailbond-update.log; }
# nothing was seeded (the agent rebuilds .ipk only), so the cached current
# is the rollback copy from the second update on: a broken one rolls back
# onto it without an explicit PREV
TB_WAIT=20 /usr/lib/tailbond/update-ipk $BROKEN ""
[ "$(res result)" = rolled_back ] && [ "$(ver)" = 99.0.0-r1 ] && [ "$(napkg)" = 2 ] \
	&& ok "the cached current is the rollback copy (no explicit PREV); the broken update rolled back onto it" || { bad "cache rollback copy"; cat $R /tmp/tailbond-update.log; }
TB_WAIT=20 /usr/lib/tailbond/update-ipk $BROKEN /etc/tailbond/pkg/tailbond_99.0.0-r1_aarch64_cortex-a53.apk
[ "$(res result)" = rolled_back ] && [ "$(res version)" = 99.1.0-r1 ] && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "an agent that never answers is rolled back (a downgrade from a file): $(res detail)" || { bad "rollback"; cat $R /tmp/tailbond-update.log; }
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "previous agent answers again" || bad "agent down after rollback"
[ "$(napkg)" = 2 ] && [ "$(cat /etc/tailbond/pkg/current)" = tailbond_99.0.0-r1_aarch64_cortex-a53.apk ] \
	&& ok "cache: both packages, current = the restored one" || { bad "cache after rollback"; ls -la /etc/tailbond/pkg; }
unset TB_RESTART TB_WAS_CONNECTED TB_ROLE TB_WAIT

echo "== firmware upgrade with 'Keep settings' (simulated)"
# sysupgrade: add_conffiles lists /etc/sysupgrade.conf + keep.d, then the
# hooks from /lib/upgrade/*.sh run with that list.
sysupgrade_list() {
	(
		IMAGE=/tmp/firmware.bin
		sysupgrade_init_conffiles="add_conffiles"
		. /lib/upgrade/tailbond.sh
		find $(sed -ne '/^[[:space:]]*$/d; /^#/d; p' /etc/sysupgrade.conf /lib/upgrade/keep.d/* 2>/dev/null) \( -type f -o -type l \) 2>/dev/null | sort -u >"$1"
		for h in $sysupgrade_init_conffiles; do [ "$h" = add_conffiles ] || "$h" "$1"; done
	)
}
# flash: the package is gone, the kept files come back from the backup
flash() {
	for p in $(pidof tailbondd); do kill "$p"; done; sleep 1
	# (minus the files podman bind-mounts, which can't be replaced)
	grep -v -E '^/etc/(hosts|hostname|resolv\.conf)$' "$1" >/tmp/conffiles.flash
	tar -czf /tmp/sysupgrade.tgz -T /tmp/conffiles.flash 2>/dev/null
	apk del tailbond >/dev/null 2>&1 # sysupgrade runs no package scripts
	rm -rf /etc/tailbond /etc/config/tailbond /tmp/tailbond-reinstall.ran
	tar -xzf /tmp/sysupgrade.tgz -C /
}
sysupgrade_list /tmp/conffiles
grep -qx /etc/tailbond/pkg/tailbond_99.0.0-r1_aarch64_cortex-a53.apk /tmp/conffiles && ! grep -q 99.1.0 /tmp/conffiles \
	&& grep -qx /etc/tailbond/pkg/reinstall /tmp/conffiles && grep -qx /etc/rc.local /tmp/conffiles \
	&& ok "keep list: current package, reinstall script, rc.local; not the previous package" || { bad "keep list"; grep -E 'tailbond|rc.local' /tmp/conffiles; }
flash /tmp/conffiles
[ -z "$(ver)" ] && [ ! -e /usr/bin/tailbondd ] && grep -q '# tailbond-reinstall$' /etc/rc.local \
	&& ok "after the flash: no package, kept rc.local line and cache" || bad "flash simulation"
sh /etc/rc.local
i=0; while [ $i -lt 120 ] && [ "$(res detail)" != reinstalled_after_firmware_upgrade ]; do sleep 1; i=$((i+1)); done
[ "$(ver)" = 99.0.0-r1 ] && [ -x /usr/bin/tailbondd ] && [ "$(res result)" = ok ] && [ "$(res detail)" = reinstalled_after_firmware_upgrade ] \
	&& ok "boot reinstalled 99.0.0-r1 from the cache (update-result: reinstalled_after_firmware_upgrade)" || { bad "reinstall at boot"; logread 2>/dev/null | tail -5; cat /tmp/tailbond-reinstall.log; cat $R; }
! world_pin >/dev/null && ok "the boot reinstall left world naming the package plainly" || { bad "world pin after boot reinstall"; world_pin; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && [ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.servers[0].name')" = homebox ] \
	&& ok "settings intact after the reinstall" || bad "settings lost in reinstall"
[ "$(grep -c '# tailbond-reinstall$' /etc/rc.local)" = 1 ] && ok "still one rc.local line" || bad "rc.local lines"
flash /tmp/conffiles
mkdir -p /tmp/tailbond-reinstall.ran # this boot's run already happened
/etc/tailbond/pkg/reinstall boot
[ -z "$(ver)" ] && ok "at most once per boot" || bad "ran twice in one boot"

echo "== half-installed after the upgrade (issue #49): apk's record kept, the files gone"
# The GL.iNet 4.9.0 upgrade kept the package manager's record and
# /usr/bin/tailbondd but dropped /usr/lib/tailbond and /usr/share/tailbond:
# apk says installed, nothing runs, and a plain `apk add` is a no-op. The
# boot hook must notice (a file, not only the record) and reinstall from
# the cache.
# reinstall_running: rc.local's line and the init script both background
# the repair; the next section must not race its apk for the lock.
# The bracket in the pattern keeps the grep from matching its own command
# line (which contains the pattern text), which would pin both waits
# below to their full 120 s.
reinstall_running() { grep -la '/etc/tailbond/pkg/re[i]nstall' /proc/[0-9]*/cmdline 2>/dev/null | grep -q .; }
apk add --allow-untrusted $V99 >/tmp/half.log 2>&1 # the package back, as the boot hook left it
sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world
rm -rf /usr/lib/tailbond /usr/share/tailbond /tmp/tailbond-reinstall.ran
[ -n "$(ver)" ] && [ -x /usr/bin/tailbondd ] && [ ! -e /usr/lib/tailbond/render-config ] \
	&& ok "half-installed: record and tailbondd kept, render-config gone" || { bad "half-install setup"; cat /tmp/half.log; }
sh /etc/rc.local
i=0; while [ $i -lt 120 ] && { reinstall_running || [ ! -x /usr/lib/tailbond/render-config ]; }; do sleep 1; i=$((i+1)); done
[ -x /usr/lib/tailbond/render-config ] && [ -x /usr/lib/tailbond/pkg-cache ] && [ -f /usr/share/tailbond/nginx-tailbond.conf ] \
	&& ok "the boot hook reinstalled the half-installed package" || { bad "half-install repair"; cat /tmp/tailbond-reinstall.log; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "settings kept through the repair" || bad "settings lost in the repair"

echo "== the init script repairs a half-installed package instead of only failing (issue #49)"
rm -rf /usr/lib/tailbond /usr/share/tailbond
/etc/init.d/tailbond start >/tmp/half-start.log 2>&1
grep -q 'half-installed' /tmp/half-start.log \
	&& ok "init start reports half-installed and kicks the cached reinstall" || { bad "init self-repair"; cat /tmp/half-start.log; }
i=0; while [ $i -lt 120 ] && { reinstall_running || [ ! -x /usr/lib/tailbond/render-config ]; }; do sleep 1; i=$((i+1)); done
[ -x /usr/lib/tailbond/render-config ] \
	&& ok "the init script's reinstall put the files back" || { bad "init repair"; cat /tmp/tailbond-reinstall.log; cat /tmp/half-start.log; }
# back to the flashed state the next section starts from, with no repair
# in flight to fight for apk's lock
flash /tmp/conffiles

echo "== reinstall from the GitHub release when the cached apk is gone"
G=/tmp/gh
n=tailbond_99.0.0-r1_aarch64_cortex-a53.apk
mkdir -p $G/dl $G/repos/kaovilai/tailbond/releases/tags
cp $V99 $G/dl/$n
# the stable name (docs/glinet.md, Download): reinstall downloads it and
# keeps it under the versioned name
sn=tailbond-openwrt-aarch64_cortex-a53.apk
cp $V99 $G/dl/$sn
chmod 644 $G/dl/* # uhttpd serves world-readable files only
rel() { # FILE: a release with the apks and SHA256SUMS
	printf '{"tag_name":"v99.0.0","assets":[{"name":"%s","browser_download_url":"http://127.0.0.1:8099/dl/%s"},{"name":"%s","browser_download_url":"http://127.0.0.1:8099/dl/%s"},{"name":"SHA256SUMS","browser_download_url":"http://127.0.0.1:8099/dl/SHA256SUMS"}]}\n' "$n" "$n" "$sn" "$sn" >"$1"
}
rel $G/repos/kaovilai/tailbond/releases/tags/v99.0.0
rel $G/repos/kaovilai/tailbond/releases/latest
printf '0000000000000000000000000000000000000000000000000000000000000000  %s\n' $n $sn >$G/dl/SHA256SUMS
chmod 644 $G/dl/SHA256SUMS $G/repos/kaovilai/tailbond/releases/latest $G/repos/kaovilai/tailbond/releases/tags/v99.0.0
uhttpd -f -p 127.0.0.1:8099 -h $G >/dev/null 2>&1 &
U1=$!
sleep 1
# the pointer still names the installed version; its apk is gone
rm -f /etc/tailbond/pkg/*.apk
echo $n >/etc/tailbond/pkg/current
TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
[ -z "$(ver)" ] && [ ! -e /etc/tailbond/pkg/$n ] && grep -q "doesn't match the release's SHA256SUMS" /tmp/reinstall.err \
	&& ok "an apk that doesn't match SHA256SUMS is refused" || { bad "bad sha256 check"; cat /tmp/reinstall.err; }
(cd $G/dl && sha256sum $n $sn >SHA256SUMS && chmod 644 SHA256SUMS)
TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
[ "$(ver)" = 99.0.0-r1 ] && [ "$(cat /etc/tailbond/pkg/current)" = $n ] && [ "$(res detail)" = reinstalled_after_firmware_upgrade ] \
	&& grep -q "downloaded $sn from the GitHub release (kept as $n)" /tmp/reinstall.err \
	&& ok "downloaded that version's apk from the GitHub release, checked it, installed it" || { bad "GitHub reinstall"; cat /tmp/reinstall.err /tmp/tailbond-reinstall.log; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "settings intact" || bad "settings lost"
# nothing cached at all: the latest release's apk for this router
cp /etc/tailbond/pkg/reinstall /tmp/reinstall
apk del tailbond >/dev/null 2>&1
mkdir -p /tmp/pkgB
TB_PKG_DIR=/tmp/pkgB TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /tmp/reinstall now 2>/tmp/reinstall.err
[ "$(ver)" = 99.0.0-r1 ] && [ "$(cat /tmp/pkgB/current)" = $n ] \
	&& ok "no cache: the latest release's apk for this architecture" || { bad "latest reinstall"; cat /tmp/reinstall.err /tmp/tailbond-reinstall.log; ls /tmp/pkgB; }
kill $U1 2>/dev/null

echo "== the signed apk feed (customfeeds.list; apk follows no redirects, so a direct URL)"
K=/etc/apk/keys/tailbond-testfeed.pem
F=/tmp/feedsrv
mkdir -p $F/f
uhttpd -f -p 127.0.0.1:8098 -h $F >/dev/null 2>&1 &
U2=$!
serve() { rm -rf $F/f && cp -r "/t/$1/feed/aarch64_cortex-a53" $F/f && chmod -R a+rX $F/f; }
serve feed1
sleep 1
apk del tailbond >/dev/null 2>&1 # takes the key it shipped with it (asserted above)
cp /etc/apk/repositories.d/customfeeds.list /tmp/customfeeds.list
echo "http://127.0.0.1:8098/f/packages.adb" >>/etc/apk/repositories.d/customfeeds.list
rejected() { # VERSION: apk warns, and nothing from the refused feed installs
	feed_update
	grep -qi 'untrusted\|signature\|invalid' /tmp/feed.log || return 1
	if apk add tailbond >/tmp/feed-add.log 2>&1 && [ "$(ver)" = "$1" ]; then return 1; fi
	return 0
}
rejected 99.2.0-r1 && ok "without the key, the feed is refused (UNTRUSTED signature)" || { bad "feed accepted without its key"; cat /tmp/feed.log /tmp/feed-add.log; }
# the documented first step: the release's tailbond-apk.pub
mkdir -p /etc/apk/keys && cp /t/keys/tailbond-testfeed.pem $K
[ -f "$K" ] && ok "the package's key (the release's tailbond-apk.pub) is in /etc/apk/keys" || bad "key"
feed_update && ! grep -qi 'untrusted\|signature\|invalid' /tmp/feed.log \
	&& ok "apk update: the feed's signature checks" || { bad "apk update"; cat /tmp/feed.log; }
apk add tailbond >/tmp/feed.log 2>&1 && [ "$(ver)" = 99.2.0-r1 ] \
	&& ok "apk add tailbond from the feed (packages.adb and the apk through http)" || { bad "install from feed"; cat /tmp/feed.log; }
[ -f "$K" ] && ok "the package owns the key (a conffile); the key added first didn't clash" || bad "key gone after feed install"
serve feed2
feed_update && apk add --upgrade tailbond >>/tmp/feed.log 2>&1 && [ "$(ver)" = 99.3.0-r1 ] \
	&& ok "a new release: apk update + add --upgrade to 99.3.0-r1" || { bad "upgrade from feed"; cat /tmp/feed.log; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && [ -f "$K" ] && ok "settings and the key stay over the upgrade" || bad "upgrade lost settings or key"
for v in tampered otherkey; do
	serve feed-$v
	rejected 99.2.0-r1 && ok "feed-$v: refused (the last good index stays; opkg deleted its list)" || { bad "feed-$v accepted"; cat /tmp/feed.log /tmp/feed-add.log; }
done
serve feed2
feed_update && ! grep -qi 'untrusted\|signature\|invalid' /tmp/feed.log && [ "$(ver)" = 99.3.0-r1 ] \
	&& ok "the good feed checks again" || { bad "good feed after rejections"; cat /tmp/feed.log; }
apk del tailbond >/tmp/feed.log 2>&1
[ ! -e "$K" ] && ok "apk del tailbond removes its key (an unmodified conffile)" || { bad "key left after del"; cat /tmp/feed.log; }
# no key by hand: the package installed by hand brings the key
serve feed1
apk add --allow-untrusted /t/feed1/feed/aarch64_cortex-a53/tailbond-99.2.0-r1.apk >/tmp/feed.log 2>&1 && sed -i 's/^tailbond[><=].*/tailbond/' /etc/apk/world && [ -f "$K" ] \
	&& feed_update && serve feed2 \
	&& feed_update && apk add --upgrade tailbond >>/tmp/feed.log 2>&1 && [ "$(ver)" = 99.3.0-r1 ] \
	&& ok "after one install by hand, the feed checks with the package's key and upgrades it" || { bad "manual install then feed"; cat /tmp/feed.log; }
apk del tailbond >/dev/null 2>&1
kill $U2 2>/dev/null
cp /tmp/customfeeds.list /etc/apk/repositories.d/customfeeds.list

echo "== apk's downloader through a real GitHub releases/latest/download redirect"
# a stable-name asset of a public repository stands in until tailbond's
# feed is published (an apk feed can't live behind one, but the downloader
# is uclient-fetch either way)
gh=https://github.com/jqlang/jq/releases/latest/download/jq-linux-arm64
got=0
for try in 1 2 3; do
	if wget -q -O /tmp/gh.bin "$gh" && [ "$(head -c 4 /tmp/gh.bin | tail -c 3)" = ELF ]; then got=1; break; fi
	sleep $((try * 3))
done
if [ "$got" = 1 ]; then
	ok "wget followed github.com's latest/download redirects over HTTPS"
elif wget -q -O /dev/null https://github.com/ 2>/dev/null; then
	bad "github.com answers, but wget didn't get $gh"
else
	echo "  SKIP GitHub redirect: github.com unreachable from the container"
fi

[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit $fail
IN
chmod +x "$TMP/test.sh"
# PODMAN: the podman command (CI: "sudo podman"; rootful, as on the arm64
# runner rootless networking can't reach the feed).
read -ra podman <<<"${PODMAN:-podman}"
"${podman[@]}" run --rm --cap-add NET_ADMIN --platform linux/arm64 -v "$TMP:/t:z" "$IMAGE" /t/test.sh 2>&1 | grep -v 'image platform'
