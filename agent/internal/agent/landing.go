package agent

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/web"
)

// The phone landing page (server role): http://<server>/ on the Tailscale
// IPs, so typing the server's MagicDNS name into a phone's browser shows
// "Get tailbond". It serves the page, the APK and the icon, nothing else:
// no API, no dashboard, whatever ui_password says. The same page is at
// /get on the dashboard port, for when port 80 isn't available.

//go:embed landing.html
var landingHTML string

var landingTmpl = template.Must(template.New("landing").Parse(landingHTML))

var errAddrNotAvail = syscall.EADDRNOTAVAIL

// androidPackage is the app the landing page's intent link opens.
const androidPackage = "dev.tailbond.app"

type landingView struct {
	Server    string // short name, e.g. homebox
	URL       string // the landing page by full MagicDNS name
	TypeName  string // what to type in a phone's browser
	Dashboard string
	APK       bool
	APKSize   string
	APKSHA256 string
	Version   string
	Android   bool
	IOS       bool
	Phone     bool
	AskLink   string
	Intent    template.URL // intent: isn't on html/template's safe-scheme list
	QR        template.HTML
}

// landingHandler serves the landing page on its own port.
func (a *Agent) landingHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.hLanding)
	mux.HandleFunc("GET /tailbond.apk", a.hAPK)
	mux.HandleFunc("GET /icon.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, web.FS, "icon.svg")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		mux.ServeHTTP(w, r)
	})
}

func setSecurityHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self'")
}

// landingNames: the server's short and full tailnet name, falling back to
// the Host the request came in on.
func (a *Agent) landingNames(ctx context.Context, host string) (short, full string) {
	ti := a.ts.Get(ctx)
	short, full = ti.Self, ti.DNSName
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if full == "" {
		full = host
	}
	if short == "" {
		short, _, _ = strings.Cut(full, ".")
		if net.ParseIP(full) != nil {
			short = full
		}
	}
	return short, full
}

// LandingURL is where phones get tailbond: port 80 on the full MagicDNS
// name when the landing page is up, else /get on the dashboard port.
func (a *Agent) landingURL(ctx context.Context, host string) (u, typed string) {
	short, full := a.landingNames(ctx, host)
	cfg := a.store.Get()
	if _, pub, ok := cfg.LandingPorts(); ok && a.landingUp.Load() {
		// the trailing slash makes a browser open it instead of searching
		if pub == 80 {
			return "http://" + full + "/", short + "/"
		}
		p := strconv.Itoa(pub)
		return "http://" + net.JoinHostPort(full, p) + "/", short + ":" + p + "/"
	}
	_, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		port = strings.TrimPrefix(cfg.Listen, ":")
	}
	return "http://" + net.JoinHostPort(full, port) + "/get", short + ":" + port + "/get"
}

func (a *Agent) hLanding(w http.ResponseWriter, r *http.Request) {
	cfg := a.store.Get()
	short, full := a.landingNames(r.Context(), r.Host)
	u, typed := a.landingURL(r.Context(), r.Host)
	ua := r.UserAgent()
	v := landingView{
		Server:   short,
		URL:      u,
		TypeName: typed,
		Version:  a.opts.Version,
		Android:  strings.Contains(ua, "Android"),
		IOS:      strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"),
	}
	v.Phone = v.Android || v.IOS || strings.Contains(ua, "Mobile")
	_, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		port = strings.TrimPrefix(cfg.Listen, ":")
	}
	v.Dashboard = "http://" + net.JoinHostPort(short, port)
	if st, err := os.Stat(cfg.APKPath()); err == nil && !st.IsDir() {
		v.APK = true
		v.APKSize = fmt.Sprintf("%.0f MB", float64(st.Size())/(1<<20))
		if st.Size() < 1<<20 {
			v.APKSize = "under 1 MB"
		}
		v.APKSHA256 = a.apkSum(cfg.APKPath(), st)
	}
	// The app asks the server by its full name (MagicDNS), like typing it.
	v.AskLink = "tailbond://ask?server=" + url.QueryEscape(full)
	back := u + "#get-app"
	v.Intent = template.URL("intent://ask?server=" + url.QueryEscape(full) + "#Intent;scheme=tailbond;package=" + androidPackage +
		";S.browser_fallback_url=" + url.QueryEscape(back) + ";end")
	if svg, err := pairing.QRSVG(u); err == nil {
		v.QR = template.HTML(svg) // generated from our own URL
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if err := landingTmpl.Execute(w, v); err != nil {
		a.Logf("landing page: %v", err)
	}
}

// apkSum is the APK's SHA-256 (hex), cached until the file changes.
func (a *Agent) apkSum(path string, st os.FileInfo) string {
	key := fmt.Sprintf("%s|%d|%d", path, st.Size(), st.ModTime().UnixNano())
	a.apkHash.Lock()
	defer a.apkHash.Unlock()
	if a.apkHash.key == key {
		return a.apkHash.sum
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	a.apkHash.key, a.apkHash.sum = key, hex.EncodeToString(h.Sum(nil))
	return a.apkHash.sum
}

// landingAddrs: the Tailscale IPs, on the landing port (server role only).
func (a *Agent) landingAddrs(ctx context.Context) []string {
	cfg := a.store.Get()
	bind, _, ok := cfg.LandingPorts()
	if !ok || cfg.Role != config.RoleServer {
		return nil
	}
	var out []string
	for _, ip := range a.ts.Get(ctx).IPs {
		out = append(out, net.JoinHostPort(ip, strconv.Itoa(bind)))
	}
	return out
}

func (a *Agent) serveLanding(ctx context.Context) {
	_ = a.serveOn(ctx, "phone page", a.landingHandler(), func() []string { return a.landingAddrs(ctx) }, false, &a.landingUp)
}
