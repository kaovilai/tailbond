package coexist

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// A trimmed `netbird status --json` (NetBird's OutputOverview).
const nbJSON = `{
 "peers": {"total": 2, "connected": 1, "details": [
   {"fqdn": "laptop.netbird.cloud", "netbirdIp": "100.92.4.9", "status": "Connected",
    "iceCandidateEndpoint": {"local": "192.168.1.20:51820", "remote": "203.0.113.50:51820"}, "relayAddress": ""},
   {"fqdn": "build.netbird.cloud", "netbirdIp": "100.92.7.7", "status": "Connected",
    "iceCandidateEndpoint": {"local": "-", "remote": "-"}, "relayAddress": "rels://streamline-de-fra1-0.relay.netbird.io:443"}]},
 "cliVersion": "0.36.0", "daemonVersion": "0.36.0",
 "management": {"url": "https://api.netbird.io:443", "connected": true},
 "signal": {"url": "https://signal.netbird.io:443", "connected": true},
 "relays": {"total": 1, "available": 1, "details": [{"uri": "rels://streamline-de-fra1-0.relay.netbird.io:443", "available": true}]},
 "netbirdIp": "100.92.1.2/16", "publicKey": "x", "usesKernelInterface": false, "fqdn": "mac.netbird.cloud",
 "networks": ["10.40.0.0/16"],
 "dnsServers": [{"servers": ["100.92.255.254:53"], "domains": ["netbird.cloud", "corp.example"], "enabled": true}]
}`

func TestParseNetBird(t *testing.T) {
	nb, err := ParseNetBirdStatus([]byte(nbJSON))
	if err != nil {
		t.Fatal(err)
	}
	if !nb.LoggedIn || !nb.Connected || nb.Addr != "100.92.1.2/16" || !slices.Equal(nb.Networks, []string{"10.40.0.0/16"}) {
		t.Fatalf("%+v", nb)
	}
	if !slices.Equal(nb.DNSDomains, []string{"netbird.cloud", "corp.example"}) {
		t.Fatalf("dns %v", nb.DNSDomains)
	}
	want := []string{"api.netbird.io", "signal.netbird.io", "streamline-de-fra1-0.relay.netbird.io", "203.0.113.50"}
	if !slices.Equal(nb.Endpoints, want) {
		t.Fatalf("endpoints %v, want %v", nb.Endpoints, want)
	}
	// logged out: NetBird's daemon answers, no address
	lo, err := ParseNetBirdStatus([]byte(`{"management":{"url":"https://api.netbird.io:443","connected":false},"netbirdIp":"","peers":{"details":[]}}`))
	if err != nil || lo.LoggedIn || len(lo.Endpoints) != 1 {
		t.Fatalf("logged out: %+v %v", lo, err)
	}
	if e := NetBirdFromError([]byte("Daemon status: NeedsLogin\n\nRun UP command to log in with SSO"), nil); !e.Running || e.Error != "logged out" {
		t.Fatalf("needs login: %+v", e)
	}
	if e := NetBirdFromError([]byte("failed to connect to daemon error: context deadline exceeded"), nil); e.Running {
		t.Fatalf("no daemon: %+v", e)
	}
}

const netstat = `Routing tables

Internet:
Destination        Gateway            Flags               Netif Expire
default            192.168.1.1        UGScg                 en0
0/1                10.77.0.1          UGSc                utun5
100.64/10          link#25            UCS                 utun3
100.92/16          link#30            UCS                utun100
100.92.1.2         100.92.1.2         UH                utun100
100.101.9.9        link#25            UHW3I               utun3
127                127.0.0.1          UCS                   lo0
128.0/1            10.77.0.1          UGSc                utun5
169.254            link#11            UCS                   en0      !
192.168.1          link#11            UCS                   en0      !

Internet6:
Destination                             Gateway                                 Flags               Netif Expire
fd7a:115c:a1e0::/48                     fe80::%utun3                            UCS                 utun3
fe80::%utun0/64                         fe80::a1b2%utun0                        UcI                 utun0
`

func TestParseNetstat(t *testing.T) {
	rs := ParseNetstat(netstat)
	find := func(p string) Route {
		for _, r := range rs {
			if r.Prefix.String() == p {
				return r
			}
		}
		t.Fatalf("no %s in %v", p, rs)
		return Route{}
	}
	if r := find("100.92.0.0/16"); r.Iface != "utun100" {
		t.Fatal(r)
	}
	if r := find("0.0.0.0/1"); r.Iface != "utun5" {
		t.Fatal(r)
	}
	find("128.0.0.0/1")
	find("100.92.1.2/32")
	find("fd7a:115c:a1e0::/48")
	find("192.168.1.0/24")
}

func TestParseIPRoute(t *testing.T) {
	rs := ParseIPRoute(`default via 10.0.0.1 dev eth0 proto dhcp
0.0.0.0/1 dev tb0 scope link
100.92.0.0/16 dev wt0 proto kernel scope link src 100.92.1.2
10.0.0.0/24 dev eth0 proto kernel scope link src 10.0.0.5
203.0.113.50 via 10.0.0.1 dev eth0`)
	if len(rs) != 4 || rs[1].Iface != "wt0" || rs[3].Gateway != "10.0.0.1" || rs[3].Prefix.Bits() != 32 {
		t.Fatalf("%+v", rs)
	}
}

func pfx(s ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, x := range s {
		out = append(out, netip.MustParsePrefix(x))
	}
	return out
}

func addrs(s ...string) []netip.Addr {
	var out []netip.Addr
	for _, x := range s {
		out = append(out, netip.MustParseAddr(x))
	}
	return out
}

func base() Input {
	return Input{
		OurTun: "utun5", TunnelSubnet: netip.MustParsePrefix("10.77.0.4/24"),
		Peers:  addrs("100.101.1.1", "100.92.5.5", "100.92.1.2", "fd7a:115c:a1e0::5", "10.1.1.1"),
		Routes: ParseNetstat(netstat), // utun3: the Tailscale app
		IfaceAddrs: map[string][]netip.Prefix{
			"en0": pfx("192.168.1.20/24"), "utun5": pfx("10.77.0.4/32"), "utun100": pfx("100.92.1.2/16"), "utun0": pfx("fe80::1/64"),
		},
		NetBird:    NetBird{CLI: true, Running: true, LoggedIn: true, Addr: "100.92.1.2/16", MTU: 1280},
		NetBirdIPs: addrs("203.0.113.50", "198.51.100.7", "192.168.1.40"),
		Uplink:     Uplink{Iface: "en0", Gateway: "192.168.1.1", MTU: 1500},
		OurMTU:     1280,
	}
}

func TestPlanRouteOwnership(t *testing.T) {
	in := base()
	in.Routes = slices.DeleteFunc(in.Routes, func(r Route) bool { return r.Iface == "utun3" }) // no Tailscale
	pl := Make(in)
	got := []string{}
	for _, p := range pl.TunRoutes {
		got = append(got, p.String())
	}
	// tailnet machines by /32 and /128 (the one inside NetBird's /16 too: a
	// host route takes only that address), never NetBird's own address,
	// never a non-tailnet address; the tunnel subnet
	want := []string{"10.77.0.0/24", "100.92.5.5/32", "100.101.1.1/32", "fd7a:115c:a1e0::5/128"}
	if !slices.Equal(got, want) {
		t.Fatalf("tun routes %v, want %v; skipped %+v", got, want, pl.Skipped)
	}
	for _, p := range pl.TunRoutes {
		if covers(p, netip.MustParsePrefix("100.92.0.0/16")) {
			t.Fatalf("%s covers NetBird's /16", p)
		}
	}
	if len(pl.Skipped) == 0 || !strings.Contains(pl.Skipped[0].Reason, "utun100") {
		t.Fatalf("NetBird's own address not skipped: %+v", pl.Skipped)
	}
	if !pl.SplitDNS || pl.TailnetDirect {
		t.Fatalf("split %v, tailnet direct %v", pl.SplitDNS, pl.TailnetDirect)
	}
	var nb Foreign
	for _, f := range pl.Foreign {
		if f.Kind == "netbird" {
			nb = f
		}
	}
	if nb.Iface != "utun100" {
		t.Fatalf("NetBird not found: %+v", pl.Foreign)
	}
	// Direct (default): NetBird's underlay via the uplink, not the LAN one
	if !slices.Equal(pl.DirectRoutes, pfx("198.51.100.7/32", "203.0.113.50/32")) || pl.WorkEffective != WorkDirect {
		t.Fatalf("direct %v %s", pl.DirectRoutes, pl.WorkEffective)
	}
	if !slices.Contains(pl.PassIfaces, "utun100") || len(pl.PassHosts) != 3 {
		t.Fatalf("pass %v %v", pl.PassIfaces, pl.PassHosts)
	}
	if len(pl.Conflicts) != 0 {
		t.Fatalf("conflicts %v", pl.Conflicts)
	}
}

func TestPlanTailscaleApp(t *testing.T) {
	pl := Make(base()) // utun3 owns 100.64/10 and fd7a:…/48: the Tailscale app
	if !pl.TailnetDirect || len(pl.TunRoutes) != 1 || pl.TunRoutes[0].String() != "10.77.0.0/24" {
		t.Fatalf("with the Tailscale app: %v %v", pl.TailnetDirect, pl.TunRoutes)
	}
	in := base()
	in.Routes = slices.DeleteFunc(in.Routes, func(r Route) bool { return r.Iface == "utun3" })
	in.TailscaleRunning = true // its CLI says so
	if pl := Make(in); !pl.TailnetDirect {
		t.Fatal("tailscale CLI running")
	}
}

func TestPlanConflictAndCovering(t *testing.T) {
	in := base()
	in.Routes = slices.DeleteFunc(in.Routes, func(r Route) bool { return r.Iface == "utun3" })
	// our tunnel already has 100.64/10 (an older tailbond): covers NetBird's /16
	in.Routes = append(in.Routes, Route{Prefix: netip.MustParsePrefix("100.64.0.0/10"), Iface: "utun5"})
	in.TunnelSubnet = netip.MustParsePrefix("100.92.0.0/16") // a tunnel subnet equal to NetBird's
	pl := Make(in)
	if len(pl.Conflicts) == 0 || !strings.Contains(pl.Conflicts[0], "utun100") {
		t.Fatalf("conflict not seen: %v", pl.Conflicts)
	}
	for _, p := range pl.TunRoutes {
		if p.String() == "100.92.0.0/16" {
			t.Fatal("added a route equal to NetBird's")
		}
	}
	// a bypass rule covering NetBird's range isn't added either
	in = base()
	in.Bypass = pfx("100.64.0.0/10", "10.20.0.0/16")
	pl = Make(in)
	if slices.Contains(pl.DirectRoutes, netip.MustParsePrefix("100.64.0.0/10")) || !slices.Contains(pl.DirectRoutes, netip.MustParsePrefix("10.20.0.0/16")) {
		t.Fatalf("bypass %v", pl.DirectRoutes)
	}
}

func TestPlanBondedMTU(t *testing.T) {
	in := base()
	in.Routes = slices.DeleteFunc(in.Routes, func(r Route) bool { return r.Iface == "utun3" })
	in.WorkMode = WorkBonded
	pl := Make(in)
	if pl.WorkEffective != WorkBonded || pl.WantMTU != 1360 || len(pl.DirectRoutes) != 0 {
		t.Fatalf("bonded on a 1500 uplink: %+v", pl)
	}
	in.OurMTU = 1400
	if pl := Make(in); pl.WantMTU != 0 || pl.WorkEffective != WorkBonded {
		t.Fatalf("already big enough: %+v", pl)
	}
	in.OurMTU, in.Uplink.MTU = 1280, 1420 // 1320 max < 1360
	pl = Make(in)
	if pl.WorkEffective != WorkDirect || pl.BondedBlocked == "" || len(pl.DirectRoutes) == 0 {
		t.Fatalf("bonded can't fit: %+v", pl)
	}
}

func TestPlanNothingElse(t *testing.T) {
	in := Input{OurTun: "utun5", TunnelSubnet: netip.MustParsePrefix("10.77.0.4/24"), Peers: addrs("100.101.1.1"),
		IfaceAddrs: map[string][]netip.Prefix{"en0": pfx("192.168.1.20/24")}, Uplink: Uplink{Iface: "en0", Gateway: "192.168.1.1", MTU: 1500}}
	pl := Make(in)
	if pl.SplitDNS || len(pl.Foreign) != 0 || len(pl.DirectRoutes) != 0 || len(pl.TunRoutes) != 2 {
		t.Fatalf("no other VPN: %+v", pl)
	}
	if !slices.Equal(DNSDomains("tail1234.ts.net"), []string{"ts.net"}) || !slices.Equal(DNSDomains("corp.ts.example."), []string{"ts.net", "corp.ts.example"}) {
		t.Fatal("dns domains")
	}
}

type fakeRun struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeRun) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name+" "+strings.ReplaceAll(strings.Join(args, " "), "\x00", "<<"))
	return nil, nil
}

func (f *fakeRun) has(s string) bool {
	for _, c := range f.calls {
		if strings.Contains(c, s) {
			return true
		}
	}
	return false
}

func TestApplyDarwin(t *testing.T) {
	dir := t.TempDir()
	f := &fakeRun{}
	ap := &Applier{OS: "darwin", RecordPath: filepath.Join(dir, "coexist.json"), ResolverDir: filepath.Join(dir, "resolver"), Run: f.run}
	in := base()
	in.Routes = slices.DeleteFunc(in.Routes, func(r Route) bool { return r.Iface == "utun3" })
	pl := Make(in)
	ctx := context.Background()
	// a resolver file of the other VPN's is never touched
	os.MkdirAll(ap.ResolverDir, 0o755)
	os.WriteFile(filepath.Join(ap.ResolverDir, "corp.example"), []byte("nameserver 100.92.255.254\n"), 0o644)
	want := Want("darwin", "utun5", pl, in.Uplink, "10.77.0.1", DNSDomains("tail1234.ts.net"), true)
	if err := ap.Apply(ctx, want); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{
		"route -n add -inet -host 100.92.5.5 -interface utun5",
		"route -n add -inet6 -host fd7a:115c:a1e0::5 -interface utun5",
		"route -n add -inet -net 10.77.0.0/24 -interface utun5",
		"route -n add -inet -host 203.0.113.50 192.168.1.1",
		"pfctl -a com.apple/249.tailbond -f - <<pass out quick on utun100 all",
	} {
		if !f.has(c) {
			t.Errorf("missing %q in %v", c, f.calls)
		}
	}
	if f.has("100.92.0.0/16") || f.has("-host 100.92.1.2 ") {
		t.Fatalf("touched NetBird's routes: %v", f.calls)
	}
	b, _ := os.ReadFile(filepath.Join(ap.ResolverDir, "ts.net"))
	if !strings.Contains(string(b), "nameserver 10.77.0.1") || !strings.HasPrefix(string(b), ResolverMarker) {
		t.Fatalf("resolver: %q", b)
	}
	// a peer goes away: only its route goes
	f.calls = nil
	in.Peers = addrs("100.101.1.1")
	want2 := Want("darwin", "utun5", Make(in), in.Uplink, "10.77.0.1", []string{"ts.net"}, true)
	if err := ap.Apply(ctx, want2); err != nil {
		t.Fatal(err)
	}
	if !f.has("route -n delete -inet -host 100.92.5.5 -interface utun5") || f.has("add -inet -host 100.101.1.1") {
		t.Fatalf("diff: %v", f.calls)
	}
	// stop (or the next start after a crash): everything goes
	f.calls = nil
	if err := ap.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if !f.has("route -n delete -inet -host 100.101.1.1 -interface utun5") || !f.has("route -n delete -inet -host 203.0.113.50 192.168.1.1") ||
		!f.has("pfctl -a com.apple/249.tailbond -F all") {
		t.Fatalf("restore: %v", f.calls)
	}
	if _, err := os.Stat(filepath.Join(ap.ResolverDir, "ts.net")); err == nil {
		t.Fatal("resolver file left")
	}
	if b, _ := os.ReadFile(filepath.Join(ap.ResolverDir, "corp.example")); string(b) != "nameserver 100.92.255.254\n" {
		t.Fatal("the other VPN's resolver changed")
	}
	if _, err := os.Stat(ap.RecordPath); err == nil {
		t.Fatal("record left")
	}
}

func TestApplyLinux(t *testing.T) {
	f := &fakeRun{}
	ap := &Applier{OS: "linux", RecordPath: filepath.Join(t.TempDir(), "coexist.json"), Run: f.run}
	in := base()
	in.OurTun = "tb0"
	in.Routes = ParseIPRoute("100.92.0.0/16 dev wt0 scope link\n0.0.0.0/1 dev tb0\n")
	in.IfaceAddrs = map[string][]netip.Prefix{"eth0": pfx("10.0.0.5/24"), "wt0": pfx("100.92.1.2/16")}
	in.Uplink = Uplink{Iface: "eth0", Gateway: "10.0.0.1", MTU: 1500}
	pl := Make(in)
	ctx := context.Background()
	if err := ap.Apply(ctx, Want("linux", "tb0", pl, in.Uplink, "10.77.0.1", []string{"ts.net", "corp.ts.example"}, true)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{
		"ip route replace 100.92.5.5/32 dev tb0",
		"ip route replace 203.0.113.50/32 via 10.0.0.1 dev eth0",
		"resolvectl dns tb0 10.77.0.1",
		"resolvectl domain tb0 ~ts.net ~corp.ts.example",
		"resolvectl default-route tb0 false",
		"iptables -I OUTPUT -o wt0 -m comment --comment tailbond-coexist -j ACCEPT",
		"iptables -I OUTPUT -d 203.0.113.50 -m comment --comment tailbond-coexist -j ACCEPT",
	} {
		if !f.has(c) {
			t.Errorf("missing %q in %v", c, f.calls)
		}
	}
	f.calls = nil
	ap.Restore(ctx)
	if !f.has("ip route del 100.92.5.5/32 dev tb0") || !f.has("resolvectl revert tb0") || !f.has("iptables -D OUTPUT -o wt0") {
		t.Fatalf("restore: %v", f.calls)
	}
}
