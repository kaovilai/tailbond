#!/usr/bin/env bash
# Install test of the .ipk in emulated OpenWrt 23.05 arm64 (podman + qemu):
# install, GL admin panel files present, the agent runs from the rendered
# config, the GL /rpc backend module ("tailbond" object, run with the
# router's own lua + cjson) answers status/api calls against it, bypass
# rules (the image's plain dnsmasq has no nftset: website rules wait, the
# rest loads; nft/ip checks need netlink, i.e. a native arm64 host),
# render-config keeps the panel's servers and rules, uninstall leaves
# nothing behind. Updates (docs/updates.md): an upgrade from an older
# package keeps the settings; update-ipk installs a newer package and rolls
# back one whose agent never answers; a simulated firmware upgrade (the
# sysupgrade hook's keep list restored onto a system without the package)
# reinstalls it at boot from the kept cache, and from a home server mirror
# when the cache is gone.
#
#   scripts/test-openwrt-ipk.sh [path/to/tailbond_*_aarch64_cortex-a53.ipk]
#
# The image has no feed for our Depends (kmod-tun, ip-full, ...), so the
# test strips them; lua + lua-cjson (present on GL firmware) and uhttpd (the
# fake mirror) come from the OpenWrt feed, so the container needs network.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IPK="${1:-$(find "$ROOT/out/ipk" -name "tailbond_*_aarch64_cortex-a53.ipk" | sort | tail -1)}"
IMAGE=docker.io/openwrt/rootfs:armsr-armv8-23.05.0
[ -f "$IPK" ] || { echo "no ipk: run packaging/openwrt/build-ipk.sh"; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
# repack OUT [VERSION [TAILBONDD]]: the ipk without Depends, optionally with
# another control Version and another /usr/bin/tailbondd
repack() {
    local w="$TMP/repack"
    rm -rf "$w" && mkdir -p "$w/x" "$w/c"
    tar -xzf "$IPK" -C "$w/x"
    tar -xzf "$w/x/control.tar.gz" -C "$w/c"
    sed -i '/^Depends:/d' "$w/c/control"
    [ -z "${2:-}" ] || sed -i "s/^Version: .*/Version: $2/" "$w/c/control"
    tar --format=ustar --owner=0 --group=0 -C "$w/c" -czf "$w/x/control.tar.gz" .
    if [ -n "${3:-}" ]; then
        mkdir -p "$w/d"
        tar -xzf "$w/x/data.tar.gz" -C "$w/d"
        install -m 0755 "$3" "$w/d/usr/bin/tailbondd"
        tar --format=ustar --owner=0 --group=0 -C "$w/d" -czf "$w/x/data.tar.gz" .
    fi
    tar --format=ustar --owner=0 --group=0 -C "$w/x" -czf "$1" ./debian-binary ./data.tar.gz ./control.tar.gz
}
repack "$TMP/tailbond.ipk"
# For the update tests: an older package, a newer one, and a newer one whose
# agent exits at once (rollback).
repack "$TMP/tailbond_0.0.1-r1_aarch64_cortex-a53.ipk" 0.0.1-r1
repack "$TMP/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk" 99.0.0-r1
printf '#!/bin/sh\necho "broken build" >&2\nexit 1\n' >"$TMP/broken-tailbondd"
repack "$TMP/tailbond_99.1.0-r1_aarch64_cortex-a53.ipk" 99.1.0-r1 "$TMP/broken-tailbondd"
# A pairing link whose payload names the server (the mirror fallback finds
# it through `tailscale ip`).
printf '%s' '{"v":1,"name":"slate7","server_name":"homebox","sni":"homebox.tail1234.ts.net","server":"203.0.113.7:443"}' |
    base64 -w0 | tr '+/' '-_' | tr -d '=' >"$TMP/pair-d.txt"

cat >"$TMP/test.sh" <<'IN'
#!/bin/sh
set -u
fail=0
ok() { echo "  PASS $*"; }
bad() { echo "  FAIL $*"; fail=1; }
mkdir -p /var/lock /var/run /tmp/sysinfo
printf 'arch all 1\narch noarch 1\narch aarch64_generic 10\narch aarch64_cortex-a53 20\n' >>/etc/opkg.conf
echo "GL.iNet GL-BE3600" >/tmp/sysinfo/model
opkg update >/dev/null 2>&1 && opkg install lua lua-cjson >/dev/null 2>&1 || { echo "need lua + lua-cjson from the feed"; exit 2; }

echo "== install"
opkg install /t/tailbond.ipk >/tmp/install.log 2>&1 && ok "opkg install" || { cat /tmp/install.log; bad "opkg install"; }
for f in /usr/bin/tailbondd /usr/bin/mqvpn /etc/init.d/tailbond /usr/share/oui/menu.d/tailbond.json \
	/www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/lib/oui-httpd/rpc/tailbond /usr/share/gl-validator.d/tailbond.lua \
	/etc/nginx/gl-conf.d/tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/tailbond/tailbond.json; do
	[ -e "$f" ] && ok "$f" || bad "missing $f"
done
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

uclient-fetch -q -O /tmp/self.ipk http://127.0.0.1:8077/tailbond-glinet.ipk &&
	tar -xzOf /tmp/self.ipk ./control.tar.gz | tar -xzOf - ./control | grep -q '^Package: tailbond' &&
	tar -xzOf /tmp/self.ipk ./data.tar.gz | tar -tzf - | grep -qx './etc/uci-defaults/90-tailbond' &&
	! tar -xzOf /tmp/self.ipk ./data.tar.gz | tar -tzf - | grep -q '^./etc/tailbond/' &&
	ok "GET /tailbond-glinet.ipk rebuilds the installed package (pristine uci-defaults, no state)" || bad "self package"

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
export TB_PKG_PURGE_DELAY=2 # postrm drops the package cache after this delay
opkg remove tailbond >/tmp/remove.log 2>&1 && ok "opkg remove" || { cat /tmp/remove.log; bad "opkg remove"; }
for f in /usr/bin/tailbondd /usr/lib/oui-httpd/rpc/tailbond /www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/share/oui/menu.d/tailbond.json \
	/usr/share/gl-validator.d/tailbond.lua /etc/nginx/gl-conf.d/tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/init.d/tailbond; do
	[ -e "$f" ] && bad "left behind: $f"
done
[ -e /etc/rc.d/S99tailbond ] && bad "init still enabled" || ok "files and init links removed"
sleep 4
! grep -q 'tailbond-reinstall' /etc/rc.local && [ ! -e /etc/tailbond/pkg ] && ok "removal drops the rc.local line and the package cache" || bad "rc.local/pkg left"

echo "== upgrade from an older package keeps the settings"
OLD=/t/tailbond_0.0.1-r1_aarch64_cortex-a53.ipk V99=/t/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk BROKEN=/t/tailbond_99.1.0-r1_aarch64_cortex-a53.ipk
ver() { sed -n 's/^Version: //p' /usr/lib/opkg/info/tailbond.control 2>/dev/null; }
opkg install $OLD >/tmp/old.log 2>&1 && [ "$(ver)" = 0.0.1-r1 ] && ok "older package installed" || { bad "install old"; cat /tmp/old.log; }
uci set tailbond.main.keep_test=kept && uci commit tailbond
export PAIR="tailbond://pair?d=$(cat /t/pair-d.txt)"
lua -e '
local cjson = require "cjson"
local f = io.open("/etc/tailbond/tailbond.json"); local c = cjson.decode(f:read("*a")); f:close()
c.client.servers = { { name = "homebox", pairing_uri = os.getenv("PAIR") } }
f = io.open("/etc/tailbond/tailbond.json", "w"); f:write(cjson.encode(c)); f:close()' 
opkg install /t/tailbond.ipk >/tmp/up.log 2>&1 && NEWV="$(ver)" && [ "$NEWV" != 0.0.1-r1 ] && ok "upgraded to $NEWV" || { bad "upgrade"; cat /tmp/up.log; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "/etc/config/tailbond kept" || bad "uci lost"
[ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.servers[0].name')" = homebox ] && ok "/etc/tailbond/tailbond.json kept (servers)" || bad "tailbond.json lost"
opkg install --force-reinstall /t/tailbond.ipk >/dev/null 2>&1
[ "$(grep -c '# tailbond-reinstall$' /etc/rc.local)" = 1 ] && ok "rc.local line stays single over re-installs" || bad "rc.local line duplicated"

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
nipk() { ls /etc/tailbond/pkg/*.ipk 2>/dev/null | wc -l; }
export TB_RESTART=/tmp/restart.sh TB_WAS_CONNECTED=0 TB_ROLE=client TB_WAIT=30
/usr/lib/tailbond/update-ipk $V99 ""
[ "$(res result)" = ok ] && [ "$(res version)" = 99.0.0-r1 ] && [ "$(res from)" = "$NEWV" ] && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "update to 99.0.0-r1: result ok, from $NEWV" || { bad "update-ipk happy path"; cat $R /tmp/tailbond-update.log; }
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "agent answers after the update" || bad "agent down after update"
[ "$(nipk)" = 2 ] && [ "$(cat /etc/tailbond/pkg/current)" = tailbond_99.0.0-r1_aarch64_cortex-a53.ipk ] \
	&& [ "$(/usr/lib/tailbond/pkg-cache ipk-version "/etc/tailbond/pkg/$(cat /etc/tailbond/pkg/previous)")" = "$NEWV" ] \
	&& ok "cache: the new package + the running one saved as the rollback copy" || { bad "cache after update"; ls -la /etc/tailbond/pkg; }
TB_WAIT=20 /usr/lib/tailbond/update-ipk $BROKEN /etc/tailbond/pkg/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk
[ "$(res result)" = rolled_back ] && [ "$(res version)" = 99.1.0-r1 ] && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "an agent that never answers is rolled back: $(res detail)" || { bad "rollback"; cat $R /tmp/tailbond-update.log; }
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "previous agent answers again" || bad "agent down after rollback"
[ "$(nipk)" = 2 ] && [ "$(cat /etc/tailbond/pkg/current)" = tailbond_99.0.0-r1_aarch64_cortex-a53.ipk ] \
	&& ok "cache: still two packages, current = the restored one" || { bad "cache after rollback"; ls -la /etc/tailbond/pkg; }
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
	TB_PKG_PURGE_DELAY=99999 opkg remove tailbond >/dev/null 2>&1 # sysupgrade runs no package scripts
	rm -rf /etc/tailbond /etc/config/tailbond /tmp/tailbond-reinstall.ran
	tar -xzf /tmp/sysupgrade.tgz -C /
}
sysupgrade_list /tmp/conffiles
grep -qx /etc/tailbond/pkg/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk /tmp/conffiles && ! grep -q 99.1.0 /tmp/conffiles \
	&& grep -qx /etc/tailbond/pkg/reinstall /tmp/conffiles && grep -qx /etc/rc.local /tmp/conffiles \
	&& ok "keep list: current package, reinstall script, rc.local; not the previous package" || { bad "keep list"; grep -E 'tailbond|rc.local' /tmp/conffiles; }
flash /tmp/conffiles
[ ! -f /usr/lib/opkg/info/tailbond.control ] && [ ! -e /usr/bin/tailbondd ] && grep -q '# tailbond-reinstall$' /etc/rc.local \
	&& ok "after the flash: no package, kept rc.local line and cache" || bad "flash simulation"
sh /etc/rc.local
i=0; while [ $i -lt 120 ] && [ "$(res detail)" != reinstalled_after_firmware_upgrade ]; do sleep 1; i=$((i+1)); done
[ "$(ver)" = 99.0.0-r1 ] && [ -x /usr/bin/tailbondd ] && [ "$(res result)" = ok ] && [ "$(res detail)" = reinstalled_after_firmware_upgrade ] \
	&& ok "boot reinstalled 99.0.0-r1 from the cache (update-result: reinstalled_after_firmware_upgrade)" || { bad "reinstall at boot"; logread 2>/dev/null | tail -5; cat /tmp/tailbond-reinstall.log; cat $R; }
[ "$(uci -q get tailbond.main.keep_test)" = kept ] && [ "$(jsonfilter -i /etc/tailbond/tailbond.json -e '@.client.servers[0].name')" = homebox ] \
	&& ok "settings intact after the reinstall" || bad "settings lost in reinstall"
[ "$(grep -c '# tailbond-reinstall$' /etc/rc.local)" = 1 ] && ok "still one rc.local line" || bad "rc.local lines"
flash /tmp/conffiles
mkdir -p /tmp/tailbond-reinstall.ran # this boot's run already happened
/etc/tailbond/pkg/reinstall boot
[ ! -f /usr/lib/opkg/info/tailbond.control ] && ok "at most once per boot" || bad "ran twice in one boot"

echo "== reinstall from the home server's mirror when the cache is gone"
rm -f /etc/tailbond/pkg/*.ipk /etc/tailbond/pkg/current /etc/tailbond/pkg/previous
if opkg install uhttpd >/dev/null 2>&1; then
	M=/tmp/mirror/updates
	mkdir -p $M/files
	n=tailbond_99.0.0-r1_aarch64_cortex-a53.ipk
	cp $V99 $M/files/$n
	chmod 644 $M/files/$n # uhttpd serves world-readable files only
	mf() { # SHA
		printf '{"format":1,"channel":"stable","version":"99.0.0","artifacts":[{"name":"tailbond-linux-x86_64.tar.gz","kind":"linux","arch":"x86_64","url":"x","sha256":"0","size":1},{"name":"%s","kind":"ipk","arch":"aarch64_cortex-a53","url":"x","sha256":"%s","size":%s}]}\n' \
			"$n" "$1" "$(wc -c <$M/files/$n | tr -d ' ')" >$M/stable.json
	}
	mf 0000000000000000000000000000000000000000000000000000000000000000
	uhttpd -f -p 127.0.0.1:8099 -h /tmp/mirror >/dev/null 2>&1 &
	U1=$!
	sleep 1
	TB_MIRROR=http://127.0.0.1:8099 TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
	[ ! -f /usr/lib/opkg/info/tailbond.control ] && [ ! -e /etc/tailbond/pkg/$n ] && grep -q "doesn't match the manifest" /tmp/reinstall.err \
		&& ok "an ipk that doesn't match the manifest's sha256 is refused" || { bad "bad sha256 check"; cat /tmp/reinstall.err; }
	mf "$(sha256sum $M/files/$n | cut -d' ' -f1)"
	# The server by name, through a stand-in tailscale CLI, on :8077 as the real mirror.
	printf '#!/bin/sh\n[ "$1 $2 $3" = "ip -4 homebox" ] && echo 127.0.0.1\n' >/usr/sbin/tailscale && chmod +x /usr/sbin/tailscale
	uhttpd -f -p 127.0.0.1:8077 -h /tmp/mirror >/dev/null 2>&1 &
	U2=$!
	sleep 1
	TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
	[ "$(ver)" = 99.0.0-r1 ] && [ "$(cat /etc/tailbond/pkg/current)" = $n ] && [ "$(res detail)" = reinstalled_after_firmware_upgrade ] \
		&& ok "found the paired server (pairing link -> tailscale ip), verified and installed its ipk" || { bad "mirror reinstall"; cat /tmp/reinstall.err /tmp/tailbond-reinstall.log; }
	[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "settings intact" || bad "settings lost"
	kill $U1 $U2 2>/dev/null
	rm -f /usr/sbin/tailscale
else
	echo "  SKIP mirror fallback: uhttpd not installable from the feed"
fi

[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit $fail
IN
chmod +x "$TMP/test.sh"
podman run --rm --cap-add NET_ADMIN --platform linux/arm64 -v "$TMP:/t:z" "$IMAGE" /t/test.sh 2>&1 | grep -v 'image platform'
