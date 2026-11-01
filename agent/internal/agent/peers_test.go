package agent

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
)

func TestAPIToken(t *testing.T) {
	// HMAC-SHA256(key "test-key-123", "tailbond-api"), the value the Android app derives too
	const want = "68c8131634495b2e11cfcb57f604aa091793b004cfac7f5ae50ae49d553c1747"
	if got := APIToken("test-key-123"); got != want {
		t.Fatalf("APIToken = %s, want %s", got, want)
	}
}

func peersAgent(t *testing.T, password string) *Agent {
	t.Helper()
	cfg := config.Default()
	cfg.Role = config.RoleServer
	cfg.DisableTailscale = true
	cfg.UIPassword = password
	cfg.StateDir = t.TempDir()
	cfg.Server.Users = []config.User{{Name: "pixel", Key: "pixel-key"}, {Name: "tab", Key: "tab-key"}}
	cfg.ApplyDefaults()
	return New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
}

// do serves one request as if it arrived on local address local.
func do(t *testing.T, a *Agent, method, path, local, token string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, nil)
	if local != "" {
		addr, err := net.ResolveTCPAddr("tcp", local)
		if err != nil {
			t.Fatal(err)
		}
		r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, addr))
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w
}

func TestPeersAuthOnTunnel(t *testing.T) {
	a := peersAgent(t, "")
	const tun = "10.77.0.1:8077"
	if w := do(t, a, "GET", "/api/peers", tun, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d, want 401", w.Code)
	}
	if w := do(t, a, "GET", "/api/peers", tun, APIToken("wrong-key")); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: %d, want 401", w.Code)
	}
	w := do(t, a, "GET", "/api/peers", tun, APIToken("tab-key"))
	if w.Code != http.StatusOK {
		t.Fatalf("good token: %d %s", w.Code, w.Body)
	}
	var v struct {
		Client string            `json:"client"`
		Peers  []json.RawMessage `json:"peers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Client != "tab" || v.Peers == nil {
		t.Fatalf("body = %s", w.Body)
	}
	// even with no UI password, the tunnel side serves nothing else
	for _, p := range []string{"/api/status", "/api/clients", "/", "/api/settings"} {
		if w := do(t, a, "GET", p, tun, APIToken("tab-key")); w.Code != http.StatusNotFound {
			t.Errorf("GET %s via tunnel: %d, want 404", p, w.Code)
		}
	}
	if w := do(t, a, "GET", "/api/peers", "[fd77:77::1]:8077", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("v6 tunnel, no token: %d, want 401", w.Code)
	}
}

func TestPeersAuthOnAdminListener(t *testing.T) {
	a := peersAgent(t, "secret")
	const lo = "127.0.0.1:8077"
	if w := do(t, a, "GET", "/api/peers", lo, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d, want 401", w.Code)
	}
	if w := do(t, a, "GET", "/api/peers", lo, APIToken("pixel-key")); w.Code != http.StatusOK {
		t.Fatalf("token on admin listener: %d, want 200", w.Code)
	}
	// a client token is not an admin session
	if w := do(t, a, "GET", "/api/status", lo, APIToken("pixel-key")); w.Code != http.StatusUnauthorized {
		t.Fatalf("token for /api/status: %d, want 401", w.Code)
	}
}

func TestListenAddrsIncludeTunnel(t *testing.T) {
	for listen, want := range map[string][]string{
		":8077":          {"127.0.0.1:8077", "10.77.0.1:8077"},
		"127.0.0.1:8077": {"127.0.0.1:8077", "10.77.0.1:8077"},
		"0.0.0.0:8077":   {"0.0.0.0:8077"},
	} {
		a := peersAgent(t, "")
		if _, err := a.store.Update(func(c *config.Config) error { c.Listen = listen; return nil }); err != nil {
			t.Fatal(err)
		}
		got := a.listenAddrs(context.Background())
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("listen %q: %v, want %v", listen, got, want)
		}
	}
}

func TestAPKDownload(t *testing.T) {
	a := peersAgent(t, "")
	if w := do(t, a, "GET", "/tailbond.apk", "127.0.0.1:8077", ""); w.Code != http.StatusNotFound {
		t.Fatalf("missing apk: %d, want 404", w.Code)
	}
	apk := filepath.Join(t.TempDir(), "app.apk")
	if err := os.WriteFile(apk, []byte("PK\x03\x04"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.Update(func(c *config.Config) error { c.AndroidAPK = apk; return nil }); err != nil {
		t.Fatal(err)
	}
	w := do(t, a, "GET", "/tailbond.apk", "127.0.0.1:8077", "")
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/vnd.android.package-archive" {
		t.Fatalf("apk: %d %s", w.Code, w.Header())
	}
	// not offered to bonded clients on the tunnel side
	if w := do(t, a, "GET", "/tailbond.apk", "10.77.0.1:8077", ""); w.Code != http.StatusNotFound {
		t.Fatalf("apk via tunnel: %d, want 404", w.Code)
	}
}
