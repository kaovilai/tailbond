package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/captive"
	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/glrepeater"
	"github.com/tailbond/tailbond/agent/internal/netif"
	"github.com/tailbond/tailbond/agent/internal/status"
)

type portalFake struct {
	mu       sync.Mutex
	portal   map[string]bool // dev -> behind a portal
	requests map[string]int  // dev -> probe requests seen
}

func captiveAgent(t *testing.T, detect bool) (*Agent, *portalFake, string) {
	t.Helper()
	pf := &portalFake{portal: map[string]bool{}, requests: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dev := r.Header.Get("X-Dev")
		pf.mu.Lock()
		defer pf.mu.Unlock()
		pf.requests[dev]++
		if pf.portal[dev] {
			http.Redirect(w, r, "http://portal.hotel.example/login", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	hooksDir := t.TempDir()
	out := t.TempDir()
	for _, hk := range []string{"portal-open", "portal-close"} {
		script := "#!/bin/sh\nenv | grep '^TB_PORTAL' | sort > " + filepath.Join(out, hk) + "\n"
		if err := os.WriteFile(filepath.Join(hooksDir, hk), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.HooksDir = hooksDir
	cfg.Client.CaptivePortalDetect = &detect
	cfg.Client.CaptiveProbeURLs = []string{srv.URL + "/generate_204"}
	cfg.ApplyDefaults()
	a := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	// the probe "leaves through dev": tag the request so the fake knows
	a.captiveDialer = func(ctx context.Context, dev, addr string) (net.Conn, error) {
		var d net.Dialer
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		return &tagConn{Conn: c, dev: dev}, nil
	}
	a.glRead = func(context.Context) glrepeater.State { return glrepeater.State{} }
	up := func(name, network string, kind string) netif.Iface {
		return netif.Iface{Name: name, Network: network, WAN: true, Kind: kind, Up: true, Carrier: true, Addrs: []string{"192.0.2.2/24"}, DNS: []string{"192.0.2.53"}}
	}
	ifs := []netif.Iface{up("eth0", "wan", netif.KindEthernet), up("phy0-sta0", "wwan", netif.KindWifi), up("usb0", "tethering", netif.KindUSB)}
	a.listIfs = func() ([]netif.Iface, error) { return ifs, nil }
	a.mu.Lock()
	a.client.remember(ifs)
	a.mu.Unlock()
	return a, pf, out
}

// tagConn adds "X-Dev: <dev>" to the request line block (plain HTTP probe).
type tagConn struct {
	net.Conn
	dev  string
	done bool
}

func (c *tagConn) Write(b []byte) (int, error) {
	if !c.done {
		c.done = true
		s := strings.Replace(string(b), "\r\n", "\r\nX-Dev: "+c.dev+"\r\n", 1)
		if _, err := c.Conn.Write([]byte(s)); err != nil {
			return 0, err
		}
		return len(b), nil
	}
	return c.Conn.Write(b)
}

func (pf *portalFake) set(dev string, on bool) { pf.mu.Lock(); pf.portal[dev] = on; pf.mu.Unlock() }
func (pf *portalFake) count(dev string) int {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	return pf.requests[dev]
}

func readHook(t *testing.T, out, name string) string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(out, name))
	return string(b)
}

func captiveOf(a *Agent, key string) captive.Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client.cap.results[key]
}

func TestCaptiveOwnProbeSigninAndRejoin(t *testing.T) {
	a, pf, out := captiveAgent(t, true)
	pf.set("usb0", true) // a hotspot that wants a sign-in
	a.captiveRound(context.Background())
	if r := captiveOf(a, "tethering"); r.State != captive.Portal || r.PortalURL != "http://portal.hotel.example/login" {
		t.Fatalf("tethering = %+v", r)
	}
	if r := captiveOf(a, "wan"); r.State != captive.Open {
		t.Fatalf("wan = %+v", r)
	}
	// held out of bonding, not failed
	a.mu.Lock()
	ifs := []netif.Iface{a.client.ifs["eth0"], a.client.ifs["usb0"]}
	want := desiredPaths(a.store.Get(), ifs, a.client.capReached, a.client.captiveBlocked)
	a.mu.Unlock()
	if strings.Join(want, ",") != "eth0" {
		t.Fatalf("paths = %v", want)
	}
	paths := []status.Path{{Iface: "usb0", State: status.PathFailed}, {Iface: "eth0", State: status.PathActive}}
	a.annotateCaptive(paths)
	if paths[0].Captive != "portal" || paths[0].State != status.PathStandby || paths[0].CaptiveSource != "tailbond" || paths[1].Captive != "open" {
		t.Fatalf("status paths = %+v", paths)
	}
	// sign-in mode only for a portal WAN
	if _, err := a.startSignin("wan"); err == nil {
		t.Fatal("sign-in on an open WAN")
	}
	if _, err := a.startSignin("tethering"); err != nil {
		t.Fatal(err)
	}
	a.drainHooks()
	if h := readHook(t, out, "portal-open"); !strings.Contains(h, "TB_PORTAL_DEV=usb0") || !strings.Contains(h, "TB_PORTAL_DNS=192.0.2.53") || !strings.Contains(h, "TB_PORTAL_SOURCE=tailbond") {
		t.Fatalf("portal-open env:\n%s", h)
	}
	// signed in on the laptop: the next probe says open, sign-in ends, path back
	pf.set("usb0", false)
	a.pokeCaptive("tethering")
	a.captiveRound(context.Background())
	a.drainHooks()
	if r := captiveOf(a, "tethering"); r.State != captive.Open {
		t.Fatalf("after login = %+v", r)
	}
	if h := readHook(t, out, "portal-close"); !strings.Contains(h, "TB_PORTAL_NET=tethering") {
		t.Fatalf("portal-close not run: %q", h)
	}
	a.mu.Lock()
	si := a.client.cap.signin
	a.mu.Unlock()
	if si != nil {
		t.Fatal("sign-in still on")
	}
}

func TestCaptiveGLRepeaterOwnsWWAN(t *testing.T) {
	a, pf, out := captiveAgent(t, true)
	gl := glrepeater.State{Present: true, Enabled: true, Portal: true, PortalURL: "http://10.20.0.1/login"}
	var mu sync.Mutex
	a.glRead = func(context.Context) glrepeater.State { mu.Lock(); defer mu.Unlock(); return gl }
	setGL := func(s glrepeater.State) { mu.Lock(); gl = s; mu.Unlock() }

	a.captiveRound(context.Background())
	if r := captiveOf(a, "wwan"); r.State != captive.Portal || r.PortalURL != "http://10.20.0.1/login" {
		t.Fatalf("wwan = %+v", r)
	}
	if n := pf.count("phy0-sta0"); n != 0 {
		t.Fatalf("probed the repeater %d times; GL owns it", n)
	}
	if pf.count("eth0") == 0 {
		t.Fatal("other WANs not probed")
	}
	paths := []status.Path{{Iface: "phy0-sta0", State: status.PathStandby}}
	a.annotateCaptive(paths)
	if paths[0].CaptiveSource != "glinet" {
		t.Fatalf("source = %q", paths[0].CaptiveSource)
	}
	// GL's Login Mode on: our bypass follows it
	setGL(glrepeater.State{Present: true, Enabled: true, Portal: true, LoginMode: true, PortalURL: "http://10.20.0.1/login"})
	a.captiveRound(context.Background())
	a.drainHooks()
	if h := readHook(t, out, "portal-open"); !strings.Contains(h, "TB_PORTAL_DEV=phy0-sta0") || !strings.Contains(h, "TB_PORTAL_SOURCE=glinet") {
		t.Fatalf("portal-open:\n%s", h)
	}
	// still one sign-in, not reopened every round
	os.Remove(filepath.Join(out, "portal-open"))
	a.captiveRound(context.Background())
	a.drainHooks()
	if readHook(t, out, "portal-open") != "" {
		t.Fatal("portal-open ran again")
	}
	// GL: passed, Login Mode quit by itself -> revert, back in bonding
	setGL(glrepeater.State{Present: true, Enabled: true})
	a.captiveRound(context.Background())
	a.drainHooks()
	if r := captiveOf(a, "wwan"); r.State != captive.Open {
		t.Fatalf("wwan after login = %+v", r)
	}
	if !strings.Contains(readHook(t, out, "portal-close"), "TB_PORTAL_SOURCE=glinet") {
		t.Fatal("portal-close not run")
	}
	if pf.count("phy0-sta0") != 0 {
		t.Fatal("probed the repeater")
	}
}

func TestCaptiveOffStillHonoursGLLoginMode(t *testing.T) {
	a, pf, out := captiveAgent(t, false)
	a.glRead = func(context.Context) glrepeater.State {
		return glrepeater.State{Present: true, Enabled: true, Portal: true, LoginMode: true}
	}
	a.captiveRound(context.Background())
	a.drainHooks()
	if pf.count("eth0")+pf.count("usb0")+pf.count("phy0-sta0") != 0 {
		t.Fatal("probed with detection off")
	}
	if !strings.Contains(readHook(t, out, "portal-open"), "TB_PORTAL_SOURCE=glinet") {
		t.Fatal("GL's Login Mode blocked by tailbond's routing")
	}
}
