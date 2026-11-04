package agent

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/update"
)

// HTTP for updates (docs/architecture.md, "Updates" → HTTP).

func (a *Agent) hUpdateGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.updateView())
}

func (a *Agent) hUpdateCheck(w http.ResponseWriter, r *http.Request) {
	select {
	case a.upd.checkNow <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

func (a *Agent) hUpdateInstall(w http.ResponseWriter, r *http.Request) {
	switch m := a.method(); m {
	case methodStatic, methodOpkg:
	case methodDnf:
		httpErr(w, http.StatusBadRequest, errors.New("this server updates with dnf: sudo dnf upgrade tailbond-server"))
		return
	default:
		httpErr(w, http.StatusBadRequest, errors.New("this install can't update itself: install the new release by hand"))
		return
	}
	a.upd.mu.Lock()
	rel := a.upd.rel
	a.upd.mu.Unlock()
	if rel == nil || !(update.Newer(rel.Version, a.opts.Version) || !isRelease(a.opts.Version)) {
		httpErr(w, http.StatusConflict, errors.New("no newer release found: check first"))
		return
	}
	select {
	case a.upd.installNw <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "version": rel.Version})
}

func (a *Agent) hUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Policy *string `json:"policy"`
		Repo   *string `json:"repo"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
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
		if body.Repo != nil {
			u.Repo = strings.TrimSpace(*body.Repo)
		}
		c.Update = &u
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if body.Policy != nil && *body.Policy == config.UpdateOff {
		a.setUpdate(func(v *status.Update) { v.State = "off" })
	} else {
		a.setUpdate(func(v *status.Update) {
			if v.State == "off" {
				v.State = "idle"
			}
		})
		select {
		case a.upd.checkNow <- struct{}{}:
		default:
		}
	}
	writeJSON(w, http.StatusOK, a.updateView())
}

// tokenOrTailnet: /api/pairing and /api/backups/push need a client token,
// an admin session, or a tailnet / loopback source.
func (a *Agent) tokenOrTailnet(r *http.Request) bool {
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
