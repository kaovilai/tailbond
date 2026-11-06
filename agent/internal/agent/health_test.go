package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

func check(t *testing.T, v healthView, id string) healthCheck {
	t.Helper()
	for _, c := range v.Checks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no check %q in %+v", id, v.Checks)
	return healthCheck{}
}

// homeStatus: canned `tailscale status --json` on the home server, built per
// test. The two relayed peers carry a handshake 30 s old — that freshness is
// what makes them relayed rather than merely unused (an idle peer has no
// CurAddr either, and used to be reported as slow). It is stamped when the
// test runs, not when the package loads: a package-level string ages past the
// two-minute window on a slow run and the relay test starts failing.
func homeStatus() string {
	fresh := hAgo(30 * time.Second)
	return `{"BackendState":"Running","Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","TailscaleIPs":["100.101.1.1"],
 "PrimaryRoutes":["10.77.0.0/24"],"AllowedIPs":["100.101.1.1/32","10.77.0.0/24"]},
 "Peer":{"a":{"HostName":"nas","Online":true,"CurAddr":"192.168.1.20:41641"},
         "b":{"HostName":"pixel-8","Online":true,"CurAddr":"198.51.100.20:41641"},
         "c":{"HostName":"slate7","Online":true,"Relay":"fra","LastHandshake":"` + fresh + `"},
         "d":{"HostName":"work-laptop","Online":true,"Relay":"nyc","LastHandshake":"` + fresh + `"},
         "e":{"HostName":"apple-tv","Online":true}}}`
}

// hAgo: a timestamp a little in the past, for LastHandshake.
func hAgo(d time.Duration) string { return time.Now().UTC().Add(-d).Format(time.RFC3339) }

func tsInfo(t *testing.T, js, route string) *tailscale.Info {
	t.Helper()
	info := &tailscale.Info{Installed: true}
	if err := tailscale.ParseStatusForTest([]byte(js), route, info); err != nil {
		t.Fatal(err)
	}
	return info
}

func TestHealthAllGood(t *testing.T) {
	js := strings.Replace(homeStatus(), `"Relay":"fra"`, `"CurAddr":"203.0.113.9:41641"`, 1)
	js = strings.Replace(js, `"Relay":"nyc"`, `"CurAddr":"203.0.113.10:41641"`, 1)
	v := buildHealth(healthInputs{Role: config.RoleServer, TS: tsInfo(t, js, "10.77.0.0/24"), Route: "10.77.0.0/24", Port: 443,
		Netcheck: &tailscale.Netcheck{UDP: true, UPnP: true, GlobalV4: "203.0.113.7:41641"}, WANIP: "203.0.113.7",
		Outside: &outsideSeen{Name: "slate7", IP: "198.51.100.9", At: time.Now().Add(-2 * time.Hour)}, Now: time.Now()})
	if !v.OK || v.Direct != 4 || v.Relayed != 0 {
		t.Fatalf("%+v", v)
	}
	if c := check(t, v, "tailscale_direct"); c.Title != "Tailscale: 4 machines direct" {
		t.Fatal(c.Title)
	}
	if c := check(t, v, "port"); c.Status != hOK || !strings.Contains(c.Detail, "slate7 connected from 198.51.100.9, 2 h ago") {
		t.Fatalf("%+v", c)
	}
}

func TestHealthRelayedHardNATRouteCGNAT(t *testing.T) {
	v := buildHealth(healthInputs{Role: config.RoleServer, TS: tsInfo(t, homeStatus(), "10.66.0.0/24"), Route: "10.66.0.0/24", Port: 443,
		Netcheck: &tailscale.Netcheck{UDP: true, MappingVariesByDestIP: true}, WANIP: "100.72.3.4", Now: time.Now()})
	if v.OK {
		t.Fatal("ok with problems")
	}
	c := check(t, v, "tailscale_direct")
	if c.Status != hWarn || c.Title != "Tailscale: 2 machines direct, 2 through a relay (slower)" || c.FixURL != urlConnTypes ||
		!strings.Contains(c.Detail, "slate7, work-laptop") || !strings.Contains(c.Fix, "41641") {
		t.Fatalf("%+v", c)
	}
	if c := check(t, v, "netcheck"); c.Status != hWarn || c.FixURL != urlFirewalls || !strings.Contains(c.Detail, "no port mapping on the router") {
		t.Fatalf("%+v", c)
	}
	if c := check(t, v, "route"); c.Status != hWarn || c.FixURL != urlSubnetRtrs {
		t.Fatalf("%+v", c)
	}
	if c := check(t, v, "cgnat"); c.Status != hBad || !strings.Contains(c.Title, "carrier-grade NAT") {
		t.Fatalf("%+v", c)
	}
	if c := check(t, v, "port"); c.Status != hUnknown {
		t.Fatalf("%+v", c)
	}
}

func TestHealthUDPBlockedBehindRouter(t *testing.T) {
	v := buildHealth(healthInputs{Role: config.RoleServer, TS: tsInfo(t, homeStatus(), "10.77.0.0/24"), Route: "10.77.0.0/24", Port: 443,
		Netcheck: &tailscale.Netcheck{UDP: false}, WANIP: "192.168.1.10", Now: time.Now()})
	if c := check(t, v, "netcheck"); c.Status != hBad || c.FixURL != urlFwPorts {
		t.Fatalf("%+v", c)
	}
	if c := check(t, v, "cgnat"); c.Status != hUnknown || !strings.Contains(c.Detail, "Forward UDP 443") {
		t.Fatalf("%+v", c)
	}
}

func TestHealthTailscaleOffAndClientRole(t *testing.T) {
	v := buildHealth(healthInputs{Role: config.RoleServer, TS: &tailscale.Info{}, Port: 443, Now: time.Now()})
	if c := check(t, v, "tailscale"); c.Status != hBad {
		t.Fatalf("%+v", c)
	}
	// client: only the path to its home server counts
	v = buildHealth(healthInputs{Role: config.RoleClient, TS: tsInfo(t, strings.Replace(homeStatus(), `"HostName":"nas"`, `"HostName":"homebox2"`, 1), ""), ServerTS: "work-laptop", Now: time.Now()})
	if c := check(t, v, "tailscale_direct"); c.Title != "Tailscale to your home server: 0 machines direct, 1 through a relay (slower)" {
		t.Fatalf("%+v", c)
	}
	for _, c := range v.Checks {
		if c.ID == "route" || c.ID == "port" || c.ID == "cgnat" {
			t.Fatalf("server check on client: %s", c.ID)
		}
	}
}

// With a relay every client arrives from the relay's private address: reaching
// the server at all proves the forwarding, so the port check confirms.
func TestNoteOutsideThroughRelay(t *testing.T) {
	a := pairServer(t)
	if _, err := a.store.Update(func(c *config.Config) error { c.Server.Relayed = true; return nil }); err != nil {
		t.Fatal(err)
	}
	// a device at home also dials the relay's public address, so it counts too
	a.noteOutside([]status.Client{{Name: "slate7", Connected: true, Endpoint: "10.200.0.1:61000"}})
	if a.outside == nil || a.outside.Name != "slate7" {
		t.Fatalf("relay-fronted client not counted: %+v", a.outside)
	}
	in := healthInputs{Role: config.RoleServer, TS: tsInfo(t, homeStatus(), "10.66.0.0/24"), Route: "10.66.0.0/24", Port: 443,
		Netcheck: &tailscale.Netcheck{UDP: true}, WANIP: "100.72.3.4", Relayed: true, Outside: a.outside, Now: time.Now()}
	if c := check(t, buildHealth(in), "port"); c.Status != hOK || c.Title != "UDP 443 reached through your relay" {
		t.Fatalf("%+v", c)
	}
	// nothing connected yet: say so, in relay terms
	in.Outside = nil
	if c := check(t, buildHealth(in), "port"); c.Status != hUnknown || c.Title != "UDP 443: not confirmed through your relay yet" {
		t.Fatalf("%+v", c)
	}
}

func TestNoteOutside(t *testing.T) {
	a := pairServer(t)
	a.noteOutside([]status.Client{{Name: "lan", Connected: true, Endpoint: "192.168.1.5:5000"}, {Name: "cg", Connected: true, Endpoint: "100.80.0.1:5000"}})
	if a.outside != nil {
		t.Fatal("private/CGNAT endpoint counted as outside")
	}
	a.noteOutside([]status.Client{{Name: "slate7", Connected: true, Endpoint: "198.51.100.9:61000"}})
	if a.outside == nil || a.outside.Name != "slate7" {
		t.Fatalf("%+v", a.outside)
	}
}

// Behind carrier-grade NAT with a relay in front: the line's own address isn't checked.
func TestHealthRelayedServerSkipsAddressCheck(t *testing.T) {
	in := healthInputs{Role: config.RoleServer, TS: tsInfo(t, homeStatus(), "10.66.0.0/24"), Route: "10.66.0.0/24", Port: 443,
		Netcheck: &tailscale.Netcheck{UDP: true}, WANIP: "100.72.3.4", Now: time.Now()}
	if c := check(t, buildHealth(in), "cgnat"); c.Status != hBad || c.FixURL != urlBehindCGNAT {
		t.Fatalf("without relay: %+v", c)
	}
	in.Relayed = true
	if c := check(t, buildHealth(in), "cgnat"); c.Status != hOK || !strings.Contains(c.Title, "relay") {
		t.Fatalf("with relay: %+v", c)
	}
}
