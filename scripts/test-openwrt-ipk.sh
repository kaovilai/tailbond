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
# reinstalls it at boot from the kept cache, and from the GitHub release
# (checked against SHA256SUMS) when the cache is gone. The signed opkg feed
# (docs/glinet.md, "Install and update from GL's Plug-ins page"): an index
# built and signed here (scripts/release/opkg-feed.sh, usign from source,
# a throwaway key), served behind a redirect like GitHub's, added to
# customfeeds.conf as GL's UI adds it, check_signature on: rejected until
# the key is added, then opkg update + install + upgrade from it; the
# router's own usign rejects a changed Packages and another key; uninstall
# removes the key the package installed. Plus opkg's downloader (wget)
# through a real GitHub releases/latest/download redirect.
#
#   scripts/test-openwrt-ipk.sh [path/to/tailbond_*_aarch64_cortex-a53.ipk]
#
# The image has no feed for our Depends (kmod-tun, ip-full, ...), so the
# test strips them; lua + lua-cjson (present on GL firmware) and uhttpd (a
# fake GitHub) come from the OpenWrt feed, so the container needs network.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IPK="${1:-$(find "$ROOT/out/ipk" -name "tailbond_*_aarch64_cortex-a53.ipk" | sort | tail -1)}"
IMAGE=docker.io/openwrt/rootfs:armsr-armv8-23.05.0
[ -f "$IPK" ] || { echo "no ipk: run packaging/openwrt/build-ipk.sh"; exit 2; }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
# repack OUT [VERSION [TAILBONDD [PUBKEY]]]: the ipk without Depends,
# optionally with another control Version, another /usr/bin/tailbondd, and
# a usign public key shipped as the feed key (/etc/opkg/keys/<fingerprint>,
# a conffile), as build-ipk.sh ships the committed one
repack() {
    local w="$TMP/repack"
    rm -rf "$w" && mkdir -p "$w/x" "$w/c"
    tar -xzf "$IPK" -C "$w/x"
    tar -xzf "$w/x/control.tar.gz" -C "$w/c"
    sed -i '/^Depends:/d' "$w/c/control"
    [ -z "${2:-}" ] || sed -i "s/^Version: .*/Version: $2/" "$w/c/control"
    if [ -n "${3:-}${4:-}" ]; then
        mkdir -p "$w/d"
        tar -xzf "$w/x/data.tar.gz" -C "$w/d"
        [ -z "${3:-}" ] || install -m 0755 "$3" "$w/d/usr/bin/tailbondd"
        if [ -n "${4:-}" ]; then
            local fp
            fp="$("$USIGN" -F -p "$4")"
            install -d -m 0755 "$w/d/etc/opkg/keys"
            install -m 0644 "$4" "$w/d/etc/opkg/keys/$fp"
            grep -qx "/etc/opkg/keys/$fp" "$w/c/conffiles" || echo "/etc/opkg/keys/$fp" >>"$w/c/conffiles"
        fi
        tar --format=ustar --owner=0 --group=0 -C "$w/d" -czf "$w/x/data.tar.gz" .
    fi
    tar --format=ustar --owner=0 --group=0 -C "$w/c" -czf "$w/x/control.tar.gz" .
    tar --format=ustar --owner=0 --group=0 -C "$w/x" -czf "$1" ./debian-binary ./data.tar.gz ./control.tar.gz
}
repack "$TMP/tailbond.ipk"
# For the update tests: an older package, a newer one, and a newer one whose
# agent exits at once (rollback).
repack "$TMP/tailbond_0.0.1-r1_aarch64_cortex-a53.ipk" 0.0.1-r1
repack "$TMP/tailbond_99.0.0-r1_aarch64_cortex-a53.ipk" 99.0.0-r1
printf '#!/bin/sh\necho "broken build" >&2\nexit 1\n' >"$TMP/broken-tailbondd"
repack "$TMP/tailbond_99.1.0-r1_aarch64_cortex-a53.ipk" 99.1.0-r1 "$TMP/broken-tailbondd"
# The feed key the repository commits (gen-opkg-key.sh) is in the package,
# as a conffile, as is: routers that install it trust the release's feed.
for k in "$ROOT"/packaging/openwrt/files/etc/opkg/keys/*; do
    [ -f "$k" ] || continue
    tar -xzOf "$IPK" ./data.tar.gz | tar -xzOf - "./etc/opkg/keys/${k##*/}" | cmp -s - "$k" &&
        tar -xzOf "$IPK" ./control.tar.gz | tar -xzOf - ./conffiles | grep -qx "/etc/opkg/keys/${k##*/}" ||
        { echo "FAIL $IPK doesn't ship $k as a conffile"; exit 1; }
    echo "  PASS the package ships the feed key ${k##*/} (a conffile)"
done
# The signed opkg feed: the release's index and signature (opkg-feed.sh,
# usign built from OpenWrt's source) for packages that ship a throwaway
# key; v1 to install, v2 to upgrade to, a changed Packages (signature of
# the original), one signed by a key the router doesn't have.
USIGN="$("$ROOT/scripts/release/usign.sh")"
"$USIGN" -G -s "$TMP/feed.sec" -p "$TMP/feed.pub"
"$USIGN" -G -s "$TMP/other.sec" -p "$TMP/other.pub"
"$USIGN" -F -p "$TMP/feed.pub" >"$TMP/feed.fp"
feed() { # DIR VERSION
    mkdir -p "$1"
    repack "$1/tailbond_$2_aarch64_cortex-a53.ipk" "$2" "" "$TMP/feed.pub"
    cp "$1/tailbond_$2_aarch64_cortex-a53.ipk" "$1/tailbond-openwrt-aarch64_cortex-a53.ipk" # the stable copy, not indexed
    "$ROOT/scripts/release/opkg-feed.sh" "$1" "$TMP/feed.sec" >/dev/null
}
feed "$TMP/feed1" 99.2.0-r1
feed "$TMP/feed2" 99.3.0-r1
cp -r "$TMP/feed1" "$TMP/feed-tampered"
sed -i 's/^Version: 99.2.0-r1$/Version: 99.2.1-r1/' "$TMP/feed-tampered/Packages"
gzip -9n -c "$TMP/feed-tampered/Packages" >"$TMP/feed-tampered/Packages.gz"
cp -r "$TMP/feed1" "$TMP/feed-otherkey"
rm "$TMP/feed-otherkey/Packages.sig"
"$USIGN" -S -m "$TMP/feed-otherkey/Packages" -s "$TMP/other.sec" -x "$TMP/feed-otherkey/Packages.sig"
chmod -R a+rX "$TMP"
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
printf 'arch all 1\narch noarch 1\narch aarch64_generic 10\narch aarch64_cortex-a53 20\n' >>/etc/opkg.conf
echo "GL.iNet GL-BE3600" >/tmp/sysinfo/model
# the feed over the network (a CI runner's network may need a retry)
feed_ok=0
for try in 1 2 3; do
	if opkg update >/tmp/opkg.log 2>&1 && opkg install lua lua-cjson >>/tmp/opkg.log 2>&1; then feed_ok=1; break; fi
	sleep $((try * 5))
done
[ "$feed_ok" = 1 ] || {
	echo "need lua + lua-cjson from the feed:"; tail -n 25 /tmp/opkg.log
	cat /etc/resolv.conf; nslookup downloads.openwrt.org; ip route
	uclient-fetch -O /dev/null https://downloads.openwrt.org/releases/23.05.0/packages/aarch64_generic/base/Packages.gz
	exit 2
}

echo "== install"
opkg install /t/tailbond.ipk >/tmp/install.log 2>&1 && ok "opkg install" || { cat /tmp/install.log; bad "opkg install"; }
for f in /usr/bin/tailbondd /usr/bin/mqvpn /etc/init.d/tailbond /usr/share/oui/menu.d/tailbond.json \
	/www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/lib/oui-httpd/rpc/tailbond /usr/share/gl-validator.d/tailbond.lua \
	/usr/share/tailbond/nginx-tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/tailbond/tailbond.json; do
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
	# bypass-status.json is there from the agent's start (no rules yet): wait
	# for the hook run that loads the two rules added above
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
# the tunnel MTU is a UCI option; without it a relayed router can't shrink the
# tunnel under a 1280-byte uplink path.
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
export TB_PKG_PURGE_DELAY=2 # postrm drops the package cache after this delay
opkg remove tailbond >/tmp/remove.log 2>&1 && ok "opkg remove" || { cat /tmp/remove.log; bad "opkg remove"; }
for f in /usr/bin/tailbondd /usr/lib/oui-httpd/rpc/tailbond /www/views/gl-sdk4-ui-tailbond.common.js.gz /usr/share/oui/menu.d/tailbond.json \
	/usr/share/gl-validator.d/tailbond.lua /etc/nginx/gl-conf.d/tailbond.conf /usr/share/tailbond/nginx-tailbond.conf /etc/hotplug.d/iface/90-tailbond /etc/init.d/tailbond; do
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

echo "== pkg-cache: seeding keeps a package that is downloaded but not installed"
# The router that installed tailbond by hand has no rollback copy, so the
# agent's next update downloads the new ipk into the pkg dir and then
# update-ipk seeds the running package. The prune used to delete the
# download, and `opkg install` failed on a missing file — the update rolled
# back with "the agent didn't answer". Caught on a Flint 3 with 0.1.15.
sd=$(mktemp -d)
cp "$V99" "$sd/new.ipk" && cp "$V99" "$sd/old.ipk" || true
TB_PKG_DIR=$sd TB_PENDING=$sd/new.ipk /usr/lib/tailbond/pkg-cache keep "$sd/old.ipk" >/dev/null 2>&1
[ -f "$sd/new.ipk" ] && [ -f "$sd/old.ipk" ] \
	&& ok "seeding spares the downloaded package" || bad "seeding deleted the downloaded package: $(ls $sd)"
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
nipk() { ls /etc/tailbond/pkg/*.ipk 2>/dev/null | wc -l; }
export TB_RESTART=/tmp/restart.sh TB_WAS_CONNECTED=0 TB_ROLE=client TB_WAIT=30
# The hand-installed router: no rollback copy in the cache, and the new
# package already sitting there, because the agent downloads it before
# update-ipk starts. Seeding then pruned the download, opkg failed on a
# missing file, and the update rolled back — a Flint 3 going 0.1.13 ->
# 0.1.15 was exactly this. The script drives it, so a missing
# TB_PENDING forward fails here too.
rm -rf /etc/tailbond/pkg && mkdir -p /etc/tailbond/pkg
cp $V99 /etc/tailbond/pkg/ && [ -f /etc/tailbond/pkg/$(basename $V99) ] \
	&& ok "no rollback copy, the download sits in the pkg dir" || bad "could not stage the download"
/usr/lib/tailbond/update-ipk /etc/tailbond/pkg/$(basename $V99) ""
[ -f /etc/tailbond/pkg/$(basename $V99) ] && ok "the download survived seeding" || bad "seeding deleted the package it was installing"
[ "$(res result)" = ok ] && [ "$(res version)" = 99.0.0-r1 ] && [ "$(res from)" = "$NEWV" ] && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "update to 99.0.0-r1 from a hand-installed router: result ok, from $NEWV" || { bad "update-ipk happy path"; cat $R /tmp/tailbond-update.log; }
uclient-fetch -q -O - http://127.0.0.1:8077/api/status | jsonfilter -e '@.role' | grep -qx client && ok "agent answers after the update" || bad "agent down after update"
[ "$(nipk)" = 2 ] && [ "$(cat /etc/tailbond/pkg/current)" = tailbond_99.0.0-r1_aarch64_cortex-a53.ipk ] \
	&& [ "$(/usr/lib/tailbond/pkg-cache ipk-version "/etc/tailbond/pkg/$(cat /etc/tailbond/pkg/previous)")" = "$NEWV" ] \
	&& ok "cache: the new package + the running one saved as the rollback copy" || { bad "cache after update"; ls -la /etc/tailbond/pkg; }
/usr/lib/tailbond/update-ipk $V99 "" || true
[ "$(res result)" = failed ] && res detail | grep -q 'already the installed' && [ "$(ver)" = 99.0.0-r1 ] \
	&& ok "the same version again is refused, not reported as done: $(res detail)" || { bad "same-version update"; cat $R /tmp/tailbond-update.log; }
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

echo "== reinstall from the GitHub release when the cached ipk is gone"
if opkg install uhttpd >/dev/null 2>&1; then
	G=/tmp/gh
	n=tailbond_99.0.0-r1_aarch64_cortex-a53.ipk
	mkdir -p $G/dl $G/repos/kaovilai/tailbond/releases/tags
	cp $V99 $G/dl/$n
	cp $V99 $G/dl/tailbond_99.0.0-r1_aarch64_generic.ipk
	# the stable name (docs/glinet.md, Download): reinstall downloads it
	# and keeps it under the versioned name
	sn=tailbond-openwrt-aarch64_cortex-a53.ipk
	cp $V99 $G/dl/$sn
	chmod 644 $G/dl/* # uhttpd serves world-readable files only
	rel() { # FILE: a release with the ipks and SHA256SUMS
		printf '{"tag_name":"v99.0.0","assets":[{"name":"%s","browser_download_url":"http://127.0.0.1:8099/dl/%s"},{"name":"%s","browser_download_url":"http://127.0.0.1:8099/dl/%s"},{"name":"tailbond_99.0.0-r1_aarch64_generic.ipk","browser_download_url":"http://127.0.0.1:8099/dl/tailbond_99.0.0-r1_aarch64_generic.ipk"},{"name":"SHA256SUMS","browser_download_url":"http://127.0.0.1:8099/dl/SHA256SUMS"}]}\n' "$n" "$n" "$sn" "$sn" >"$1"
	}
	rel $G/repos/kaovilai/tailbond/releases/tags/v99.0.0
	rel $G/repos/kaovilai/tailbond/releases/latest
	printf '0000000000000000000000000000000000000000000000000000000000000000  %s\n' $n $sn >$G/dl/SHA256SUMS
	chmod 644 $G/dl/SHA256SUMS $G/repos/kaovilai/tailbond/releases/latest $G/repos/kaovilai/tailbond/releases/tags/v99.0.0
	uhttpd -f -p 127.0.0.1:8099 -h $G >/dev/null 2>&1 &
	U1=$!
	sleep 1
	# the pointer still names the installed version; its ipk is gone
	rm -f /etc/tailbond/pkg/*.ipk
	echo $n >/etc/tailbond/pkg/current
	TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
	[ ! -f /usr/lib/opkg/info/tailbond.control ] && [ ! -e /etc/tailbond/pkg/$n ] && grep -q "doesn't match the release's SHA256SUMS" /tmp/reinstall.err \
		&& ok "an ipk that doesn't match SHA256SUMS is refused" || { bad "bad sha256 check"; cat /tmp/reinstall.err; }
	(cd $G/dl && sha256sum $n $sn tailbond_99.0.0-r1_aarch64_generic.ipk >SHA256SUMS && chmod 644 SHA256SUMS)
	TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /etc/tailbond/pkg/reinstall now 2>/tmp/reinstall.err
	[ "$(ver)" = 99.0.0-r1 ] && [ "$(cat /etc/tailbond/pkg/current)" = $n ] && [ "$(res detail)" = reinstalled_after_firmware_upgrade ] \
		&& grep -q "downloaded $sn from the GitHub release (kept as $n)" /tmp/reinstall.err \
		&& ok "downloaded that version's ipk from the GitHub release, checked it, installed it" || { bad "GitHub reinstall"; cat /tmp/reinstall.err /tmp/tailbond-reinstall.log; }
	[ "$(uci -q get tailbond.main.keep_test)" = kept ] && ok "settings intact" || bad "settings lost"
	# nothing cached at all: the latest release's ipk for this router (cortex-a53 beats generic)
	cp /etc/tailbond/pkg/reinstall /tmp/reinstall
	opkg remove tailbond >/dev/null 2>&1
	mkdir -p /tmp/pkgB
	TB_PKG_DIR=/tmp/pkgB TB_API_URL=http://127.0.0.1:8099 TB_TRIES=1 /tmp/reinstall now 2>/tmp/reinstall.err
	[ "$(ver)" = 99.0.0-r1 ] && [ "$(cat /tmp/pkgB/current)" = $n ] \
		&& ok "no cache: the latest release's ipk for this architecture" || { bad "latest reinstall"; cat /tmp/reinstall.err /tmp/tailbond-reinstall.log; ls /tmp/pkgB; }
	kill $U1 2>/dev/null
else
	echo "  SKIP GitHub fallback: uhttpd not installable from the feed"
fi

echo "== the signed opkg feed (GL: Plug-ins -> Manage Sources -> Add Custom Software Source)"
grep -q '^option check_signature' /etc/opkg.conf && ok "check_signature is on (as on GL 4.x)" || { bad "no check_signature in /etc/opkg.conf"; echo 'option check_signature' >>/etc/opkg.conf; }
if command -v uhttpd >/dev/null 2>&1 || opkg install uhttpd >/dev/null 2>&1; then
	FP="$(cat /t/feed.fp)"
	K="/etc/opkg/keys/$FP"
	F=/tmp/feedsrv
	mkdir -p $F/cgi-bin $F/f
	# GitHub's releases/latest/download/<name> answers with a redirect (to
	# releases/download/<tag>/<name>, then the file's host); so does this
	cat >$F/cgi-bin/latest <<'SH'
#!/bin/sh
printf 'Status: 302 Found\r\nLocation: http://127.0.0.1:8098/f%s\r\n\r\n' "$PATH_INFO"
SH
	chmod 755 $F/cgi-bin/latest
	uhttpd -f -p 127.0.0.1:8098 -h $F -x /cgi-bin >/dev/null 2>&1 &
	U2=$!
	serve() { rm -rf $F/f && cp -r "/t/$1" $F/f && chmod -R a+rX $F/f; }
	serve feed1
	sleep 1
	opkg remove tailbond >/dev/null 2>&1
	rm -f "$K"
	# only this feed: opkg update's result is then this feed's
	mv /etc/opkg/distfeeds.conf /tmp/distfeeds.conf
	cp /etc/opkg/customfeeds.conf /tmp/customfeeds.conf
	# the line GL's "Add Custom Software Source" writes
	echo 'src/gz tailbond http://127.0.0.1:8098/cgi-bin/latest' >>/etc/opkg/customfeeds.conf
	L=/var/opkg-lists/tailbond
	# opkg update exits 0 on a bad signature; it deletes the list instead
	rejected() { opkg update >/tmp/feed.log 2>&1; [ ! -e $L ] && grep -q 'Signature check failed' /tmp/feed.log && ! opkg list 2>/dev/null | grep -qE '^tailbond - 99[.]2[.]'; }
	rejected && ok "before the key is added, opkg update rejects the feed (signature check failed, list deleted)" || { bad "feed accepted without its key"; cat /tmp/feed.log; }
	# the documented first step: the release's tailbond-opkg.pub
	wget -q -O /tmp/tailbond-opkg.pub http://127.0.0.1:8098/cgi-bin/latest/tailbond-opkg.pub && opkg-key add /tmp/tailbond-opkg.pub
	[ -f "$K" ] && ok "opkg-key add put the release's key in /etc/opkg/keys/$FP" || bad "opkg-key add"
	opkg update >/tmp/feed.log 2>&1 && [ -s $L ] && grep -q 'Signature check passed' /tmp/feed.log \
		&& ok "opkg update: the feed's signature checks (router's usign, key from the release)" || { bad "opkg update"; cat /tmp/feed.log; }
	opkg list 2>/dev/null | grep -q '^tailbond - 99.2.0-r1' && ok "opkg list shows tailbond 99.2.0-r1 (the stable copy isn't a second entry)" || { bad "opkg list"; opkg list; }
	opkg install tailbond >/tmp/feed.log 2>&1 && [ "$(ver)" = 99.2.0-r1 ] \
		&& ok "opkg install tailbond from the feed (Packages.gz, .sig and the ipk through the redirect)" || { bad "install from feed"; cat /tmp/feed.log; }
	if grep -qiE 'clash|already provided' /tmp/feed.log; then
		bad "the key added first clashed with the package's"; cat /tmp/feed.log
	elif grep -qx "$K" /usr/lib/opkg/info/tailbond.list && awk 'BEGIN { RS = "" } /^Package: tailbond\n/' /usr/lib/opkg/status | grep -q "^ $K "; then
		ok "the package owns the key (a conffile); the key added first didn't clash"
	else
		bad "key not owned as a conffile"; cat /usr/lib/opkg/info/tailbond.list; cat /usr/lib/opkg/status
	fi
	serve feed2
	opkg update >/tmp/feed.log 2>&1 && opkg upgrade tailbond >>/tmp/feed.log 2>&1 && [ "$(ver)" = 99.3.0-r1 ] \
		&& ok "a new release: opkg update + upgrade to 99.3.0-r1" || { bad "upgrade from feed"; cat /tmp/feed.log; }
	[ "$(uci -q get tailbond.main.keep_test)" = kept ] && [ -f "$K" ] && ok "settings and the key stay over the upgrade" || bad "upgrade lost settings or key"
	for v in tampered otherkey; do
		serve feed-$v
		rejected && ok "feed-$v: opkg update rejects it (the good list from before is deleted too)" || { bad "feed-$v accepted"; cat /tmp/feed.log; }
	done
	serve feed2
	opkg update >/dev/null 2>&1 && [ -s $L ] && opkg list 2>/dev/null | grep -q '^tailbond - 99.3.0-r1' && ok "the good feed checks again" || bad "good feed after rejections"
	opkg remove tailbond >/tmp/feed.log 2>&1
	[ ! -e "$K" ] && ok "opkg remove tailbond removes its key" || { bad "key left after remove"; cat /tmp/feed.log; }
	# no opkg-key: the package installed by hand (LuCI upload) brings the key
	serve feed1
	opkg install /t/feed1/tailbond_99.2.0-r1_aarch64_cortex-a53.ipk >/tmp/feed.log 2>&1 && [ -f "$K" ] \
		&& opkg update >>/tmp/feed.log 2>&1 && grep -q 'Signature check passed' /tmp/feed.log && serve feed2 \
		&& opkg update >>/tmp/feed.log 2>&1 && opkg upgrade tailbond >>/tmp/feed.log 2>&1 && [ "$(ver)" = 99.3.0-r1 ] \
		&& ok "after one install by hand, the source checks with the package's key and upgrades it" || { bad "manual install then feed"; cat /tmp/feed.log; }
	opkg remove tailbond >/dev/null 2>&1
	kill $U2 2>/dev/null
	mv /tmp/distfeeds.conf /etc/opkg/distfeeds.conf
	cp /tmp/customfeeds.conf /etc/opkg/customfeeds.conf
else
	echo "  SKIP opkg feed: uhttpd not installable from the feed"
fi

echo "== opkg's downloader through a real GitHub releases/latest/download redirect"
# opkg runs "wget -q -O FILE URL" (here uclient-fetch); a stable-name asset
# of a public repository stands in until tailbond's feed is published
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
# PODMAN: the podman command (CI: "sudo podman"; rootless networking on the
# arm64 runner can't reach the feed).
read -ra podman <<<"${PODMAN:-podman}"
"${podman[@]}" run --rm --cap-add NET_ADMIN --platform linux/arm64 -v "$TMP:/t:z" "$IMAGE" /t/test.sh 2>&1 | grep -v 'image platform'
