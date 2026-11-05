// Package tailscale wraps the optional `tailscale` CLI. Every call degrades to
// "not installed / not running" when the binary or daemon is missing.
package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Info struct {
	Installed      bool   `json:"installed"`
	Running        bool   `json:"running"`
	BackendState   string `json:"backend_state,omitempty"`
	Self           string `json:"self"`
	DNSName        string `json:"dns_name,omitempty"`
	MagicDNSSuffix string `json:"magic_dns_suffix,omitempty"`
	Tailnet        string `json:"tailnet,omitempty"`
	// Owner is the login of the user this machine belongs to; empty for a
	// tagged machine.
	Owner            string   `json:"owner,omitempty"`
	IPs              []string `json:"ips,omitempty"`
	AdvertisedRoutes []string `json:"advertised_routes"`
	ApprovedRoutes   []string `json:"approved_routes"`
	// Route is the tunnel subnet the agent wants advertised.
	Route           string `json:"route,omitempty"`
	RouteAdvertised bool   `json:"route_advertised"`
	RoutesApproved  bool   `json:"routes_approved"`
	ExitNode        bool   `json:"exit_node_advertised"`
	AdminURL        string `json:"admin_url,omitempty"`
	Peers           int    `json:"peers"`
	PeersOnline     int    `json:"peers_online"`
	Error           string `json:"error,omitempty"`
	// SelfNode and PeerList back GET /api/peers (not part of /api/tailscale).
	SelfNode *Node  `json:"-"`
	PeerList []Node `json:"-"`
}

// Node is one tailnet machine as GET /api/peers reports it.
type Node struct {
	HostName     string    `json:"hostname"`
	DNSName      string    `json:"dns_name"`
	TailscaleIPs []string  `json:"tailscale_ips"`
	OS           string    `json:"os"`
	Online       bool      `json:"online"`
	LastSeen     time.Time `json:"last_seen,omitzero"`
	// Additive: how this machine reaches the peer right now. "direct"
	// (CurAddr set), "relay" (only a DERP relay, e.g. "fra"), or "" when
	// there is no active connection.
	Conn  string `json:"conn,omitempty"`
	Relay string `json:"relay,omitempty"`
}

type nodeJSON struct {
	HostName     string
	DNSName      string
	OS           string
	TailscaleIPs []string
	Online       bool
	LastSeen     time.Time
	CurAddr      string
	Relay        string
	// LastHandshake: last time traffic (direct or relayed) was exchanged.
	// Without it, an empty CurAddr only means "no direct path right now",
	// which is every peer that hasn't been used since the tunnel came up —
	// not a peer that is going through a relay.
	LastHandshake time.Time
}

func (n nodeJSON) node() Node {
	ips := n.TailscaleIPs
	if ips == nil {
		ips = []string{}
	}
	out := Node{HostName: n.HostName, DNSName: strings.TrimSuffix(n.DNSName, "."), TailscaleIPs: ips, OS: n.OS, Online: n.Online}
	switch {
	case n.CurAddr != "":
		out.Conn = "direct"
	case n.Relay != "" && n.Online && recently(n.LastHandshake, now()):
		// traffic in the last couple of minutes with no direct path: this one
		// really is going through the relay
		out.Conn, out.Relay = "relay", n.Relay
	}
	// tailscale reports the zero time ("0001-01-01T00:00:00Z") for never/now-online peers
	if !n.LastSeen.IsZero() && n.LastSeen.Year() > 1 {
		out.LastSeen = n.LastSeen.UTC()
	}
	return out
}

// relayWindow: how fresh a handshake must be for "no direct path" to mean
// "relayed" rather than "not used since it connected".
const relayWindow = 2 * time.Minute

func recently(t, at time.Time) bool { return !t.IsZero() && at.Sub(t) < relayWindow && t.Before(at.Add(time.Minute)) }

// now is swapped in tests.
var now = time.Now

type CLI struct {
	Bin      string
	Disabled bool
	// Route is the subnet whose approval state is reported.
	Route string

	mu      sync.Mutex
	fake    bool
	cached  *Info
	fetched time.Time
}

func (c *CLI) bin() string {
	if c.Bin != "" {
		return c.Bin
	}
	if _, err := exec.LookPath("tailscale"); err != nil {
		// macOS: the Tailscale app keeps its CLI inside the app bundle
		for _, p := range appCLIs {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return "tailscale"
}

// appCLIs: where a Tailscale app (not on PATH) keeps its CLI.
var appCLIs = []string{"/Applications/Tailscale.app/Contents/MacOS/Tailscale"}

func (c *CLI) run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.bin(), args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return out, errors.New(strings.TrimSpace(string(ee.Stderr)))
		}
	}
	return out, err
}

type statusJSON struct {
	BackendState   string
	MagicDNSSuffix string
	CurrentTailnet *struct {
		Name           string
		MagicDNSSuffix string
	}
	Self *struct {
		nodeJSON
		PrimaryRoutes []string
		AllowedIPs    []string
		UserID        int64
		Tags          []string
	}
	Peer map[string]nodeJSON
	User map[string]struct{ LoginName string }
}

type prefsJSON struct {
	AdvertiseRoutes []string
}

// Get returns cached info, refreshing it at most every 10 s.
func (c *CLI) Get(ctx context.Context) *Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached != nil && time.Since(c.fetched) < 10*time.Second {
		cp := *c.cached
		return &cp
	}
	info := c.fetch(ctx)
	c.cached, c.fetched = info, time.Now()
	cp := *info
	return &cp
}

// SetRoute changes the subnet whose approval is reported (a server that
// moved to another tunnel subnet).
func (c *CLI) SetRoute(route string) {
	c.mu.Lock()
	c.Route = route
	if !c.fake {
		c.cached = nil
	}
	c.mu.Unlock()
}

// Invalidate drops the cache (after route changes).
func (c *CLI) Invalidate() {
	c.mu.Lock()
	if !c.fake {
		c.cached = nil
	}
	c.mu.Unlock()
}

func (c *CLI) fetch(ctx context.Context) *Info {
	info := &Info{AdvertisedRoutes: []string{}, ApprovedRoutes: []string{}, Route: c.Route}
	if c.Disabled {
		info.Error = "tailscale integration disabled"
		return info
	}
	if _, err := exec.LookPath(c.bin()); err != nil {
		info.Error = "tailscale not installed"
		return info
	}
	info.Installed = true
	out, err := c.run(ctx, "status", "--json")
	if len(out) == 0 && err != nil {
		info.Error = err.Error()
		return info
	}
	if err := parseStatus(out, c.Route, info); err != nil {
		info.Error = err.Error()
		return info
	}
	if pout, err := c.run(ctx, "debug", "prefs"); err == nil {
		var p prefsJSON
		if json.Unmarshal(pout, &p) == nil {
			for _, r := range p.AdvertiseRoutes {
				switch r {
				case "0.0.0.0/0", "::/0":
					info.ExitNode = true
				default:
					info.AdvertisedRoutes = append(info.AdvertisedRoutes, r)
					if canon(r) == canon(c.Route) {
						info.RouteAdvertised = true
					}
				}
			}
		}
	}
	if info.Running {
		info.AdminURL = "https://login.tailscale.com/admin/machines"
	}
	return info
}

// parseStatus fills info from `tailscale status --json` output.
func parseStatus(out []byte, route string, info *Info) error {
	var st statusJSON
	if err := json.Unmarshal(out, &st); err != nil {
		return errors.New("parse tailscale status: " + err.Error())
	}
	info.BackendState = st.BackendState
	info.Running = st.BackendState == "Running"
	info.MagicDNSSuffix = st.MagicDNSSuffix
	if st.CurrentTailnet != nil {
		info.Tailnet = st.CurrentTailnet.Name
		if info.MagicDNSSuffix == "" {
			info.MagicDNSSuffix = st.CurrentTailnet.MagicDNSSuffix
		}
	}
	info.PeerList = []Node{}
	for _, p := range st.Peer {
		info.Peers++
		if p.Online {
			info.PeersOnline++
		}
		info.PeerList = append(info.PeerList, p.node())
	}
	sort.Slice(info.PeerList, func(i, j int) bool {
		a, b := info.PeerList[i], info.PeerList[j]
		if a.Online != b.Online {
			return a.Online
		}
		return strings.ToLower(a.HostName) < strings.ToLower(b.HostName)
	})
	if st.Self != nil {
		self := st.Self.node()
		self.Online = true
		info.SelfNode = &self
		info.Self = st.Self.HostName
		info.DNSName = strings.TrimSuffix(st.Self.DNSName, ".")
		info.IPs = st.Self.TailscaleIPs
		if u, ok := st.User[strconv.FormatInt(st.Self.UserID, 10)]; ok && len(st.Self.Tags) == 0 && u.LoginName != "tagged-devices" {
			info.Owner = u.LoginName
		}
		approved := map[string]bool{}
		for _, r := range append(append([]string{}, st.Self.PrimaryRoutes...), st.Self.AllowedIPs...) {
			if pfx, err := netip.ParsePrefix(r); err == nil && pfx.Bits() < pfx.Addr().BitLen() {
				if !approved[pfx.String()] {
					approved[pfx.String()] = true
					info.ApprovedRoutes = append(info.ApprovedRoutes, pfx.String())
				}
			}
		}
		info.RoutesApproved = route != "" && approved[canon(route)]
	}
	return nil
}

func canon(s string) string {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return s
	}
	return p.Masked().String()
}

// Cert fetches a Let's Encrypt cert for the machine's ts.net name. It needs
// HTTPS certificates enabled for the tailnet.
func (c *CLI) Cert(ctx context.Context, domain, certFile, keyFile string) error {
	if c.Disabled {
		return errors.New("tailscale disabled")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.bin(), "cert", "--cert-file", certFile, "--key-file", keyFile, domain)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return nil
}

// Whois is who a tailnet address belongs to (`tailscale whois --json`).
type Whois struct {
	NodeID   string `json:"node_id"`  // stable node ID
	Node     string `json:"node"`     // machine name (MagicDNS short name)
	DNSName  string `json:"dns_name"` // full MagicDNS name, no trailing dot
	Hostname string `json:"hostname"` // the OS hostname the device reports
	OS       string `json:"os,omitempty"`
	User     string `json:"user"` // login name
	UserName string `json:"user_name,omitempty"`
}

type whoisJSON struct {
	Node *struct {
		ID           int64
		StableID     string
		Name         string
		ComputedName string
		Hostinfo     *struct {
			Hostname string
			OS       string
		}
	}
	UserProfile *struct {
		LoginName   string
		DisplayName string
	}
}

// ParseWhois decodes `tailscale whois --json` output.
func ParseWhois(out []byte) (*Whois, error) {
	var w whoisJSON
	if err := json.Unmarshal(out, &w); err != nil {
		return nil, errors.New("parse tailscale whois: " + err.Error())
	}
	if w.Node == nil {
		return nil, errors.New("tailscale whois: no node")
	}
	r := &Whois{NodeID: w.Node.StableID, DNSName: strings.TrimSuffix(w.Node.Name, "."), Node: w.Node.ComputedName}
	if r.NodeID == "" && w.Node.ID != 0 {
		r.NodeID = strconv.FormatInt(w.Node.ID, 10)
	}
	if r.Node == "" {
		r.Node, _, _ = strings.Cut(r.DNSName, ".")
	}
	if w.Node.Hostinfo != nil {
		r.Hostname, r.OS = w.Node.Hostinfo.Hostname, w.Node.Hostinfo.OS
	}
	if w.UserProfile != nil {
		r.User, r.UserName = w.UserProfile.LoginName, w.UserProfile.DisplayName
	}
	if r.NodeID == "" {
		return nil, errors.New("tailscale whois: node has no ID")
	}
	return r, nil
}

// Whois asks the local tailscaled who owns a tailnet IP.
func (c *CLI) Whois(ctx context.Context, ip string) (*Whois, error) {
	if c.Disabled {
		return nil, errors.New("tailscale integration disabled")
	}
	out, err := c.run(ctx, "whois", "--json", ip)
	if err != nil {
		return nil, err
	}
	return ParseWhois(out)
}

var tailnetPrefixes = []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("fd7a:115c:a1e0::/48")}

// IsTailnetIP reports whether ip is in Tailscale's address ranges.
func IsTailnetIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	for _, p := range tailnetPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// NewFake returns a CLI that always reports info (tests).
func NewFake(info *Info) *CLI {
	return &CLI{cached: info, fetched: time.Now().Add(100 * 365 * 24 * time.Hour), fake: true}
}

// Netcheck is the part of `tailscale netcheck --format=json` the setup
// health check uses.
type Netcheck struct {
	UDP                   bool   `json:"udp"`
	IPv4                  bool   `json:"ipv4"`
	IPv6                  bool   `json:"ipv6"`
	GlobalV4              string `json:"global_v4,omitempty"`
	MappingVariesByDestIP bool   `json:"mapping_varies_by_dest_ip"` // hard NAT
	UPnP                  bool   `json:"upnp"`
	PMP                   bool   `json:"pmp"`
	PCP                   bool   `json:"pcp"`
	PreferredDERP         int    `json:"preferred_derp,omitempty"`
}

type netcheckJSON struct {
	UDP                   bool
	IPv4                  bool
	IPv6                  bool
	GlobalV4              string
	MappingVariesByDestIP optBool
	UPnP                  optBool
	PMP                   optBool
	PCP                   optBool
	PreferredDERP         int
}

// optBool reads Tailscale's opt.Bool ("true" / "false" / "") or a JSON bool.
type optBool bool

func (b *optBool) UnmarshalJSON(d []byte) error {
	*b = optBool(strings.Trim(string(d), `"`) == "true")
	return nil
}

func deref(b optBool) bool { return bool(b) }

// ParseNetcheck decodes `tailscale netcheck --format=json`.
func ParseNetcheck(out []byte) (*Netcheck, error) {
	var n netcheckJSON
	if err := json.Unmarshal(out, &n); err != nil {
		return nil, errors.New("parse tailscale netcheck: " + err.Error())
	}
	return &Netcheck{UDP: n.UDP, IPv4: n.IPv4, IPv6: n.IPv6, GlobalV4: n.GlobalV4,
		MappingVariesByDestIP: deref(n.MappingVariesByDestIP), UPnP: deref(n.UPnP), PMP: deref(n.PMP), PCP: deref(n.PCP),
		PreferredDERP: n.PreferredDERP}, nil
}

// Netcheck runs `tailscale netcheck` (a few seconds); callers cache it.
func (c *CLI) Netcheck(ctx context.Context) (*Netcheck, error) {
	if c.Disabled || c.fake {
		return nil, errors.New("tailscale integration disabled")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.bin(), "netcheck", "--format=json").Output()
	if err != nil {
		return nil, err
	}
	return ParseNetcheck(out)
}

// ParseStatusForTest exposes parseStatus to other packages' tests.
func ParseStatusForTest(out []byte, route string, info *Info) error {
	return parseStatus(out, route, info)
}
