package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/web"
)

const sessionCookie = "tb_session"

func (a *Agent) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", a.hLogin)
	mux.HandleFunc("POST /api/logout", a.hLogout)
	mux.HandleFunc("GET /api/status", a.hStatus)
	mux.HandleFunc("GET /api/events", a.hEvents)
	mux.HandleFunc("GET /api/logs", a.hLogs)
	mux.HandleFunc("POST /api/mode", a.hMode)
	mux.HandleFunc("GET /api/paths", a.hPathsGet)
	mux.HandleFunc("POST /api/paths", a.hPathsPost)
	mux.HandleFunc("POST /api/connect", a.hConnect)
	mux.HandleFunc("POST /api/disconnect", a.hDisconnect)
	mux.HandleFunc("GET /api/clients", a.hClientsGet)
	mux.HandleFunc("POST /api/clients", a.hClientsPost)
	mux.HandleFunc("GET /api/clients/{name}", a.hClientGet)
	mux.HandleFunc("DELETE /api/clients/{name}", a.hClientDelete)
	mux.HandleFunc("POST /api/pair", a.hPair)
	mux.HandleFunc("GET /api/tailscale", a.hTailscale)
	mux.HandleFunc("GET /api/peers", a.hPeers)
	mux.HandleFunc("GET /api/settings", a.hSettingsGet)
	mux.HandleFunc("POST /api/settings", a.hSettingsPost)
	mux.HandleFunc("GET /tailbond.apk", a.hAPK)
	mux.HandleFunc("GET /tailbond-glinet.ipk", a.hIPK)
	mux.HandleFunc("DELETE /api/pair", a.hUnpair)
	// Pairing over Tailscale (docs/architecture.md).
	mux.HandleFunc("GET /api/hello", a.hHello)
	mux.HandleFunc("POST /api/pair-requests", a.hPairRequestCreate)
	mux.HandleFunc("GET /api/pair-requests", a.hPairRequestList)
	mux.HandleFunc("GET /api/pair-requests/{id}", a.hPairRequestPoll)
	mux.HandleFunc("POST /api/pair-requests/{id}/{action}", a.hPairRequestDecide)
	mux.HandleFunc("GET /api/discover", a.hDiscover)
	mux.HandleFunc("GET /api/join", a.hJoinGet)
	mux.HandleFunc("POST /api/join", a.hJoin)
	mux.HandleFunc("DELETE /api/join", a.hJoinCancel)
	mux.HandleFunc("GET /api/health", a.hHealth)
	// Captive portals (client role)
	mux.HandleFunc("POST /api/captive/signin", a.hCaptiveSignin)
	mux.HandleFunc("DELETE /api/captive/signin", a.hCaptiveSigninEnd)
	mux.HandleFunc("POST /api/captive/check", a.hCaptiveCheck)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { httpErr(w, http.StatusNotFound, errNotFound) })

	var assets fs.FS = web.FS
	if a.opts.WebDir != "" {
		assets = os.DirFS(a.opts.WebDir)
	}
	static := http.FileServer(http.FS(assets))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		static.ServeHTTP(w, r)
	}))
	return a.secure(mux)
}

func (a *Agent) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self'")
		// Bonded clients reach the server's tunnel address; there only the
		// token-authenticated peer list exists (no UI, no admin API).
		if a.viaTunnel(r) {
			if r.URL.Path != "/api/peers" || r.Method != http.MethodGet {
				httpErr(w, http.StatusNotFound, errNotFound)
				return
			}
			if a.clientForToken(r) == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="tailbond"`)
				httpErr(w, http.StatusUnauthorized, errors.New("client token required"))
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/login" {
			if !a.authorized(r) && !tailnetPublic(r) && !(r.URL.Path == "/api/peers" && a.clientForToken(r) != "") {
				httpErr(w, http.StatusUnauthorized, errors.New("login required"))
				return
			}
			// Mutations must come from our own page (no CSRF via simple forms).
			if r.Method != http.MethodGet && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.Method != http.MethodDelete {
				httpErr(w, http.StatusUnsupportedMediaType, errors.New("send application/json"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// tailnetPublic: the requests a device on the tailnet makes before it is
// paired. Their handlers check the source and identity themselves.
func tailnetPublic(r *http.Request) bool {
	p := r.URL.Path
	switch {
	case p == "/api/hello" && r.Method == http.MethodGet:
		return true
	case p == "/api/pair-requests" && r.Method == http.MethodPost:
		return true
	case r.Method == http.MethodGet && strings.HasPrefix(p, "/api/pair-requests/"):
		return !strings.Contains(strings.TrimPrefix(p, "/api/pair-requests/"), "/")
	}
	return false
}

func (a *Agent) authorized(r *http.Request) bool {
	if a.store.Get().UIPassword == "" {
		return true
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	if !ok || time.Now().After(exp) {
		delete(a.sessions, c.Value)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"ok": false, "error": err.Error()})
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("bad JSON body: %w", err)
	}
	return nil
}

func (a *Agent) hLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	pw := a.store.Get().UIPassword
	if pw != "" && subtle.ConstantTimeCompare([]byte(body.Password), []byte(pw)) != 1 {
		time.Sleep(500 * time.Millisecond)
		httpErr(w, http.StatusUnauthorized, errors.New("wrong password"))
		return
	}
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	a.mu.Lock()
	a.sessions[tok] = time.Now().Add(30 * 24 * time.Hour)
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 30 * 24 * 3600, Secure: r.TLS != nil})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.Snapshot())
}

func (a *Agent) hEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		httpErr(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		b, _ := json.Marshal(a.Snapshot())
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return
		}
		fl.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-t.C:
		}
	}
}

func (a *Agent) hLogs(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": a.logs.Since(since, limit)})
}

func (a *Agent) hMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if !config.ValidMode(body.Mode) {
		httpErr(w, http.StatusBadRequest, fmt.Errorf("unknown mode %q", body.Mode))
		return
	}
	prev := a.store.Get().Mode
	if _, err := a.store.Update(func(c *config.Config) error { c.Mode = body.Mode; return nil }); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	if prev != body.Mode {
		a.Logf("mode %s -> %s", prev, body.Mode)
		if a.sup.Active() {
			go a.sup.Restart()
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "mode": body.Mode})
}

func (a *Agent) hPathsGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	paths, err := a.listPaths()
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

func (a *Agent) hPathsPost(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var u pathUpdate
	if err := readJSON(r, &u); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if err := a.updatePath(u); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// connect/disconnect are client-role in the contract; on the server they
// start/stop mqvpn as an additive convenience.
func (a *Agent) hConnect(w http.ResponseWriter, r *http.Request) {
	var err error
	if a.role == config.RoleClient {
		err = a.connect()
	} else {
		a.sup.Start()
	}
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hDisconnect(w http.ResponseWriter, r *http.Request) {
	if a.role == config.RoleClient {
		if err := a.disconnect(); err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		go a.sup.Stop()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hClientsGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	writeJSON(w, http.StatusOK, a.Snapshot().Clients)
}

func (a *Agent) hClientsPost(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := a.addClient(r.Context(), strings.TrimSpace(body.Name))
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, res)
}

// hClientGet re-issues the pairing URI/QR of an existing client (additive).
func (a *Agent) hClientGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	name := r.PathValue("name")
	for _, u := range a.store.Get().Server.Users {
		if u.Name == name {
			res, err := a.pairingFor(u)
			if err != nil {
				httpErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, res)
			return
		}
	}
	httpErr(w, http.StatusNotFound, errNotFound)
}

func (a *Agent) hClientDelete(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	err := a.removeClient(r.Context(), r.PathValue("name"))
	switch {
	case errors.Is(err, errNotFound):
		httpErr(w, http.StatusNotFound, err)
	case err != nil:
		httpErr(w, http.StatusInternalServerError, err)
	default:
		a.poll(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func (a *Agent) hPair(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		URI string `json:"uri"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	p, err := a.importPairing(body.URI)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": p.Name, "server": p.Server, "mode": p.Mode})
}

func (a *Agent) hTailscale(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.ts.Get(r.Context()))
}

type settingsView struct {
	Role        string `json:"role"`
	Mode        string `json:"mode"`
	ConfigPath  string `json:"config_path"`
	Password    bool   `json:"password_set"`
	PublicAddr  string `json:"public_addr,omitempty"`
	Effective   string `json:"effective_public_addr,omitempty"`
	Guessed     bool   `json:"public_addr_guessed,omitempty"`
	Port        int    `json:"port,omitempty"`
	Subnet      string `json:"subnet,omitempty"`
	TLS         string `json:"tls,omitempty"`
	TLSName     string `json:"tls_name,omitempty"`
	CertSHA256  string `json:"cert_sha256,omitempty"`
	PairedWith  string `json:"paired_with,omitempty"`
	PairedAs    string `json:"paired_as,omitempty"`
	ManageRoute bool   `json:"manage_routes,omitempty"`
	// client: the user turned tailbond off (client.disconnected)
	Disconnected bool `json:"disconnected,omitempty"`
	// client: captive-portal detection on (client.captive_portal_detect)
	CaptiveDetect *bool  `json:"captive_portal_detect,omitempty"`
	Version       string `json:"version"`
	AndroidAPK    bool   `json:"android_apk"`
	GLiNetIPK     bool   `json:"glinet_ipk"`
}

// hAPK offers the Android app to phones opening the dashboard (additive).
func (a *Agent) hAPK(w http.ResponseWriter, r *http.Request) {
	serveDownload(w, r, a.store.Get().APKPath(), "tailbond.apk", "application/vnd.android.package-archive")
}

func serveDownload(w http.ResponseWriter, r *http.Request, path, name, ctype string) {
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeContent(w, r, name, st.ModTime(), f)
}

func apkAvailable(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func (a *Agent) hSettingsGet(w http.ResponseWriter, r *http.Request) {
	cfg := a.store.Get()
	v := settingsView{Role: cfg.Role, Mode: cfg.Mode, ConfigPath: a.store.Path(), Password: cfg.UIPassword != "", Version: a.opts.Version, AndroidAPK: apkAvailable(cfg.APKPath()), GLiNetIPK: a.glinetIPKAvailable()}
	if cfg.Role == config.RoleServer {
		a.mu.Lock()
		tls := a.tlsInfo
		a.mu.Unlock()
		v.PublicAddr, v.Effective, v.Guessed = cfg.Server.PublicAddr, a.publicAddr(), cfg.Server.PublicAddr == ""
		v.Port, v.Subnet, v.TLS, v.TLSName, v.CertSHA256 = cfg.Server.Port, cfg.Server.Subnet, tls.source, tls.sni, tls.sha256
	} else {
		a.mu.Lock()
		if p := a.client.payload; p != nil {
			v.PairedWith, v.PairedAs = p.Server, p.User
		}
		a.mu.Unlock()
		v.ManageRoute = cfg.Client.RoutesManaged()
		v.Disconnected = cfg.Client.Disconnected
		on := cfg.Client.CaptiveDetect()
		v.CaptiveDetect = &on
	}
	writeJSON(w, http.StatusOK, v)
}

// hSettingsPost updates the few settings the UI exposes (additive).
func (a *Agent) hSettingsPost(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PublicAddr *string `json:"public_addr"`
		UIPassword *string `json:"ui_password"`
		Captive    *bool   `json:"captive_portal_detect"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if body.PublicAddr != nil && *body.PublicAddr != "" {
		pa := strings.TrimSpace(*body.PublicAddr)
		if strings.ContainsAny(pa, " /\\@?#") {
			httpErr(w, http.StatusBadRequest, errors.New("public_addr must be host or host:port"))
			return
		}
		body.PublicAddr = &pa
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		if body.PublicAddr != nil {
			c.Server.PublicAddr = *body.PublicAddr
		}
		if body.UIPassword != nil {
			c.UIPassword = *body.UIPassword
		}
		if body.Captive != nil {
			v := *body.Captive
			c.Client.CaptivePortalDetect = &v
		}
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// listenAddrs resolves the "listen" setting. With no host the server role
// binds loopback plus its Tailscale IPs, the client role every interface.
// The server role also binds its tunnel address (for bonded clients' GET
// /api/peers) unless the listen host is a wildcard that already covers it.
func (a *Agent) listenAddrs(ctx context.Context) []string {
	cfg := a.store.Get()
	host, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		host, port = "", strings.TrimPrefix(cfg.Listen, ":")
	}
	if cfg.Role == config.RoleClient {
		return []string{net.JoinHostPort(host, port)}
	}
	var out []string
	switch host {
	case "":
		out = append(out, net.JoinHostPort("127.0.0.1", port))
		for _, ip := range a.ts.Get(ctx).IPs {
			out = append(out, net.JoinHostPort(ip, port))
		}
	case "0.0.0.0", "::":
		return []string{net.JoinHostPort(host, port)}
	default:
		out = append(out, net.JoinHostPort(host, port))
	}
	// appears once mqvpn is up; serveHTTP retries the bind
	if ip := cfg.ServerTunnelIP(); ip != "" && ip != host {
		out = append(out, net.JoinHostPort(ip, port))
	}
	return out
}

func (a *Agent) serveHTTP(ctx context.Context) error {
	h := a.Handler()
	var mu sync.Mutex
	bound := map[string]*http.Server{}
	errc := make(chan error, 1)
	bind := func(first bool) error {
		for _, addr := range a.listenAddrs(ctx) {
			mu.Lock()
			_, ok := bound[addr]
			mu.Unlock()
			if ok {
				continue
			}
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				// loopback/explicit failures are fatal at startup; Tailscale
				// IPs may simply not be up yet.
				if first && len(bound) == 0 {
					return err
				}
				continue
			}
			srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
			mu.Lock()
			bound[addr] = srv
			mu.Unlock()
			a.Logf("web UI listening on http://%s/", addr)
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					a.Logf("http %s: %v", addr, err)
				}
			}()
		}
		return nil
	}
	if err := bind(true); err != nil {
		return err
	}
	// retry often enough that the tunnel address is served soon after mqvpn starts
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, s := range bound {
				sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.Shutdown(sctx)
				cancel()
			}
			mu.Unlock()
			return nil
		case err := <-errc:
			return err
		case <-t.C:
			_ = bind(false)
		}
	}
}
