package agent

import (
	"context"
	"fmt"
	"net"
	goruntime "runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/netif"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/share"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// a computer's interfaces as the sharer sees them: en5 the hotel Ethernet,
// en0 the travel router's Wi-Fi, en9 an adapter with no cable, lo0
func testShareIfs() map[string]shareIfFacts {
	return map[string]shareIfFacts{
		"en5": {Up: true, Addr: true},
		"en0": {Up: true, Addr: true},
		"en9": {Up: true},
		"lo0": {Loopback: true, Up: true, Addr: true},
	}
}

func lookIn(m map[string]shareIfFacts) func(string) (shareIfFacts, bool) {
	return func(name string) (shareIfFacts, bool) {
		f, ok := m[name]
		return f, ok
	}
}

func TestShareIfaceProblem(t *testing.T) {
	for _, c := range []struct {
		iface, server string
		want          string // substring; "" = shareable
		fixed         bool
	}{
		{"en5", "198.51.100.1:443", "", false},
		{"tb0", "198.51.100.1:443", "is a VPN tunnel", true},
		{"utun4", "198.51.100.1:443", "is a VPN tunnel", true},
		{"wg0", "198.51.100.1:443", "is a VPN tunnel", true},
		{"tap0", "198.51.100.1:443", "is a VPN tunnel", true},
		{"tb1", "198.51.100.1:443", "is a VPN tunnel", true},
		{"en7", "198.51.100.1:443", "no connection called en7", false},
		{"en9", "198.51.100.1:443", "isn't connected right now", false},
		{"lo0", "198.51.100.1:443", "loopback", true},
		{"lo0", "home.example.net:443", "loopback", true},
		// a home server on this machine's loopback (a VM's forwarded port)
		{"lo0", "127.0.0.1:4443", "", false},
		{"lo0", "[::1]:4443", "", false},
	} {
		got, fixed := shareIfaceProblem(c.iface, c.server, lookIn(testShareIfs()))
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) || fixed != c.fixed {
			t.Errorf("%s via %s: %q fixed=%v, want %q fixed=%v", c.iface, c.server, got, fixed, c.want, c.fixed)
		}
	}
}

func TestDefaultShareLabel(t *testing.T) {
	for _, c := range []struct{ kind, iface, name, want string }{
		{netif.KindEthernet, "en5", "tiger-mbp", "Ethernet from tiger-mbp"},
		{netif.KindWifi, "wlan1", "thinkpad", "Wi-Fi from thinkpad"},
		{netif.KindCellular, "wwan0", "thinkpad", "Mobile data from thinkpad"},
		{netif.KindUSB, "en7", "tiger-mbp", "USB tethering from tiger-mbp"},
		{netif.KindUnknown, "lo0", "macbook", "lo0 from macbook"},
		{"", "eth9", "box", "eth9 from box"},
	} {
		if got := defaultShareLabel(c.kind, c.iface, c.name); got != c.want {
			t.Errorf("%s %s: %q, want %q", c.kind, c.iface, got, c.want)
		}
	}
	long := defaultShareLabel(netif.KindEthernet, "en5", strings.Repeat("é", 40))
	if len(long) > share.MaxIDLen || !utf8.ValidString(long) || !strings.HasPrefix(long, "Ethernet from é") {
		t.Errorf("long name: %q (%d bytes)", long, len(long))
	}
	if l := defaultShareLabel(netif.KindEthernet, "en5", ""); !strings.HasPrefix(l, "Ethernet from ") || len(l) <= len("Ethernet from ") {
		t.Errorf("no name, the hostname: %q", l)
	}
}

// The hub keeps a label that says what it is ("Ethernet from tiger-mbp",
// "Mobile data from sam-pixel") and marks the rest as shared.
func TestHubShareLabel(t *testing.T) {
	for in, want := range map[string]string{
		"Ethernet from tiger-mbp":    "Ethernet from tiger-mbp",
		"Mobile data from sam-pixel": "Mobile data from sam-pixel",
		"ThinkPad LTE":               "ThinkPad LTE (shared)",
		"":                           "Nearby device (shared)",
	} {
		hl := &hubLink{found: share.Found{Label: in}}
		if got := shareLabel(hl, share.LinkStatus{}); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func sharerAgent(t *testing.T, edit func(*config.Config)) (*Agent, map[string]shareIfFacts) {
	t.Helper()
	a := clientAgent(t, edit)
	ifm := testShareIfs()
	a.shareIf = lookIn(ifm)
	ifs := []netif.Iface{
		{Name: "en0", Kind: netif.KindWifi, Label: "Wi-Fi", Up: true, Carrier: true, Addrs: []string{"192.168.8.42/24"}},
		{Name: "en5", Kind: netif.KindEthernet, Label: "USB 10/100/1000 LAN", Up: true, Carrier: true, Addrs: []string{"10.20.0.7/16"}},
		{Name: "sh0a1b2c", Kind: netif.KindShared, Up: true, Carrier: true, Addrs: []string{"127.0.0.1/8"}},
		{Name: "lo0", Kind: netif.KindUnknown, Up: true, Carrier: true, Addrs: []string{"127.0.0.1/8"}},
	}
	a.mu.Lock()
	a.client.payload = &pairing.Payload{V: pairing.Version, Server: "198.51.100.1:443", User: "tiger-mbp", Key: "k", Subnet: "10.77.0.0/24"}
	a.client.remember(ifs)
	a.mu.Unlock()
	return a, ifm
}

func sharingOf(a *Agent) *status.Sharing {
	st := &status.Status{}
	a.shareStatus(st)
	return st.Sharing
}

func TestShareSettingsAPI(t *testing.T) {
	a, _ := sharerAgent(t, nil)
	sh := sharingOf(a)
	if sh == nil || sh.State != "off" || sh.Enabled {
		t.Fatalf("before: %+v", sh)
	}
	// the dashboard's choices: Wi-Fi and Ethernet, never a share or the loopback
	var names []string
	for _, i := range sh.Ifaces {
		names = append(names, i.Name+"="+i.Kind+"="+i.ShareLabel)
	}
	if got := strings.Join(names, " "); got != "en0=wifi=Wi-Fi from tiger-mbp en5=ethernet=Ethernet from tiger-mbp" {
		t.Fatalf("ifaces: %s", got)
	}
	if w := req(t, a, "POST", "/api/share", lan, "", map[string]any{"share_enabled": true}); w.Code != 400 || !strings.Contains(w.Body.String(), "pick the connection") {
		t.Fatalf("on with nothing set up: %d %s", w.Code, w.Body)
	}
	for _, bad := range []string{"utun4", "tb0", "en7", "lo0"} {
		if w := req(t, a, "POST", "/api/share", lan, "", map[string]any{"share": map[string]any{"enabled": true, "iface": bad}}); w.Code != 400 {
			t.Errorf("share %s: %d %s", bad, w.Code, w.Body)
		}
	}
	// not connected yet: fine to set up ahead
	if w := req(t, a, "POST", "/api/share", lan, "", map[string]any{"share": map[string]any{"enabled": true, "iface": "en9"}}); w.Code != 200 {
		t.Fatalf("en9: %d %s", w.Code, w.Body)
	}
	w := req(t, a, "POST", "/api/share", lan, "", map[string]any{"share": map[string]any{
		"enabled": true, "iface": " en5 ", "label": "", "cap_mb": 5000, "allowed_hubs": []string{" slate7 ", "", "slate7", "pixel-9"},
	}})
	if w.Code != 200 {
		t.Fatalf("en5: %d %s", w.Code, w.Body)
	}
	sc := a.store.Get().Client.Share
	if sc == nil || !sc.Enabled || sc.Iface != "en5" || sc.CapMB != 5000 || strings.Join(sc.AllowedHubs, ",") != "slate7,pixel-9" {
		t.Fatalf("stored: %+v", sc)
	}
	sh = sharingOf(a)
	if sh.Label != "Ethernet from tiger-mbp" || sh.CustomLabel != "" || sh.CapMB != 5000 || len(sh.AllowedHubs) != 2 {
		t.Fatalf("status: %+v", sh)
	}
	// off and on again keep the rest
	for _, on := range []bool{false, true} {
		if w := req(t, a, "POST", "/api/share", lan, "", map[string]any{"share_enabled": on}); w.Code != 200 {
			t.Fatalf("share_enabled %v: %d %s", on, w.Code, w.Body)
		}
		if sc := a.store.Get().Client.Share; sc.Enabled != on || sc.Iface != "en5" || sc.CapMB != 5000 {
			t.Fatalf("share_enabled %v: %+v", on, sc)
		}
	}
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := c.LocalAddr().(*net.UDPAddr).Port
	c.Close()
	return p
}

func TestSharerStates(t *testing.T) {
	port := freeUDPPort(t)
	a, ifm := sharerAgent(t, func(c *config.Config) {
		c.Client.Share = &config.ShareConfig{Enabled: true, Iface: "en5", Port: port}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.mu.Lock()
	p := a.client.payload
	a.mu.Unlock()
	round := func() *status.Sharing {
		a.runSharer(ctx, a.store.Get(), p, a.clientShareKey(p), false)
		return sharingOf(a)
	}
	// a pairing without the share key, the tunnel never up
	if sh := round(); sh.State != "no_key" || sh.Error != errNoShareKey {
		t.Fatalf("no key: %+v", sh)
	}
	p.ShareKey = "the-share-key"
	sh := round()
	if sh.State != "sharing" || sh.Label != "Ethernet from tiger-mbp" || sh.Port != port || len(sh.Hubs) != 0 {
		t.Fatalf("sharing: %+v", sh)
	}
	a.mu.Lock()
	id := a.client.share.sharerID
	a.mu.Unlock()
	if want := map[string]string{"darwin": "mac-"}[goruntime.GOOS]; want != "" && !strings.HasPrefix(id, want) || want == "" && !strings.HasPrefix(id, "lx-") {
		t.Fatalf("sharer id %q on %s", id, goruntime.GOOS)
	}
	if k := sharerKind(); k != map[string]string{"darwin": "macos"}[goruntime.GOOS] && k != "linux" {
		t.Fatalf("kind %q", k)
	}
	// the sharer listens: a second one can't have the port
	if c, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port)); err == nil {
		c.Close()
		t.Fatal("the port isn't held")
	}

	// the adapter is unplugged: sharing stops and says why; back: it starts again
	delete(ifm, "en5")
	sh = round()
	a.mu.Lock()
	stopped := a.client.share.sharer == nil
	a.mu.Unlock()
	if sh.State != "error" || !strings.Contains(sh.Error, "no connection called en5") || !stopped {
		t.Fatalf("unplugged: %+v (stopped %v)", sh, stopped)
	}
	ifm["en5"] = shareIfFacts{Up: true, Addr: true}
	if sh := round(); sh.State != "sharing" {
		t.Fatalf("plugged in again: %+v", sh)
	}

	// another program holds the port
	held, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	busy := held.LocalAddr().(*net.UDPAddr).Port
	if _, err := a.store.Update(func(c *config.Config) error {
		c.Client.Share = &config.ShareConfig{Enabled: true, Iface: "en5", Port: busy, Label: "Hotel Ethernet"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if sh := round(); sh.State != "error" || sh.Error != fmt.Sprintf("UDP port %d is in use by another program: pick another port for sharing.", busy) || sh.Label != "Hotel Ethernet" {
		t.Fatalf("port busy: %+v", sh)
	}

	// off
	if _, err := a.store.Update(func(c *config.Config) error { c.Client.Share.Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if sh := round(); sh.State != "off" || sh.Error != "" {
		t.Fatalf("off: %+v", sh)
	}
}
