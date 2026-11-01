package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// Setup health (GET /api/health, docs/architecture.md): a few checks that
// decide whether tailbond and Tailscale run at their best, each with a
// one-line fix and a link to Tailscale's own docs.

const (
	urlConnTypes    = "https://tailscale.com/docs/reference/connection-types"
	urlFwPorts      = "https://tailscale.com/docs/reference/faq/firewall-ports"
	urlFirewalls    = "https://tailscale.com/docs/integrations/firewalls"
	urlDeviceConn   = "https://tailscale.com/docs/reference/device-connectivity"
	urlSubnetRtrs   = "https://tailscale.com/docs/features/subnet-routers"
	urlTroubleshoot = "https://tailscale.com/docs/reference/troubleshooting"
)

const (
	hOK      = "ok"
	hWarn    = "warn"
	hBad     = "bad"
	hUnknown = "unknown"
)

type healthCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"` // ok | warn | bad | unknown
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`     // one sentence: what to do
	FixURL string `json:"fix_url,omitempty"` // Tailscale's docs
}

type healthView struct {
	OK       bool                `json:"all_good"` // nothing to fix (unknowns allowed)
	Checks   []healthCheck       `json:"checks"`
	Direct   int                 `json:"direct"`
	Relayed  int                 `json:"relayed"`
	Relays   []string            `json:"relayed_peers,omitempty"`
	Netcheck *tailscale.Netcheck `json:"netcheck,omitempty"`
}

// outsideSeen: the last client that reached the server from the internet.
type outsideSeen struct {
	Name string
	IP   string
	At   time.Time
}

type healthInputs struct {
	Role     string
	TS       *tailscale.Info
	Netcheck *tailscale.Netcheck
	WANIP    string // this machine's own address toward the internet
	Outside  *outsideSeen
	Port     int
	Route    string
	ServerTS string // client role: the paired server's tailnet name
	// server: tagged (no owner), no approvers, no password
	OpenApproval bool
	Now          time.Time
}

// lanIP is ip when it's a home-network address (what a port forward points at).
func lanIP(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil || !a.IsPrivate() {
		return ""
	}
	return a.String()
}

func isPrivate(ip netip.Addr) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// directFix: how to get direct Tailscale connections to this machine.
func directFix(lan string) string {
	to := "this machine"
	if lan != "" {
		to += " (" + lan + ")"
	}
	return "On your home router's admin page (Port forwarding, also called Virtual server or NAT), forward UDP port 41641 to " + to + ". " +
		"(Turning on UPnP on that router also works, but lets any device on your network open ports.)"
}

func buildHealth(in healthInputs) healthView {
	v := healthView{Checks: []healthCheck{}}
	ts := in.TS
	if ts == nil {
		ts = &tailscale.Info{}
	}
	add := func(c healthCheck) { v.Checks = append(v.Checks, c) }

	if !ts.Running {
		add(healthCheck{ID: "tailscale", Status: hBad, Title: "Tailscale is off on this machine",
			Fix: "Turn Tailscale on and sign in.", FixURL: urlTroubleshoot})
	} else {
		// direct vs relayed, among peers with an active connection
		var relays []string
		for _, p := range ts.PeerList {
			if in.Role == config.RoleClient && in.ServerTS != "" && !strings.EqualFold(p.HostName, in.ServerTS) && !strings.HasPrefix(strings.ToLower(p.DNSName), strings.ToLower(in.ServerTS)+".") {
				continue
			}
			switch p.Conn {
			case "direct":
				v.Direct++
			case "relay":
				v.Relayed++
				relays = append(relays, p.HostName)
			}
		}
		sort.Strings(relays)
		v.Relays = relays
		c := healthCheck{ID: "tailscale_direct", FixURL: urlConnTypes}
		switch {
		case v.Relayed > 0:
			c.Status = hWarn
			c.Title = fmt.Sprintf("Tailscale: %s direct, %d through a relay (slower)", plural(v.Direct, "machine"), v.Relayed)
			those := "that machine"
			if v.Relayed > 1 {
				those = "those machines"
			}
			c.Detail = "Through a relay: " + strings.Join(relays, ", ") + ". Only traffic between this machine and " + those +
				" is slower (including bonded devices reaching it); bonding itself doesn't go through Tailscale. The cause can also be on that machine's own network."
			c.Fix = directFix(lanIP(in.WANIP))
		case v.Direct > 0:
			c.Status, c.Title = hOK, fmt.Sprintf("Tailscale: %s direct", plural(v.Direct, "machine"))
		default:
			c.Status, c.Title = hUnknown, "Tailscale: no active connections to check right now"
		}
		if in.Role == config.RoleClient {
			c.Title = strings.Replace(c.Title, "Tailscale:", "Tailscale to your home server:", 1)
		}
		add(c)
	}

	if n := in.Netcheck; n != nil {
		maps := []string{}
		for name, on := range map[string]bool{"UPnP": n.UPnP, "NAT-PMP": n.PMP, "PCP": n.PCP} {
			if on {
				maps = append(maps, name)
			}
		}
		sort.Strings(maps)
		pm := "no port mapping on the router"
		if len(maps) > 0 {
			pm = "port mapping: " + strings.Join(maps, ", ")
		}
		switch {
		case !n.UDP:
			add(healthCheck{ID: "netcheck", Status: hBad, Title: "UDP is blocked on this network: Tailscale goes through relays",
				Detail: pm, Fix: "Allow outbound UDP (Tailscale uses 41641 and 3478).", FixURL: urlFwPorts})
		case n.MappingVariesByDestIP:
			add(healthCheck{ID: "netcheck", Status: hWarn, Title: "This network makes direct Tailscale connections hard",
				Detail: "The router in front of this machine picks a new outside port for every destination (\"hard NAT\"), so other machines can't connect straight back; " + pm + ".", Fix: directFix(lanIP(in.WANIP)), FixURL: urlFirewalls})
		default:
			add(healthCheck{ID: "netcheck", Status: hOK, Title: "Network is friendly to direct connections"})
		}
	}

	if in.Role != config.RoleServer {
		v.OK = healthOK(v.Checks)
		return v
	}

	if ts.Running && in.Route != "" {
		if ts.RoutesApproved {
			add(healthCheck{ID: "route", Status: hOK, Title: "Optional route " + in.Route + " approved: your Tailscale machines can reach tailbond devices"})
		} else {
			add(healthCheck{ID: "route", Status: hWarn, Title: "Route " + in.Route + " isn't approved yet (optional)",
				Detail: "Devices connected through tailbond reach your Tailscale machines without it. Approving it lets your Tailscale machines reach those devices too, under your access rules.",
				Fix: "If you want that: Tailscale admin console → Machines → this machine → Edit route settings (an autoApprovers rule in your policy approves it automatically). " +
					"If you do not want it: leave it, or add \"no_advertise_route\": true inside the \"server\" section of /etc/tailbond/tailbond.json, then restart tailbond (Fedora: sudo systemctl restart tailbond; GL.iNet: /etc/init.d/tailbond restart) to stop advertising it.", FixURL: urlSubnetRtrs})
		}
	}

	// carrier-grade NAT on the home line: nothing from outside can reach UDP 443
	if ip, err := netip.ParseAddr(in.WANIP); err == nil {
		ip = ip.Unmap()
		pub := ""
		if in.Netcheck != nil && in.Netcheck.GlobalV4 != "" {
			if ap, err := netip.ParseAddrPort(in.Netcheck.GlobalV4); err == nil {
				pub = ap.Addr().String()
			}
		}
		switch {
		case cgnat.Contains(ip):
			add(healthCheck{ID: "cgnat", Status: hBad, Title: "Your internet line uses carrier-grade NAT",
				Detail: "This machine's internet address " + ip.String() + " is shared by your provider, so travel routers and phones away from home can't reach UDP " + fmt.Sprint(in.Port) + " here. " +
					"Bonding needs that direct connection; it can't run through Tailscale's relays (they'd make it slower than a single link).",
				Fix: "Ask your provider for a public IPv4 address (often free or a small monthly fee; say it's for reaching your home from outside). " +
					"Advanced: a public IPv6 address also works, but only from networks that have IPv6 too (most mobile networks, few hotels).", FixURL: urlDeviceConn})
		case isPrivate(ip) && pub != "":
			add(healthCheck{ID: "cgnat", Status: hOK, Title: "Behind your home router (public address " + pub + ")",
				Detail: "Forward UDP " + fmt.Sprint(in.Port) + " on that router to " + ip.String() + "."})
		case isPrivate(ip):
			add(healthCheck{ID: "cgnat", Status: hUnknown, Title: "Behind your home router",
				Detail: "Forward UDP " + fmt.Sprint(in.Port) + " on that router to " + ip.String() + "."})
		default:
			add(healthCheck{ID: "cgnat", Status: hOK, Title: "Public internet address " + ip.String()})
		}
	}

	if o := in.Outside; o != nil {
		add(healthCheck{ID: "port", Status: hOK, Title: fmt.Sprintf("UDP %d reached from the internet", in.Port),
			Detail: fmt.Sprintf("%s connected from %s, %s.", o.Name, o.IP, ago(in.Now.Sub(o.At)))})
	} else {
		add(healthCheck{ID: "port", Status: hUnknown, Title: fmt.Sprintf("UDP %d: not confirmed from outside yet", in.Port),
			Detail: "It can't be checked from inside your network; the first device that connects from away confirms it. Test it now: a phone with tailbond, Wi-Fi off (on mobile data), confirms it within a minute."})
	}
	if in.OpenApproval {
		add(healthCheck{ID: "approvers", Status: hWarn, Title: "Anyone on your Tailscale network can approve devices",
			Detail: "This machine is tagged, so it has no owner account, and no approvers or password are set.",
			Fix:    "Name who may approve, or set a password: Devices → Who can approve and add devices."})
	}
	v.OK = healthOK(v.Checks)
	return v
}

func healthOK(cs []healthCheck) bool {
	for _, c := range cs {
		if c.Status == hWarn || c.Status == hBad {
			return false
		}
	}
	return true
}

func plural(n int, w string) string {
	if n == 1 {
		return "1 " + w
	}
	return fmt.Sprintf("%d %ss", n, w)
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}

// noteOutside remembers a client that connected from a public address.
func (a *Agent) noteOutside(clients []status.Client) {
	for _, c := range clients {
		if !c.Connected || c.Endpoint == "" {
			continue
		}
		host, _, err := net.SplitHostPort(c.Endpoint)
		if err != nil {
			host = c.Endpoint
		}
		ip, err := netip.ParseAddr(strings.Trim(host, "[]"))
		if err != nil {
			continue
		}
		ip = ip.Unmap()
		if isPrivate(ip) || cgnat.Contains(ip) || ip.IsUnspecified() {
			continue
		}
		a.mu.Lock()
		a.outside = &outsideSeen{Name: c.Name, IP: ip.String(), At: time.Now()}
		a.mu.Unlock()
	}
}

type netcheckCache struct {
	mu      sync.Mutex
	res     *tailscale.Netcheck
	at      time.Time
	running bool
}

// netcheck returns the cached result and refreshes it in the background
// at most every 10 minutes.
func (a *Agent) netcheck() *tailscale.Netcheck {
	a.nc.mu.Lock()
	defer a.nc.mu.Unlock()
	if !a.nc.running && time.Since(a.nc.at) > 10*time.Minute {
		a.nc.running = true
		go func() {
			n, err := a.ts.Netcheck(context.Background())
			a.nc.mu.Lock()
			if err == nil {
				a.nc.res = n
			}
			a.nc.at, a.nc.running = time.Now(), false
			a.nc.mu.Unlock()
		}()
	}
	return a.nc.res
}

func (a *Agent) hHealth(w http.ResponseWriter, r *http.Request) {
	cfg := a.store.Get()
	in := healthInputs{Role: a.role, TS: a.ts.Get(r.Context()), Netcheck: a.netcheck(), Port: cfg.Server.Port, Now: time.Now()}
	if a.role == config.RoleServer {
		in.WANIP = outboundIP()
		if !cfg.Server.NoAdvertiseRoute {
			in.Route = cfg.Server.Subnet
		}
		in.OpenApproval = in.TS.Running && in.TS.Owner == "" && len(cfg.Approvers) == 0 && cfg.UIPassword == ""
		a.mu.Lock()
		in.Outside = a.outside
		a.mu.Unlock()
	} else {
		a.mu.Lock()
		if p := a.client.payload; p != nil {
			in.ServerTS, _, _ = strings.Cut(p.SNI, ".")
		}
		a.mu.Unlock()
	}
	writeJSON(w, http.StatusOK, buildHealth(in))
}
