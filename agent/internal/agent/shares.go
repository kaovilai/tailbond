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
	"path/filepath"
	goruntime "runtime"
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
	"github.com/kaovilai/tailbond/agent/internal/sockbind"
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

// shareSecret is the server's share secret (state dir share.key, random,
// created once, in backups). Devices' share keys are derived from it.
func (a *Agent) shareSecret() []byte {
	k, err := a.stateSecret("share.key")
	if err != nil {
		a.Logf("share key: %v", err)
		return nil
	}
	return []byte(k)
}

// shareKeyFor is device u's share key ("" without a secret).
func (a *Agent) shareKeyFor(u config.User) string {
	sec := a.shareSecret()
	if sec == nil || u.Key == "" {
		return ""
	}
	return share.ClientKey(sec, u.Name, u.Key)
}

// passExpiry: passes run to the end of tomorrow (UTC), so a hub's pass stays
// the same all day and lasts at least a day without the server; a removed
// device's last pass runs out within two.
func passExpiry(now time.Time) uint64 {
	y, m, d := now.UTC().Date()
	return uint64(time.Date(y, m, d+2, 0, 0, 0, 0, time.UTC).Unix())
}

// sharePass is a hub's pass for one sharer (by its key id).
type sharePass struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"` // hex
	Exp   uint64 `json:"exp"`
}

type sharesView struct {
	// server, to a paired device: its own share key, and its passes for
	// every other device's shares
	Key     string      `json:"key,omitempty"`
	KeyID   string      `json:"key_id,omitempty"`
	Passes  []sharePass `json:"passes,omitempty"`
	Sharers []regSharer `json:"sharers"`
	// client role
	Shares  []status.Share  `json:"shares,omitempty"`
	Note    string          `json:"note,omitempty"`
	Sharing *status.Sharing `json:"sharing,omitempty"`
}

// hShares: server — to a paired device (client token) its share key and
// passes, and the sharers registered in the last few minutes; client —
// this hub's shares.
func (a *Agent) hShares(w http.ResponseWriter, r *http.Request) {
	if a.role == config.RoleClient {
		st := &status.Status{}
		a.shareStatus(st)
		writeJSON(w, http.StatusOK, sharesView{Sharers: []regSharer{}, Shares: st.Shares, Note: st.ShareNote, Sharing: st.Sharing})
		return
	}
	v := sharesView{Sharers: []regSharer{}}
	users := a.store.Get().Server.Users
	if c := a.clientForToken(r); c != "" {
		if sec := a.shareSecret(); sec != nil {
			exp := passExpiry(time.Now())
			for _, u := range users {
				if u.Key == "" {
					continue
				}
				k := share.ClientKey(sec, u.Name, u.Key)
				if u.Name == c {
					v.Key, v.KeyID = k, share.KeyID([]byte(k))
					continue
				}
				v.Passes = append(v.Passes, sharePass{KeyID: share.KeyID([]byte(k)), Key: hex.EncodeToString(share.PassKey([]byte(k), c, exp)), Exp: exp})
			}
		}
	}
	paired := map[string]bool{}
	for _, u := range users {
		paired[u.Name] = true
	}
	a.shareReg.mu.Lock()
	now := time.Now()
	for id, s := range a.shareReg.m {
		if now.Sub(s.Seen) > shareRegTTL || !paired[s.Client] {
			delete(a.shareReg.m, id)
			continue
		}
		v.Sharers = append(v.Sharers, s)
	}
	a.shareReg.mu.Unlock()
	sort.Slice(v.Sharers, func(i, j int) bool { return v.Sharers[i].ID < v.Sharers[j].ID })
	writeJSON(w, http.StatusOK, v)
}

// dropSharers forgets a removed device's registered shares.
func (a *Agent) dropSharers(client string) {
	a.shareReg.mu.Lock()
	defer a.shareReg.mu.Unlock()
	for id, s := range a.shareReg.m {
		if s.Client == client {
			delete(a.shareReg.m, id)
		}
	}
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
	// the key id is the server's to say: hubs pick the pass by it
	for _, u := range a.store.Get().Server.Users {
		if u.Name == client {
			in.KeyID = share.KeyID([]byte(a.shareKeyFor(u)))
		}
	}
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
	kick  chan struct{} // run a round now (settings changed)

	sharer       *share.Sharer
	sharerCancel context.CancelFunc
	sharerSig    string
	sharerErr    string
	sharerID     string
	sharerLabel  string
	sharerNote   string
	lastReg      time.Time
	// the server's list from the last round (phone hubs use it too)
	reg *sharesView

	// what the home server said, per pairing (share-cache.json)
	cache       map[string]shareCacheEntry
	cacheLoaded bool
}

// shareCacheEntry is one pairing's share key (for pairings made before
// pairings carried it) and the hub's passes, kept so sharing still works
// while the server can't be reached (a share may be the only way out).
type shareCacheEntry struct {
	Key    string      `json:"key,omitempty"`
	Passes []sharePass `json:"passes,omitempty"`
}

func newShareState() *shareState {
	return &shareState{links: map[string]*hubLink{}, kick: make(chan struct{}, 1)}
}

// kickShares runs the next share round now, not in up to 10 s.
func (a *Agent) kickShares() {
	a.mu.Lock()
	ss := a.client.share
	a.mu.Unlock()
	if ss != nil {
		select {
		case ss.kick <- struct{}{}:
		default:
		}
	}
}

// sharePathName is the path name for a sharer: short, stable, a valid
// interface-like label.
func sharePathName(id string) string {
	h := sha256.Sum256([]byte(id))
	return "sh" + hex.EncodeToString(h[:3])
}

// pairingID names a pairing in the share cache (not its key).
func pairingID(p *pairing.Payload) string {
	h := sha256.Sum256([]byte("tbs1-pairing\x00" + p.Server + "\x00" + p.User + "\x00" + p.Key))
	return hex.EncodeToString(h[:8])
}

func (a *Agent) shareCachePath() string {
	return filepath.Join(a.store.Get().StateDir, "share-cache.json")
}

// shareCacheLocked loads the cache once. Caller holds a.mu.
func (a *Agent) shareCacheLocked() map[string]shareCacheEntry {
	ss := a.client.share
	if !ss.cacheLoaded {
		ss.cacheLoaded = true
		ss.cache = map[string]shareCacheEntry{}
		if b, err := os.ReadFile(a.shareCachePath()); err == nil {
			_ = json.Unmarshal(b, &ss.cache)
		}
	}
	return ss.cache
}

// setShareCache stores what the server said for p's pairing; entries of
// pairings no longer configured go.
func (a *Agent) setShareCache(p *pairing.Payload, e shareCacheEntry) {
	id := pairingID(p)
	a.mu.Lock()
	c := a.shareCacheLocked()
	old, had := c[id]
	same := had && old.Key == e.Key && slices.Equal(old.Passes, e.Passes)
	known := map[string]bool{id: true}
	if a.client.payload != nil {
		known[pairingID(a.client.payload)] = true
	}
	for _, s := range a.client.servers {
		if s.payload != nil {
			known[pairingID(s.payload)] = true
		}
	}
	for k := range c {
		if !known[k] {
			delete(c, k)
			same = false
		}
	}
	c[id] = e
	var b []byte
	if !same {
		b, _ = json.Marshal(c)
	}
	a.mu.Unlock()
	if b != nil {
		if err := writeFileAtomic(a.shareCachePath(), b, 0o600); err != nil {
			a.Logf("share: %v", err)
		}
	}
}

func (a *Agent) shareCacheFor(p *pairing.Payload) shareCacheEntry {
	if p == nil {
		return shareCacheEntry{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.shareCacheLocked()[pairingID(p)]
}

// clientShareKey: this device's share key, the pairing's, else the one the
// server gave this pairing.
func (a *Agent) clientShareKey(p *pairing.Payload) []byte {
	if p != nil && p.ShareKey != "" {
		return []byte(p.ShareKey)
	}
	if k := a.shareCacheFor(p).Key; k != "" {
		return []byte(k)
	}
	return nil
}

// hubPass is a pass decoded for a link.
type hubPass struct {
	key []byte
	exp uint64
}

// hubPasses: this pairing's unexpired passes, by sharer key id.
func (a *Agent) hubPasses(p *pairing.Payload) map[string]hubPass {
	out := map[string]hubPass{}
	now := uint64(max(time.Now().Unix(), 0))
	for _, ps := range a.shareCacheFor(p).Passes {
		k, err := hex.DecodeString(ps.Key)
		if err != nil || len(k) == 0 || ps.Exp <= now {
			continue
		}
		out[ps.KeyID] = hubPass{key: k, exp: ps.Exp}
	}
	return out
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
	a.mu.Lock()
	kick := a.client.share.kick
	a.mu.Unlock()
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
		case <-kick:
			t.Stop() // (Go 1.23+ timers: no stale tick after Stop)
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
	if ready && p != nil {
		if b, err := a.serverAPI(ctx, http.MethodGet, "/api/shares", nil); err == nil {
			var v sharesView
			if json.Unmarshal(b, &v) == nil {
				reg = &v
				e := shareCacheEntry{Passes: v.Passes}
				if p.ShareKey == "" {
					e.Key = v.Key // pairings made before they carried it
				}
				a.setShareCache(p, e)
			}
		}
	}
	a.mu.Lock()
	a.client.share.reg = reg
	a.mu.Unlock()
	key := a.clientShareKey(p)
	a.runSharer(ctx, cfg, p, key, ready)
	changed := a.runHub(ctx, cfg, p, a.hubPasses(p), reg)
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
func (a *Agent) runHub(ctx context.Context, cfg *config.Config, p *pairing.Payload, passes map[string]hubPass, reg *sharesView) bool {
	a.mu.Lock()
	ss := a.client.share
	own := ss.sharerID
	a.mu.Unlock()
	if !cfg.Client.SharesUsed() || p == nil || len(passes) == 0 {
		stopped := a.stopLinks(func(*hubLink) bool { return true })
		a.mu.Lock()
		ss.note = ""
		a.mu.Unlock()
		return stopped > 0 // the path set lost them
	}
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
			if _, ok := passes[r.KeyID]; !ok || r.Client == p.User || r.Kind == "proxy" {
				continue // no pass for it; or our own; or a phone hub's proxy (phonehub.go), not a relay
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
		pass, ok := passes[f.KeyID]
		if !ok || f.ID == own || f.Addr == nil {
			continue // another server's, or a removed device's
		}
		hl := ss.links[f.ID]
		// a working link stays put: a sharer on several of our networks (a
		// computer on the hub's Wi-Fi and Ethernet) answers on each of them
		if hl != nil && hl.found.Addr.String() != f.Addr.String() && (f.Via == "mdns" || !hl.mdns) && hl.link.Status().State != share.StateReady {
			// moved: new link, same local port so mqvpn's path stays valid
			port := hl.link.Local().Port
			hl.cancel()
			delete(ss.links, f.ID)
			hl = a.newHubLinkLocked(ss, f, p.User, pass, port)
		} else if hl == nil {
			hl = a.newHubLinkLocked(ss, f, p.User, pass, 0)
			changed = true
		} else {
			hl.link.SetPass(pass.key, pass.exp) // renewed daily
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
		if _, ok := passes[hl.found.KeyID]; !ok {
			// the server no longer gives a pass for it: removed
			hl.cancel()
			delete(ss.links, id)
			changed = true
			continue
		}
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
func (a *Agent) newHubLinkLocked(ss *shareState, f share.Found, hubID string, pass hubPass, localPort int) *hubLink {
	onChange := func(st share.LinkStatus) {
		// ready <-> not ready changes the path set: re-check now, not at
		// the next round (a dead relay path loses most of what it's given)
		go a.reconcilePaths(true)
	}
	l, err := share.NewLink(share.LinkConfig{Sharer: f.Addr, Key: pass.key, Exp: pass.exp, HubID: hubID, LocalPort: localPort, Logf: a.Logf, OnChange: onChange})
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

// stopLinks ends the links which picks; how many it ended.
func (a *Agent) stopLinks(which func(*hubLink) bool) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for id, hl := range a.client.share.links {
		if which(hl) {
			hl.cancel()
			delete(a.client.share.links, id)
			n++
		}
	}
	return n
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
		l = "Nearby device"
	}
	if strings.Contains(l, " from ") {
		return l // already says what it is: "Mobile data from sam-pixel", "Ethernet from tiger-mbp"
	}
	return l + " (shared)"
}

// allIfs is the discovered interfaces, the interfaces named in the path
// settings that discovery skips, and the shares.
func (a *Agent) allIfs() ([]netif.Iface, error) {
	ifs, err := a.listIfs()
	ifs = append(ifs, a.namedIfs(ifs)...)
	return append(ifs, a.shareIfaces()...), err
}

// namedIfs: a path setting can name any interface (an adapter discovery
// doesn't recognise, lo0 in the macOS e2e); it is used as named.
func (a *Agent) namedIfs(found []netif.Iface) []netif.Iface {
	if a.lookupIf == nil {
		return nil
	}
	have := map[string]bool{config.DefaultTunName: true}
	for _, i := range found {
		have[i.Name], have[i.Key()] = true, true
	}
	var out []netif.Iface
	for _, ps := range a.store.Get().Client.Paths {
		if have[ps.Iface] || strings.HasPrefix(ps.Iface, "sh") {
			continue
		}
		if it, ok := a.lookupIf(ps.Iface); ok {
			out = append(out, it)
			have[ps.Iface] = true
		}
	}
	return out
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
	use, local := cfg.Client.SharesUsed(), cfg.Client.LocalUsed()
	st.UseShares, st.UseLocal = &use, &local
	st.Platform = goruntime.GOOS
	user := ""
	if a.client.payload != nil {
		user = a.client.payload.User
	}
	sh := &status.Sharing{State: "off", Port: share.DefaultPort, Hubs: []status.SharingHub{}, Ifaces: a.shareCandidatesLocked(user)}
	if sc := cfg.Client.Share; sc != nil {
		sh.Enabled, sh.Iface, sh.CustomLabel, sh.Port, sh.CapMB, sh.AllowedHubs = sc.Enabled, sc.Iface, sc.Label, sharePort(sc), sc.CapMB, sc.AllowedHubs
		if sc.Iface != "" {
			sh.Label = a.sharerLabelLocked(sc, user)
		}
		switch {
		case !sc.Enabled:
		case ss.sharerErr != "":
			sh.State, sh.Error = "error", ss.sharerErr
			if ss.sharerErr == errNoShareKey {
				sh.State = "no_key"
			}
		case ss.sharer != nil:
			sh.State, sh.Label, sh.Note = "sharing", ss.sharerLabel, ss.sharerNote
			sh.Used = ss.sharer.Used()
			for _, h := range ss.sharer.Sessions() {
				sh.Hubs = append(sh.Hubs, status.SharingHub{Hub: h.HubID, Addr: h.Addr, UpBytes: h.Up, DnBytes: h.Down})
			}
		}
	}
	st.Sharing = sh
}

// ---- sharer (a Mac or Linux client lending one of its uplinks) ----

// errNoShareKey: a pairing made before pairings carried the share key gets
// it from the home server, through the tunnel.
const errNoShareKey = "Waiting for your home server's share key: turn tailbond on here once so it can fetch it."

func sharePort(sc *config.ShareConfig) int {
	if sc.Port > 0 {
		return sc.Port
	}
	return share.DefaultPort
}

// sharerKind is what this machine registers as with the home server. Hubs
// skip only "proxy" entries (phone hubs); this one says what lends it.
func sharerKind() string {
	if goruntime.GOOS == "darwin" {
		return "macos"
	}
	return "linux"
}

// shareNouns name a connection by its kind in the default label.
var shareNouns = map[string]string{
	netif.KindEthernet: "Ethernet", netif.KindWifi: "Wi-Fi", netif.KindCellular: "Mobile data", netif.KindUSB: "USB tethering",
}

// defaultShareLabel is what hubs call a computer's share unless it has a
// label: "Ethernet from tiger-mbp", like a phone's "Mobile data from
// sam-pixel". name is this device's name on the home server.
func defaultShareLabel(kind, iface, name string) string {
	noun := shareNouns[kind]
	if noun == "" {
		noun = iface
	}
	if name == "" {
		h, _ := os.Hostname()
		name, _, _ = strings.Cut(h, ".")
	}
	l := noun + " from " + name
	if len(l) > share.MaxIDLen {
		l = strings.ToValidUTF8(l[:share.MaxIDLen], "")
	}
	return l
}

// sharerLabelLocked is what hubs see for sc, the interface's kind from the
// last discovery. Caller holds a.mu.
func (a *Agent) sharerLabelLocked(sc *config.ShareConfig, user string) string {
	if sc.Label != "" {
		return sc.Label
	}
	return defaultShareLabel(a.client.ifs[sc.Iface].Kind, sc.Iface, user)
}

// sharerLabel is sharerLabelLocked, asking the OS about an interface
// discovery skipped.
func (a *Agent) sharerLabel(sc *config.ShareConfig, user string) string {
	a.mu.Lock()
	_, known := a.client.ifs[sc.Iface]
	l := a.sharerLabelLocked(sc, user)
	a.mu.Unlock()
	if sc.Label == "" && !known && a.lookupIf != nil {
		if it, ok := a.lookupIf(sc.Iface); ok {
			l = defaultShareLabel(it.Kind, sc.Iface, user)
		}
	}
	return l
}

// shareCandidatesLocked: this machine's connections that can be shared,
// from the last discovery. Caller holds a.mu.
func (a *Agent) shareCandidatesLocked(user string) []status.SharingIface {
	out := []status.SharingIface{}
	for _, name := range slices.Sorted(maps.Keys(a.client.ifs)) {
		it := a.client.ifs[name]
		if it.Kind == netif.KindShared || isTunnelName(name) || name == "lo" || name == "lo0" {
			continue
		}
		out = append(out, status.SharingIface{Name: name, Kind: it.Kind, Label: it.Label, Up: it.Usable(), ShareLabel: defaultShareLabel(it.Kind, name, user)})
	}
	return out
}

// shareIfFacts is what the sharer checks about the interface it lends.
type shareIfFacts struct {
	Loopback bool
	Up       bool
	Addr     bool // an IPv4 (not link-local) or global IPv6 address
}

func realShareIf(name string) (shareIfFacts, bool) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return shareIfFacts{}, false
	}
	f := shareIfFacts{Loopback: ifi.Flags&net.FlagLoopback != 0, Up: ifi.Flags&net.FlagUp != 0}
	// an Ethernet port stays UP with its address after the cable is pulled:
	// also need the carrier (netif.Lookup: sysfs on Linux, ifconfig's media
	// status on macOS)
	if f.Up && !f.Loopback {
		if it, ok := netif.Lookup(name); ok && !it.Carrier {
			f.Up = false
		}
	}
	addrs, _ := ifi.Addrs()
	for _, ad := range addrs {
		if n, ok := ad.(*net.IPNet); ok && !n.IP.IsLinkLocalUnicast() && (n.IP.To4() != nil || n.IP.IsGlobalUnicast() || n.IP.IsLoopback()) {
			f.Addr = true
		}
	}
	return f, true
}

// errShareChanged: the connection to share changed while turning it on.
var errShareChanged = errors.New("the share settings changed meanwhile; check them and try again")

// isTunnelName: tailbond's own tunnel or another VPN's, never shared.
// tb<N> is tailbond's tunnel on Linux and OpenWrt (config.DefaultTunName is
// "utun" on macOS), refused on every platform alike.
func isTunnelName(name string) bool {
	if name == config.DefaultTunName || isTbTun(name) {
		return true
	}
	for _, p := range []string{"utun", "tun", "tap", "wg", "tailscale", "mqvpn", "ipsec"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// isTbTun: "tb" and one or more digits.
func isTbTun(name string) bool {
	if len(name) < 3 || !strings.HasPrefix(name, "tb") {
		return false
	}
	for _, c := range name[2:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// loopbackServer: the pairing's server is on this machine's loopback (a
// server in a local VM, as in scripts/e2e-macos.sh), so the loopback
// interface does reach it.
func loopbackServer(server string) bool {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		host = server
	}
	ip, err := netip.ParseAddr(host)
	return host == "localhost" || err == nil && ip.IsLoopback()
}

// shareIfaceProblem says why iface can't be shared now ("" when it can).
// fixed: it never can (the API refuses it); otherwise it may later
// (plugged in, connected).
func shareIfaceProblem(iface, server string, look func(string) (shareIfFacts, bool)) (problem string, fixed bool) {
	if isTunnelName(iface) {
		return iface + " is a VPN tunnel, not an internet connection: pick Ethernet, Wi-Fi or a USB adapter.", true
	}
	f, ok := look(iface)
	switch {
	case !ok:
		return "There's no connection called " + iface + " on this computer right now (unplugged?).", false
	case f.Loopback && !loopbackServer(server):
		return iface + " is this computer's loopback, not an internet connection.", true
	case !f.Up || !f.Addr:
		return iface + " isn't connected right now; sharing starts when it is.", false
	}
	return "", false
}

// checkShareIface is what POST /api/share refuses: an interface that isn't
// there, a tunnel, the loopback.
func (a *Agent) checkShareIface(iface string) error {
	a.mu.Lock()
	p := a.client.payload
	a.mu.Unlock()
	server := ""
	if p != nil {
		server = p.Server
	}
	if _, ok := a.shareIf(iface); !ok && !isTunnelName(iface) {
		return fmt.Errorf("there's no connection called %s on this computer", iface)
	}
	if msg, fixed := shareIfaceProblem(iface, server, a.shareIf); fixed {
		return errors.New(msg)
	}
	return nil
}

// runSharer starts, restarts or stops this machine's own share. The share
// doesn't need this machine's own tunnel: only the pairing and its share
// key (registering with the home server does, so hubs off mDNS find it).
func (a *Agent) runSharer(ctx context.Context, cfg *config.Config, p *pairing.Payload, key []byte, ready bool) {
	sc := cfg.Client.Share
	on := sc != nil && sc.Enabled
	problem := ""
	switch {
	case !on:
	case sc.Iface == "":
		problem = "Pick the connection to share."
	case p == nil:
		problem = "Pair this computer with your home server first."
	case len(key) == 0:
		problem = errNoShareKey
	default:
		problem, _ = shareIfaceProblem(sc.Iface, p.Server, a.shareIf)
	}
	want := on && problem == ""
	sig := ""
	if want {
		sig = strings.Join([]string{sc.Iface, fmt.Sprint(sharePort(sc)), sc.Label, fmt.Sprint(sc.CapMB), strings.Join(sc.AllowedHubs, ","), string(key), p.Server, time.Now().Format("2006-01")}, "|")
	}
	a.mu.Lock()
	ss := a.client.share
	if ss.sharerSig != sig {
		if ss.sharerCancel != nil {
			ss.sharerCancel()
		}
		ss.sharer, ss.sharerCancel, ss.sharerSig, ss.sharerErr, ss.sharerNote = nil, nil, "", "", ""
	}
	if !want {
		ss.sharerErr = problem
	}
	running := ss.sharer != nil
	a.mu.Unlock()
	if !want {
		return
	}
	if !running {
		a.startSharer(ctx, sc, p, key, sig)
	}
	// hubs find the share on this machine's other networks (the hub's
	// Wi-Fi), never on the uplink it lends
	ifaces := share.MulticastIfaces(config.DefaultTunName, sc.Iface)
	note := ""
	if len(ifaces) == 0 {
		note = "This computer isn't on any other network, so no hub can find it: join the hub's Wi-Fi (or plug into its LAN) too."
	}
	a.mu.Lock()
	ss.sharerNote = note
	sh, last := ss.sharer, ss.lastReg
	a.mu.Unlock()
	if sh != nil && ready && time.Since(last) > shareRegEvery {
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
		body := regSharer{ID: ss.sharerID, Label: ss.sharerLabel, KeyID: share.KeyID(key), LAN: lan, Port: sharePort(sc), Kind: sharerKind()}
		a.mu.Unlock()
		if _, err := a.serverAPI(ctx, http.MethodPost, "/api/shares", body); err == nil {
			a.mu.Lock()
			ss.lastReg = time.Now()
			a.mu.Unlock()
		}
	}
}

func (a *Agent) startSharer(ctx context.Context, sc *config.ShareConfig, p *pairing.Payload, key []byte, sig string) {
	host, _ := os.Hostname()
	prefix := "lx-"
	if goruntime.GOOS == "darwin" {
		prefix = "mac-"
	}
	id := prefix + hex.EncodeToString(func() []byte { h := sha256.Sum256([]byte(host + "/" + p.User + "/" + sc.Iface)); return h[:5] }())
	label := a.sharerLabel(sc, p.User)
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
			// the upstream leaves through the shared interface only
			// (SO_BINDTODEVICE, IP_BOUND_IF), whatever the default route
			d := net.Dialer{Control: sockbind.Control(iface)}
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
		msg := err.Error()
		if errors.Is(err, syscall.EADDRINUSE) {
			msg = fmt.Sprintf("UDP port %d is in use by another program: pick another port for sharing.", sharePort(sc))
		}
		a.mu.Lock()
		cs.share.sharerErr = msg
		a.mu.Unlock()
		return
	}
	sctx, cancel := context.WithCancel(ctx)
	// stopping frees the port at once: a restart (new settings) binds it again
	stop := func() { cancel(); _ = conn.Close() }
	a.mu.Lock()
	cs.share.sharer, cs.share.sharerCancel, cs.share.sharerSig, cs.share.sharerErr = sh, stop, sig, ""
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

// hShareSettings: client role — POST {"use_shares":bool}, {"use_local":bool},
// {"share":{enabled, iface, label, cap_mb, allowed_hubs, port}} (replaces
// this machine's share settings) and/or {"share_enabled":bool} (turns the
// share on or off as set up).
func (a *Agent) hShareSettings(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	var in struct {
		UseShares    *bool               `json:"use_shares"`
		UseLocal     *bool               `json:"use_local"`
		Share        *config.ShareConfig `json:"share"`
		ShareEnabled *bool               `json:"share_enabled"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	next := a.store.Get().Client.Share
	if in.Share != nil {
		s := *in.Share
		s.Iface, s.Label = strings.TrimSpace(s.Iface), strings.TrimSpace(s.Label)
		var hubs []string
		for _, h := range s.AllowedHubs {
			if h = strings.TrimSpace(h); h != "" && !slices.Contains(hubs, h) {
				hubs = append(hubs, h)
			}
		}
		s.AllowedHubs = hubs
		if s.Enabled && s.Iface == "" {
			httpErr(w, http.StatusBadRequest, errors.New("pick the connection to share"))
			return
		}
		if s.CapMB < 0 || len(s.Label) > share.MaxIDLen || s.Port < 0 || s.Port > 65535 || len(s.AllowedHubs) > 32 {
			httpErr(w, http.StatusBadRequest, errors.New("bad share settings"))
			return
		}
		next = &s
	}
	if in.ShareEnabled != nil {
		if next == nil || next.Iface == "" {
			if *in.ShareEnabled {
				httpErr(w, http.StatusBadRequest, errors.New("pick the connection to share first (dashboard: Share this computer's connection)"))
				return
			}
		} else {
			s := *next
			s.Enabled = *in.ShareEnabled
			next = &s
		}
	}
	if next != nil && next.Enabled && (in.Share != nil || in.ShareEnabled != nil) {
		if err := a.checkShareIface(next.Iface); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		if in.UseShares != nil {
			v := *in.UseShares
			c.Client.UseShares = &v
		}
		if in.UseLocal != nil {
			v := *in.UseLocal
			c.Client.UseLocal = &v
		}
		switch {
		case in.Share != nil && next != nil:
			s := *next
			c.Client.Share = &s
		case in.ShareEnabled != nil && c.Client.Share != nil && c.Client.Share.Iface != "":
			// only the switch: flip it on what's saved now, not on the copy
			// read above, so a save that landed in between isn't undone
			if *in.ShareEnabled && (next == nil || c.Client.Share.Iface != next.Iface) {
				return errShareChanged
			}
			s := *c.Client.Share
			s.Enabled = *in.ShareEnabled
			c.Client.Share = &s
		}
		return nil
	}); errors.Is(err, errShareChanged) {
		httpErr(w, http.StatusConflict, err)
		return
	} else if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	if in.UseLocal != nil || in.UseShares != nil {
		a.reconcilePaths(true)
	}
	if in.Share != nil || in.ShareEnabled != nil {
		a.kickShares()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
