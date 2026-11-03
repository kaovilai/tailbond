package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/captive"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/glrepeater"
	"github.com/kaovilai/tailbond/agent/internal/hooks"
	"github.com/kaovilai/tailbond/agent/internal/netif"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// Captive portals (client role; docs/architecture.md, "Captive portals").
// Each usable WAN is probed over itself. A WAN behind a portal stays out of
// mqvpn's paths (it doesn't count as a failed path or for failover) until
// the probe says open; the other WANs keep the tunnel up. Sign-in mode lets
// LAN devices reach one portal WAN directly (portal-open hook) for up to
// signinTTL, and ends by itself once the WAN is open (portal-close).

const (
	probePortalEvery  = 60 * time.Second
	probeOfflineEvery = 60 * time.Second
	probeOpenEvery    = 10 * time.Minute
	probeSigninEvery  = 5 * time.Second
	signinTTL         = 10 * time.Minute
	captiveTick       = 2 * time.Second
)

// captiveState is guarded by Agent.mu (inside clientState).
type captiveState struct {
	results  map[string]captive.Result // by settings key (network or device)
	usable   map[string]bool           // last seen usable, by key
	recheck  map[string]bool           // probe at the next tick
	signin   *signin
	probeNow chan struct{}
	source   map[string]string // key -> "glinet" when GL's repeater owns it
}

type signin struct {
	Key, Dev, URL string
	Source        string // "tailbond" (our button) or "glinet" (GL's Login Mode)
	DNS           []string
	Until         time.Time
}

func newCaptiveState() *captiveState {
	return &captiveState{results: map[string]captive.Result{}, usable: map[string]bool{}, recheck: map[string]bool{}, probeNow: make(chan struct{}, 1), source: map[string]string{}}
}

// captiveBlocked: a WAN behind a portal is not bonded. Caller holds a.mu.
func (cs *clientState) captiveBlocked(i netif.Iface) bool {
	r, ok := cs.cap.results[i.Key()]
	return ok && r.State == captive.Portal
}

func (a *Agent) prober() *captive.Prober {
	cfg := a.store.Get()
	return &captive.Prober{
		Probes: captive.ProbesFor(cfg.Client.CaptiveProbeURLs),
		Dial:   a.captiveDial,
		DNS: func(dev string) []string {
			a.mu.Lock()
			defer a.mu.Unlock()
			return a.client.ifs[dev].DNS
		},
	}
}

// pokeCaptive asks the watcher to re-probe now (all WANs, or one key).
func (a *Agent) pokeCaptive(key string) {
	a.mu.Lock()
	if key == "" {
		for k := range a.client.cap.results {
			a.client.cap.recheck[k] = true
		}
		for k := range a.client.cap.usable {
			a.client.cap.recheck[k] = true
		}
	} else {
		a.client.cap.recheck[key] = true
	}
	ch := a.client.cap.probeNow
	a.mu.Unlock()
	select {
	case ch <- struct{}{}:
	default:
	}
}

// captiveWatcher probes WANs on link up, on hotplug (Rescan), when mqvpn
// reports a path failed, every minute while behind a portal or offline,
// every 5 s during sign-in, and every 10 minutes otherwise.
func (a *Agent) captiveWatcher(ctx context.Context) {
	t := time.NewTicker(captiveTick)
	defer t.Stop()
	a.mu.Lock()
	ch := a.client.cap.probeNow
	a.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			a.endSignin("shutdown")
			return
		case <-t.C:
		case <-ch:
		}
		a.captiveRound(ctx)
	}
}

type probeJob struct {
	key, dev string
}

func (a *Agent) captiveRound(ctx context.Context) {
	cfg := a.store.Get()
	now := time.Now()
	glChanged := false
	a.mu.Lock()
	cs := a.client
	cp := cs.cap
	if !cfg.Client.CaptiveDetect() {
		changed := len(cp.results) > 0
		cp.results, cp.recheck = map[string]captive.Result{}, map[string]bool{}
		si := cp.signin
		var wwan netif.Iface
		for _, it := range cs.ifs {
			if it.Network == glrepeater.Network {
				wwan = it
			}
		}
		a.mu.Unlock()
		// Even with our detection off, GL's Login Mode must not be blocked
		// by tailbond's routing and kill switch.
		gl := a.glRead(ctx)
		switch {
		case gl.Present && gl.LoginMode && wwan.Name != "":
			if si == nil || si.Source != "glinet" {
				a.openSignin(&signin{Key: wwan.Key(), Dev: wwan.Name, URL: gl.PortalURL, DNS: wwan.DNS, Source: "glinet"})
			}
		case si != nil:
			a.endSignin("captive portal detection turned off")
		}
		if changed {
			a.reconcilePaths(true)
		}
		return
	}
	a.mu.Unlock()
	// GL's repeater portal handling, when present and on, owns the repeater
	// uplink: no probe of ours there (GL probes it already), and its Login
	// Mode drives our sign-in bypass.
	gl := a.glRead(ctx)
	glOwns := gl.Present && gl.Enabled
	a.mu.Lock()
	var jobs []probeJob
	var glKey, glDev string
	var glDNS []string
	seen := map[string]bool{}
	for dev, it := range cs.ifs {
		ps, ok := effectiveSetting(cfg, it)
		key := it.Key()
		usable := ok && ps.Enabled && it.Usable()
		seen[key] = true
		was := cp.usable[key]
		cp.usable[key] = usable
		if !usable {
			delete(cp.results, key)
			continue
		}
		r, have := cp.results[key]
		if glOwns && it.Network == glrepeater.Network {
			glKey, glDev, glDNS = key, dev, it.DNS
			cp.source[key] = "glinet"
			next := captive.Result{State: captive.Open, Checked: now}
			switch {
			case gl.Portal:
				next = captive.Result{State: captive.Portal, PortalURL: gl.PortalURL, Checked: now}
			case gl.Detecting && have:
				next = r // keep the last verdict while GL checks
			}
			if next.State != r.State {
				a.logs.Add("agent", "captive (GL repeater) "+key+": "+next.State+portalSuffix(next))
				if r.State == captive.Portal || next.State == captive.Portal {
					glChanged = true
				}
			}
			cp.results[key] = next
			continue
		}
		delete(cp.source, key)
		every := probeOpenEvery
		switch {
		case cp.signin != nil && cp.signin.Key == key:
			every = probeSigninEvery
		case r.State == captive.Portal:
			every = probePortalEvery
		case r.State == captive.Offline:
			every = probeOfflineEvery
		}
		if !have || !was || cp.recheck[key] || now.Sub(r.Checked) >= every {
			jobs = append(jobs, probeJob{key, dev})
			delete(cp.recheck, key)
		}
	}
	for k := range cp.results {
		if !seen[k] {
			delete(cp.results, k)
		}
	}
	si := cp.signin
	a.mu.Unlock()

	// GL's Login Mode on: LAN devices must reach the portal past our
	// routing too. Off (signed in, or quit): put our routing back.
	switch {
	case glOwns && gl.LoginMode && glKey != "":
		if si == nil || si.Key != glKey || si.Source != "glinet" {
			a.openSignin(&signin{Key: glKey, Dev: glDev, URL: gl.PortalURL, DNS: glDNS, Source: "glinet"})
		} else {
			a.mu.Lock()
			si.Until = now.Add(signinTTL) // GL decides when it ends
			a.mu.Unlock()
		}
	case si != nil && si.Source == "glinet":
		a.endSignin("GL's Login Mode ended")
	case si != nil && now.After(si.Until):
		a.endSignin("sign-in time ran out")
	}
	changed := glChanged
	p := a.prober()
	for _, j := range jobs {
		r := p.Check(ctx, j.dev)
		a.mu.Lock()
		prev := cp.results[j.key]
		cp.results[j.key] = r
		sig := cp.signin
		a.mu.Unlock()
		if prev.State != r.State {
			a.Logf("captive check %s (%s): %s%s", j.key, j.dev, r.State, portalSuffix(r))
			if prev.State == captive.Portal || r.State == captive.Portal {
				changed = true
			}
		}
		if sig != nil && sig.Key == j.key && r.State == captive.Open {
			a.endSignin("signed in")
		}
	}
	if changed {
		a.reconcilePaths(true)
	}
}

func portalSuffix(r captive.Result) string {
	if r.PortalURL != "" {
		return " (" + r.PortalURL + ")"
	}
	return ""
}

// markFailedPaths: mqvpn says a path failed while its link is up; check
// whether a portal appeared (e.g. the hotel's session expired).
func (a *Agent) markFailedPaths(paths []status.Path) {
	for _, p := range paths {
		if p.State != status.PathFailed {
			continue
		}
		a.mu.Lock()
		it, ok := a.client.ifs[p.Iface]
		key := ""
		if ok && it.Usable() {
			if r := a.client.cap.results[it.Key()]; r.State == captive.Open && time.Since(r.Checked) > 30*time.Second {
				key = it.Key()
				a.client.cap.recheck[key] = true
			}
		}
		a.mu.Unlock()
		if key != "" {
			select {
			case a.client.cap.probeNow <- struct{}{}:
			default:
			}
		}
	}
}

func (a *Agent) portalEnv(si *signin) map[string]string {
	a.mu.Lock()
	env := a.hookEnvLocked()
	a.mu.Unlock()
	env["TB_PORTAL_DEV"] = si.Dev
	env["TB_PORTAL_NET"] = si.Key
	env["TB_PORTAL_DNS"] = strings.Join(si.DNS, ",")
	env["TB_PORTAL_URL"] = si.URL
	env["TB_PORTAL_SOURCE"] = si.Source
	return env
}

// startSignin: POST /api/captive/signin {"name"}.
func (a *Agent) startSignin(name string) (*signin, error) {
	a.mu.Lock()
	var key, dev string
	var dns []string
	for d, it := range a.client.ifs {
		if it.Key() == name || d == name {
			key, dev, dns = it.Key(), d, it.DNS
		}
	}
	r, ok := a.client.cap.results[key]
	if key == "" || !ok || r.State != captive.Portal {
		a.mu.Unlock()
		return nil, errNotPortal
	}
	a.mu.Unlock()
	si := &signin{Key: key, Dev: dev, URL: r.PortalURL, DNS: dns, Source: "tailbond"}
	a.openSignin(si)
	a.pokeCaptive(key)
	return si, nil
}

// openSignin turns sign-in mode on for one WAN (portal-open), replacing
// any other one.
func (a *Agent) openSignin(si *signin) {
	si.Until = time.Now().Add(signinTTL)
	a.mu.Lock()
	prev := a.client.cap.signin
	a.client.cap.signin = si
	a.mu.Unlock()
	if prev != nil && prev.Key != si.Key {
		a.runHook(hooks.PortalClose, a.portalEnv(prev))
	}
	a.Logf("sign-in mode on %s (%s, %s): LAN devices reach it directly", si.Key, si.Dev, si.Source)
	a.runHook(hooks.PortalOpen, a.portalEnv(si))
}

// endSignin turns sign-in mode off (portal-close) if it is on.
func (a *Agent) endSignin(why string) {
	a.mu.Lock()
	if a.client == nil || a.client.cap.signin == nil {
		a.mu.Unlock()
		return
	}
	si := a.client.cap.signin
	a.client.cap.signin = nil
	a.mu.Unlock()
	a.Logf("sign-in mode on %s off: %s", si.Key, why)
	env := a.portalEnv(si)
	if why == "shutdown" {
		a.hooks.Run(hooks.PortalClose, env) // synchronous: the hook queue is stopping
		return
	}
	a.runHook(hooks.PortalClose, env)
}

var errNotPortal = errors.New("that connection doesn't need a sign-in right now")

// annotateCaptive fills the captive fields of status paths. Caller holds no lock.
func (a *Agent) annotateCaptive(paths []status.Path) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := a.client.cap
	for i := range paths {
		p := &paths[i]
		it, ok := a.client.ifs[p.Iface]
		key := p.Iface
		if ok {
			key = it.Key()
		}
		r, have := cp.results[key]
		if !have {
			continue
		}
		p.Captive = r.State
		if r.State == captive.Portal {
			p.PortalURL = r.PortalURL
			if p.State == status.PathFailed || p.State == status.PathActive {
				p.State = status.PathStandby // held back on purpose, not broken
			}
		}
		if cp.signin != nil && cp.signin.Key == key {
			u := cp.signin.Until
			p.SigninUntil = &u
		}
		p.CaptiveSource = "tailbond"
		if cp.source[key] != "" {
			p.CaptiveSource = cp.source[key]
		}
	}
}

func (a *Agent) hCaptiveSignin(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	si, err := a.startSignin(body.Name)
	if err != nil {
		httpErr(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": si.Key, "portal_url": si.URL, "until": si.Until.UTC()})
}

func (a *Agent) hCaptiveSigninEnd(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	a.endSignin("stopped from the dashboard")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// hCaptiveCheck: POST /api/captive/check {"name"} re-probes now ("" = all).
func (a *Agent) hCaptiveCheck(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = readJSON(r, &body)
	a.pokeCaptive(body.Name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// captiveDial is the probe's connection over one WAN (tests replace it).
func (a *Agent) captiveDial(ctx context.Context, dev, addr string) (net.Conn, error) {
	if a.captiveDialer != nil {
		return a.captiveDialer(ctx, dev, addr)
	}
	return captive.DialDevice(ctx, dev, addr)
}
