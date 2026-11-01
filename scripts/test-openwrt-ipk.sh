#!/usr/bin/env bash
# Install test of the .ipk in emulated OpenWrt 23.05 arm64 (podman + qemu):
# install, GL admin panel files present, the agent runs from the rendered
# config, the GL /rpc backend module ("tailbond" object, run with the
# router's own lua + cjson) answers status/api calls against it, uninstall
# leaves nothing behind.
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
podman run --rm --platform linux/arm64 -v "$TMP:/t:z" "$IMAGE" /t/test.sh 2>&1 | grep -v 'image platform'
