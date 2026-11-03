// Package status defines the normalized status JSON (docs/architecture.md,
// "Normalized status") and converts mqvpn control API replies into it.
package status

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/mqctl"
)

const (
	StateConnected    = "connected"
	StateConnecting   = "connecting"
	StateDisconnected = "disconnected"
	StateError        = "error"

	PathActive  = "active"
	PathStandby = "standby"
	PathFailed  = "failed"
	PathPending = "pending"
)

type Totals struct {
	TxBps   int64  `json:"tx_bps"`
	RxBps   int64  `json:"rx_bps"`
	TxBytes uint64 `json:"tx_bytes"`
	RxBytes uint64 `json:"rx_bytes"`
}

type Path struct {
	ID       uint64  `json:"id"`
	Iface    string  `json:"iface"`
	Kind     string  `json:"kind"`
	State    string  `json:"state"`
	SrttMs   uint64  `json:"srtt_ms"`
	MinRttMs uint64  `json:"min_rtt_ms"`
	LossPct  float64 `json:"loss_pct"`
	TxBps    int64   `json:"tx_bps"`
	RxBps    int64   `json:"rx_bps"`
	TxBytes  uint64  `json:"tx_bytes"`
	RxBytes  uint64  `json:"rx_bytes"`
	Priority string  `json:"priority,omitempty"`
	Enabled  bool    `json:"enabled"`
	// Additive fields.
	DataCapMB  int64  `json:"data_cap_mb,omitempty"`
	UsedBytes  uint64 `json:"used_bytes,omitempty"` // counted against data_cap_mb
	CapReached bool   `json:"cap_reached,omitempty"`
	// OpenWrt: the network the device belongs to (wan, wwan, tethering,
	// modem_0001...) and a plain label for it (Ethernet, Repeater, ...).
	Network string `json:"network,omitempty"`
	Label   string `json:"label,omitempty"`
	// Captive portal check (client role, additive): open | portal | offline;
	// portal_url for a portal; signin_until while sign-in mode is on.
	Captive     string     `json:"captive,omitempty"`
	PortalURL   string     `json:"portal_url,omitempty"`
	SigninUntil *time.Time `json:"signin_until,omitempty"`
	// "glinet" when GL's own repeater portal handling owns this WAN's
	// sign-in (the panel then points to GL's Repeater page), else "tailbond".
	CaptiveSource string `json:"captive_source,omitempty"`
}

// Client is one registered or connected client in the server role (additive shape).
type Client struct {
	Name         string `json:"name"`
	Registered   bool   `json:"registered"`
	Connected    bool   `json:"connected"`
	Endpoint     string `json:"endpoint,omitempty"`
	Scheduler    string `json:"scheduler,omitempty"`
	ConnectedSec uint64 `json:"connected_sec"`
	TxBps        int64  `json:"tx_bps"`
	RxBps        int64  `json:"rx_bps"`
	TxBytes      uint64 `json:"tx_bytes"`
	RxBytes      uint64 `json:"rx_bytes"`
	Paths        []Path `json:"paths"`
	// Additive: what the client last reported (X-Tailbond-Version/Kind)
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

type Tailscale struct {
	Running        bool     `json:"running"`
	Self           string   `json:"self"`
	RoutesApproved bool     `json:"routes_approved"`
	Installed      bool     `json:"installed"`
	DNSName        string   `json:"dns_name,omitempty"`
	IPs            []string `json:"ips,omitempty"`
}

type Status struct {
	Role      string     `json:"role"`
	Mode      string     `json:"mode"`
	State     string     `json:"state"`
	Server    string     `json:"server"`
	TunnelIP  string     `json:"tunnel_ip"`
	UptimeSec uint64     `json:"uptime_sec"`
	Totals    Totals     `json:"totals"`
	Paths     []Path     `json:"paths"`
	Clients   []Client   `json:"clients"`
	Tailscale *Tailscale `json:"tailscale"`
	// Additive fields.
	Error      string `json:"error,omitempty"`
	ControlAPI bool   `json:"control_api"` // live per-path stats from mqvpn
	Paired     bool   `json:"paired,omitempty"`
	Version    string `json:"version,omitempty"`
	// Server role: pending pair requests. Client role: asking a server to
	// pair (state of POST /api/join).
	PairRequests any `json:"pair_requests,omitempty"`
	Join         any `json:"join,omitempty"`
	// Client role: the server list, the active one and the failover
	// policy (docs/architecture.md, "Multiple servers and failover").
	Servers      []ServerState `json:"servers,omitempty"`
	ActiveServer string        `json:"active_server,omitempty"`
	Failover     *Failover     `json:"failover,omitempty"`
	// Updates (docs/architecture.md, "Updates")
	Protocol int     `json:"protocol,omitempty"`
	Update   *Update `json:"update,omitempty"`
	// Client role, Pair & Share (docs/pair-and-share.md): mobile data that
	// nearby devices share with this hub, why sharing can't work here
	// (ShareNote, e.g. client isolation), and this machine's own share.
	Shares    []Share  `json:"shares,omitempty"`
	UseShares *bool    `json:"use_shares,omitempty"`
	ShareNote string   `json:"share_note,omitempty"`
	Sharing   *Sharing `json:"sharing,omitempty"`
}

// Share is a nearby device's shared mobile data, as the hub sees it.
type Share struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`            // "Pixel's 5G (shared)"
	Path     string  `json:"path"`             // the path's name in paths[] (iface)
	Addr     string  `json:"addr"`             // the sharer's LAN address
	Via      string  `json:"via"`              // "mdns" or "server"
	State    string  `json:"state"`            // connecting, ready, refused, unreachable, lost
	Reason   string  `json:"reason,omitempty"` // why not ready, in plain words
	UpBytes  uint64  `json:"up_bytes"`
	DnBytes  uint64  `json:"down_bytes"`
	CapLeft  *uint64 `json:"cap_left_bytes,omitempty"` // the sharer's own cap
	Battery  int     `json:"battery"`                  // -1 unknown
	Charging bool    `json:"charging"`
}

// Sharing is this machine sharing one of its uplinks.
type Sharing struct {
	Enabled bool         `json:"enabled"`
	Iface   string       `json:"iface"`
	Label   string       `json:"label"`
	Port    int          `json:"port"`
	State   string       `json:"state"` // sharing, off, no_key, error
	Error   string       `json:"error,omitempty"`
	Used    uint64       `json:"used_bytes"`
	CapMB   int64        `json:"cap_mb,omitempty"`
	Hubs    []SharingHub `json:"hubs"`
}

// SharingHub is a hub using this machine's share.
type SharingHub struct {
	Hub     string `json:"hub"`
	Addr    string `json:"addr"`
	UpBytes uint64 `json:"up_bytes"`
	DnBytes uint64 `json:"down_bytes"`
}

// Update is the status of auto-updates on this machine.
type Update struct {
	Current   string `json:"current"`
	Available string `json:"available,omitempty"`
	NotesURL  string `json:"notes_url,omitempty"` // the release page (changelog)
	Policy    string `json:"policy"`              // notify | auto | off
	Repo      string `json:"repo,omitempty"`
	// Method: static (install.sh bundle) | opkg (OpenWrt) | dnf (RPM: dnf
	// upgrade) | none; one-click install for static and opkg
	Method     string     `json:"method"`
	State      string     `json:"state"` // idle | checking | downloading | installing | failed | rolled_back | off
	LastCheck  *time.Time `json:"last_check,omitempty"`
	LastResult string     `json:"last_result,omitempty"` // ok | rolled_back | failed
	// the version the last result is about, when, and a detail (e.g.
	// reinstalled_after_firmware_upgrade)
	LastResultVersion string `json:"last_result_version,omitempty"`
	LastResultAt      string `json:"last_result_at,omitempty"`
	Detail            string `json:"detail,omitempty"`
	Error             string `json:"error,omitempty"`
	// AutoWindow: when policy auto installs (local time, "03:00-05:00")
	AutoWindow string `json:"auto_window,omitempty"`
	// DnfAutomatic (method dnf): install | download (Fedora's default: a
	// dnf-automatic timer that only downloads) | off; for every package
	DnfAutomatic string     `json:"dnf_automatic,omitempty"`
	Outdated     []Outdated `json:"outdated,omitempty"`
}

// Outdated is a client running an older version than its server.
type Outdated struct {
	Name    string `json:"name"`
	Kind    string `json:"kind,omitempty"`
	Version string `json:"version"`
}

// ServerState is one entry of the client's server list.
type ServerState struct {
	Name   string     `json:"name"`
	Role   string     `json:"role"`  // primary | backup
	State  string     `json:"state"` // active | connecting | standby | unreachable | unknown | off
	Active bool       `json:"active"`
	LastOK *time.Time `json:"last_ok,omitempty"`
	Subnet string     `json:"subnet"`
	Server string     `json:"server"`
}

type Failover struct {
	Policy           string     `json:"policy"`
	Server           string     `json:"server"`
	SwitchAfterSec   int        `json:"switch_after_sec"`
	FailbackAfterSec int        `json:"failback_after_sec"`
	Reason           string     `json:"reason"`
	Since            *time.Time `json:"since,omitempty"`
	FailbackInSec    *int       `json:"failback_in_sec,omitempty"`
	// the last switches, newest first (in memory: since the agent started)
	History []Switch `json:"history,omitempty"`
}

// Switch is one change of the active server.
type Switch struct {
	At     time.Time `json:"at"`
	From   string    `json:"from"`
	To     string    `json:"to"`
	Reason string    `json:"reason"` // failover | failback | manual | policy | list
	// why the server left behind stopped working, for failovers
	Detail string `json:"detail,omitempty"`
}

// MapPathState turns mqvpn path labels (control API state_label or the client
// log's path status) into the contract's path states.
func MapPathState(label string) string {
	switch label {
	case "active", "degraded":
		return PathActive
	case "standby":
		return PathStandby
	case "init", "validating", "pending":
		return PathPending
	case "closing", "closed":
		return PathFailed
	default:
		return PathPending
	}
}

type sample struct {
	t            time.Time
	tx, rx       uint64
	sent, lost   uint64
	txBps, rxBps int64
	loss         float64
	seen         bool
}

// Rater derives rates from successive cumulative counters, keyed by series.
type Rater struct {
	prev map[string]*sample
}

func NewRater() *Rater { return &Rater{prev: map[string]*sample{}} }

// Begin starts a poll round; series not touched before End are forgotten.
func (r *Rater) Begin() {
	for _, s := range r.prev {
		s.seen = false
	}
}

func (r *Rater) End() {
	for k, s := range r.prev {
		if !s.seen {
			delete(r.prev, k)
		}
	}
}

// Rate returns bits/s for tx/rx and a smoothed loss percentage.
func (r *Rater) Rate(key string, now time.Time, tx, rx, sent, lost uint64) (txBps, rxBps int64, lossPct float64) {
	p, ok := r.prev[key]
	if !ok {
		r.prev[key] = &sample{t: now, tx: tx, rx: rx, sent: sent, lost: lost, seen: true, loss: cumLoss(sent, lost)}
		return 0, 0, round1(cumLoss(sent, lost))
	}
	p.seen = true
	dt := now.Sub(p.t).Seconds()
	if dt < 0.2 {
		return p.txBps, p.rxBps, round1(p.loss)
	}
	// A counter going backwards means mqvpn restarted or the path was replaced.
	if tx < p.tx || rx < p.rx || sent < p.sent {
		*p = sample{t: now, tx: tx, rx: rx, sent: sent, lost: lost, seen: true, loss: cumLoss(sent, lost)}
		return 0, 0, round1(p.loss)
	}
	p.txBps = int64(float64(tx-p.tx) * 8 / dt)
	p.rxBps = int64(float64(rx-p.rx) * 8 / dt)
	if ds := sent - p.sent; ds > 0 {
		dl := float64(0)
		if lost >= p.lost {
			dl = float64(lost - p.lost)
		}
		inst := math.Min(100, dl*100/float64(ds))
		p.loss = 0.7*p.loss + 0.3*inst
	}
	p.t, p.tx, p.rx, p.sent, p.lost = now, tx, rx, sent, lost
	return p.txBps, p.rxBps, round1(p.loss)
}

func cumLoss(sent, lost uint64) float64 {
	if sent == 0 {
		return 0
	}
	return math.Min(100, float64(lost)*100/float64(sent))
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

// PathMeta supplies client-side settings for a path's interface.
type PathMeta func(iface string) (kind, priority string, enabled bool)

func convertPath(r *Rater, key string, now time.Time, p mqctl.Path, meta PathMeta) Path {
	tx, rx, loss := r.Rate(key, now, p.BytesTx, p.BytesRx, p.PktSent, p.PktLost)
	out := Path{
		ID: p.PathID, Iface: p.Iface, Kind: "unknown", State: MapPathState(p.StateLabel),
		SrttMs: p.SrttMs, MinRttMs: p.MinRttMs, LossPct: loss,
		TxBps: tx, RxBps: rx, TxBytes: p.BytesTx, RxBytes: p.BytesRx, Enabled: true,
	}
	if out.State != PathActive {
		out.TxBps, out.RxBps = 0, 0
	}
	if meta != nil && p.Iface != "" {
		out.Kind, out.Priority, out.Enabled = meta(p.Iface)
	}
	return out
}

// trimClosed keeps every live path and at most keepClosed closed ones (the
// newest), since mqvpn lists replaced paths until the connection ends.
func trimClosed(paths []Path, keepClosed int) []Path {
	var live, closed []Path
	for _, p := range paths {
		if p.State == PathFailed {
			closed = append(closed, p)
		} else {
			live = append(live, p)
		}
	}
	if len(closed) > keepClosed {
		closed = closed[len(closed)-keepClosed:]
	}
	return append(live, closed...)
}

// dedupeByIface keeps one entry per interface, preferring a live path, then the
// highest path id (the newest).
func dedupeByIface(paths []Path) []Path {
	best := map[string]int{}
	var out []Path
	for _, p := range paths {
		if p.Iface == "" {
			out = append(out, p)
			continue
		}
		i, ok := best[p.Iface]
		if !ok {
			best[p.Iface] = len(out)
			out = append(out, p)
			continue
		}
		cur := out[i]
		if (cur.State == PathFailed && p.State != PathFailed) ||
			((cur.State == PathFailed) == (p.State == PathFailed) && p.ID > cur.ID) {
			out[i] = p
		}
	}
	return out
}

// Server converts a server-mode get_status/get_stats pair. registered lists
// configured user names so offline clients show up too.
func Server(r *Rater, now time.Time, st *mqctl.Status, stats *mqctl.Stats, registered []string) ([]Client, Totals, uint64) {
	var tot Totals
	var uptime uint64
	byName := map[string]*Client{}
	var order []string
	for _, n := range registered {
		byName[n] = &Client{Name: n, Registered: true, Paths: []Path{}}
		order = append(order, n)
	}
	var live []Client
	if st != nil {
		for _, c := range st.Clients {
			key := "c/" + c.User + "/" + c.Endpoint
			txb, rxb, _ := r.Rate(key, now, c.BytesTx, c.BytesRx, 0, 0)
			cl := Client{
				Name: c.User, Connected: true, Endpoint: c.Endpoint, Scheduler: c.Scheduler,
				ConnectedSec: c.ConnectedSec, TxBps: txb, RxBps: rxb, TxBytes: c.BytesTx, RxBytes: c.BytesRx,
				Paths: []Path{},
			}
			for _, p := range c.Paths {
				cl.Paths = append(cl.Paths, convertPath(r, key+"/"+strconv.FormatUint(p.PathID, 10), now, p, nil))
			}
			cl.Paths = trimClosed(cl.Paths, 2)
			if reg, ok := byName[c.User]; ok && !reg.Connected {
				cl.Registered = true
				*reg = cl
			} else {
				live = append(live, cl)
			}
		}
	}
	if stats != nil {
		tot.TxBytes, tot.RxBytes = stats.BytesTx, stats.BytesRx
		tot.TxBps, tot.RxBps, _ = r.Rate("totals", now, stats.BytesTx, stats.BytesRx, 0, 0)
		uptime = stats.UptimeSec
	}
	out := make([]Client, 0, len(order)+len(live))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	out = append(out, live...)
	return out, tot, uptime
}

// ClientFromControl converts the patched client-mode get_status (one client
// object whose paths carry "iface").
func ClientFromControl(r *Rater, now time.Time, st *mqctl.Status, meta PathMeta) ([]Path, Totals, uint64, bool) {
	var tot Totals
	if st == nil || len(st.Clients) == 0 {
		return []Path{}, tot, 0, false
	}
	c := st.Clients[0]
	tot.TxBytes, tot.RxBytes = c.BytesTx, c.BytesRx
	tot.TxBps, tot.RxBps, _ = r.Rate("totals", now, c.BytesTx, c.BytesRx, 0, 0)
	paths := make([]Path, 0, len(c.Paths))
	connected := false
	for _, p := range c.Paths {
		np := convertPath(r, "p/"+strconv.FormatUint(p.PathID, 10), now, p, meta)
		if np.State == PathActive {
			connected = true
		}
		paths = append(paths, np)
	}
	paths = dedupeByIface(paths)
	sort.SliceStable(paths, func(i, j int) bool { return paths[i].ID < paths[j].ID })
	return paths, tot, c.ConnectedSec, connected
}
