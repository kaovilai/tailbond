package agent

import (
	"net/http"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

// Without a password, only the server owner's devices (alex) use the admin
// API from the tailnet; others (sam) can still ask to connect.
func TestAdminAPIOwnerOnlyFromTailnet(t *testing.T) {
	a := pairServer(t)
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox", Owner: "alex@example.com"})
	if w := req(t, a, http.MethodGet, "/api/status", "100.101.8.8:4000", "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("sam's device: %d, want 403", w.Code)
	}
	var e struct{ Owner string }
	decode(t, req(t, a, http.MethodGet, "/api/clients", "100.101.8.8:4000", "", nil), &e)
	if e.Owner != "alex@example.com" {
		t.Errorf("403 body owner = %q", e.Owner)
	}
	for _, remote := range []string{"100.101.9.9:4000", "127.0.0.1:4000"} {
		if w := req(t, a, http.MethodGet, "/api/status", remote, "", nil); w.Code != http.StatusOK {
			t.Errorf("%s: %d, want 200", remote, w.Code)
		}
	}
	// sam may still ask to connect
	if w := req(t, a, http.MethodPost, "/api/pair-requests", "100.101.8.8:4000", "", map[string]string{"name": "pixel-8", "kind": "phone"}); w.Code != http.StatusAccepted {
		t.Errorf("sam's request: %d", w.Code)
	}
	// a tagged server (no owner): anyone who reaches it, as before
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox"})
	if w := req(t, a, http.MethodGet, "/api/status", "100.101.8.8:4000", "", nil); w.Code != http.StatusOK {
		t.Errorf("tagged server: %d", w.Code)
	}
	// with a password, the password decides (no session: 401)
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox", Owner: "alex@example.com"})
	if _, err := a.store.Update(func(c *config.Config) error { c.UIPassword = "pw"; return nil }); err != nil {
		t.Fatal(err)
	}
	if w := req(t, a, http.MethodGet, "/api/status", "100.101.9.9:4000", "", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("password set, no session: %d", w.Code)
	}
}

func TestStatusOwner(t *testing.T) {
	var info tailscale.Info
	js := `{"BackendState":"Running","Self":{"HostName":"homebox","DNSName":"homebox.tail1234.ts.net.","TailscaleIPs":["100.101.1.1"],"UserID":7},
	        "User":{"7":{"LoginName":"alex@example.com"}}}`
	if err := tailscale.ParseStatusForTest([]byte(js), "", &info); err != nil || info.Owner != "alex@example.com" {
		t.Fatalf("owner %q, %v", info.Owner, err)
	}
	info = tailscale.Info{}
	js = `{"BackendState":"Running","Self":{"HostName":"homebox","UserID":9,"Tags":["tag:server"]},"User":{"9":{"LoginName":"tagged-devices"}}}`
	if err := tailscale.ParseStatusForTest([]byte(js), "", &info); err != nil || info.Owner != "" {
		t.Fatalf("tagged: owner %q, %v", info.Owner, err)
	}
}

func TestApproversWithoutPassword(t *testing.T) {
	a := pairServer(t)
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox", Owner: "alex@example.com"})
	if w := req(t, a, http.MethodPost, "/api/settings", "127.0.0.1:4000", "", map[string]any{"approvers": []string{"Sam@example.com", " "}}); w.Code != http.StatusOK {
		t.Fatalf("set approvers: %d %s", w.Code, w.Body)
	}
	if got := a.store.Get().Approvers; len(got) != 1 || got[0] != "Sam@example.com" {
		t.Fatalf("approvers = %v", got)
	}
	if w := req(t, a, http.MethodGet, "/api/status", "100.101.8.8:4000", "", nil); w.Code != http.StatusOK {
		t.Errorf("sam as approver: %d", w.Code)
	}
	// a tagged server with approvers: only they
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox"})
	if w := req(t, a, http.MethodGet, "/api/status", "100.101.9.9:4000", "", nil); w.Code != http.StatusForbidden {
		t.Errorf("tagged server, alex not an approver: %d", w.Code)
	}
	if w := req(t, a, http.MethodPost, "/api/settings", "127.0.0.1:4000", "", map[string]any{"approvers": []string{"bad login"}}); w.Code != http.StatusBadRequest {
		t.Errorf("bad login accepted: %d", w.Code)
	}
}
