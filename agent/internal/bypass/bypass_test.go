package bypass

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
)

func rules(t *testing.T) []config.BypassRule {
	t.Helper()
	off := false
	in := []config.BypassRule{
		{ID: "bank", Kind: "domain", Match: "bank.example", Target: "direct"},
		{ID: "stream", Kind: "domain", Match: "netflix.com", Target: "wan", WAN: "wwan"},
		{ID: "laptop", Kind: "device", Match: "aa:bb:cc:dd:ee:ff", Target: "tunnel"},
		{ID: "tv", Kind: "device", IP: "192.168.8.40", Target: "direct"},
		{ID: "net", Kind: "ip", Match: "203.0.113.0/24", Target: "wan", WAN: "wan"},
		{ID: "net6", Kind: "ip", Match: "2001:db8::/32", Target: "direct"},
		{ID: "wifical", Kind: "port", Match: "udp/4500", Target: "direct"},
		{ID: "sip", Kind: "port", Match: "udp/5060-5061", Target: "direct", Enabled: &off},
	}
	for i, r := range in {
		n, err := config.NormalizeBypass(r)
		if err != nil {
			t.Fatal(err)
		}
		in[i] = n
	}
	return in
}

func opts() Options {
	return Options{
		LocalDNS: true,
		WANDNS:   map[string][]string{"wan": {"192.168.1.1"}, "wwan": {"10.1.0.1", "10.1.0.2"}},
		WANDev:   map[string]string{"wan": "eth1", "wwan": "phy0-sta0"},
	}
}

func TestMarks(t *testing.T) {
	d, tun, w := Marks(DefaultMask)
	if d != 0x01000000 || tun != 0x02000000 || w(0) != 0x03000000 || w(4) != 0x07000000 {
		t.Fatalf("%#x %#x %#x %#x", d, tun, w(0), w(4))
	}
	if MaxWANsFor(DefaultMask) != 5 || MaxWANsFor(0x0300) != 1 || MaxWANsFor(0x0100) != 0 {
		t.Fatal("MaxWANsFor")
	}
	d, _, w = Marks(0x00000f00)
	if d != 0x100 || w(1) != 0x400 {
		t.Fatalf("shifted %#x %#x", d, w(1))
	}
}

func TestMarksLine(t *testing.T) {
	if got := MarksLine(0); got != "0x07000000 0x01000000 0x02000000\n" {
		t.Fatal(got)
	}
}

func TestRenderNft(t *testing.T) {
	f := Render(rules(t), opts())
	n := f.Nft
	for _, want := range []string{
		"table inet tailbond\ndelete table inet tailbond\ntable inet tailbond {",
		"set lan_ifs {",
		"set bp_bank_4 {\n\t\ttype ipv4_addr",
		"set bp_bank_6 {\n\t\ttype ipv6_addr",
		"ct mark & 0x07000000 != 0 meta mark set ct mark & 0x07000000 return\n\t\tiifname != @lan_ifs return\n",
		`ether saddr aa:bb:cc:dd:ee:ff counter name "c_laptop" meta mark set meta mark & 0xf8ffffff | 0x02000000 ct mark set ct mark & 0xf8ffffff | 0x02000000 accept`,
		`ip saddr 192.168.8.40 counter name "c_tv" meta mark set meta mark & 0xf8ffffff | 0x01000000`,
		`ip daddr 203.0.113.0/24 counter name "c_net" meta mark set meta mark & 0xf8ffffff | 0x03000000`,
		`ip6 daddr 2001:db8::/32 counter name "c_net6"`,
		`udp dport 4500 counter name "c_wifical"`,
		`ip daddr @bp_stream_4 counter name "c_stream" meta mark set meta mark & 0xf8ffffff | 0x04000000`,
		`ip6 daddr @bp_bank_6 counter name "c_bank"`,
	} {
		if !strings.Contains(n, want) {
			t.Errorf("nft missing %q\n%s", want, n)
		}
	}
	if strings.Contains(n, "c_sip") || strings.Contains(n, "5060") {
		t.Error("disabled rule rendered")
	}
	// devices first, then ports, IPs, domains
	pos := func(s string) int { return strings.Index(n, "counter name \""+s+"\"") }
	if !(pos("c_laptop") < pos("c_tv") && pos("c_tv") < pos("c_wifical") && pos("c_wifical") < pos("c_net") && pos("c_net") < pos("c_bank")) {
		t.Errorf("rule order wrong\n%s", n)
	}
	// WAN marks by first use: wwan (domain "stream") comes after wan (ip "net")
	if len(f.WANs) != 2 || f.WANs[0].Name != "wan" || f.WANs[0].Table != 7900 || f.WANs[1].Name != "wwan" || f.WANs[1].Mark != 0x04000000 {
		t.Errorf("wans %+v", f.WANs)
	}
	if f.WANList() != "wan 0x03000000 7900\nwwan 0x04000000 7901\n" {
		t.Errorf("wan list %q", f.WANList())
	}
}

func TestRenderNftSyntax(t *testing.T) {
	nft, err := exec.LookPath("nft")
	if err != nil || os.Geteuid() != 0 {
		t.Skip("needs root and nft (checked by the e2e test otherwise)")
	}
	p := filepath.Join(t.TempDir(), "b.nft")
	os.WriteFile(p, []byte(Render(rules(t), opts()).Nft), 0o600)
	if out, err := exec.Command(nft, "-c", "-f", p).CombinedOutput(); err != nil {
		t.Fatalf("nft -c: %v\n%s", err, out)
	}
}

func TestRenderIPTables(t *testing.T) {
	f := Render(rules(t), opts())
	for _, want := range []string{
		"create tb_bank_4 hash:ip family inet -exist\nflush tb_bank_4\n",
		"create tb_bank_6 hash:ip family inet6 -exist",
	} {
		if !strings.Contains(f.IPSet, want) {
			t.Errorf("ipset missing %q\n%s", want, f.IPSet)
		}
	}
	for _, want := range []string{
		"*mangle\n:tailbond_restore - [0:0]\n-F tailbond_restore\n-A tailbond_restore -m connmark ! --mark 0x0/0x07000000 -j CONNMARK --restore-mark --nfmask 0x07000000 --ctmask 0x07000000\n",
		":tailbond_bypass - [0:0]\n-F tailbond_bypass\n-A tailbond_bypass -m mark ! --mark 0x0/0x07000000 -j RETURN\n",
		`-A tailbond_bypass -m mac --mac-source aa:bb:cc:dd:ee:ff -m mark --mark 0x0/0x07000000 -m comment --comment "tb:laptop" -j MARK --set-xmark 0x02000000/0x07000000`,
		`-A tailbond_bypass -s 192.168.8.40 -m mark --mark 0x0/0x07000000 -m comment --comment "tb:tv"`,
		`-A tailbond_bypass -p udp --dport 4500 -m mark`,
		`-A tailbond_bypass -m set --match-set tb_bank_4 dst -m mark --mark 0x0/0x07000000 -m comment --comment "tb:bank" -j MARK --set-xmark 0x01000000/0x07000000`,
		"-j CONNMARK --save-mark --nfmask 0x07000000 --ctmask 0x07000000\nCOMMIT\n",
	} {
		if !strings.Contains(f.IPTables, want) {
			t.Errorf("iptables missing %q\n%s", want, f.IPTables)
		}
	}
	// family split: v4 prefixes/IP devices only in iptables, v6 only in ip6tables
	if strings.Contains(f.IPTables, "2001:db8") || strings.Contains(f.IP6Tables, "203.0.113") || strings.Contains(f.IP6Tables, "192.168.8.40") {
		t.Error("family mix-up")
	}
	if !strings.Contains(f.IP6Tables, "-d 2001:db8::/32") || !strings.Contains(f.IP6Tables, "--match-set tb_bank_6 dst") || !strings.Contains(f.IP6Tables, "--mac-source aa:bb:cc:dd:ee:ff") {
		t.Errorf("ip6tables\n%s", f.IP6Tables)
	}
}

func TestRenderDnsmasq(t *testing.T) {
	f := Render(rules(t), opts())
	for _, want := range []string{
		"nftset=/bank.example/4#inet#tailbond#bp_bank_4,6#inet#tailbond#bp_bank_6\n",
		"server=/bank.example/192.168.1.1\nserver=/bank.example/10.1.0.1\nserver=/bank.example/10.1.0.2\n",
		"server=/netflix.com/10.1.0.1@phy0-sta0\nserver=/netflix.com/10.1.0.2@phy0-sta0\n",
	} {
		if !strings.Contains(f.DnsmasqNftset, want) {
			t.Errorf("nftset conf missing %q\n%s", want, f.DnsmasqNftset)
		}
	}
	if !strings.Contains(f.DnsmasqIPSet, "ipset=/bank.example/tb_bank_4,tb_bank_6\n") || !strings.Contains(f.DnsmasqIPSet, "server=/bank.example/192.168.1.1") {
		t.Errorf("ipset conf\n%s", f.DnsmasqIPSet)
	}
	if strings.Join(f.Domains, " ") != "bank.example netflix.com" {
		t.Errorf("domains %v", f.Domains)
	}
	o := opts()
	o.LocalDNS = false
	if strings.Contains(Render(rules(t), o).DnsmasqNftset, "server=") {
		t.Error("server= lines with local DNS off")
	}
	tun := []config.BypassRule{{ID: "x", Kind: "domain", Match: "corp.example", Target: "tunnel"}}
	if strings.Contains(Render(tun, opts()).DnsmasqNftset, "server=") {
		t.Error("tunnel domain resolves locally")
	}
}

func TestTooManyWANs(t *testing.T) {
	var rs []config.BypassRule
	for i, w := range []string{"a", "b", "c", "d", "e", "f"} {
		rs = append(rs, config.BypassRule{ID: "r" + string(rune('0'+i)), Kind: "port", Match: "udp/" + string(rune('1'+i)), Target: "wan", WAN: w})
	}
	f := Render(rs, Options{})
	if len(f.WANs) != 5 || len(f.Skipped) != 1 || f.Skipped[0] != "r5" {
		t.Fatalf("wans %d skipped %v", len(f.WANs), f.Skipped)
	}
}

func TestEmpty(t *testing.T) {
	f := Render(nil, opts())
	if !strings.Contains(f.Nft, "chain prerouting") || strings.Contains(f.Nft, "counter name") || f.IPSet != "" || len(f.Domains) != 0 {
		t.Fatalf("%+v", f)
	}
}
