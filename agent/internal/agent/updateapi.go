package agent

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/update"
)

// HTTP for updates (docs/architecture.md, "Updates" → HTTP).

func (a *Agent) hUpdateGet(w http.ResponseWriter, r *http.Request) {
	a.upd.mu.Lock()
	m := a.upd.manifest
	a.upd.mu.Unlock()
	out := map[string]any{"update": a.updateView()}
	if m != nil {
		out["manifest"] = map[string]any{"version": m.Version, "channel": m.Channel, "released": m.Released, "security": m.Security,
			"notes_url": m.NotesURL, "protocol": m.Protocol, "requires_server_protocol": m.RequiresServerProtocol, "artifacts": len(m.Artifacts)}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *Agent) hUpdateCheck(w http.ResponseWriter, r *http.Request) {
	select {
	case a.upd.checkNow <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (a *Agent) hUpdateInstall(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version string `json:"version"`
	}
	if r.ContentLength != 0 {
		if err := readJSON(r, &body); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
	}
	a.upd.mu.Lock()
	m := a.upd.manifest
	a.upd.mu.Unlock()
	if m == nil {
		httpErr(w, http.StatusConflict, errors.New("no update found yet: check first"))
		return
	}
	if body.Version != "" && !update.Same(body.Version, m.Version) {
		httpErr(w, http.StatusConflict, fmt.Errorf("%s isn't the version on offer (%s)", body.Version, m.Version))
		return
	}
	select {
	case a.upd.installNw <- m.Version:
	default:
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "version": m.Version})
}

func (a *Agent) hUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Policy  *string `json:"policy"`
		Channel *string `json:"channel"`
		Window  *string `json:"window"`
		Source  *string `json:"source"`
		Pin     *string `json:"pin"`
		// attestation | sha256-only (setups without GitHub), and the
		// repository the attestations must come from (forks)
		Verify *string `json:"verify"`
		Repo   *string `json:"repo"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if body.Window != nil && !update.ValidWindow(*body.Window) {
		httpErr(w, http.StatusBadRequest, errors.New("window: HH:MM-HH:MM"))
		return
	}
	if body.Pin != nil && *body.Pin != "" && !isRelease(*body.Pin) {
		httpErr(w, http.StatusBadRequest, errors.New("pin: a version like 0.2.0"))
		return
	}
	if body.Source != nil && *body.Source != "" && !strings.HasPrefix(*body.Source, "https://") && !strings.HasPrefix(*body.Source, "http://") {
		httpErr(w, http.StatusBadRequest, errors.New("source: an http(s) URL"))
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		u := config.UpdateSettings{}
		if c.Update != nil {
			u = *c.Update
		}
		if body.Policy != nil {
			u.Policy = *body.Policy
		}
		if body.Channel != nil {
			u.Channel = *body.Channel
		}
		if body.Window != nil {
			u.Window = *body.Window
		}
		if body.Source != nil {
			u.Source = *body.Source
		}
		if body.Pin != nil {
			u.Pin = *body.Pin
		}
		if body.Verify != nil {
			u.Verify = *body.Verify
		}
		if body.Repo != nil {
			u.Repo = *body.Repo
		}
		c.Update = &u
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	select {
	case a.upd.checkNow <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "update": a.updateView()})
}

// hUpdateUpload: one file into the server's mirror (offline setups). The
// manifest is verified as soon as it can be: at once with update.verify
// sha256-only; with attestations, when its .sigstore.json is there too (or
// GitHub's API has it, if the server is online).
func (a *Agent) hUpdateUpload(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	name := r.URL.Query().Get("name")
	mir := a.mirror()
	u := a.store.Get().UpdateOrDefault()
	isManifest := name == u.Channel+".json" || name == u.Channel+".json"+update.BundleSuffix
	var dst string
	if isManifest {
		dst = filepath.Join(mir.Dir, "upload", name)
	} else {
		dst = mir.FilePath(name)
	}
	if dst == "" || name == "" {
		httpErr(w, http.StatusBadRequest, errors.New("name: a manifest (<channel>.json, <channel>.json.sigstore.json) or an artifact file name"))
		return
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	f, err := os.Create(dst + ".part")
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	n, err := io.Copy(f, io.LimitReader(r.Body, 512<<20+1))
	f.Close()
	if err != nil || n > 512<<20 {
		os.Remove(dst + ".part")
		httpErr(w, http.StatusBadRequest, errors.New("upload failed or too large"))
		return
	}
	if err := os.Rename(dst+".part", dst); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	res := map[string]any{"ok": true, "name": name, "size": n}
	up := filepath.Join(mir.Dir, "upload", u.Channel+".json")
	t := a.trust()
	if raw, err := os.ReadFile(up); err == nil {
		bundle, _ := os.ReadFile(up + update.BundleSuffix)
		if t.Mode == update.VerifySHA256 || bundle != nil || name == u.Channel+".json" {
			m, how, b, err := t.Check(r.Context(), raw, bundle)
			switch {
			case err == nil:
				if err := mir.Store(u.Channel, raw, b); err != nil {
					httpErr(w, http.StatusInternalServerError, err)
					return
				}
				mir.SetOrigin(u.Channel, "upload")
				os.Remove(up)
				os.Remove(up + update.BundleSuffix)
				res["manifest"], res["verified"] = m.Version, how
				a.Logf("update mirror: uploaded manifest %s (%s)", m.Version, how)
			case bundle == nil && errors.Is(err, update.ErrNoAttestation):
				// offline: wait for the .sigstore.json from the release
				res["waiting_for"] = u.Channel + ".json" + update.BundleSuffix
			default:
				// start over: upload the manifest and its attestation again
				os.Remove(up)
				os.Remove(up + update.BundleSuffix)
				httpErr(w, http.StatusBadRequest, fmt.Errorf("the uploaded manifest doesn't verify: %w", err))
				return
			}
		}
	}
	if !isManifest {
		if raw, bundle, ok := mir.Manifest(u.Channel); ok {
			if m, _, _, err := t.Check(r.Context(), raw, bundle); err == nil {
				for _, art := range m.Artifacts {
					if art.Name == name {
						if err := update.CheckFile(dst, art); err != nil {
							os.Remove(dst)
							httpErr(w, http.StatusBadRequest, err)
							return
						}
						res["checksum_ok"] = true
					}
				}
			}
		}
	}
	select {
	case a.upd.checkNow <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusOK, res)
}

// hMirror serves /updates/<channel>.json, its .sigstore.json (when the
// manifest had a GitHub attestation) and /updates/files/<name>.
func (a *Agent) hMirror(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		http.NotFound(w, r)
		return
	}
	a.noteClientVersion(r, a.clientForToken(r))
	mir := a.mirror()
	p := strings.TrimPrefix(r.URL.Path, "/updates/")
	if name, ok := strings.CutPrefix(p, "files/"); ok {
		fp := mir.FilePath(name)
		if fp == "" {
			http.NotFound(w, r)
			return
		}
		serveDownload(w, r, fp, name, "application/octet-stream")
		return
	}
	channel, wantBundle := strings.CutSuffix(p, update.BundleSuffix)
	channel, ok := strings.CutSuffix(channel, ".json")
	if !ok || (channel != "stable" && channel != "beta") {
		http.NotFound(w, r)
		return
	}
	raw, bundle, found := mir.Manifest(channel)
	if !found || (wantBundle && bundle == nil) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	if wantBundle {
		_, _ = w.Write(bundle)
	} else {
		_, _ = w.Write(raw)
	}
}

// mirrorAllowed: /updates/ and /api/pairing need a client token, an admin
// session, or a tailnet / loopback source.
func (a *Agent) mirrorAllowed(r *http.Request) bool {
	if a.clientForToken(r) != "" || a.authorized(r) {
		return true
	}
	ip := remoteAddr(r)
	return ip.IsValid() && (ip.IsLoopback() || isTailnet(ip))
}

// hPairing: GET /api/pairing, a client's own current pairing URI (same
// credential), so it learns a new public address after the server moved.
func (a *Agent) hPairing(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleServer {
		httpErr(w, http.StatusBadRequest, errServerOnly)
		return
	}
	name := a.clientForToken(r)
	if name == "" {
		httpErr(w, http.StatusUnauthorized, errors.New("client token required"))
		return
	}
	a.noteClientVersion(r, name)
	for _, u := range a.store.Get().Server.Users {
		if u.Name == name {
			res, err := a.pairingFor(u)
			if err != nil {
				httpErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"name": name, "uri": res.URI})
			return
		}
	}
	httpErr(w, http.StatusNotFound, errNotFound)
}

// refreshPairings (client): ask each server for this client's current
// pairing and adopt it when only the address/TLS details changed (same user
// and key): every 6 h, and every 2 min while no server works.
func (a *Agent) refreshPairings(ctxDone <-chan struct{}) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	var last time.Time
	var downSince time.Time
	for {
		select {
		case <-ctxDone:
			return
		case <-t.C:
		}
		if !a.paired() {
			continue
		}
		working := a.Snapshot().State == status.StateConnected
		if working {
			downSince = time.Time{}
		} else if downSince.IsZero() {
			downSince = time.Now()
		}
		due := time.Since(last) > 6*time.Hour || (!working && time.Since(downSince) > 2*time.Minute && time.Since(last) > 2*time.Minute)
		if !due {
			continue
		}
		last = time.Now()
		a.refreshPairingsOnce()
	}
}

func (a *Agent) refreshPairingsOnce() {
	cfg := a.store.Get()
	a.mu.Lock()
	active, ready := a.client.activeServer, a.rt.ready
	a.mu.Unlock()
	for _, e := range cfg.Client.Servers {
		p, err := pairing.Decode(e.PairingURI)
		if err != nil {
			continue
		}
		var hosts []string
		if e.Name == active && ready {
			host := p.TunnelSubnet().Addr().Next().String()
			if len(p.DNS) > 0 {
				host = p.DNS[0]
			}
			hosts = append(hosts, host)
		}
		if p.SNI != "" && strings.Contains(p.SNI, ".ts.net") {
			hosts = append(hosts, p.SNI)
		}
		for _, h := range hosts {
			var res struct {
				URI string `json:"uri"`
			}
			hd := a.clientHeaders()
			hd.Set("Authorization", "Bearer "+APIToken(p.Key))
			code, err := a.joinCallH(h, "/api/pairing", hd, &res)
			if err != nil || code != http.StatusOK || res.URI == "" {
				continue
			}
			np, err := pairing.Decode(res.URI)
			if err != nil || np.User != p.User || subtle.ConstantTimeCompare([]byte(np.Key), []byte(p.Key)) != 1 {
				break
			}
			if res.URI != e.PairingURI {
				name := e.Name
				if _, err := a.store.Update(func(c *config.Config) error {
					if i := c.Client.ServerIndex(name); i >= 0 {
						c.Client.Servers[i].PairingURI = res.URI
					}
					return nil
				}); err == nil {
					a.Logf("server %s: new pairing details (%s -> %s)", name, p.Server, np.Server)
					a.afterServersChanged(switchList)
					if name == active {
						a.switchServer(name, switchList)
					}
				}
			}
			break
		}
	}
}

func (a *Agent) joinCallH(host, path string, h http.Header, out any) (int, error) {
	req, err := http.NewRequest(http.MethodGet, "http://"+hostPort(host, a.apiPort)+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header = h
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_ = jsonDecode(resp.Body, out)
	return resp.StatusCode, nil
}
