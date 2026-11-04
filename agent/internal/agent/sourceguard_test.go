package agent

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
)

func TestSourceGuard(t *testing.T) {
	cfg := config.Default()
	cfg.Role = config.RoleServer
	cfg.Server.Subnet = "10.78.0.0/24"
	cfg.Server.Subnet6 = "fd77:77::/112"
	a := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	get := func(listen, from string) int {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/hello", nil)
		r.RemoteAddr = from
		a.sourceGuard(listen, ok).ServeHTTP(rec, r)
		return rec.Code
	}
	for _, tc := range []struct {
		listen, from string
		want         int
	}{
		// the Tailscale address: tailnet, loopback and tunnel clients only
		{"100.101.0.1:8077", "100.101.9.9:5000", 200},
		{"100.101.0.1:8077", "127.0.0.1:5000", 200},
		{"100.101.0.1:8077", "10.78.0.2:5000", 200},
		{"100.101.0.1:8077", "[fd7a:115c:a1e0::5]:5000", 200},
		{"100.101.0.1:8077", "[fd77:77::2]:5000", 200},
		{"100.101.0.1:8077", "192.168.1.20:5000", 403}, // a LAN host via the weak host model
		{"100.101.0.1:80", "192.0.2.9:5000", 403},      // the phone page too
		{"[fd7a:115c:a1e0::1]:8077", "[2001:db8::9]:5000", 403},
		// the tunnel address
		{"10.78.0.1:8077", "10.78.0.2:5000", 200},
		{"10.78.0.1:8077", "192.168.1.20:5000", 403},
		// loopback and explicit/wildcard listeners are as configured
		{"127.0.0.1:8077", "127.0.0.1:5000", 200},
		{"0.0.0.0:8077", "192.168.1.20:5000", 200},
		{"192.168.1.5:8077", "192.168.1.20:5000", 200},
	} {
		if got := get(tc.listen, tc.from); got != tc.want {
			t.Errorf("listener %s, from %s: %d, want %d", tc.listen, tc.from, got, tc.want)
		}
	}
	// a client role is never guarded (it serves its LAN)
	cfg.Role = config.RoleClient
	c := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.20:5000"
	c.sourceGuard("100.101.0.1:8077", ok).ServeHTTP(rec, r)
	if rec.Code != 200 {
		t.Errorf("client role guarded: %d", rec.Code)
	}
}
