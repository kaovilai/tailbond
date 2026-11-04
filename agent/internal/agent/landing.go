package agent

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/web"
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
	Version   string
	Repo      string // GitHub owner/name (update.repo)

	// the visitor: the detected platform, the section shown first
	Detected  string
	Section   string
	Windows   bool
	OnAndroid bool
	OnMac     bool

	// Android
	APK           bool
	APKSize       string
	APKSHA256     string
	AskLink       string
	Intent        template.URL // intent: isn't on html/template's safe-scheme list
	ObtainiumLink template.URL // obtainium:// isn't either
	QRAndroid     template.HTML

	// Mac and Linux
	MacBundle   string
	MacSize     string
	MacSHA256   string
	MacOneLiner string
	MacInstall  string
	BrewCmds    []string
	AskURL      string // the Mac's / laptop's own dashboard, the server filled in
	LinuxClient string
	LinuxServer string
}

// landingHandler serves the landing page on its own port.
func (a *Agent) landingHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.hLanding)
	mux.HandleFunc("GET /tailbond.apk", a.hAPK)
	mux.HandleFunc("GET /"+macBundleName, a.hMacBundle)
	mux.HandleFunc("GET /landing.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, web.FS, "landing.js")
	})
	// the brand: icon, favicon, wordmarks and the colour tokens (static)
	for _, f := range []string{"icon.svg", "favicon.svg", "wordmark-dark.svg", "wordmark-light.svg", "tokens.css"} {
		mux.HandleFunc("GET /"+f, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, web.FS, f)
		})
	}
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
	plat := landingPlatform(r.UserAgent(), r.Header.Get("Sec-CH-UA-Platform"))
	repo := a.updateRepo()
	v := landingView{
		Server:    short,
		URL:       u,
		TypeName:  typed,
		Version:   a.opts.Version,
		Repo:      repo,
		Detected:  plat,
		Section:   landingSection(plat),
		Windows:   plat == platWindows,
		OnAndroid: plat == platAndroid,
		OnMac:     plat == platMac,
		MacBundle: macBundleName,
	}
	_, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		port = strings.TrimPrefix(cfg.Listen, ":")
	}
	v.Dashboard = "http://" + net.JoinHostPort(short, port)
	if st, err := os.Stat(cfg.APKPath()); err == nil && !st.IsDir() {
		v.APK = true
		v.APKSize = sizeLabel(st.Size())
		v.APKSHA256 = a.sumOf(cfg.APKPath(), st)
	}
	// The app asks the server by its full name (MagicDNS), like typing it.
	v.AskLink = "tailbond://ask?server=" + url.QueryEscape(full)
	back := u + "#get-app"
	v.Intent = template.URL("intent://ask?server=" + url.QueryEscape(full) + "#Intent;scheme=tailbond;package=" + androidPackage +
		";S.browser_fallback_url=" + url.QueryEscape(back) + ";end")
	v.ObtainiumLink = template.URL("obtainium://add/https://github.com/" + repo)
	if plat != platAndroid {
		if svg, err := pairing.QRSVG(u + "#android"); err == nil {
			v.QRAndroid = template.HTML(svg) // generated from our own URL
		}
	}

	// Mac: the bundle from this server (or GitHub's, fetched once), Homebrew
	if p, sum, size := a.macBundle(); p != "" {
		v.MacSHA256, v.MacSize = sum, sizeLabel(size)
	} else if plat == platMac {
		a.prefetchMacBundle() // ready, with its checksum, by the time they tap
	}
	base := strings.TrimSuffix(u, "get") // the page's own origin, /get or /
	v.MacOneLiner = "cd ~/Downloads && curl -fLO " + base + macBundleName + " && tar -xzf " + macBundleName + " && sudo ./tailbond-macos-arm64/install.sh"
	// Safari may unpack the archive by itself ("Open safe files")
	v.MacInstall = "cd ~/Downloads && { [ -d tailbond-macos-arm64 ] || tar -xzf " + macBundleName + "; } && sudo ./tailbond-macos-arm64/install.sh"
	rel := "https://github.com/" + repo + "/releases/latest/download/"
	if isRelease(a.opts.Version) {
		rel = "https://github.com/" + repo + "/releases/download/v" + a.opts.Version + "/"
	}
	v.BrewCmds = []string{
		"brew tap-new --no-git local/tailbond",
		`curl -fsSLo "$(brew --repository local/tailbond)/Formula/tailbond.rb" ` + rel + "tailbond.rb",
		"brew install local/tailbond/tailbond",
		"sudo brew services start tailbond",
	}
	v.AskURL = "http://localhost:8077/?ask=" + url.QueryEscape(full)
	v.LinuxClient = "curl -fsSL " + rel + "tailbond-linux-$(uname -m).tar.gz | tar -xz && sudo ./tailbond-linux-$(uname -m)/install.sh --client"
	v.LinuxServer = "curl -fsSL " + rel + "tailbond-linux-$(uname -m).tar.gz | tar -xz && sudo ./tailbond-linux-$(uname -m)/install.sh"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	// the platform the page picks depends on these
	w.Header().Set("Vary", "User-Agent, Sec-CH-UA-Platform")
	w.Header().Set("Accept-CH", "Sec-CH-UA-Platform")
	if err := landingTmpl.Execute(w, v); err != nil {
		a.Logf("landing page: %v", err)
	}
}

// sizeLabel: "27 MB", "under 1 MB".
func sizeLabel(n int64) string {
	if n < 1<<20 {
		return "under 1 MB"
	}
	return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
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
