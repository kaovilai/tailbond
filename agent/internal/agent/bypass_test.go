package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/netif"
)

func TestBypassAPIAndHook(t *testing.T) {
	out := t.TempDir()
	a := clientAgent(t, nil)
	// a fake bypass-apply: records the env and the rule file, answers
	// support and counters like the OpenWrt hook does
	hook := `#!/bin/sh
echo "$TB_BYPASS_ACTION" >> ` + filepath.Join(out, "actions") + `
case "$TB_BYPASS_ACTION" in
apply) cp "$TB_BYPASS_FILE" ` + filepath.Join(out, "rules.json") + `
       echo '{"domains":"nftset","firewall":"nft"}' > "$TB_BYPASS_STATUS" ;;
counters) echo '{"counters":{"r1":{"packets":3,"bytes":1200}}}' > "$TB_BYPASS_STATUS" ;;
esac
`
	if err := os.WriteFile(filepath.Join(a.store.Get().HooksDir, "bypass-apply"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	go a.hookWorker(t.Context())
	ifs := []netif.Iface{
		{Name: "eth0", Network: "wan", Label: "Ethernet", WAN: true, Up: true, Carrier: true, Addrs: []string{"192.0.2.2/24"}},
		{Name: "phy0-sta0", Network: "wwan", Label: "Repeater", WAN: true},
	}
	a.mu.Lock()
	a.client.remember(ifs)
	a.mu.Unlock()

	var r config.BypassRule
	w := req(t, a, "POST", "/api/bypass", lan, "", map[string]any{"id": "r1", "kind": "domain", "match": "https://www.Bank.example/login", "target": "direct", "note": "Bank"})
	if w.Code != 200 {
		t.Fatalf("add: %d %s", w.Code, w.Body)
	}
	decode(t, w, &r)
	if r.ID != "r1" || r.Match != "www.bank.example" {
		t.Fatalf("stored %+v", r)
	}
	w = req(t, a, "POST", "/api/bypass", lan, "", map[string]any{"kind": "device", "match": "AA:BB:CC:DD:EE:FF", "ip": "192.168.8.23", "target": "tunnel", "note": "Work laptop"})
	decode(t, w, &r)
	if w.Code != 200 || len(r.ID) != 8 || r.Match != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("device: %d %+v", w.Code, r)
	}
	laptop := r.ID
	if w := req(t, a, "POST", "/api/bypass", lan, "", map[string]any{"kind": "device", "match": "aa-bb-cc-dd-ee-ff", "target": "direct"}); w.Code != 400 {
		t.Fatalf("duplicate device rule: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/bypass", lan, "", map[string]any{"kind": "port", "match": "udp/4500", "target": "wan"}); w.Code != 400 {
		t.Fatalf("wan target without a wan: %d", w.Code)
	}
	off := false
	if w := req(t, a, "POST", "/api/bypass", lan, "", map[string]any{"kind": "port", "match": "udp/4500", "target": "wan", "wan": "wwan", "enabled": off}); w.Code != 200 {
		t.Fatalf("port: %d %s", w.Code, w.Body)
	}
	waitFor(t, func() bool {
		b, _ := os.ReadFile(filepath.Join(out, "rules.json"))
		return strings.Contains(string(b), "aa:bb:cc:dd:ee:ff")
	})
	time.Sleep(100 * time.Millisecond)
	b, _ := os.ReadFile(filepath.Join(out, "rules.json"))
	if strings.Contains(string(b), "udp/4500") {
		t.Fatalf("disabled rule handed to the hook: %s", b)
	}

	var got struct {
		Rules    []config.BypassRule
		Support  bypassSupport
		WANs     []wanView
		Counters map[string]bypassCounter
	}
	decode(t, req(t, a, "GET", "/api/bypass", lan, "", nil), &got)
	if len(got.Rules) != 3 || got.Support.Domains != "nftset" || got.Counters["r1"].Bytes != 1200 {
		t.Fatalf("GET %+v", got)
	}
	if len(got.WANs) != 2 || got.WANs[0].Name != "wan" || !got.WANs[0].Up || got.WANs[1].Up {
		t.Fatalf("wans %+v", got.WANs)
	}

	if w := req(t, a, "DELETE", "/api/bypass/"+laptop, lan, "", nil); w.Code != 200 {
		t.Fatalf("delete: %d", w.Code)
	}
	if w := req(t, a, "DELETE", "/api/bypass/"+laptop, lan, "", nil); w.Code != 404 {
		t.Fatalf("delete twice: %d", w.Code)
	}
	if n := len(a.store.Get().Client.Bypass); n != 2 {
		t.Fatalf("%d rules left", n)
	}
	a.clearBypass()
	acts, _ := os.ReadFile(filepath.Join(out, "actions"))
	if !strings.HasSuffix(strings.TrimSpace(string(acts)), "clear") || !strings.Contains(string(acts), "counters") {
		t.Fatalf("hook actions %q", acts)
	}
}

func TestLANClients(t *testing.T) {
	dir := t.TempDir()
	leases := filepath.Join(dir, "dhcp.leases")
	arp := filepath.Join(dir, "arp")
	os.WriteFile(leases, []byte(`1793000000 aa:bb:cc:dd:ee:01 192.168.8.23 work-laptop 01:aa:bb:cc:dd:ee:01
1793000000 aa:bb:cc:dd:ee:02 192.168.8.40 * *
1793000000 AA:BB:CC:DD:EE:03 192.168.8.50 Pixel-8 *
garbage
`), 0o644)
	os.WriteFile(arp, []byte(`IP address       HW type     Flags       HW address            Mask     Device
192.168.8.50     0x1         0x2         aa:bb:cc:dd:ee:03     *        br-lan
192.168.8.61     0x1         0x2         aa:bb:cc:dd:ee:04     *        br-lan
192.168.8.23     0x1         0x0         aa:bb:cc:dd:ee:01     *        br-lan
192.168.1.1      0x1         0x2         11:22:33:44:55:66     *        eth0
`), 0o644)
	got := lanClients(leases, arp, map[string]bool{"eth0": true})
	want := []lanClient{
		{MAC: "aa:bb:cc:dd:ee:03", IP: "192.168.8.50", Name: "Pixel-8", Online: true},
		{MAC: "aa:bb:cc:dd:ee:04", IP: "192.168.8.61", Online: true},
		{MAC: "aa:bb:cc:dd:ee:01", IP: "192.168.8.23", Name: "work-laptop"},
		{MAC: "aa:bb:cc:dd:ee:02", IP: "192.168.8.40"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] got %+v want %+v", i, got[i], want[i])
		}
	}
	if out := lanClients(filepath.Join(dir, "none"), filepath.Join(dir, "none"), nil); out == nil || len(out) != 0 {
		t.Fatalf("missing files: %#v", out)
	}
}
