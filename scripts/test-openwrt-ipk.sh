#!/usr/bin/env bash
# Install test of the .ipk in emulated OpenWrt 23.05 arm64 (podman + qemu):
# install, GL admin panel files present, the agent runs from the rendered
# config, the GL /rpc backend module ("tailbond" object, run with the
# router's own lua + cjson) answers status/api calls against it, bypass
# rules (the image's plain dnsmasq has no nftset: website rules wait, the
# rest loads; nft/ip checks need netlink, i.e. a native arm64 host),
# render-config keeps the panel's servers and rules, uninstall leaves
# nothing behind.
#
#   scripts/test-openwrt-ipk.sh [path/to/tailbond_*_aarch64_cortex-a53.ipk]
#
# The image has no feed for our Depends (kmod-tun, ip-full, ...), so the
# test strips them; lua + lua-cjson (present on GL firmware) come from the
# OpenWrt feed, so the container needs network.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IPK="${1:-$(find "$ROOT/out/ipk" -name "tailbond_*_aarch64_cortex-a53.ipk" | sort | tail -1)}"
IMAGE=docker.io/openwrt/rootfs:armsr-armv8-23.05.0
[ -f "$IPK" ] || { echo "no ipk: run packaging/openwrt/build-ipk.sh"; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
# repack without Depends
mkdir -p "$TMP/x" "$TMP/c"
tar -xzf "$IPK" -C "$TMP/x"
tar -xzf "$TMP/x/control.tar.gz" -C "$TMP/c"
sed -i '/^Depends:/d' "$TMP/c/control"
tar --format=ustar --owner=0 --group=0 -C "$TMP/c" -czf "$TMP/x/control.tar.gz" .
tar --format=ustar --owner=0 --group=0 -C "$TMP/x" -czf "$TMP/tailbond.ipk" ./debian-binary ./data.tar.gz ./control.tar.gz

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
opkg remove tailbond >/tmp/remove.log 2>&1 && ok "opkg remove" || { cat /tmp/remove.log; bad "opkg remove"; }
for f in /usr/bin/tailbondd /usr/lib/oui-httpd/rpc/tailbond /www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/share/oui/menu.d/tailbond.json \
	/usr/share/gl-validator.d/tailbond.lua /etc/nginx/gl-conf.d/tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/init.d/tailbond; do
	[ -e "$f" ] && bad "left behind: $f"
done
[ -e /etc/rc.d/S99tailbond ] && bad "init still enabled" || ok "files and init links removed"
[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit $fail
IN
chmod +x "$TMP/test.sh"
podman run --rm --cap-add NET_ADMIN --platform linux/arm64 -v "$TMP:/t:z" "$IMAGE" /t/test.sh 2>&1 | grep -v 'image platform'
