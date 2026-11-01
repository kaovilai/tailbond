package captive

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	g := DefaultProbes[0] // generate_204
	f := DefaultProbes[1] // success.txt
	for _, c := range []struct {
		name   string
		p      Probe
		code   int
		loc    string
		body   string
		state  string
		portal string
	}{
		{"204 open", g, 204, "", "", Open, ""},
		{"success open", f, 200, "", "success\n", Open, ""},
		{"302 redirect", g, 302, "https://portal.hotel.example/login?x=1", "", Portal, "https://portal.hotel.example/login?x=1"},
		{"relative redirect", g, 307, "/splash", "", Portal, "http://connectivitycheck.gstatic.com/splash"},
		{"200 html meta refresh", g, 200, "", `<html><head><meta http-equiv="refresh" content="0; url=http://10.0.0.1:8002/index.php?zone=guest"></head></html>`, Portal, "http://10.0.0.1:8002/index.php?zone=guest"},
		{"200 html js", f, 200, "", `<script>window.location.href="https://wifi.airport.example/";</script>`, Portal, "https://wifi.airport.example/"},
		{"200 html plain", g, 200, "", "<html>Welcome! Accept the terms</html>", Portal, g.URL},
		{"511", g, 511, "", "", Portal, g.URL},
		{"502 not portal", g, 502, "", "bad gateway", Offline, ""},
	} {
		st, u := Classify(c.p, c.code, c.loc, []byte(c.body))
		if st != c.state || u != c.portal {
			t.Errorf("%s: got %s %q, want %s %q", c.name, st, u, c.state, c.portal)
		}
	}
}

func TestProbesFor(t *testing.T) {
	ps := ProbesFor([]string{"http://10.200.9.9/generate_204", "http://example.net/success.txt"})
	if ps[0].WantCode != 204 || ps[1].WantCode != 200 || ps[1].WantBody != "success" {
		t.Fatalf("%+v", ps)
	}
	if len(ProbesFor(nil)) != 2 {
		t.Fatal("defaults")
	}
}

// Check end to end against local servers; Dial ignores the device.
func TestCheck(t *testing.T) {
	loggedIn := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			loggedIn = true
		}
		if !loggedIn {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	dial := func(ctx context.Context, _ string, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	}
	p := &Prober{Probes: ProbesFor([]string{srv.URL + "/generate_204"}), Dial: dial}
	r := p.Check(context.Background(), "wlan0")
	if r.State != Portal || r.PortalURL != srv.URL+"/login" {
		t.Fatalf("before login: %+v", r)
	}
	http.Get(srv.URL + "/login")
	if r := p.Check(context.Background(), "wlan0"); r.State != Open {
		t.Fatalf("after login: %+v", r)
	}

	// timeout -> offline
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // accept, never answer
		}
	}()
	slow := &Prober{Probes: ProbesFor([]string{"http://" + ln.Addr().String() + "/generate_204"}), Dial: dial, Timeout: 300 * time.Millisecond}
	if r := slow.Check(context.Background(), "wlan0"); r.State != Offline {
		t.Fatalf("timeout: %+v", r)
	}
	// unresolvable name -> offline
	dns := &Prober{Probes: ProbesFor([]string{"http://does-not-exist.invalid/generate_204"}), Dial: dial, Timeout: time.Second}
	if r := dns.Check(context.Background(), ""); r.State != Offline {
		t.Fatalf("dns failure: %+v", r)
	}
	if !strings.HasPrefix(DefaultProbes[0].URL, "http://") {
		t.Fatal("probes must be plain http (portals intercept those)")
	}
}
