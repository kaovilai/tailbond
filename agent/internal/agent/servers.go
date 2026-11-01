package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/tailbond/tailbond/agent/internal/captive"
	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/failover"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/qprobe"
	"github.com/tailbond/tailbond/agent/internal/status"
)

// Client role: an ordered list of servers, one tunnel at a time to the
// active one, failover and failback between them (docs/architecture.md,
// "Multiple servers and failover").

// serverRT is one entry of client.servers, decoded. Guarded by Agent.mu.
type serverRT struct {
	name    string
	uri     string
	payload *pairing.Payload
	on      bool
}

const probeTimeout = 2 * time.Second

func policyOf(f config.Failover) failover.Policy {
	return failover.Policy{
		Mode: f.Policy, Pinned: f.Server,
		SwitchAfter:   time.Duration(f.SwitchAfterSec) * time.Second,
		FailbackAfter: time.Duration(f.FailbackAfterSec) * time.Second,
	}
}

// loadServers rebuilds the runtime list from the config and keeps the
// failover machine in step. It returns the server to switch to when the
// active one changed or went away ("" otherwise). Caller holds no lock.
func (a *Agent) loadServers() string {
	cfg := a.store.Get()
	var list []serverRT
	var order []string
	for _, e := range cfg.Client.Servers {
		p, err := pairing.Decode(e.PairingURI)
		if err != nil {
			a.Logf("server %s: stored pairing is invalid: %v", e.Name, err)
			continue
		}
		list = append(list, serverRT{name: e.Name, uri: e.PairingURI, payload: p, on: e.On()})
		if e.On() {
			order = append(order, e.Name)
		}
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	cs := a.client
	prevURI := ""
	if cur := cs.server(cs.activeServer); cur != nil {
		prevURI = cur.uri
	}
	cs.servers = list
	pol := policyOf(cfg.Client.FailoverOrDefault())
	to := ""
	if cs.fo == nil {
		cs.fo = failover.New(pol, order, "", now)
	} else {
		to = cs.fo.SetOrder(order, now)
		if t := cs.fo.SetPolicy(pol, now); t != "" {
			to = t
		}
	}
	cs.activeServer = cs.fo.Active()
	cs.payload = nil
	if s := cs.server(cs.activeServer); s != nil {
		cs.payload = s.payload
		// the active server was re-paired (new key or address)
		if prevURI != "" && s.uri != prevURI && to == "" {
			to = s.name
		}
	}
	return to
}

// server returns the runtime entry by name. Caller holds a.mu.
func (cs *clientState) server(name string) *serverRT {
	for i := range cs.servers {
		if cs.servers[i].name == name {
			return &cs.servers[i]
		}
	}
	return nil
}

// switchServer makes name the active server: mqvpn restarts, client-down
// runs with the old server's env (TB_SWITCH=1), client-up with the new one.
func (a *Agent) switchServer(name, why string) {
	a.mu.Lock()
	cs := a.client
	s := cs.server(name)
	if s == nil {
		a.mu.Unlock()
		return
	}
	prev := cs.activeServer
	cs.activeServer, cs.payload = name, s.payload
	cs.peers, cs.peersAt = nil, time.Time{}
	running := a.sup.Active()
	if running {
		cs.switchPending = true
	}
	a.mu.Unlock()
	if prev != name {
		a.Logf("server %s -> %s (%s)", prev, name, why)
	}
	if running {
		a.sup.Restart()
	}
}

// serverWatcher feeds the failover machine once a second: the active
// tunnel's health (mqvpn's state + control-API paths) and, every
// probe_every_sec, a probe of each standby server.
func (a *Agent) serverWatcher(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if !a.paired() || !a.sup.Active() {
			continue
		}
		cfg := a.store.Get()
		every := time.Duration(cfg.Client.FailoverOrDefault().ProbeEverySec) * time.Second
		now := time.Now()
		a.mu.Lock()
		cs := a.client
		var due []serverRT
		if len(cs.fo.Order()) > 1 {
			for _, s := range cs.servers {
				if !s.on || s.name == cs.activeServer || cs.probing[s.name] || now.Sub(cs.lastProbe[s.name]) < every {
					continue
				}
				cs.probing[s.name], cs.lastProbe[s.name] = true, now
				due = append(due, s)
			}
		}
		dev := a.probeDevLocked()
		a.mu.Unlock()
		for _, s := range due {
			go a.probeServer(ctx, s, dev)
		}
		working := a.tunnelWorking()
		a.mu.Lock()
		to := cs.fo.Step(now, working)
		a.mu.Unlock()
		if to != "" {
			a.switchServer(to, "failover policy")
		}
	}
}

// probeDevLocked picks a WAN for probes, so they leave outside the tunnel
// whatever the routing says ("" = let routing decide). Caller holds a.mu.
func (a *Agent) probeDevLocked() string {
	for _, dev := range a.client.active {
		if it, ok := a.client.ifs[dev]; ok && it.Usable() {
			return dev
		}
	}
	return ""
}

func (a *Agent) probeServer(ctx context.Context, s serverRT, dev string) {
	err := a.probe(ctx, s.payload.Server, dev)
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	cs := a.client
	delete(cs.probing, s.name)
	if cs.fo != nil {
		cs.fo.Probe(s.name, err == nil, now)
	}
	was, had := cs.probeOK[s.name]
	cs.probeOK[s.name] = err == nil
	if had && was != (err == nil) {
		if err == nil {
			a.logs.Add("agent", fmt.Sprintf("server %s answers again", s.name))
		} else {
			a.logs.Add("agent", fmt.Sprintf("server %s doesn't answer: %v", s.name, err))
		}
	}
}

// probe sends one QUIC stateless-reset probe to a server (tests stub it).
func (a *Agent) probe(ctx context.Context, server, dev string) error {
	if a.serverProbe != nil {
		return a.serverProbe(ctx, server, dev)
	}
	return qprobe.Probe(ctx, func(ctx context.Context, addr string) (net.Conn, error) {
		return captive.DialDeviceUDP(ctx, dev, addr)
	}, server, probeTimeout)
}

// tunnelWorking: the last status snapshot's verdict (workingFrom).
func (a *Agent) tunnelWorking() bool {
	return a.workingFrom(a.Snapshot(), time.Now())
}

// serversView fills the additive status fields. Caller holds no lock.
func (a *Agent) serversView(st *status.Status, working bool) {
	cfg := a.store.Get()
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	cs := a.client
	if cs.fo == nil || len(cs.servers) == 0 {
		return
	}
	v := cs.fo.View(now, working)
	byName := map[string]failover.ServerView{}
	for _, s := range v.Servers {
		byName[s.Name] = s
	}
	out := make([]status.ServerState, 0, len(cs.servers))
	for i, s := range cs.servers {
		sv := status.ServerState{Name: s.name, Role: "backup", Subnet: s.payload.TunnelSubnet().String(), Server: s.payload.Server, State: failover.StateUnknown}
		if i == 0 {
			sv.Role = "primary"
		}
		if x, ok := byName[s.name]; ok {
			sv.State, sv.Active = x.State, x.Active
			if !x.LastOK.IsZero() {
				t := x.LastOK.UTC()
				sv.LastOK = &t
			}
		}
		if !s.on {
			sv.State = "off"
		}
		out = append(out, sv)
	}
	f := cfg.Client.FailoverOrDefault()
	fv := &status.Failover{Policy: f.Policy, Server: f.Server, SwitchAfterSec: f.SwitchAfterSec, FailbackAfterSec: f.FailbackAfterSec, Reason: v.Reason}
	if !v.Since.IsZero() {
		t := v.Since.UTC()
		fv.Since = &t
	}
	if v.Counting {
		n := int((v.FailbackIn + time.Second - 1) / time.Second)
		fv.FailbackInSec = &n
	}
	st.Servers, st.ActiveServer, st.Failover = out, v.Active, fv
}

// subnetClash returns the server in the config whose tunnel subnet
// overlaps p's ("" if none), skipping the entry named skip.
func subnetClash(cfg *config.Config, p *pairing.Payload, skip string) (string, netip.Prefix) {
	want := p.TunnelSubnet()
	for _, e := range cfg.Client.Servers {
		if e.Name == skip {
			continue
		}
		q, err := pairing.Decode(e.PairingURI)
		if err != nil {
			continue
		}
		if config.Overlaps(q.TunnelSubnet(), want) {
			return e.Name, want
		}
	}
	return "", want
}

// sameServer: the same server and credential (adding it twice).
func sameServer(cfg *config.Config, p *pairing.Payload) string {
	for _, e := range cfg.Client.Servers {
		q, err := pairing.Decode(e.PairingURI)
		if err == nil && q.Server == p.Server && q.User == p.User {
			return e.Name
		}
	}
	return ""
}

type addServerError struct {
	code     string // subnet_in_use | exists
	conflict string
	subnet   string
}

func (e *addServerError) Error() string {
	if e.code == "exists" {
		return fmt.Sprintf("this server is already in the list as %s", e.conflict)
	}
	return fmt.Sprintf("this server uses the tunnel addresses %s, like %s; change server.subnet on the new server", e.subnet, e.conflict)
}

// addServer appends a pairing as a backup server (POST /api/servers).
func (a *Agent) addServer(uri, name string) (string, error) {
	p, err := pairing.Decode(uri)
	if err != nil {
		return "", err
	}
	var added string
	if _, err := a.store.Update(func(c *config.Config) error {
		if n := sameServer(c, p); n != "" {
			return &addServerError{code: "exists", conflict: n}
		}
		if n, sub := subnetClash(c, p, ""); n != "" {
			return &addServerError{code: "subnet_in_use", conflict: n, subnet: sub.String()}
		}
		if name == "" {
			name = p.Label()
		}
		added = config.UniqueServerName(c.Client.Servers, name)
		c.Client.Servers = append(c.Client.Servers, config.ServerEntry{Name: added, PairingURI: uri})
		c.Client.Disconnected = false
		return nil
	}); err != nil {
		return "", err
	}
	a.Logf("added server %s (%s) as a backup", added, p.Server)
	a.afterServersChanged()
	a.connectIfIdle()
	return added, nil
}

// afterServersChanged reloads the list and switches if the active server
// changed or is gone.
func (a *Agent) afterServersChanged() {
	to := a.loadServers()
	if !a.paired() {
		a.mu.Lock()
		a.userStop = true
		a.mu.Unlock()
		a.sup.Stop()
		return
	}
	if to != "" {
		a.switchServer(to, "server list changed")
	}
}

// connectIfIdle starts the tunnel unless the user turned tailbond off.
func (a *Agent) connectIfIdle() {
	if a.paired() && !a.store.Get().Client.Disconnected && !a.sup.Active() {
		a.mu.Lock()
		a.userStop = false
		a.mu.Unlock()
		a.sup.Start()
	}
}

// --- HTTP ---------------------------------------------------------------

func (a *Agent) hServersGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	st := a.Snapshot()
	servers := st.Servers
	if servers == nil {
		servers = []status.ServerState{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers, "active_server": st.ActiveServer, "failover": st.Failover})
}

func (a *Agent) hServersAdd(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name != "" && !validServerName(name) {
		httpErr(w, http.StatusBadRequest, errors.New("name: 1-32 letters, digits, '.', '_' or '-'"))
		return
	}
	a.replyAddServer(w, r, body.URI, name)
}

func (a *Agent) replyAddServer(w http.ResponseWriter, r *http.Request, uri, name string) {
	added, err := a.addServer(uri, name)
	if writeAddServerError(w, err) {
		return
	}
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": added})
}

// writeAddServerError answers 409 for a clashing or duplicate server.
func writeAddServerError(w http.ResponseWriter, err error) bool {
	var ae *addServerError
	if !errors.As(err, &ae) {
		return false
	}
	writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": ae.code, "message": ae.Error(), "conflict": ae.conflict, "subnet": ae.subnet})
	return true
}

func validServerName(s string) bool {
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func (a *Agent) hServersDelete(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	name := r.PathValue("name")
	found := false
	if _, err := a.store.Update(func(c *config.Config) error {
		if i := c.Client.ServerIndex(name); i >= 0 {
			found = true
			c.Client.Servers = append(c.Client.Servers[:i], c.Client.Servers[i+1:]...)
			if c.Client.Failover != nil && c.Client.Failover.Server == name {
				c.Client.Failover.Policy, c.Client.Failover.Server = config.PolicyPrefer, ""
			}
		}
		return nil
	}); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.Logf("server %s removed", name)
	a.afterServersChanged()
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hServersActive(w http.ResponseWriter, r *http.Request) {
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
	cfg := a.store.Get()
	i := cfg.Client.ServerIndex(body.Name)
	if i < 0 || !cfg.Client.Servers[i].On() {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	// under "Always use <server>", choosing another one changes the policy
	if f := cfg.Client.FailoverOrDefault(); f.Policy == config.PolicyAlways && f.Server != body.Name {
		if _, err := a.store.Update(func(c *config.Config) error {
			ff := c.Client.FailoverOrDefault()
			ff.Server = body.Name
			c.Client.Failover = &ff
			return nil
		}); err != nil {
			httpErr(w, http.StatusInternalServerError, err)
			return
		}
		if to := a.loadServers(); to != "" {
			a.switchServer(to, "always use "+body.Name)
		}
	} else {
		a.mu.Lock()
		to := a.client.fo.Manual(body.Name, time.Now())
		a.mu.Unlock()
		if to != "" {
			a.switchServer(to, "chosen in the dashboard")
		}
	}
	a.connectIfIdle()
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active_server": body.Name})
}

func (a *Agent) hServersOrder(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		Names []string `json:"names"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		if len(body.Names) != len(c.Client.Servers) {
			return errors.New("send every server's name, in the new order")
		}
		next := make([]config.ServerEntry, 0, len(body.Names))
		seen := map[string]bool{}
		for _, n := range body.Names {
			i := c.Client.ServerIndex(n)
			if i < 0 || seen[n] {
				return fmt.Errorf("unknown or repeated server %q", n)
			}
			seen[n] = true
			next = append(next, c.Client.Servers[i])
		}
		c.Client.Servers = next
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.afterServersChanged()
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Agent) hServersPolicy(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var body struct {
		Policy           *string `json:"policy"`
		Server           *string `json:"server"`
		SwitchAfterSec   *int    `json:"switch_after_sec"`
		FailbackAfterSec *int    `json:"failback_after_sec"`
	}
	if err := readJSON(r, &body); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		f := c.Client.FailoverOrDefault()
		if body.Policy != nil {
			f.Policy = *body.Policy
		}
		if body.Server != nil {
			f.Server = *body.Server
		}
		if body.SwitchAfterSec != nil {
			if *body.SwitchAfterSec < 5 || *body.SwitchAfterSec > 600 {
				return errors.New("switch_after_sec: 5-600")
			}
			f.SwitchAfterSec = *body.SwitchAfterSec
		}
		if body.FailbackAfterSec != nil {
			if *body.FailbackAfterSec < 10 || *body.FailbackAfterSec > 86400 {
				return errors.New("failback_after_sec: 10-86400")
			}
			f.FailbackAfterSec = *body.FailbackAfterSec
		}
		if f.Policy == config.PolicyAlways && c.Client.ServerIndex(f.Server) < 0 {
			return errors.New("pick which server to always use")
		}
		if f.Policy == config.PolicyPrefer {
			f.Server = ""
		}
		c.Client.Failover = &f
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.afterServersChanged()
	a.poll(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
