package agent

import (
	"context"
	"encoding/json"
	"net/netip"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

// A Linux client next to a NetBird-like VPN (wt0 owns 100.92.0.0/16): the
// plan is applied with ip/resolvectl, taken away when the tunnel goes down,
// and the API switches Work VPN traffic.
func TestCoexistLinuxClient(t *testing.T) {
	if goruntime.GOOS != "linux" {
		t.Skip("the Linux client's commands (ip, resolvectl); darwin's are in internal/coexist and e2e-macos")
	}
	if !(func() bool { a := clientAgent(t, nil); return a.coexistOn() })() {
		t.Skip("coexistence runs on a Mac or a Linux client (this host looks like OpenWrt)")
	}
	a := clientAgent(t, func(c *config.Config) { c.Client.KillSwitch = true })
	a.cxIfaces = func() map[string][]netip.Prefix {
		return map[string][]netip.Prefix{"eth0": {netip.MustParsePrefix("10.0.0.5/24")}, "wt0": {netip.MustParsePrefix("100.92.1.2/16")}}
	}
	var mu sync.Mutex
	var calls []string
	a.cxRun = func(_ context.Context, name string, args ...string) ([]byte, error) {
		mu.Lock()
		calls = append(calls, name+" "+strings.Join(args, " "))
		mu.Unlock()
		if name == "ip" && strings.Join(args, " ") == "-4 route show table main" {
			return []byte("default via 10.0.0.1 dev eth0\n0.0.0.0/1 dev tb0\n100.92.0.0/16 dev wt0 scope link\n"), nil
		}
		return nil, nil
	}
	p := &pairing.Payload{V: pairing.Version, Server: "198.51.100.1:443", User: "laptop", Key: "k", Subnet: "10.77.0.0/24", DNS: []string{"10.77.0.1"}}
	peers, _ := json.Marshal(peersView{Server: "homebox", MagicDNSSuffix: "tail1234.ts.net",
		Peers: []tailscale.Node{{HostName: "nas", TailscaleIPs: []string{"100.101.1.1"}}, {HostName: "build", TailscaleIPs: []string{"100.92.5.5"}}}})
	a.mu.Lock()
	a.client.payload = p
	a.rt.ready, a.rt.tunName = true, "tb0"
	a.client.peers = peers
	a.client.peersAt = time.Now()
	a.mu.Unlock()
	ctx := context.Background()
	a.coexistRound(ctx)
	has := func(s string) bool {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range calls {
			if strings.Contains(c, s) {
				return true
			}
		}
		return false
	}
	for _, c := range []string{
		"ip route replace 100.101.1.1/32 dev tb0",
		"ip route replace 100.92.5.5/32 dev tb0",
		"ip route replace 10.77.0.0/24 dev tb0",
		"resolvectl dns tb0 10.77.0.1",
		"resolvectl domain tb0 ~ts.net",
		"resolvectl default-route tb0 false",
		"iptables -I OUTPUT -o wt0",
	} {
		if !has(c) {
			t.Errorf("missing %q in %v", c, calls)
		}
	}
	if has("100.92.0.0/16") || has("100.64.0.0/10") {
		t.Fatalf("a covering route: %v", calls)
	}
	var st status.Coexist
	w := req(t, a, "GET", "/api/coexist", lan, "", nil)
	decode(t, w, &st)
	if !st.SplitDNS || st.WorkVPN != "direct" || len(st.Foreign) != 1 || st.Foreign[0].Iface != "wt0" || st.TailnetRoutes != 3 {
		t.Fatalf("status: %s", w.Body)
	}
	if w := req(t, a, "POST", "/api/coexist", lan, "", map[string]any{"work_vpn": "sideways"}); w.Code != 400 {
		t.Fatalf("bad mode: %d", w.Code)
	}
	w = req(t, a, "POST", "/api/coexist", lan, "", map[string]any{"work_vpn": "bonded"})
	decode(t, w, &st)
	if st.WorkVPN != "bonded" {
		t.Fatalf("bonded: %s", w.Body)
	}
	// the tunnel goes down: everything of ours goes
	mu.Lock()
	calls = nil
	mu.Unlock()
	a.mu.Lock()
	a.rt.ready = false
	a.mu.Unlock()
	a.coexistRound(ctx)
	if !has("ip route del 100.92.5.5/32 dev tb0") || !has("resolvectl revert tb0") || !has("iptables -D OUTPUT -o wt0") {
		t.Fatalf("restore: %v", calls)
	}
	// pre-start: mqvpn gets no DNS (split instead)
	split, mtu := a.coexistPreStart(ctx, a.store.Get())
	if !split || mtu < config.DefaultClientMTU {
		t.Fatalf("pre-start: split %v mtu %d", split, mtu)
	}
}
