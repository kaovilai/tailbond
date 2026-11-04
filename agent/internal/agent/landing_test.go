package agent

import (
	"crypto/sha256"
	"encoding/hex"
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
		"On this phone", `class="plat first" id="android"`,
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

func TestLandingPlatform(t *testing.T) {
	for _, c := range []struct{ ua, ch, want string }{
		{pixelUA, "", platAndroid},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148", "", platIPhone},
		{"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148", "", platIPhone},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15", "", platMac},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/141.0", "", platWindows},
		{"Mozilla/5.0 (X11; Linux x86_64) Firefox/140.0", "", platLinux},
		{"Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) Chrome/141.0", "", platLinux},
		// client hints win over a frozen or desktop-mode User-Agent
		{"Mozilla/5.0 (X11; Linux x86_64) Chrome/141.0", `"Android"`, platAndroid},
		{"Mozilla/5.0 (Windows NT 10.0) Chrome/141.0", `"macOS"`, platMac},
		{"curl/8.0", "", platAndroid},
	} {
		if got := landingPlatform(c.ua, c.ch); got != c.want {
			t.Errorf("%q %q: %s, want %s", c.ua, c.ch, got, c.want)
		}
	}
	if landingSection(platWindows) != platAndroid || landingSection(platMac) != platMac {
		t.Error("landingSection")
	}
}

const macUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15"

// Every platform's section is on the page (the switcher and #mac etc.
// pick one); the detected one shows first.
func TestLandingSectionsPerPlatform(t *testing.T) {
	a := landingServer(t, nil)
	a.landingUp.Store(true)
	h := a.landingHandler()
	for _, c := range []struct{ ua, first string }{
		{pixelUA, "android"}, {macUA, "mac"}, {"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)", "iphone"},
		{"Mozilla/5.0 (X11; Linux x86_64) Firefox/140.0", "linux"}, {"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "android"},
	} {
		body := landingGet(a, h, "/", c.ua).Body.String()
		for _, id := range []string{"android", "mac", "iphone", "linux"} {
			if !strings.Contains(body, `id="`+id+`"`) || !strings.Contains(body, `href="#`+id+`"`) {
				t.Errorf("%s: no section/tab %s", c.first, id)
			}
		}
		if !strings.Contains(body, `class="plat first" id="`+c.first+`"`) || strings.Count(body, `plat first`) != 1 {
			t.Errorf("%q: first section isn't %s", c.ua, c.first)
		}
	}
	mac := landingGet(a, h, "/", macUA).Body.String()
	for _, want := range []string{
		"Open tailbond and ask homebox", `href="http://localhost:8077/?ask=homebox.tail1234.ts.net"`,
		"curl -fLO http://homebox.tail1234.ts.net/tailbond-macos-arm64.tar.gz",
		"sudo ./tailbond-macos-arm64/install.sh", "brew install local/tailbond/tailbond",
		"releases/latest/download/tailbond.rb", "xattr -dr com.apple.quarantine",
		"iPhone (USB)", "Shared by nearby phones",
		"install.sh --client", "dnf copr enable kaovilai/tailbond",
		"tailbond Share", "7 days", "SideStore", "docs/ios.md",
		`<script src="/landing.js"></script>`,
	} {
		if !strings.Contains(mac, want) {
			t.Errorf("Mac view lacks %q", want)
		}
	}
	// a computer gets the Android QR (this page, #android) to scan
	if !strings.Contains(mac, "<svg") || !strings.Contains(mac, "type <b>homebox/</b> into the phone") {
		t.Error("Mac view: no QR for the phone")
	}
	win := landingGet(a, h, "/", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)").Body.String()
	if !strings.Contains(win, "doesn't run on Windows") {
		t.Error("Windows: no note")
	}
	if w := landingGet(a, h, "/landing.js", macUA); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "execCommand") {
		t.Errorf("landing.js = %d", w.Code)
	}
}

// The macOS bundle: an admin's copy (with its SHA-256 on the page), else
// GitHub's release asset fetched once (checked against SHA256SUMS) and
// cached, else 404; never asked for with updates off.
func TestLandingMacDownload(t *testing.T) {
	a := landingServer(t, nil)
	a.landingUp.Store(true)
	h := a.landingHandler()
	off := config.UpdateSettings{Policy: config.UpdateOff}
	_, _ = a.store.Update(func(c *config.Config) error { c.Update = &off; return nil })
	if w := landingGet(a, h, "/"+macBundleName, macUA); w.Code != http.StatusNotFound {
		t.Fatalf("no bundle, updates off: %d", w.Code)
	}

	// an admin's copy
	local := filepath.Join(t.TempDir(), macBundleName)
	if err := os.WriteFile(local, []byte("mac bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = a.store.Update(func(c *config.Config) error { c.MacOSBundle = local; return nil })
	w := landingGet(a, h, "/"+macBundleName, macUA)
	if w.Code != http.StatusOK || w.Body.String() != "mac bundle" || !strings.Contains(w.Header().Get("Content-Disposition"), macBundleName) {
		t.Fatalf("local copy: %d %q", w.Code, w.Body.String())
	}
	if body := landingGet(a, h, "/", macUA).Body.String(); !strings.Contains(body, "SHA-256 <code>"+sha256hex([]byte("mac bundle"))) {
		t.Error("page doesn't show the bundle's SHA-256")
	}

	// GitHub: fetched once, checked, cached
	g := newFakeGitHub(t, "0.2.0")
	g.mu.Lock()
	g.files[macBundleName] = []byte("from github")
	g.mu.Unlock()
	_, _ = a.store.Update(func(c *config.Config) error {
		c.MacOSBundle = filepath.Join(t.TempDir(), "none.tar.gz")
		c.Update = &config.UpdateSettings{APIURL: g.srv.URL}
		return nil
	})
	if w := landingGet(a, a.Handler(), "/"+macBundleName, macUA); w.Code != http.StatusOK || w.Body.String() != "from github" {
		t.Fatalf("fetched on the dashboard port: %d %q", w.Code, w.Body.String())
	}
	calls := g.calls
	if w := landingGet(a, h, "/"+macBundleName, macUA); w.Code != http.StatusOK || g.calls != calls {
		t.Errorf("second request: %d, asked GitHub again (%d -> %d)", w.Code, calls, g.calls)
	}
	if _, err := os.Stat(a.macCachePath()); err != nil {
		t.Errorf("not cached: %v", err)
	}

	// a checksum that doesn't match: nothing cached, 404, no retry right away
	os.Remove(a.macCachePath())
	g.mu.Lock()
	g.sums[macBundleName] = strings.Repeat("0", 64)
	g.mu.Unlock()
	if w := landingGet(a, h, "/"+macBundleName, macUA); w.Code != http.StatusNotFound {
		t.Errorf("bad checksum: %d", w.Code)
	}
	if _, err := os.Stat(a.macCachePath()); err == nil {
		t.Error("a bad download was cached")
	}
	calls = g.calls
	landingGet(a, h, "/"+macBundleName, macUA)
	if g.calls != calls {
		t.Error("retried GitHub right after a failure")
	}
}

func sha256hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
