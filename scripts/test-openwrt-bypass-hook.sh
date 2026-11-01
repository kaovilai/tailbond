#!/usr/bin/env bash
# The OpenWrt bypass-apply hook on real OpenWrt userland: busybox sh and
# ip, firewall4's nft, the image's plain dnsmasq (no nftset), run natively
# in the x86-64 OpenWrt 23.05 rootfs (podman). The rules come from the
# agent's own renderer (internal/bypass), so this checks that what tailbondd
# writes is what the router's tools accept:
#   apply: table inet tailbond with the rules and the LAN set, ip rules
#   1070-1073, the WAN table, website rules waiting (no nftset) with the
#   reason; re-apply changes nothing and keeps the counters; counters;
#   clear removes everything.
#
#   scripts/test-openwrt-bypass-hook.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE=docker.io/openwrt/rootfs:x86-64-23.05.0
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/state" "$TMP/hooks" "$TMP/render"

cp "$ROOT/packaging/openwrt/hooks/bypass-apply" "$ROOT/packaging/openwrt/hooks/lib.sh" "$TMP/hooks/"
# render with the agent's code: a website (direct), a device (always
# tailbond), Wi-Fi calling (only the wan connection)
cat >"$TMP/render/main_test.go" <<'EOF'
package bypass

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
)

func TestWriteFixture(t *testing.T) {
	dir := os.Getenv("TB_FIXTURE_DIR")
	rules := []config.BypassRule{
		{ID: "bank", Kind: "domain", Match: "bank.example", Target: "direct"},
		{ID: "laptop", Kind: "device", Match: "aa:bb:cc:dd:ee:ff", Target: "tunnel"},
		{ID: "wifical", Kind: "port", Match: "udp/4500", Target: "wan", WAN: "wan"},
	}
	f := Render(rules, Options{LocalDNS: true, WANDNS: map[string][]string{"wan": {"192.0.2.53"}}, WANDev: map[string]string{"wan": "eth1"}})
	w := func(n, s string) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w("bypass.json", `{"rules":[{"id":"bank"},{"id":"laptop"},{"id":"wifical"}]}`)
	w("bypass.nft", f.Nft)
	w("bypass.ipset", f.IPSet)
	w("bypass.iptables", f.IPTables)
	w("bypass.ip6tables", f.IP6Tables)
	w("bypass-nftset.conf", f.DnsmasqNftset)
	w("bypass-ipset.conf", f.DnsmasqIPSet)
	w("bypass-wans", f.WANList())
	w("bypass-marks", MarksLine(0))
	w("bypass-domains", "bank.example\n")
}
EOF
cp "$TMP/render/main_test.go" "$ROOT/agent/internal/bypass/zz_fixture_test.go"
(cd "$ROOT/agent" && TB_FIXTURE_DIR="$TMP/state" go test ./internal/bypass -run TestWriteFixture -count=1 >/dev/null)
rm -f "$ROOT/agent/internal/bypass/zz_fixture_test.go"

cat >"$TMP/test.sh" <<'IN'
#!/bin/sh
set -u
fail=0
ok() { echo "  PASS $*"; }
bad() { echo "  FAIL $*"; fail=1; }
ip link add br-lan type dummy && ip link set br-lan up
ip link add eth1 type dummy && ip addr add 192.0.2.2/24 dev eth1 && ip link set eth1 up
ip route add default via 192.0.2.1 dev eth1 metric 10
touch /etc/config/network; uci set network.lan=interface; uci set network.lan.device=br-lan
uci set network.wan=interface; uci set network.wan.device=eth1; uci commit network
# no ubus/netifd in the container: net_device falls back to UCI
cat >/tmp/ifstatus <<'S'
#!/bin/sh
exit 1
S
chmod +x /tmp/ifstatus
export PATH="/tmp:$PATH" TB_LIB=/t/hooks/lib.sh TB_STATE=/tmp/tbs TB_BYPASS_FILE=/t/state/bypass.json TB_BYPASS_STATUS=/tmp/st.json TB_DNS_WARM=-
H=/t/hooks/bypass-apply
. /t/hooks/lib.sh; echo "lan device: $(net_device lan), wan device: $(net_device wan)"

echo "== apply"
sh $H >/tmp/apply.log 2>&1 && ok "apply exits 0" || { bad "apply"; cat /tmp/apply.log; }
grep -q '"domains":"none"' /tmp/st.json && grep -q dnsmasq-full /tmp/st.json && ok "plain dnsmasq: website rules wait, reason names dnsmasq-full" || bad "status: $(cat /tmp/st.json)"
nft list table inet tailbond >/tmp/t.nft 2>&1 && ok "table inet tailbond loaded by firewall4's nft" || { bad "nft"; cat /tmp/t.nft; }
grep -q 'ether saddr aa:bb:cc:dd:ee:ff counter name "c_laptop"' /tmp/t.nft && grep -q 'udp dport 4500 counter name "c_wifical"' /tmp/t.nft \
	&& ok "device and port rules" || bad "rules: $(cat /tmp/t.nft)"
nft list set inet tailbond lan_ifs | grep -q '"br-lan"' && ok "LAN set holds br-lan" || bad "lan_ifs"
ip rule >/tmp/rules
grep -q '^1070:.*fwmark 0x2000000/0x7000000 lookup main suppress_prefixlength 0' /tmp/rules && grep -q '^1070:.*fwmark 0x3000000/0x7000000 lookup main suppress_prefixlength 0' /tmp/rules \
	&& grep -q '^1071:.*fwmark 0x2000000/0x7000000 lookup 77' /tmp/rules && grep -q '^1072:.*fwmark 0x1000000/0x7000000 lookup main' /tmp/rules \
	&& grep -q '^1073:.*fwmark 0x3000000/0x7000000 lookup 80' /tmp/rules && ok "ip rules 1070-1073 (busybox ip)" || { bad "ip rules"; cat /tmp/rules; }
ip route show table 80 | grep -q 'default via 192.0.2.1 dev eth1' && ok "table 80: default via the wan" || bad "table 80: $(ip route show table 80)"
[ ! -e /tmp/dnsmasq.d/tailbond-bypass.conf ] && ok "no dnsmasq drop-in without nftset" || bad "drop-in written"

echo "== re-apply keeps everything (and the counters)"
before="$(nft -a list table inet tailbond | sed -n 's/.*table inet tailbond { # handle \([0-9]*\).*/\1/p')"
sh $H >/dev/null 2>&1 && ok "re-apply exits 0" || bad "re-apply"
[ "$(ip rule | grep -c '^1072:')" = 1 ] && [ "$(ip rule | grep -c '^1073:')" = 1 ] && ok "no duplicate ip rules" || bad "rules: $(ip rule)"
after="$(nft -a list table inet tailbond | sed -n 's/.*table inet tailbond { # handle \([0-9]*\).*/\1/p')"
[ -n "$before" ] && [ "$before" = "$after" ] && ok "table kept, not reloaded (same rules), so counters survive" || bad "table handle $before -> $after"

echo "== counters"
TB_BYPASS_ACTION=counters TB_BYPASS_STATUS=/tmp/c.json sh $H && grep -q '"laptop":{"connections":0}' /tmp/c.json && grep -q '"bank":' /tmp/c.json \
	&& ok "counters: $(cat /tmp/c.json)" || bad "counters: $(cat /tmp/c.json 2>&1)"

echo "== clear"
TB_BYPASS_ACTION=clear sh $H >/tmp/clear.log 2>&1 && ok "clear exits 0" || { bad "clear"; cat /tmp/clear.log; }
! nft list table inet tailbond >/dev/null 2>&1 && ok "table removed" || bad "table left"
[ "$(ip rule | grep -c 0x7000000)" = 0 ] && [ -z "$(ip route show table 80)" ] && ok "ip rules and table 80 removed" || bad "left: $(ip rule; ip route show table 80)"
grep -qi "error\|can't" /tmp/clear.log && bad "clear printed errors: $(cat /tmp/clear.log)" || ok "clear is quiet"
[ "$fail" = 0 ] && echo "RESULT: PASS" || echo "RESULT: FAIL"
exit $fail
IN
chmod +x "$TMP/test.sh"
podman run --rm --cap-add NET_ADMIN -v "$TMP:/t:z" "$IMAGE" /t/test.sh
