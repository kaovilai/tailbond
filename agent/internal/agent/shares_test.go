package agent

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/share"
)

// The first pairings and /api/shares calls can ask for the share secret at
// once: every caller must get the one that ends up on disk.
func TestShareSecretIsCreatedOnce(t *testing.T) {
	for round := 0; round < 20; round++ {
		a := peersAgent(t, "")
		var wg sync.WaitGroup
		got := make([]string, 16)
		for i := range got {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				got[i] = string(a.shareSecret())
			}(i)
		}
		wg.Wait()
		disk, err := os.ReadFile(filepath.Join(a.store.Get().StateDir, "share.key"))
		if err != nil {
			t.Fatal(err)
		}
		for i, g := range got {
			if g == "" || g != strings.TrimSpace(string(disk)) {
				t.Fatalf("round %d caller %d got %q, disk has %q", round, i, g, disk)
			}
		}
		left, _ := filepath.Glob(filepath.Join(a.store.Get().StateDir, "*.tmp"))
		if len(left) > 0 {
			t.Fatalf("temp files left: %v", left)
		}
	}
}

type sharesResp struct {
	Key     string      `json:"key"`
	KeyID   string      `json:"key_id"`
	Passes  []sharePass `json:"passes"`
	Sharers []regSharer `json:"sharers"`
}

func getShares(t *testing.T, a *Agent, token string) sharesResp {
	t.Helper()
	w := do(t, a, "GET", "/api/shares", "10.77.0.1:8077", token)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/shares: %d %s", w.Code, w.Body)
	}
	var v sharesResp
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// Each device has its own share key; a hub gets passes bound to its own
// name, which the sharer derives from its key; removing a device ends both.
func TestSharesPerDeviceKeysAndPasses(t *testing.T) {
	a := peersAgent(t, "") // users pixel, tab
	pixel := getShares(t, a, APIToken("pixel-key"))
	tab := getShares(t, a, APIToken("tab-key"))
	if pixel.Key == "" || tab.Key == "" || pixel.Key == tab.Key {
		t.Fatalf("share keys pixel %q tab %q", pixel.Key, tab.Key)
	}
	if pixel.KeyID != share.KeyID([]byte(pixel.Key)) {
		t.Fatal("key id")
	}
	// the pairing carries the same key
	if k := a.shareKeyFor(config.User{Name: "pixel", Key: "pixel-key"}); k != pixel.Key {
		t.Fatalf("pairing key %q, api %q", k, pixel.Key)
	}
	// pixel's pass for tab's share: what tab derives for "pixel"
	if len(pixel.Passes) != 1 || pixel.Passes[0].KeyID != tab.KeyID {
		t.Fatalf("pixel's passes %+v", pixel.Passes)
	}
	ps := pixel.Passes[0]
	if ps.Exp < uint64(time.Now().Add(24*time.Hour).Unix()) || ps.Exp > uint64(time.Now().Add(49*time.Hour).Unix()) {
		t.Fatalf("pass expiry %d", ps.Exp)
	}
	if ps.Key != hex.EncodeToString(share.PassKey([]byte(tab.Key), "pixel", ps.Exp)) {
		t.Fatal("pass doesn't match what the sharer derives")
	}
	if ps.Key == hex.EncodeToString(share.PassKey([]byte(tab.Key), "tab", ps.Exp)) {
		t.Fatal("pass not bound to the hub's name")
	}
	// the admin (no client token): no key, no passes; nobody gets the secret
	w0 := do(t, a, "GET", "/api/shares", "127.0.0.1:8077", "")
	if w0.Code != http.StatusOK || strings.Contains(w0.Body.String(), `"key"`) || strings.Contains(w0.Body.String(), "passes") {
		t.Fatalf("admin got %d %s", w0.Code, w0.Body)
	}

	// tab registers its share, then is removed
	body := `{"id":"an-1","label":"Tab","key_id":"forged","lan":["192.168.8.23"],"port":47077}`
	r := httptestRequest(t, "POST", "/api/shares", body, APIToken("tab-key"))
	w := serve(a, r)
	if w.Code != http.StatusOK {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	if reg := getShares(t, a, APIToken("pixel-key")).Sharers; len(reg) != 1 || reg[0].KeyID != tab.KeyID {
		t.Fatalf("registry %+v (key id must be the server's)", reg)
	}
	if err := a.removeClient(context.Background(), "tab"); err != nil {
		t.Fatal(err)
	}
	after := getShares(t, a, APIToken("pixel-key"))
	if len(after.Passes) != 0 || len(after.Sharers) != 0 {
		t.Fatalf("removed device still shares: %+v", after)
	}
	if w := do(t, a, "GET", "/api/shares", "10.77.0.1:8077", APIToken("tab-key")); w.Code != http.StatusUnauthorized {
		t.Fatalf("removed device's token: %d", w.Code)
	}
	// paired again under the same name: a new key, the old one opens nothing
	if _, err := a.store.Update(func(c *config.Config) error {
		c.Server.Users = append(c.Server.Users, config.User{Name: "tab", Key: "tab-key-2"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if k := getShares(t, a, APIToken("tab-key-2")).Key; k == "" || k == tab.Key {
		t.Fatalf("re-paired key %q", k)
	}
}

func TestPassExpiry(t *testing.T) {
	now := time.Date(2026, 11, 3, 23, 59, 0, 0, time.UTC)
	if got := passExpiry(now); got != uint64(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC).Unix()) {
		t.Fatalf("expiry %d", got)
	}
	// the same all day
	if passExpiry(now) != passExpiry(now.Add(-23*time.Hour)) {
		t.Fatal("expiry changes within a day")
	}
}

// httptestRequest is a request with a JSON body and a client token, on the
// server's tunnel address.
func httptestRequest(t *testing.T, method, path, body, token string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	addr, err := net.ResolveTCPAddr("tcp", "10.77.0.1:8077")
	if err != nil {
		t.Fatal(err)
	}
	return r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, addr))
}

func serve(a *Agent, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w
}
