package config

import (
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/pairing"
)

func uri(t *testing.T, p pairing.Payload) string {
	t.Helper()
	p.V, p.Key = pairing.Version, "k"
	u, err := pairing.Encode(&p)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestMigrateSingleServerConfig(t *testing.T) {
	home := uri(t, pairing.Payload{Server: "203.0.113.7:443", SNI: "homebox.tail1234.ts.net", DNS: []string{"10.77.0.1"}})
	dir := t.TempDir()
	path := filepath.Join(dir, "tailbond.json")
	old := `{"role":"client","client":{"pairing_uri":"` + home + `","paths":[]}}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c := st.Get().Client
	if len(c.Servers) != 1 || c.Servers[0].Name != "homebox" || c.Servers[0].PairingURI != home || c.PairingURI != home {
		t.Fatalf("migrated %+v", c)
	}
	// saved that way on the next change, pairing_uri still written
	if _, err := st.Update(func(c *Config) error { c.Mode = ModeStreaming; return nil }); err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	b, _ := os.ReadFile(path)
	_ = json.Unmarshal(b, &raw)
	cl := raw["client"].(map[string]any)
	if cl["pairing_uri"] != home || len(cl["servers"].([]any)) != 1 {
		t.Fatalf("saved %s", b)
	}
}

func TestMigrateHandSetURIReplacesPrimary(t *testing.T) {
	home := uri(t, pairing.Payload{Server: "203.0.113.7:443", ServerName: "homebox"})
	vps := uri(t, pairing.Payload{Server: "198.51.100.9:8443", ServerName: "vps", Subnet: "10.83.1.0/24"})
	home2 := uri(t, pairing.Payload{Server: "203.0.113.8:443", ServerName: "homebox"})
	c := Client{PairingURI: home2, Servers: []ServerEntry{{Name: "homebox", PairingURI: home}, {Name: "vps", PairingURI: vps}}}
	c.MigrateServers()
	if len(c.Servers) != 2 || c.Servers[0].PairingURI != home2 || c.Servers[1].Name != "vps" {
		t.Fatalf("%+v", c.Servers)
	}
	// reordering: pairing_uri follows the primary
	c.Servers[0], c.Servers[1] = c.Servers[1], c.Servers[0]
	c.MigrateServers()
	if c.PairingURI != vps || len(c.Servers) != 2 {
		t.Fatalf("%+v", c)
	}
	c.Servers = nil
	c.PairingURI = ""
	c.MigrateServers()
	if c.PairingURI != "" || len(c.Servers) != 0 {
		t.Fatal("unpair")
	}
}

func TestLabelAndSubnet(t *testing.T) {
	cases := []struct {
		p      pairing.Payload
		name   string
		subnet string
	}{
		{pairing.Payload{ServerName: "vps", Server: "1.2.3.4:443", Subnet: "10.83.142.0/24"}, "vps", "10.83.142.0/24"},
		{pairing.Payload{SNI: "homebox.tail1234.ts.net", Server: "1.2.3.4:443", DNS: []string{"10.77.0.1"}}, "homebox", "10.77.0.0/24"},
		{pairing.Payload{Server: "home.example.net:443"}, "home.example.net", "10.77.0.0/24"},
		{pairing.Payload{Server: "1.2.3.4:443", SNI: "1.2.3.4", DNS: []string{"10.80.3.1"}}, "1.2.3.4", "10.80.3.0/24"},
	}
	for _, c := range cases {
		if got := c.p.Label(); got != c.name {
			t.Errorf("label %q, want %q", got, c.name)
		}
		if got := c.p.TunnelSubnet().String(); got != c.subnet {
			t.Errorf("subnet %q, want %q", got, c.subnet)
		}
	}
}

func TestNextFreeSubnet(t *testing.T) {
	avoid := []netip.Prefix{netip.MustParsePrefix("10.77.0.0/24"), netip.MustParsePrefix("10.78.0.0/16"), netip.MustParsePrefix("10.79.0.0/24")}
	p, ok := NextFreeSubnet(avoid)
	if !ok || p.String() != "10.79.1.0/24" {
		t.Fatalf("got %v", p)
	}
	p, _ = NextFreeSubnet(nil)
	if p.String() != "10.78.0.0/24" {
		t.Fatalf("got %v", p)
	}
	if _, ok := NextFreeSubnet([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}); ok {
		t.Fatal("found a subnet inside 10/8")
	}
}

func TestUniqueServerName(t *testing.T) {
	list := []ServerEntry{{Name: "vps"}, {Name: "vps-2"}}
	if got := UniqueServerName(list, "vps"); got != "vps-3" {
		t.Fatal(got)
	}
}

func TestNormalizeBypass(t *testing.T) {
	ok := []struct{ in, kind, match string }{
		{"https://www.Netflix.com/browse", BypassDomain, "www.netflix.com"},
		{"*.bank.example", BypassDomain, "bank.example"},
		{"netflix.com.", BypassDomain, "netflix.com"},
		{"AA-BB-CC-DD-EE-FF", BypassDevice, "aa:bb:cc:dd:ee:ff"},
		{"203.0.113.9", BypassIP, "203.0.113.9/32"},
		{"203.0.113.77/24", BypassIP, "203.0.113.0/24"},
		{"2001:db8::1", BypassIP, "2001:db8::1/128"},
		{"UDP/4500", BypassPort, "udp/4500"},
		{"udp/5060-5061", BypassPort, "udp/5060-5061"},
	}
	for _, c := range ok {
		r, err := NormalizeBypass(BypassRule{Kind: c.kind, Match: c.in, Target: TargetDirect})
		if err != nil || r.Match != c.match {
			t.Errorf("%s %q → %q, %v; want %q", c.kind, c.in, r.Match, err, c.match)
		}
	}
	bad := []BypassRule{
		{Kind: BypassDomain, Match: "localhost", Target: TargetDirect},
		{Kind: BypassDomain, Match: "1.2.3.4", Target: TargetDirect},
		{Kind: BypassDomain, Match: "bad_name.com", Target: TargetDirect},
		{Kind: BypassDevice, Match: "", Target: TargetDirect},
		{Kind: BypassDevice, Match: "zz:bb", Target: TargetDirect},
		{Kind: BypassIP, Match: "300.1.1.1", Target: TargetDirect},
		{Kind: BypassPort, Match: "udp/0", Target: TargetDirect},
		{Kind: BypassPort, Match: "udp/6000-5000", Target: TargetDirect},
		{Kind: BypassPort, Match: "sctp/1", Target: TargetDirect},
		{Kind: BypassDomain, Match: "a.com", Target: TargetWAN},
		{Kind: BypassDomain, Match: "a.com", Target: "home"},
		{Kind: "app", Match: "x", Target: TargetDirect},
		{Kind: BypassDomain, Match: "a.com", Target: TargetDirect, Note: strings.Repeat("x", 65)},
		{Kind: BypassDomain, Match: "a.com", Target: TargetDirect, Note: "a\nb"},
	}
	for _, r := range bad {
		if _, err := NormalizeBypass(r); err == nil {
			t.Errorf("accepted %+v", r)
		}
	}
	r, err := NormalizeBypass(BypassRule{Kind: BypassDevice, IP: "192.168.8.23", Target: TargetWAN, WAN: "wwan"})
	if err != nil || r.Match != "" || r.IP != "192.168.8.23" || r.WAN != "wwan" {
		t.Fatalf("device by IP: %+v %v", r, err)
	}
	r, _ = NormalizeBypass(BypassRule{Kind: BypassDomain, Match: "a.com", Target: TargetDirect, WAN: "wwan"})
	if r.WAN != "" {
		t.Fatal("wan kept for direct")
	}
}

func TestValidateBypassList(t *testing.T) {
	r := BypassRule{ID: "a1", Kind: BypassDomain, Match: "a.com", Target: TargetDirect}
	if err := ValidateBypass([]BypassRule{r, r}); err == nil {
		t.Fatal("duplicate id")
	}
	r2 := r
	r2.ID = "Bad!"
	if err := ValidateBypass([]BypassRule{r2}); err == nil {
		t.Fatal("bad id")
	}
	many := make([]BypassRule, MaxBypassRules+1)
	if err := ValidateBypass(many); err == nil {
		t.Fatal("too many")
	}
	c := Default()
	c.Role = RoleClient
	c.Client.Bypass = []BypassRule{{ID: "x", Kind: BypassPort, Match: "tcp/99999", Target: TargetDirect}}
	if err := c.Validate(); err == nil {
		t.Fatal("config with a bad rule validated")
	}
}
