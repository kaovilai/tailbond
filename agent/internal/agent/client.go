package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/mqconf"
	"github.com/tailbond/tailbond/agent/internal/netif"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/supervisor"
	"github.com/tailbond/tailbond/agent/internal/tlsutil"
)

// clientState is guarded by Agent.mu.
type clientState struct {
	payload  *pairing.Payload
	serverIP string
	active   []string // Path interfaces in the running config
	pending  []string // desired set awaiting debounce
	pendingN int
	usage    usageFile
	lastCtr  map[string][2]uint64
	dirty    bool
	ifs      map[string]netif.Iface // last discovery, by device
	peers    []byte                 // server's GET /api/peers, cached
	peersAt  time.Time
	cap      *captiveState
}

type usageFile struct {
	Month string            `json:"month"` // usage resets monthly
	Bytes map[string]uint64 `json:"bytes"`
}

func newClientState() *clientState {
	return &clientState{usage: usageFile{Bytes: map[string]uint64{}}, lastCtr: map[string][2]uint64{}, ifs: map[string]netif.Iface{}, cap: newCaptiveState()}
}

func (a *Agent) usagePath() string { return filepath.Join(a.store.Get().StateDir, "usage.json") }

func (a *Agent) startClient(ctx context.Context) {
	cfg := a.store.Get()
	if b, err := os.ReadFile(a.usagePath()); err == nil {
		_ = json.Unmarshal(b, &a.client.usage)
		if a.client.usage.Bytes == nil {
			a.client.usage.Bytes = map[string]uint64{}
		}
	}
	if cfg.Client.PairingURI != "" {
		p, err := pairing.Decode(cfg.Client.PairingURI)
		if err != nil {
			a.Logf("stored pairing is invalid: %v", err)
		} else {
			a.mu.Lock()
			a.client.payload = p
			a.mu.Unlock()
		}
	}
	go a.pathWatcher(ctx)
	go a.captiveWatcher(ctx)
	if a.paired() && !cfg.Client.Disconnected {
		a.sup.Start()
	} else if !a.paired() {
		a.Logf("not paired yet: import a pairing URI in the Setup tab")
	}
}

func (a *Agent) paired() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client != nil && a.client.payload != nil
}

func priorityRank(p string) int {
	switch p {
	case config.PriorityPrimary:
		return 0
	case config.PrioritySecondary:
		return 1
	default:
		return 2
	}
}

// capReached reports whether a path has used up its data cap this month.
func (cs *clientState) capReached(p config.PathSetting) bool {
	return p.DataCapMB > 0 && cs.usage.Bytes[p.Iface] >= uint64(p.DataCapMB)<<20
}

// settingFor finds the path setting for a discovered interface: by its
// OpenWrt network name first, then by device. ok is false when there is none.
func settingFor(cfg *config.Config, i netif.Iface) (config.PathSetting, bool) {
	if i.Network != "" {
		if ps, ok := cfg.PathSetting(i.Network); ok {
			return ps, true
		}
	}
	return cfg.PathSetting(i.Name)
}

// effectiveSetting is the setting a discovered interface runs with: its
// own, or for an unconfigured WAN (OpenWrt) the default "use it": primary
// for Ethernet and Wi-Fi, secondary for tethering and cellular, so that
// "Save mobile data" (failover) keeps those on standby.
func effectiveSetting(cfg *config.Config, i netif.Iface) (config.PathSetting, bool) {
	if ps, ok := settingFor(cfg, i); ok {
		return ps, true
	}
	if i.WAN {
		prio := config.PriorityPrimary
		if i.Kind == netif.KindCellular || i.Kind == netif.KindUSB {
			prio = config.PrioritySecondary
		}
		return config.PathSetting{Iface: i.Key(), Enabled: true, Priority: prio}, true
	}
	return config.PathSetting{}, false
}

// desiredPaths applies priorities, mode and data caps to the interfaces that
// exist right now and returns the devices to bind. Failover keeps only the
// best tier that has a usable link; the bonding modes use primary+secondary
// and add backup paths only when none of those is usable.
// blocked (may be nil) holds WANs back that sit behind a captive portal:
// not bonded, and not counted as usable, so failover moves on instead of
// flapping.
func desiredPaths(cfg *config.Config, ifs []netif.Iface, capReached func(config.PathSetting) bool, blocked func(netif.Iface) bool) []string {
	tiers := [3][]string{}
	usable := [3]bool{}
	for _, it := range ifs {
		p, ok := effectiveSetting(cfg, it)
		if !ok || !p.Enabled || capReached(p) || (blocked != nil && blocked(it)) {
			continue
		}
		r := priorityRank(p.Priority)
		tiers[r] = append(tiers[r], it.Name)
		if it.Usable() {
			usable[r] = true
		}
	}
	var out []string
	if cfg.Mode == config.ModeFailover {
		for r := 0; r < 3; r++ {
			if usable[r] {
				out = tiers[r]
				break
			}
		}
		if out == nil {
			for r := 0; r < 3 && out == nil; r++ {
				if len(tiers[r]) > 0 {
					out = tiers[r]
				}
			}
		}
	} else {
		out = append(append(out, tiers[0]...), tiers[1]...)
		if !usable[0] && !usable[1] || len(out) == 0 {
			out = append(out, tiers[2]...)
		}
	}
	out = slices.Clone(out)
	sort.Strings(out)
	return out
}

// remember stores the latest discovery. Caller holds a.mu.
func (cs *clientState) remember(ifs []netif.Iface) {
	cs.ifs = make(map[string]netif.Iface, len(ifs))
	for _, i := range ifs {
		cs.ifs[i.Name] = i
	}
}

// keyOf maps a device to its settings/usage key. Caller holds a.mu.
func (cs *clientState) keyOf(dev string) string {
	if i, ok := cs.ifs[dev]; ok {
		return i.Key()
	}
	return dev
}

// pathWatcher re-evaluates the path set every 5 s and tracks data usage.
func (a *Agent) pathWatcher(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	lastSave := time.Now()
	for {
		select {
		case <-ctx.Done():
			a.saveUsage()
			return
		case <-t.C:
		case <-a.rescan:
			a.Logf("network change: re-checking connections")
		}
		a.accountUsage()
		if time.Since(lastSave) > time.Minute {
			a.saveUsage()
			lastSave = time.Now()
		}
		a.reconcilePaths(false)
	}
}

func (a *Agent) accountUsage() {
	month := time.Now().Format("2006-01")
	a.mu.Lock()
	defer a.mu.Unlock()
	cs := a.client
	if cs.usage.Month != month {
		cs.usage = usageFile{Month: month, Bytes: map[string]uint64{}}
		cs.dirty = true
	}
	tunUp := a.rt.ready
	for _, iface := range cs.active {
		tx, rx, ok := netif.Counters(iface)
		if !ok {
			delete(cs.lastCtr, iface)
			continue
		}
		prev, had := cs.lastCtr[iface]
		cs.lastCtr[iface] = [2]uint64{tx, rx}
		if !had || !tunUp || tx < prev[0] || rx < prev[1] {
			continue
		}
		cs.usage.Bytes[cs.keyOf(iface)] += (tx - prev[0]) + (rx - prev[1])
		cs.dirty = true
	}
}

func (a *Agent) saveUsage() {
	a.mu.Lock()
	if a.client == nil || !a.client.dirty {
		a.mu.Unlock()
		return
	}
	b, _ := json.Marshal(a.client.usage)
	a.client.dirty = false
	a.mu.Unlock()
	_ = writeFileAtomic(a.usagePath(), b, 0o600)
}

// reconcilePaths restarts mqvpn when the desired path set gains an interface
// or loses one by policy (failover tier change, data cap). Interfaces that
// vanish are left to mqvpn's own path handling. immediate skips the debounce.
func (a *Agent) reconcilePaths(immediate bool) {
	ifs, err := a.listIfs()
	if err != nil {
		return
	}
	if !a.sup.Active() {
		a.mu.Lock()
		a.client.remember(ifs)
		a.mu.Unlock()
		return
	}
	cfg := a.store.Get()
	present := map[string]bool{}
	for _, i := range ifs {
		present[i.Name] = true
	}
	a.mu.Lock()
	cs := a.client
	cs.remember(ifs)
	want := desiredPaths(cfg, ifs, cs.capReached, cs.captiveBlocked)
	active := cs.active
	change := false
	for _, w := range want {
		if !slices.Contains(active, w) {
			change = true
		}
	}
	for _, x := range active {
		if present[x] && !slices.Contains(want, x) {
			change = true
		}
	}
	if !change {
		cs.pending, cs.pendingN = nil, 0
		a.mu.Unlock()
		return
	}
	if !immediate {
		if slices.Equal(cs.pending, want) {
			cs.pendingN++
		} else {
			cs.pending, cs.pendingN = want, 1
		}
		if cs.pendingN < 2 {
			a.mu.Unlock()
			return
		}
	}
	cs.pending, cs.pendingN = nil, 0
	a.mu.Unlock()
	a.Logf("path set %v -> %v, restarting mqvpn", active, want)
	a.sup.Restart()
}

func (a *Agent) clientPreStart(ctx context.Context) error {
	cfg := a.store.Get()
	a.mu.Lock()
	p := a.client.payload
	a.mu.Unlock()
	if p == nil {
		return errors.New("not paired")
	}
	ifs, _ := a.listIfs()
	a.mu.Lock()
	a.client.remember(ifs)
	paths := desiredPaths(cfg, ifs, a.client.capReached, a.client.captiveBlocked)
	a.mu.Unlock()

	host, _, _ := net.SplitHostPort(p.Server)
	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	addrs, err := net.DefaultResolver.LookupHost(rctx, host)
	cancel()
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("resolve %s: %v", host, err)
	}
	serverIP := addrs[0]

	if p.Insecure {
		if err := tlsutil.CheckPin(ctx, p.Server, p.SNI, p.CertSHA256); err != nil {
			return err
		}
	}
	dns := p.DNS
	if cfg.Client.NoDNS {
		dns = nil
	}
	ini, err := mqconf.Client(mqconf.ClientParams{
		Server: p.Server, ServerName: p.SNI, Insecure: p.Insecure, Key: p.Key, DNS: dns,
		Paths: paths, TunName: config.DefaultTunName, Mode: cfg.Mode, ControlAddr: cfg.ControlAddr,
		ManageRoutes: cfg.Client.RoutesManaged(), LogLevel: cfg.LogLevel,
	})
	if err != nil {
		return err
	}
	if err := writeFileAtomic(a.confPath(), []byte(ini), 0o600); err != nil {
		return err
	}
	a.mu.Lock()
	a.client.active = paths
	a.client.serverIP = serverIP
	a.mu.Unlock()
	return nil
}

// pathMeta returns kind, priority and enabled for a device.
func (a *Agent) pathMeta(cfg *config.Config) status.PathMeta {
	a.mu.Lock()
	ifs := maps.Clone(a.client.ifs)
	a.mu.Unlock()
	return func(iface string) (string, string, bool) {
		it, known := ifs[iface]
		if !known {
			it = netif.Iface{Name: iface, Kind: netif.KindOf(iface)}
		}
		ps, ok := effectiveSetting(cfg, it)
		kind := it.Kind
		if ok && ps.Kind != "" {
			kind = ps.Kind
		}
		if !ok {
			return kind, config.PriorityPrimary, true
		}
		return kind, ps.Priority, ps.Enabled
	}
}

func (a *Agent) clientStatus(ctx context.Context) *status.Status {
	cfg := a.store.Get()
	a.mu.Lock()
	rt := a.rt
	logPaths := make(map[string]logPath, len(rt.logPaths))
	for k, v := range rt.logPaths {
		logPaths[k] = v
	}
	cs := a.client
	active := slices.Clone(cs.active)
	payload := cs.payload
	usage := make(map[string]uint64, len(cs.usage.Bytes))
	for k, v := range cs.usage.Bytes {
		usage[k] = v
	}
	a.mu.Unlock()

	st := &status.Status{Role: cfg.Role, Mode: cfg.Mode, Paths: []status.Path{}, Clients: []status.Client{}, Paired: payload != nil}
	if payload != nil {
		st.Server = payload.SNI
		if st.Server == "" || payload.Insecure {
			st.Server = payload.Server
		}
	}
	meta := a.pathMeta(cfg)
	now := time.Now()
	ready := rt.ready
	a.rater.Begin()
	running := a.sup.Info().State == supervisor.Running
	if running {
		if mst, err := a.ctl.Status(ctx); err == nil {
			st.ControlAPI = true
			var connected bool
			var up uint64
			st.Paths, st.Totals, up, connected = status.ClientFromControl(a.rater, now, mst, meta)
			ready = ready || connected
			if connected {
				st.UptimeSec = up
			}
		}
	}
	if !st.ControlAPI && running {
		// Unpatched mqvpn: totals from the tun device, per-path bytes from the
		// kernel counters of each bound interface, RTT from the 30 s log lines.
		if tx, rx, ok := netif.Counters(config.DefaultTunName); ok {
			st.Totals.TxBytes, st.Totals.RxBytes = tx, rx
			st.Totals.TxBps, st.Totals.RxBps, _ = a.rater.Rate("totals", now, tx, rx, 0, 0)
		}
		for i, iface := range active {
			kind, prio, en := meta(iface)
			p := status.Path{ID: uint64(i), Iface: iface, Kind: kind, Priority: prio, Enabled: en, State: status.PathPending}
			if ready {
				p.State = status.PathActive
			}
			if lp, ok := logPaths[iface]; ok && time.Since(lp.updated) < 90*time.Second {
				p.ID, p.SrttMs, p.State = lp.id, lp.srttMs, status.MapPathState(lp.state)
			}
			if tx, rx, ok := netif.Counters(iface); ok {
				p.TxBytes, p.RxBytes = tx, rx
				p.TxBps, p.RxBps, _ = a.rater.Rate("if/"+iface, now, tx, rx, 0, 0)
			}
			st.Paths = append(st.Paths, p)
		}
		if ready && !rt.readyAt.IsZero() {
			st.UptimeSec = uint64(time.Since(rt.readyAt).Seconds())
		}
	}
	a.rater.End()

	// Known WANs and configured paths that mqvpn is not using: standby (by
	// policy, or not connected yet) or failed (link gone or down).
	a.mu.Lock()
	ifs := maps.Clone(cs.ifs)
	a.mu.Unlock()
	seen := map[string]bool{}
	for _, p := range st.Paths {
		seen[p.Iface] = true
	}
	devs := slices.Sorted(maps.Keys(ifs))
	keyed := map[string]bool{}
	for _, dev := range devs {
		it := ifs[dev]
		if _, ok := effectiveSetting(cfg, it); ok {
			keyed[it.Key()], keyed[it.Name] = true, true
		} else {
			continue
		}
		if seen[dev] {
			continue
		}
		kind, prio, en := meta(dev)
		state := status.PathStandby
		if !it.Usable() {
			state = status.PathFailed
		}
		st.Paths = append(st.Paths, status.Path{ID: 1000 + uint64(len(st.Paths)), Iface: dev, Kind: kind, Priority: prio, Enabled: en, State: state})
		seen[dev] = true
	}
	for _, ps := range cfg.Client.Paths {
		if seen[ps.Iface] || keyed[ps.Iface] {
			continue
		}
		kind, prio, en := meta(ps.Iface)
		st.Paths = append(st.Paths, status.Path{ID: 1000 + uint64(len(st.Paths)), Iface: ps.Iface, Kind: kind, Priority: prio, Enabled: en, State: status.PathFailed})
	}
	for i := range st.Paths {
		p := &st.Paths[i]
		it, ok := ifs[p.Iface]
		if !ok {
			it = netif.Iface{Name: p.Iface}
		}
		p.Network, p.Label = it.Network, it.Label
		if ps, ok := effectiveSetting(cfg, it); ok {
			p.DataCapMB = ps.DataCapMB
			p.UsedBytes = usage[it.Key()]
			p.CapReached = ps.DataCapMB > 0 && p.UsedBytes >= uint64(ps.DataCapMB)<<20
		}
	}
	a.annotateCaptive(st.Paths)
	if running {
		a.markFailedPaths(st.Paths)
	}
	st.State, st.Error = a.supState(ready)
	if payload == nil {
		st.State = status.StateDisconnected
	}
	st.TunnelIP = rt.tunnelIP
	if st.State != status.StateConnected {
		st.Totals.TxBps, st.Totals.RxBps = 0, 0
	}
	st.Tailscale = a.tsStatus(ctx)
	if j := a.joinView(); j.State != joinIdle {
		st.Join = j
	}
	return st
}

// importPairing stores a pairing URI and (re)connects.
func (a *Agent) importPairing(uri string) (*pairing.Payload, error) {
	p, err := pairing.Decode(uri)
	if err != nil {
		return nil, err
	}
	ifs, _ := a.listIfs()
	if _, err := a.store.Update(func(c *config.Config) error {
		c.Client.PairingURI = uri
		if p.Mode != "" && config.ValidMode(p.Mode) {
			c.Mode = p.Mode
		}
		c.Client.Disconnected = false
		// First pairing: offer every usable uplink as a primary path (on
		// OpenWrt the WAN zone is used without configuration).
		if !anyPresent(c.Client.Paths, ifs) && !anyWAN(ifs) {
			c.Client.Paths = nil
			for _, i := range ifs {
				if i.Usable() {
					c.Client.Paths = append(c.Client.Paths, config.PathSetting{Iface: i.Name, Enabled: true, Priority: config.PriorityPrimary})
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.client.payload = p
	a.userStop = false
	a.mu.Unlock()
	a.Logf("paired with %s as %s", p.Server, p.User)
	a.sup.Restart()
	return p, nil
}

func anyWAN(ifs []netif.Iface) bool {
	for _, i := range ifs {
		if i.WAN {
			return true
		}
	}
	return false
}

func anyPresent(ps []config.PathSetting, ifs []netif.Iface) bool {
	for _, p := range ps {
		for _, i := range ifs {
			if i.Name == p.Iface || (i.Network != "" && i.Network == p.Iface) {
				return true
			}
		}
	}
	return false
}

// pathView is one entry of GET /api/paths.
type pathView struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Up         bool     `json:"up"`
	Carrier    bool     `json:"carrier"`
	Addrs      []string `json:"addrs"`
	Present    bool     `json:"present"`
	Enabled    bool     `json:"enabled"`
	Priority   string   `json:"priority"`
	DataCapMB  int64    `json:"data_cap_mb"`
	UsedBytes  uint64   `json:"used_bytes"`
	InUse      bool     `json:"in_use"`
	Configured bool     `json:"configured"`
	// OpenWrt: settings are keyed by network; send it as "name" to update.
	Network string `json:"network,omitempty"`
	Label   string `json:"label,omitempty"`
	WAN     bool   `json:"wan,omitempty"`
}

func (a *Agent) listPaths() ([]pathView, error) {
	cfg := a.store.Get()
	ifs, err := a.listIfs()
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.client.remember(ifs)
	active := slices.Clone(a.client.active)
	usage := maps.Clone(a.client.usage.Bytes)
	a.mu.Unlock()
	out := []pathView{}
	seen := map[string]bool{}
	for _, i := range ifs {
		v := pathView{Name: i.Name, Kind: i.Kind, Up: i.Up, Carrier: i.Carrier, Addrs: i.Addrs, Present: true,
			Priority: config.PriorityPrimary, Network: i.Network, Label: i.Label, WAN: i.WAN}
		_, configured := settingFor(cfg, i)
		if eff, ok := effectiveSetting(cfg, i); ok {
			v.Enabled, v.Priority, v.DataCapMB = eff.Enabled, eff.Priority, eff.DataCapMB
			if eff.Kind != "" {
				v.Kind = eff.Kind
			}
		}
		v.Configured = configured
		v.UsedBytes, v.InUse = usage[i.Key()], slices.Contains(active, i.Name)
		seen[i.Name], seen[i.Key()] = true, true
		out = append(out, v)
	}
	for _, ps := range cfg.Client.Paths {
		if seen[ps.Iface] {
			continue
		}
		out = append(out, pathView{Name: ps.Iface, Kind: netif.KindOf(ps.Iface), Addrs: []string{}, Configured: true,
			Enabled: ps.Enabled, Priority: ps.Priority, DataCapMB: ps.DataCapMB, UsedBytes: usage[ps.Iface]})
	}
	return out, nil
}

type pathUpdate struct {
	Name       string  `json:"name"`
	Enabled    *bool   `json:"enabled"`
	Priority   *string `json:"priority"`
	DataCapMB  *int64  `json:"data_cap_mb"`
	Kind       *string `json:"kind"`
	ResetUsage bool    `json:"reset_usage"`
}

func (a *Agent) updatePath(u pathUpdate) error {
	if u.Name == "" {
		return errors.New("name required")
	}
	if u.Priority != nil && !config.ValidPriority(*u.Priority) {
		return fmt.Errorf("bad priority %q", *u.Priority)
	}
	if u.DataCapMB != nil && *u.DataCapMB < 0 {
		return errors.New("data_cap_mb must be >= 0")
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		idx := -1
		for i, p := range c.Client.Paths {
			if p.Iface == u.Name {
				idx = i
			}
		}
		if idx < 0 {
			c.Client.Paths = append(c.Client.Paths, config.PathSetting{Iface: u.Name, Enabled: true, Priority: config.PriorityPrimary})
			idx = len(c.Client.Paths) - 1
		}
		p := &c.Client.Paths[idx]
		if u.Enabled != nil {
			p.Enabled = *u.Enabled
		}
		if u.Priority != nil {
			p.Priority = *u.Priority
		}
		if u.DataCapMB != nil {
			p.DataCapMB = *u.DataCapMB
		}
		if u.Kind != nil {
			p.Kind = *u.Kind
		}
		return nil
	}); err != nil {
		return err
	}
	if u.ResetUsage {
		a.mu.Lock()
		delete(a.client.usage.Bytes, u.Name)
		delete(a.client.usage.Bytes, a.client.keyOf(u.Name))
		a.client.dirty = true
		a.mu.Unlock()
		a.saveUsage()
	}
	a.reconcilePaths(true)
	return nil
}

func (a *Agent) connect() error {
	if !a.paired() {
		return errors.New("not paired: import a pairing URI first")
	}
	if _, err := a.store.Update(func(c *config.Config) error { c.Client.Disconnected = false; return nil }); err != nil {
		return err
	}
	a.mu.Lock()
	a.userStop = false
	a.mu.Unlock()
	a.sup.Start()
	return nil
}

func (a *Agent) disconnect() error {
	if _, err := a.store.Update(func(c *config.Config) error { c.Client.Disconnected = true; return nil }); err != nil {
		return err
	}
	a.mu.Lock()
	a.userStop = true
	a.mu.Unlock()
	a.sup.Stop()
	return nil
}

// hUnpair forgets the pairing (DELETE /api/pair, additive): the tunnel
// stops and the LAN goes back to the WANs.
func (a *Agent) hUnpair(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		c.Client.PairingURI, c.Client.Disconnected = "", false
		return nil
	}); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	a.mu.Lock()
	a.client.payload = nil
	a.userStop = true
	if a.join.cancel != nil {
		a.join.cancel()
	}
	a.join = &joinState{State: joinIdle}
	a.mu.Unlock()
	a.sup.Stop()
	a.Logf("pairing removed")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
