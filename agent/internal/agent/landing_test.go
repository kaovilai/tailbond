package agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
)

func landingServer(t *testing.T, edit func(*config.Config)) *Agent {
	t.Helper()
	a := pairServer(t)
	apk := filepath.Join(t.TempDir(), "tailbond.apk")
	if err := os.WriteFile(apk, make([]byte, 3<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		c.AndroidAPK = apk
		if edit != nil {
			edit(c)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	a.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "homebox",
		DNSName: "homebox.tail1234.ts.net", IPs: []string{"100.101.1.1", "fd7a:115c:a1e0::1"}})
	return a
}

func landingGet(a *Agent, h http.Handler, path, ua string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.Host = "homebox"
	r.RemoteAddr = "100.101.8.8:40000"
	r.Header.Set("User-Agent", ua)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const pixelUA = "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Mobile Safari/537.36"

func TestLandingPageOnPhone(t *testing.T) {
	a := landingServer(t, nil)
	a.landingUp.Store(true)
	w := landingGet(a, a.landingHandler(), "/", pixelUA)
	if w.Code != http.StatusOK {
		t.Fatalf("GET / = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Get tailbond on this phone",
		`href="/tailbond.apk"`, "Download tailbond (3 MB)",
		"intent://ask?server=homebox.tail1234.ts.net#Intent;scheme=tailbond;package=dev.tailbond.app;S.browser_fallback_url=http%3A%2F%2Fhomebox.tail1234.ts.net%2F%23get-app;end",
		"Allow from this source",
		`href="http://homebox:8077"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("landing page lacks %q", want)
		}
	}
	if strings.Contains(body, "<svg") {
		t.Error("phone view shows the QR code")
	}
	if got := w.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("no CSP on the landing page")
	}
}

func TestLandingPageOnComputerShowsQR(t *testing.T) {
	a := landingServer(t, nil)
	a.landingUp.Store(true)
	w := landingGet(a, a.landingHandler(), "/", "Mozilla/5.0 (X11; Linux x86_64) Firefox/140.0")
	body := w.Body.String()
	if !strings.Contains(body, "<svg") || !strings.Contains(body, "type <b>homebox/</b> into the phone") {
		t.Errorf("computer view: no QR / typing hint:\n%s", body)
	}
}

func TestLandingServesNothingElse(t *testing.T) {
	a := landingServer(t, func(c *config.Config) { c.UIPassword = "secret" })
	h := a.landingHandler()
	if w := landingGet(a, h, "/tailbond.apk", pixelUA); w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/vnd.android.package-archive" {
		t.Errorf("APK: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	for _, p := range []string{"/api/status", "/api/clients", "/api/pair-requests", "/index.html", "/app.js", "/get"} {
		w := landingGet(a, h, p, pixelUA)
		if w.Code != http.StatusFound || w.Header().Get("Location") != "/" {
			t.Errorf("GET %s = %d %q, want a redirect to /", p, w.Code, w.Header().Get("Location"))
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Error("POST / accepted")
	}
}

func TestLandingOnDashboardPortNeedsNoLogin(t *testing.T) {
	a := landingServer(t, func(c *config.Config) { c.UIPassword = "secret" })
	w := landingGet(a, a.Handler(), "/get", pixelUA)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Get tailbond") {
		t.Fatalf("GET /get = %d", w.Code)
	}
	// port 80 isn't up: the page and the app's fallback point at /get
	if !strings.Contains(w.Body.String(), "S.browser_fallback_url=http%3A%2F%2Fhomebox.tail1234.ts.net%3A8077%2Fget%23get-app") {
		t.Error("fallback URL doesn't point at /get")
	}
}

func TestLandingURLAndAddrs(t *testing.T) {
	a := landingServer(t, nil)
	if u, typed := a.landingURL(t.Context(), "127.0.0.1:8077"); u != "http://homebox.tail1234.ts.net:8077/get" || typed != "homebox:8077/get" {
		t.Errorf("not up: %s %s", u, typed)
	}
	a.landingUp.Store(true)
	if u, typed := a.landingURL(t.Context(), "127.0.0.1:8077"); u != "http://homebox.tail1234.ts.net/" || typed != "homebox/" {
		t.Errorf("up: %s %s", u, typed)
	}
	if got := strings.Join(a.landingAddrs(t.Context()), " "); got != "100.101.1.1:80 [fd7a:115c:a1e0::1]:80" {
		t.Errorf("addrs: %s", got)
	}
	// OpenWrt: binds 8078, the firewall redirects tailnet tcp/80 to it
	a = landingServer(t, func(c *config.Config) { c.LandingPort, c.LandingPublicPort = 8078, 80 })
	a.landingUp.Store(true)
	if u, _ := a.landingURL(t.Context(), ""); u != "http://homebox.tail1234.ts.net/" {
		t.Errorf("8078→80: %s", u)
	}
	if got := strings.Join(a.landingAddrs(t.Context()), " "); got != "100.101.1.1:8078 [fd7a:115c:a1e0::1]:8078" {
		t.Errorf("addrs 8078: %s", got)
	}
	a = landingServer(t, func(c *config.Config) { c.LandingPort = -1 })
	if got := a.landingAddrs(t.Context()); len(got) != 0 {
		t.Errorf("off: %v", got)
	}
}

func TestSettingsReportLanding(t *testing.T) {
	a := landingServer(t, nil)
	a.landingUp.Store(true)
	w := req(t, a, http.MethodGet, "/api/settings", "127.0.0.1:5000", "", nil)
	var s settingsView
	decode(t, w, &s)
	if s.LandingURL != "http://homebox.tail1234.ts.net/" || s.LandingTyped != "homebox/" || !strings.HasPrefix(s.LandingQR, "<svg") {
		t.Errorf("settings: %+v", s)
	}
}
