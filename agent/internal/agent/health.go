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
	Now      time.Time
}

func isPrivate(ip netip.Addr) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

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
			c.Detail = "Through a relay: " + strings.Join(relays, ", ") + ". To check one: tailscale ping " + relays[0]
			c.Fix = "Usually fixed by forwarding UDP port 41641 to this machine, or by turning on UPnP on the router in front of it."
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
				Detail: "Hard NAT (the outside port changes per destination); " + pm + ".", Fix: "Forward UDP port 41641 to this machine, or turn on UPnP on the router in front of it.", FixURL: urlFirewalls})
		default:
			add(healthCheck{ID: "netcheck", Status: hOK, Title: "Network is friendly to direct connections", Detail: pm})
		}
	}

	if in.Role != config.RoleServer {
		v.OK = healthOK(v.Checks)
		return v
	}

	if ts.Running && in.Route != "" {
		if ts.RoutesApproved {
			add(healthCheck{ID: "route", Status: hOK, Title: "Route " + in.Route + " approved"})
		} else {
			add(healthCheck{ID: "route", Status: hWarn, Title: "Route " + in.Route + " isn't approved yet",
				Fix: "Approve it once in the Tailscale admin console (Machines → this machine → Edit route settings).", FixURL: urlSubnetRtrs})
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
				Detail: "This machine's internet address " + ip.String() + " is shared by your provider, so travel routers can't reach UDP " + fmt.Sprint(in.Port) + " here.",
				Fix:    "Ask your provider for a public IPv4 address (or use public IPv6).", FixURL: urlDeviceConn})
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
			Detail: "It can't be checked from inside your network; the first device that connects from away confirms it."})
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
		in.WANIP, in.Route = outboundIP(), cfg.Server.Subnet
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
