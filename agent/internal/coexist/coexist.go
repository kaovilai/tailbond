// Package coexist keeps tailbond out of other VPNs' way on a Mac or a Linux
// client (docs/architecture.md, "Living next to other VPNs"): NetBird (a
// work VPN), the Tailscale app, anything else that owns routes on its own
// tunnel interface.
//
// The rules, decided by Plan (pure, unit tested) and carried out by an
// Applier (darwin: route, pfctl, /etc/resolver; Linux: ip, iptables,
// resolvectl), which records everything it adds so a stop or the next
// start after a crash takes it away again:
//
//   - tailnet machines are reached through tailbond by host routes (/32,
//     /128) to their actual addresses, never a prefix that equals or covers
//     a route another interface owns; none at all while the Tailscale app
//     runs (it reaches them itself)
//   - DNS: split, never the global resolver, once another VPN is there:
//     ts.net and the tailnet's suffix go to the tunnel's DNS
//   - the work VPN's own traffic (its WireGuard UDP to peers, relays, signal
//     and management): Direct (host routes via the physical uplink, the
//     default) or Bonded (through tailbond, if tailbond's MTU can be at
//     least the work VPN's + 80)
//   - with tailbond's kill switch on, the work VPN's interface and its
//     endpoints are allowed
//   - bypass rules (IP ranges, domains) go direct via the uplink
package coexist

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Work VPN traffic modes (client.work_vpn).
const (
	WorkDirect = "direct"
	WorkBonded = "bonded"
	// BondedOverhead: tailbond's tunnel must carry the work VPN's packets
	// whole: its MTU at least the work VPN's interface MTU + this
	// (WireGuard's 60-80 bytes of IPv6/UDP/WireGuard header).
	BondedOverhead = 80
	// TailbondOverhead is what tailbond's own encapsulation takes from the
	// uplink MTU (IPv6 + UDP + QUIC short header + MASQUE + reorder header,
	// rounded up): the largest tunnel MTU an uplink allows.
	TailbondOverhead = 100
)

var (
	cgnat      = netip.MustParsePrefix("100.64.0.0/10")
	tailnet6   = netip.MustParsePrefix("fd7a:115c:a1e0::/48")
	clatOrLink = []netip.Prefix{netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("fe80::/10")}
)

// Route is one entry of the OS routing table.
type Route struct {
	Prefix  netip.Prefix
	Iface   string
	Gateway string // "" or "link#N" for on-link
}

// NetBird is what `netbird status --json` says (or that it isn't there).
type NetBird struct {
	CLI       bool   `json:"cli"`       // the netbird CLI exists
	Running   bool   `json:"running"`   // its daemon answers
	LoggedIn  bool   `json:"logged_in"` // it has an address (management login done)
	Connected bool   `json:"connected"` // management connected
	Iface     string `json:"iface,omitempty"`
	Addr      string `json:"addr,omitempty"` // 100.92.1.2/16
	MTU       int    `json:"mtu,omitempty"`
	// Networks are the routes NetBird installs (its "networks").
	Networks []string `json:"networks,omitempty"`
	// DNSDomains are the domains NetBird's DNS answers (match domains).
	DNSDomains []string `json:"dns_domains,omitempty"`
	// Endpoints: host or host:port of peers (ICE remote candidates),
	// relays, signal and management: its underlay.
	Endpoints []string `json:"endpoints,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ParseNetBirdStatus reads `netbird status --json` (NetBird's
// OutputOverview). Unknown fields are ignored; a logged-out daemon gives no
// address.
func ParseNetBirdStatus(b []byte) (NetBird, error) {
	var o struct {
		Management struct {
			URL       string `json:"url"`
			Connected bool   `json:"connected"`
		} `json:"management"`
		Signal struct {
			URL string `json:"url"`
		} `json:"signal"`
		Relays struct {
			Details []struct {
				URI string `json:"uri"`
			} `json:"details"`
		} `json:"relays"`
		Peers struct {
			Details []struct {
				IP       string `json:"netbirdIp"`
				Endpoint struct {
					Remote string `json:"remote"`
				} `json:"iceCandidateEndpoint"`
				RelayAddress string `json:"relayAddress"`
			} `json:"details"`
		} `json:"peers"`
		IP         string   `json:"netbirdIp"`
		Networks   []string `json:"networks"`
		DNSServers []struct {
			Domains []string `json:"domains"`
			Enabled *bool    `json:"enabled"`
		} `json:"dnsServers"`
	}
	nb := NetBird{CLI: true, Running: true}
	if err := json.Unmarshal(b, &o); err != nil {
		return nb, fmt.Errorf("netbird status: %v", err)
	}
	nb.Connected = o.Management.Connected
	if p, err := netip.ParsePrefix(o.IP); err == nil {
		nb.Addr, nb.LoggedIn = p.String(), true
	}
	nb.Networks = o.Networks
	for _, g := range o.DNSServers {
		if g.Enabled != nil && !*g.Enabled {
			continue
		}
		nb.DNSDomains = append(nb.DNSDomains, g.Domains...)
	}
	add := func(s string) {
		if h := endpointHost(s); h != "" && !slices.Contains(nb.Endpoints, h) {
			nb.Endpoints = append(nb.Endpoints, h)
		}
	}
	add(o.Management.URL)
	add(o.Signal.URL)
	for _, r := range o.Relays.Details {
		add(r.URI)
	}
	for _, p := range o.Peers.Details {
		add(p.Endpoint.Remote)
		add(p.RelayAddress)
	}
	return nb, nil
}

// endpointHost: the host of "https://api.netbird.io:443",
// "rels://relay.netbird.io:443", "203.0.113.5:51820", "[2001:db8::1]:51820"
// or a bare host; "" for nothing usable (a "-" ICE placeholder).
func endpointHost(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return ""
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		return u.Hostname()
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		return h
	}
	return strings.Trim(s, "[]")
}

// NetBirdFromError classifies a failed `netbird status --json`: its daemon
// not running, or not logged in.
func NetBirdFromError(out []byte, err error) NetBird {
	nb := NetBird{CLI: true}
	t := strings.ToLower(string(out))
	switch {
	case strings.Contains(t, "needslogin") || strings.Contains(t, "needs login") || strings.Contains(t, "not logged") || strings.Contains(t, "login"):
		nb.Running, nb.Error = true, "logged out"
	case strings.Contains(t, "daemon") || strings.Contains(t, "connect"):
		nb.Error = "its service isn't running"
	default:
		nb.Error = strings.TrimSpace(firstLine(string(out)))
		if nb.Error == "" && err != nil {
			nb.Error = err.Error()
		}
	}
	return nb
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ParseNetstat reads darwin's `netstat -rn` (both families).
func ParseNetstat(out string) []Route {
	var rs []Route
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 4 || f[0] == "Destination" || f[0] == "Routing" || f[0] == "Internet:" || f[0] == "Internet6:" {
			continue
		}
		p, ok := darwinDest(f[0])
		if !ok {
			continue
		}
		rs = append(rs, Route{Prefix: p, Gateway: f[1], Iface: f[3]})
	}
	return rs
}

// darwinDest reads netstat's destination column: "default", "100.92/16",
// "10.77.0.4" (a host), "fd7a:115c:a1e0::/48", "fe80::%utun0/64".
func darwinDest(s string) (netip.Prefix, bool) {
	if s == "default" {
		return netip.Prefix{}, false
	}
	if i := strings.IndexByte(s, '%'); i >= 0 {
		rest := s[i:]
		j := strings.IndexByte(rest, '/')
		if j < 0 {
			s = s[:i]
		} else {
			s = s[:i] + rest[j:]
		}
	}
	addr, bits, hasBits := strings.Cut(s, "/")
	implied := -1
	if !strings.Contains(addr, ":") {
		// netstat abbreviates IPv4 networks: "100.92/16"; without a length
		// the octets given are the network ("192.168.1" = /24, "127" = /8)
		parts := strings.Split(addr, ".")
		if len(parts) < 4 {
			implied = len(parts) * 8
		}
		for len(parts) < 4 {
			parts = append(parts, "0")
		}
		addr = strings.Join(parts, ".")
	}
	a, err := netip.ParseAddr(addr)
	if err != nil {
		return netip.Prefix{}, false
	}
	n := a.BitLen()
	if implied >= 0 {
		n = implied
	}
	if hasBits {
		v, err := strconv.Atoi(bits)
		if err != nil {
			return netip.Prefix{}, false
		}
		n = v
	}
	p, err := a.Prefix(n)
	return p, err == nil
}

// ParseIPRoute reads Linux `ip route show` / `ip -6 route show` lines.
func ParseIPRoute(out string) []Route {
	var rs []Route
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 3 || f[0] == "default" || f[0] == "unreachable" || f[0] == "blackhole" {
			continue
		}
		if f[0] == "local" || f[0] == "broadcast" || f[0] == "multicast" {
			continue
		}
		p, err := netip.ParsePrefix(f[0])
		if err != nil {
			a, aerr := netip.ParseAddr(f[0])
			if aerr != nil {
				continue
			}
			p = netip.PrefixFrom(a, a.BitLen())
		}
		r := Route{Prefix: p}
		for i := 1; i+1 < len(f); i++ {
			switch f[i] {
			case "dev":
				r.Iface = f[i+1]
			case "via":
				r.Gateway = f[i+1]
			}
		}
		rs = append(rs, r)
	}
	return rs
}

// TunnelLike: an interface that belongs to some VPN.
func TunnelLike(iface string) bool {
	for _, p := range []string{"utun", "wt", "tailscale", "tun", "wg", "ipsec", "ppp", "zt"} {
		if strings.HasPrefix(iface, p) {
			return true
		}
	}
	return false
}

// Foreign is another VPN's interface and what it owns.
type Foreign struct {
	Name     string   `json:"name"` // NetBird, Tailscale, VPN
	Kind     string   `json:"kind"` // netbird | tailscale | vpn
	Iface    string   `json:"iface"`
	Addrs    []string `json:"addrs,omitempty"`
	Prefixes []string `json:"prefixes,omitempty"` // routes it owns (not link-local)
}

// Uplink is the physical default route.
type Uplink struct {
	Iface   string
	Gateway string
	MTU     int
}

// Input is everything Plan looks at.
type Input struct {
	OurTun       string
	TunnelSubnet netip.Prefix
	// Peers: the tailnet machines' addresses (the server's /api/peers).
	Peers  []netip.Addr
	Routes []Route
	// IfaceAddrs: every interface's addresses (with their prefix length).
	IfaceAddrs       map[string][]netip.Prefix
	TailscaleRunning bool
	NetBird          NetBird
	// NetBirdIPs: NetBird.Endpoints resolved.
	NetBirdIPs []netip.Addr
	WorkMode   string
	Uplink     Uplink
	OurMTU     int
	// Bypass: destinations that always go direct (IP rules, resolved
	// domain rules).
	Bypass []netip.Prefix
	// SplitDNS forces split DNS (client.dns "split"); false = when another
	// VPN is there.
	ForceSplit bool
}

// Plan is what tailbond should have in place.
type Plan struct {
	Foreign []Foreign `json:"foreign"`
	// TunRoutes go to tailbond's tunnel interface.
	TunRoutes []netip.Prefix `json:"tun_routes"`
	// DirectRoutes go via the uplink's gateway.
	DirectRoutes []netip.Prefix `json:"direct_routes"`
	Skipped      []Skip         `json:"skipped,omitempty"`
	SplitDNS     bool           `json:"split_dns"`
	// TailnetDirect: the Tailscale app reaches tailnet machines itself.
	TailnetDirect bool   `json:"tailnet_via_tailscale_app"`
	WorkMode      string `json:"work_vpn"`           // what the user chose
	WorkEffective string `json:"work_vpn_effective"` // what's in force
	// WantMTU > 0: Bonded needs tailbond's tunnel MTU raised to this.
	WantMTU int `json:"want_mtu,omitempty"`
	// PassIfaces and PassHosts: the kill switch lets these through.
	PassIfaces []string     `json:"pass_ifaces,omitempty"`
	PassHosts  []netip.Addr `json:"pass_hosts,omitempty"`
	Warnings   []string     `json:"warnings,omitempty"`
	// Conflicts: found in the table, not caused by this plan (health).
	Conflicts []string `json:"conflicts,omitempty"`
}

// Skip is a route Plan left out, and why.
type Skip struct {
	Prefix string `json:"prefix"`
	Reason string `json:"reason"`
}

func host(a netip.Addr) netip.Prefix { return netip.PrefixFrom(a, a.BitLen()) }

// covers: p equals or contains q (a route for p would take q's traffic or
// be q itself).
func covers(p, q netip.Prefix) bool {
	return p.Addr().Is4() == q.Addr().Is4() && p.Bits() <= q.Bits() && p.Contains(q.Addr())
}

func linkLocal(p netip.Prefix) bool {
	for _, l := range clatOrLink {
		if l.Overlaps(p) {
			return true
		}
	}
	return p.Addr().IsMulticast() || p.Addr().IsLoopback()
}

// Make decides (pure).
func Make(in Input) Plan {
	pl := Plan{WorkMode: in.WorkMode, WorkEffective: in.WorkMode}
	if pl.WorkMode == "" {
		pl.WorkMode, pl.WorkEffective = WorkDirect, WorkDirect
	}
	// other VPNs: tunnel-like interfaces that aren't ours, with routes or
	// addresses outside link-local
	byIface := map[string]*Foreign{}
	var owned []Route // routes other VPNs own
	nbAddr, _ := netip.ParsePrefix(in.NetBird.Addr)
	for _, r := range in.Routes {
		if r.Iface == "" || r.Iface == in.OurTun || !TunnelLike(r.Iface) || linkLocal(r.Prefix) || r.Prefix.Bits() == 0 {
			continue
		}
		owned = append(owned, r)
		f := byIface[r.Iface]
		if f == nil {
			f = &Foreign{Iface: r.Iface}
			byIface[r.Iface] = f
		}
		f.Prefixes = append(f.Prefixes, r.Prefix.String())
	}
	for ifn, addrs := range in.IfaceAddrs {
		if ifn == in.OurTun || !TunnelLike(ifn) {
			continue
		}
		for _, a := range addrs {
			if linkLocal(a) {
				continue
			}
			f := byIface[ifn]
			if f == nil {
				f = &Foreign{Iface: ifn}
				byIface[ifn] = f
			}
			f.Addrs = append(f.Addrs, a.String())
		}
	}
	nbIface := in.NetBird.Iface
	for ifn, f := range byIface {
		switch {
		case ifn == nbIface || (nbAddr.IsValid() && slices.Contains(f.Addrs, nbAddr.String())):
			f.Name, f.Kind = "NetBird", "netbird"
			nbIface = ifn
		case in.TailscaleRunning && ownsTailnet(f):
			f.Name, f.Kind = "Tailscale", "tailscale"
		case ownsTailnet(f):
			f.Name, f.Kind = "Tailscale", "tailscale"
		default:
			f.Name, f.Kind = "VPN", "vpn"
		}
		pl.Foreign = append(pl.Foreign, *f)
	}
	sort.Slice(pl.Foreign, func(i, j int) bool { return pl.Foreign[i].Iface < pl.Foreign[j].Iface })
	tailscaleApp := in.TailscaleRunning
	for _, f := range pl.Foreign {
		if f.Kind == "tailscale" {
			tailscaleApp = true
		}
	}
	otherVPN := len(pl.Foreign) > 0 || in.NetBird.LoggedIn
	pl.SplitDNS = in.ForceSplit || otherVPN || in.TailscaleRunning

	// foreign addresses: another VPN's own IPs are never ours to route
	foreignAddr := map[netip.Addr]bool{}
	for _, f := range pl.Foreign {
		for _, a := range f.Addrs {
			if p, err := netip.ParsePrefix(a); err == nil {
				foreignAddr[p.Addr()] = true
			}
		}
	}
	allowed := func(p netip.Prefix, what string) bool {
		for _, o := range owned {
			if covers(p, o.Prefix) {
				pl.Skipped = append(pl.Skipped, Skip{p.String(), fmt.Sprintf("%s: %s owns %s", what, o.Iface, o.Prefix)})
				return false
			}
		}
		if p.IsSingleIP() && foreignAddr[p.Addr()] {
			pl.Skipped = append(pl.Skipped, Skip{p.String(), what + ": another VPN's own address"})
			return false
		}
		return true
	}

	// tailnet machines: host routes, none while the Tailscale app runs
	if tailscaleApp {
		pl.TailnetDirect = true
	} else {
		seen := map[netip.Addr]bool{}
		for _, a := range in.Peers {
			a = a.Unmap()
			if seen[a] || !(cgnat.Contains(a) || tailnet6.Contains(a)) {
				continue
			}
			seen[a] = true
			if p := host(a); allowed(p, "tailnet machine") {
				pl.TunRoutes = append(pl.TunRoutes, p)
			}
		}
	}
	if in.TunnelSubnet.IsValid() && allowed(in.TunnelSubnet.Masked(), "tunnel subnet") {
		pl.TunRoutes = append(pl.TunRoutes, in.TunnelSubnet.Masked())
	}

	// conflicts already in the table: a route on our tunnel that equals or
	// covers another VPN's (the catch-alls 0/1, 128/1 excepted: longer
	// prefixes win over them)
	for _, r := range in.Routes {
		if r.Iface != in.OurTun || r.Prefix.Bits() <= 1 {
			continue
		}
		for _, o := range owned {
			if covers(r.Prefix, o.Prefix) {
				pl.Conflicts = append(pl.Conflicts, fmt.Sprintf("%s on %s covers %s's %s", r.Prefix, in.OurTun, o.Iface, o.Prefix))
			}
		}
	}

	// the work VPN's own traffic
	if nbIface != "" || in.NetBird.LoggedIn {
		if nbIface != "" {
			pl.PassIfaces = append(pl.PassIfaces, nbIface)
		}
		if pl.WorkMode == WorkBonded {
			nbMTU := in.NetBird.MTU
			if nbMTU <= 0 {
				nbMTU = 1280 // NetBird's default
			}
			need := nbMTU + BondedOverhead
			switch {
			case in.OurMTU >= need:
			case in.Uplink.MTU > 0 && in.Uplink.MTU-TailbondOverhead >= need:
				pl.WantMTU = need
			default:
				pl.WorkEffective = WorkDirect
				pl.Warnings = append(pl.Warnings, fmt.Sprintf(
					"Bonded isn't possible on this connection: NetBird's packets (%d bytes) plus tailbond's wrapping (%d) are more than this connection carries inside tailbond (%d). Work traffic goes Direct.",
					nbMTU, BondedOverhead, max(in.Uplink.MTU-TailbondOverhead, 0)))
			}
		}
		if pl.WorkEffective == WorkDirect {
			for _, a := range in.NetBirdIPs {
				a = a.Unmap()
				if !a.Is4() || onLink(in.IfaceAddrs, a) || a.IsLoopback() {
					continue // on the LAN already; IPv6 underlay keeps the system's v6 route
				}
				p := host(a)
				if allowed(p, "work VPN endpoint") && !slices.Contains(pl.DirectRoutes, p) {
					pl.DirectRoutes = append(pl.DirectRoutes, p)
				}
			}
			pl.PassHosts = append(pl.PassHosts, in.NetBirdIPs...)
		}
	}
	for _, f := range pl.Foreign {
		if f.Kind != "netbird" && !slices.Contains(pl.PassIfaces, f.Iface) {
			pl.PassIfaces = append(pl.PassIfaces, f.Iface)
		}
	}
	// bypass rules: direct via the uplink
	for _, p := range in.Bypass {
		p = p.Masked()
		if !p.Addr().Is4() || slices.Contains(pl.DirectRoutes, p) {
			continue
		}
		if allowed(p, "bypass rule") {
			pl.DirectRoutes = append(pl.DirectRoutes, p)
		}
	}
	if len(pl.DirectRoutes) > 0 && in.Uplink.Gateway == "" {
		pl.Warnings = append(pl.Warnings, "No physical uplink with a gateway: direct routes (work VPN, bypass rules) aren't added.")
		pl.DirectRoutes = nil
	}
	sortPrefixes(pl.TunRoutes)
	sortPrefixes(pl.DirectRoutes)
	return pl
}

func ownsTailnet(f *Foreign) bool {
	for _, s := range f.Prefixes {
		if p, err := netip.ParsePrefix(s); err == nil && (p == cgnat || p == tailnet6) {
			return true
		}
	}
	for _, s := range f.Addrs {
		if p, err := netip.ParsePrefix(s); err == nil && p.Bits() == 32 && cgnat.Contains(p.Addr()) {
			return true // Tailscale's utun: a /32 in 100.64/10
		}
	}
	return false
}

func onLink(ifs map[string][]netip.Prefix, a netip.Addr) bool {
	for ifn, ps := range ifs {
		if TunnelLike(ifn) {
			continue
		}
		for _, p := range ps {
			if p.Bits() < p.Addr().BitLen() && p.Masked().Contains(a) {
				return true
			}
		}
	}
	return false
}

func sortPrefixes(ps []netip.Prefix) {
	sort.Slice(ps, func(i, j int) bool {
		if c := ps[i].Addr().Compare(ps[j].Addr()); c != 0 {
			return c < 0
		}
		return ps[i].Bits() < ps[j].Bits()
	})
}

// DNSDomains are the split-DNS domains for the tunnel's resolver: ts.net and
// the tailnet's suffix (when not under ts.net).
func DNSDomains(suffix string) []string {
	out := []string{"ts.net"}
	s := strings.Trim(strings.ToLower(suffix), ".")
	if s != "" && s != "ts.net" && !strings.HasSuffix(s, ".ts.net") {
		out = append(out, s)
	}
	return out
}
