package agent

// Pair & Share (docs/pair-and-share.md).
//
// Server role: the share key (state dir share.key; in every new pairing,
// and from GET /api/shares for older ones) and a short-lived registry of
// sharers (POST /api/shares), so hubs find sharers mDNS can't reach.
//
// Client role, as a hub: nearby sharers (mDNS _tailbond-share._udp, or the
// server's registry when they're on one of our subnets) each get a
// share.Link; a link is one more path, "shNNNNNN", that mqvpn reaches as a
// relay path (patches/mqvpn/0002: Path = shNNNNNN@127.0.0.1:port). Settings
// (Always / Backup only / Off, data cap) and usage work as for any WAN.
//
// Client role, as a sharer: client.share lends one of this machine's
// uplinks (an LTE dongle) to hubs on its LAN.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/netif"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/share"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

const (
	shareRegTTL     = 3 * time.Minute // registry entries without a refresh
	shareLinkTTL    = 2 * time.Minute // links to sharers no longer found
	shareRound      = 10 * time.Second
	shareRegEvery   = time.Minute
	shareRecoverAft = 20 * time.Second // ready, yet mqvpn's path isn't: restart
	// IsolationNote is what the hub says when the Wi-Fi keeps devices apart.
	IsolationNote = "This Wi-Fi blocks devices from talking to each other; sharing isn't available here."
)

// ---- server role ----

type regSharer struct {
	ID     string    `json:"id"`
	Label  string    `json:"label"`
	KeyID  string    `json:"key_id"`
	Client string    `json:"client"`
	LAN    []string  `json:"lan"`
	Port   int       `json:"port"`
	Kind   string    `json:"kind,omitempty"`
	Seen   time.Time `json:"seen"`
}

type shareRegistry struct {
	mu sync.Mutex
	m  map[string]regSharer
}

// serverShareKey is this server's share key (created once).
func (a *Agent) serverShareKey() string {
	k, err := a.stateSecret("share.key")
	if err != nil {
		a.Logf("share key: %v", err)
		return ""
	}
	return k
}

type sharesView struct {
	Key     string      `json:"key,omitempty"`
	KeyID   string      `json:"key_id,omitempty"`
	Sharers []regSharer `json:"sharers"`
	// client role
	Shares  []status.Share  `json:"shares,omitempty"`
	Note    string          `json:"note,omitempty"`
	Sharing *status.Sharing `json:"sharing,omitempty"`
}

// hShares: server — the share key (to paired clients) and the sharers
// registered in the last few minutes; client — this hub's shares.
func (a *Agent) hShares(w http.ResponseWriter, r *http.Request) {
	if a.role == config.RoleClient {
		st := &status.Status{}
		a.shareStatus(st)
		writeJSON(w, http.StatusOK, sharesView{Sharers: []regSharer{}, Shares: st.Shares, Note: st.ShareNote, Sharing: st.Sharing})
		return
	}
	v := sharesView{Sharers: []regSharer{}}
	if a.clientForToken(r) != "" || a.authorized(r) {
		if k := a.serverShareKey(); k != "" {
			v.Key, v.KeyID = k, share.KeyID([]byte(k))
		}
	}
	a.shareReg.mu.Lock()
	now := time.Now()
	for id, s := range a.shareReg.m {
		if now.Sub(s.Seen) > shareRegTTL {
			delete(a.shareReg.m, id)
			continue
		}
		v.Sharers = append(v.Sharers, s)
	}
	a.shareReg.mu.Unlock()
	sort.Slice(v.Sharers, func(i, j int) bool { return v.Sharers[i].ID < v.Sharers[j].ID })
	writeJSON(w, http.StatusOK, v)
}

// hShareRegister: a sharer (with its client token) says where it is.
func (a *Agent) hShareRegister(w http.ResponseWriter, r *http.Request) {
	client := a.clientForToken(r)
	if client == "" || a.role != config.RoleServer {
		httpErr(w, http.StatusUnauthorized, errors.New("client token required"))
		return
	}
	var in regSharer
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	in.ID, in.Label = strings.TrimSpace(in.ID), strings.TrimSpace(in.Label)
	if in.ID == "" || len(in.ID) > share.MaxIDLen || len(in.Label) > share.MaxIDLen || in.Port < 1 || in.Port > 65535 || len(in.LAN) > 8 {
		httpErr(w, http.StatusBadRequest, errors.New("id, label (≤64), port and up to 8 LAN addresses"))
		return
	}
	var lan []string
	for _, s := range in.LAN {
		if ip, err := netip.ParseAddr(s); err == nil && ip.Is4() && (ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
			lan = append(lan, ip.String())
		}
	}
	in.LAN, in.Client, in.Seen = lan, client, time.Now()
	a.shareReg.mu.Lock()
	if len(a.shareReg.m) >= 64 && a.shareReg.m[in.ID].ID == "" {
		a.shareReg.mu.Unlock()
		httpErr(w, http.StatusTooManyRequests, errors.New("too many sharers"))
		return
	}
	a.shareReg.m[in.ID] = in
	a.shareReg.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- client role ----

type hubLink struct {
	link     *share.Link
	cancel   context.CancelFunc
	found    share.Found
	path     string
	seen     time.Time
	mdns     bool // found over mDNS at least once
	server   bool // listed by the home server
	ready    bool
	readyAt  time.Time
	restarts time.Time // last mqvpn restart for this link
}

type shareState struct {
	links map[string]*hubLink // by sharer id
	note  string

	sharer       *share.Sharer
	sharerCancel context.CancelFunc
	sharerSig    string
	sharerErr    string
	sharerID     string
	sharerLabel  string
	lastReg      time.Time
}

func newShareState() *shareState { return &shareState{links: map[string]*hubLink{}} }

// sharePathName is the path name for a sharer: short, stable, a valid
// interface-like label.
func sharePathName(id string) string {
	h := sha256.Sum256([]byte(id))
	return "sh" + hex.EncodeToString(h[:3])
}

// clientShareKey: the pairing's, else the one cached from the server.
func (a *Agent) clientShareKey(p *pairing.Payload) []byte {
	if p != nil && p.ShareKey != "" {
		return []byte(p.ShareKey)
	}
	if k := a.store.Get().Client.ShareKey; k != "" {
		return []byte(k)
	}
	return nil
}

// serverAPI calls the active server's API through the tunnel with this
// client's token.
func (a *Agent) serverAPI(ctx context.Context, method, path string, body any) ([]byte, error) {
	a.mu.Lock()
	p := a.client.payload
	ready := a.rt.ready
	a.mu.Unlock()
	if p == nil || !ready {
		return nil, errors.New("not connected")
	}
	host := p.TunnelSubnet().Addr().Next().String()
	if len(p.DNS) > 0 {
		host = p.DNS[0]
	}
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, method, "http://"+net.JoinHostPort(host, a.apiPort)+path, rd)
	req.Header.Set("Authorization", "Bearer "+APIToken(p.Key))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home server: HTTP %d", resp.StatusCode)
	}
	return b, nil
}

// shareWatcher runs the hub and sharer sides.
func (a *Agent) shareWatcher(ctx context.Context) {
	t := time.NewTimer(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			a.mu.Lock()
			ss := a.client.share
			for _, l := range ss.links {
				l.cancel()
			}
			if ss.sharerCancel != nil {
				ss.sharerCancel()
			}
			a.mu.Unlock()
			return
		case <-t.C:
		}
		a.shareRound(ctx)
		t.Reset(shareRound)
	}
}

func (a *Agent) shareRound(ctx context.Context) {
	cfg := a.store.Get()
	a.mu.Lock()
	p := a.client.payload
	ready := a.rt.ready
	a.mu.Unlock()

	var reg *sharesView
	if ready {
		if b, err := a.serverAPI(ctx, http.MethodGet, "/api/shares", nil); err == nil {
			var v sharesView
			if json.Unmarshal(b, &v) == nil {
				reg = &v
				// cache the key for pairings made before share keys existed
				if v.Key != "" && (p == nil || p.ShareKey == "") && cfg.Client.ShareKey != v.Key {
					_, _ = a.store.Update(func(c *config.Config) error { c.Client.ShareKey = v.Key; return nil })
				}
			}
		}
	}
	key := a.clientShareKey(p)
	a.runSharer(ctx, cfg, p, key, ready)
	changed := a.runHub(ctx, cfg, p, key, reg)
	if changed {
		select {
		case a.rescan <- struct{}{}:
		default:
		}
	}
}

// ---- hub ----

// runHub finds sharers and keeps a link to each; true when a link's
// readiness changed (the path set may change).
func (a *Agent) runHub(ctx context.Context, cfg *config.Config, p *pairing.Payload, key []byte, reg *sharesView) bool {
	a.mu.Lock()
	ss := a.client.share
	own := ss.sharerID
	a.mu.Unlock()
	if !cfg.Client.SharesUsed() || p == nil || len(key) == 0 {
		a.stopLinks(func(*hubLink) bool { return true })
		a.mu.Lock()
		ss.note = ""
		a.mu.Unlock()
		return false
	}
	kid := share.KeyID(key)
	skip := []string{config.DefaultTunName}
	if cfg.Client.Share != nil && cfg.Client.Share.Enabled {
		skip = append(skip, cfg.Client.Share.Iface)
	}
	ifaces := share.MulticastIfaces(skip...)
	var found []share.Found
	if len(ifaces) > 0 {
		if f, err := share.Browse(ctx, ifaces, 1500*time.Millisecond); err == nil {
			found = f
		}
	}
	// the server's list: sharers on one of our subnets
	subnets := localSubnets(ifaces)
	fromServer := map[string]bool{}
	if reg != nil {
		for _, r := range reg.Sharers {
			if r.Client == p.User || r.KeyID != kid {
				continue
			}
			for _, ip := range r.LAN {
				if a, err := netip.ParseAddr(ip); err == nil && inAny(subnets, a) {
					found = append(found, share.Found{ID: r.ID, Label: r.Label, KeyID: r.KeyID, Addr: &net.UDPAddr{IP: net.ParseIP(ip), Port: r.Port}, Via: "server"})
					fromServer[r.ID] = true
					break
				}
			}
		}
	}
	now := time.Now()
	changed := false
	a.mu.Lock()
	for _, f := range found {
		if f.KeyID != kid || f.ID == own || f.Addr == nil {
			continue
		}
		hl := ss.links[f.ID]
		if hl != nil && hl.found.Addr.String() != f.Addr.String() && (f.Via == "mdns" || !hl.mdns) {
			// moved: new link, same local port so mqvpn's path stays valid
			port := hl.link.Local().Port
			hl.cancel()
			delete(ss.links, f.ID)
			hl = a.newHubLinkLocked(ss, f, p.User, key, port)
		} else if hl == nil {
			hl = a.newHubLinkLocked(ss, f, p.User, key, 0)
			changed = true
		}
		if hl == nil {
			continue
		}
		hl.seen = now
		if f.Via == "mdns" {
			hl.mdns = true
		} else {
			hl.server = true
		}
		if f.Label != "" && hl.found.Label == "" {
			hl.found.Label = f.Label
		}
	}
	note := ""
	for id, hl := range ss.links {
		st := hl.link.Status()
		ready := st.State == share.StateReady
		if ready != hl.ready {
			hl.ready, changed = ready, true
			if ready {
				hl.readyAt = now
			}
		}
		if !ready && now.Sub(hl.seen) > shareLinkTTL {
			hl.cancel()
			delete(ss.links, id)
			changed = true
			continue
		}
		// known from the server, on our subnet, but no answer and never
		// heard over mDNS: the Wi-Fi keeps devices apart
		if hl.server && !hl.mdns && st.State == share.StateUnreachable {
			note = IsolationNote
		}
	}
	ss.note = note
	// ready for a while, yet mqvpn's relay path for it isn't active (it gave
	// up while the sharer was away): restart mqvpn once per 2 minutes
	var restart bool
	if snap := a.snap; snap != nil {
		state := map[string]string{}
		for _, sp := range snap.Paths {
			state[sp.Iface] = sp.State
		}
		for _, hl := range ss.links {
			if hl.ready && now.Sub(hl.readyAt) > shareRecoverAft && slices.Contains(a.client.active, hl.path) &&
				state[hl.path] != status.PathActive && now.Sub(hl.restarts) > 2*time.Minute {
				hl.restarts, restart = now, true
			}
		}
	}
	a.mu.Unlock()
	if restart && a.sup.Active() {
		a.Logf("share: a shared connection is back, restarting mqvpn to use it again")
		a.sup.Restart()
	}
	return changed
}

// newHubLinkLocked starts a link. Caller holds a.mu.
func (a *Agent) newHubLinkLocked(ss *shareState, f share.Found, hubID string, key []byte, localPort int) *hubLink {
	onChange := func(st share.LinkStatus) {
		// ready <-> not ready changes the path set: re-check now, not at
		// the next round (a dead relay path loses most of what it's given)
		go a.reconcilePaths(true)
	}
	l, err := share.NewLink(share.LinkConfig{Sharer: f.Addr, Key: key, HubID: hubID, LocalPort: localPort, Logf: a.Logf, OnChange: onChange})
	if err != nil {
		a.Logf("share: link to %s: %v", f.Addr, err)
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	go l.Run(ctx)
	hl := &hubLink{link: l, cancel: cancel, found: f, path: sharePathName(f.ID)}
	ss.links[f.ID] = hl
	a.Logf("share: found %q at %s (%s)", f.Label, f.Addr, f.Via)
	return hl
}

func (a *Agent) stopLinks(which func(*hubLink) bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, hl := range a.client.share.links {
		if which(hl) {
			hl.cancel()
			delete(a.client.share.links, id)
		}
	}
}

func localSubnets(ifaces []net.Interface) []netip.Prefix {
	var out []netip.Prefix
	for _, ifi := range ifaces {
		addrs, _ := ifi.Addrs()
		for _, ad := range addrs {
			if n, ok := ad.(*net.IPNet); ok && n.IP.To4() != nil {
				if pf, err := netip.ParsePrefix(n.String()); err == nil {
					out = append(out, pf.Masked())
				}
			}
		}
	}
	return out
}

func inAny(ps []netip.Prefix, a netip.Addr) bool {
	for _, p := range ps {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// shareIfaces are the shares as interfaces for path selection: usable
// while their session is open.
func (a *Agent) shareIfaces() []netif.Iface {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil || a.client.share == nil {
		return nil
	}
	var out []netif.Iface
	for _, hl := range a.client.share.links {
		st := hl.link.Status()
		it := netif.Iface{Name: hl.path, Kind: netif.KindShared, Label: shareLabel(hl, st), WAN: true}
		if st.State == share.StateReady {
			it.Up, it.Carrier, it.Addrs = true, true, []string{"127.0.0.1/8"}
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func shareLabel(hl *hubLink, st share.LinkStatus) string {
	l := st.Label
	if l == "" {
		l = hl.found.Label
	}
	if l == "" {
		l = "Nearby phone"
	}
	return l + " (shared)"
}

// allIfs is the discovered interfaces plus the shares.
func (a *Agent) allIfs() ([]netif.Iface, error) {
	ifs, err := a.listIfs()
	return append(ifs, a.shareIfaces()...), err
}

// pathSpecs turns path names into mqvpn Path values: a share becomes its
// relay path, NAME@127.0.0.1:PORT.
func (a *Agent) pathSpecs(paths []string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	byPath := map[string]*hubLink{}
	if a.client.share != nil {
		for _, hl := range a.client.share.links {
			byPath[hl.path] = hl
		}
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if hl := byPath[p]; hl != nil {
			out = append(out, p+"@"+hl.link.Local().String())
		} else {
			out = append(out, p)
		}
	}
	return out
}

// pathCounters: a share's payload counters, else the interface's.
func (a *Agent) pathCounters(iface string) (tx, rx uint64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pathCountersLocked(iface)
}

// pathCountersLocked is pathCounters for a caller holding a.mu.
func (a *Agent) pathCountersLocked(iface string) (tx, rx uint64, ok bool) {
	if strings.HasPrefix(iface, "sh") && a.client != nil && a.client.share != nil {
		for _, hl := range a.client.share.links {
			if hl.path == iface {
				st := hl.link.Status()
				return st.Up, st.Down, true
			}
		}
	}
	return netif.Counters(iface)
}

// shareStatus fills the Pair & Share part of the status.
func (a *Agent) shareStatus(st *status.Status) {
	cfg := a.store.Get()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil || a.client.share == nil {
		return
	}
	ss := a.client.share
	ids := slices.Sorted(maps.Keys(ss.links))
	for _, id := range ids {
		hl := ss.links[id]
		ls := hl.link.Status()
		via := hl.found.Via
		if hl.mdns {
			via = "mdns"
		}
		s := status.Share{
			ID: id, Label: shareLabel(hl, ls), Path: hl.path, Addr: ls.Addr, Via: via, State: ls.State, Reason: ls.Reason,
			UpBytes: ls.Up, DnBytes: ls.Down, Battery: ls.Battery, Charging: ls.Charging,
		}
		if ls.CapLeft != share.NoCap {
			c := ls.CapLeft
			s.CapLeft = &c
		}
		st.Shares = append(st.Shares, s)
	}
	st.ShareNote = ss.note
	use := cfg.Client.SharesUsed()
	st.UseShares = &use
	if sc := cfg.Client.Share; sc != nil {
		sh := &status.Sharing{Enabled: sc.Enabled, Iface: sc.Iface, Label: ss.sharerLabel, Port: sharePort(sc), CapMB: sc.CapMB, Hubs: []status.SharingHub{}, State: "off"}
		switch {
		case !sc.Enabled:
		case ss.sharerErr != "":
			sh.State, sh.Error = "error", ss.sharerErr
		case ss.sharer != nil:
			sh.State = "sharing"
			sh.Used = ss.sharer.Used()
			for _, h := range ss.sharer.Sessions() {
				sh.Hubs = append(sh.Hubs, status.SharingHub{Hub: h.HubID, Addr: h.Addr, UpBytes: h.Up, DnBytes: h.Down})
			}
		}
		st.Sharing = sh
	}
}

// ---- sharer (Linux) ----

func sharePort(sc *config.ShareConfig) int {
	if sc.Port > 0 {
		return sc.Port
	}
	return share.DefaultPort
}

// runSharer starts, restarts or stops this machine's own share.
func (a *Agent) runSharer(ctx context.Context, cfg *config.Config, p *pairing.Payload, key []byte, ready bool) {
	sc := cfg.Client.Share
	a.mu.Lock()
	ss := a.client.share
	a.mu.Unlock()
	want := sc != nil && sc.Enabled && sc.Iface != "" && p != nil && len(key) > 0
	sig := ""
	if want {
		sig = strings.Join([]string{sc.Iface, fmt.Sprint(sharePort(sc)), sc.Label, fmt.Sprint(sc.CapMB), strings.Join(sc.AllowedHubs, ","), string(key), p.Server, time.Now().Format("2006-01")}, "|")
	}
	a.mu.Lock()
	if ss.sharerSig != sig {
		if ss.sharerCancel != nil {
			ss.sharerCancel()
		}
		ss.sharer, ss.sharerCancel, ss.sharerSig, ss.sharerErr = nil, nil, "", ""
	}
	running := ss.sharer != nil
	a.mu.Unlock()
	if sc != nil && sc.Enabled && len(key) == 0 {
		a.mu.Lock()
		ss.sharerErr = "waiting for the home server's share key (connect once)"
		a.mu.Unlock()
	}
	if !want {
		return
	}
	if !running {
		a.startSharer(ctx, cfg, sc, p, key, sig)
	}
	a.mu.Lock()
	sh, last := ss.sharer, ss.lastReg
	a.mu.Unlock()
	if sh != nil && ready && time.Since(last) > shareRegEvery {
		ifaces := share.MulticastIfaces(config.DefaultTunName, sc.Iface)
		var lan []string
		for _, ifi := range ifaces {
			addrs, _ := ifi.Addrs()
			for _, ad := range addrs {
				if n, ok := ad.(*net.IPNet); ok && n.IP.To4() != nil {
					lan = append(lan, n.IP.String())
				}
			}
		}
		a.mu.Lock()
		body := regSharer{ID: ss.sharerID, Label: ss.sharerLabel, KeyID: share.KeyID(key), LAN: lan, Port: sharePort(sc), Kind: "linux"}
		a.mu.Unlock()
		if _, err := a.serverAPI(ctx, http.MethodPost, "/api/shares", body); err == nil {
			a.mu.Lock()
			ss.lastReg = time.Now()
			a.mu.Unlock()
		}
	}
}

func (a *Agent) startSharer(ctx context.Context, cfg *config.Config, sc *config.ShareConfig, p *pairing.Payload, key []byte, sig string) {
	host, _ := os.Hostname()
	id := "lx-" + hex.EncodeToString(func() []byte { h := sha256.Sum256([]byte(host + "/" + p.User + "/" + sc.Iface)); return h[:5] }())
	label := sc.Label
	if label == "" {
		label = host + "'s " + sc.Iface
	}
	a.mu.Lock()
	cs := a.client
	usedKey := "shared:" + sc.Iface
	used := cs.usage.Bytes[usedKey]
	a.mu.Unlock()
	iface := sc.Iface
	sh, err := share.NewSharer(share.SharerConfig{
		ID: id, Label: label, Key: key, Server: p.Server,
		CapBytes: uint64(max(sc.CapMB, 0)) << 20, UsedBytes: used, AllowedHubs: sc.AllowedHubs,
		Dial: func(ctx context.Context, server string) (net.Conn, error) {
			d := net.Dialer{Control: func(network, address string, c syscall.RawConn) error {
				var serr error
				if err := c.Control(func(fd uintptr) { serr = syscall.BindToDevice(int(fd), iface) }); err != nil {
					return err
				}
				return serr
			}}
			return d.DialContext(ctx, "udp", server)
		},
		Logf: a.Logf,
	})
	if err != nil {
		a.mu.Lock()
		cs.share.sharerErr = err.Error()
		a.mu.Unlock()
		return
	}
	conn, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", sharePort(sc)))
	if err != nil {
		a.mu.Lock()
		cs.share.sharerErr = err.Error()
		a.mu.Unlock()
		return
	}
	sctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	cs.share.sharer, cs.share.sharerCancel, cs.share.sharerSig, cs.share.sharerErr = sh, cancel, sig, ""
	cs.share.sharerID, cs.share.sharerLabel, cs.share.lastReg = id, label, time.Time{}
	a.mu.Unlock()
	go func() { _ = sh.Serve(sctx, conn) }()
	go func() {
		adv := share.Advert{ID: id, Label: label, KeyID: share.KeyID(key), Port: sharePort(sc)}
		for sctx.Err() == nil {
			if err := share.Advertise(sctx, share.MulticastIfaces(config.DefaultTunName, iface), adv); err != nil {
				a.Logf("share: mDNS: %v", err)
			}
			select {
			case <-sctx.Done():
			case <-time.After(10 * time.Second):
			}
		}
	}()
	// usage: what's shared counts against the month, like a WAN's own traffic
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-sctx.Done():
				return
			case <-t.C:
			}
			a.mu.Lock()
			if cs.usage.Bytes != nil {
				cs.usage.Bytes[usedKey] = sh.Used()
				cs.dirty = true
			}
			a.mu.Unlock()
		}
	}()
	a.Logf("share: sharing %s as %q on UDP %d", iface, label, sharePort(sc))
}

// hShareSettings: client role — POST {"use_shares":bool} and/or
// {"share":{enabled, iface, label, cap_mb, allowed_hubs}}.
func (a *Agent) hShareSettings(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	var in struct {
		UseShares *bool               `json:"use_shares"`
		Share     *config.ShareConfig `json:"share"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if in.Share != nil {
		if in.Share.Enabled && in.Share.Iface == "" {
			httpErr(w, http.StatusBadRequest, errors.New("pick the connection to share"))
			return
		}
		if in.Share.CapMB < 0 || len(in.Share.Label) > share.MaxIDLen || in.Share.Port < 0 || in.Share.Port > 65535 {
			httpErr(w, http.StatusBadRequest, errors.New("bad share settings"))
			return
		}
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		if in.UseShares != nil {
			v := *in.UseShares
			c.Client.UseShares = &v
		}
		if in.Share != nil {
			s := *in.Share
			c.Client.Share = &s
		}
		return nil
	}); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
