// Package tailscale wraps the optional `tailscale` CLI. Every call degrades to
// "not installed / not running" when the binary or daemon is missing.
package tailscale

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Info struct {
	Installed        bool     `json:"installed"`
	Running          bool     `json:"running"`
	BackendState     string   `json:"backend_state,omitempty"`
	Self             string   `json:"self"`
	DNSName          string   `json:"dns_name,omitempty"`
	MagicDNSSuffix   string   `json:"magic_dns_suffix,omitempty"`
	Tailnet          string   `json:"tailnet,omitempty"`
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
}

type CLI struct {
	Bin      string
	Disabled bool
	// Route is the subnet whose approval state is reported.
	Route string

	mu      sync.Mutex
	cached  *Info
	fetched time.Time
}

func (c *CLI) bin() string {
	if c.Bin == "" {
		return "tailscale"
	}
	return c.Bin
}

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
		HostName      string
		DNSName       string
		TailscaleIPs  []string
		PrimaryRoutes []string
		AllowedIPs    []string
	}
	Peer map[string]struct {
		Online bool
	}
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

// Invalidate drops the cache (after route changes).
func (c *CLI) Invalidate() {
	c.mu.Lock()
	c.cached = nil
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
	var st statusJSON
	if err := json.Unmarshal(out, &st); err != nil {
		info.Error = "parse tailscale status: " + err.Error()
		return info
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
	for _, p := range st.Peer {
		info.Peers++
		if p.Online {
			info.PeersOnline++
		}
	}
	if st.Self != nil {
		info.Self = st.Self.HostName
		info.DNSName = strings.TrimSuffix(st.Self.DNSName, ".")
		info.IPs = st.Self.TailscaleIPs
		approved := map[string]bool{}
		for _, r := range append(append([]string{}, st.Self.PrimaryRoutes...), st.Self.AllowedIPs...) {
			if pfx, err := netip.ParsePrefix(r); err == nil && pfx.Bits() < pfx.Addr().BitLen() {
				if !approved[pfx.String()] {
					approved[pfx.String()] = true
					info.ApprovedRoutes = append(info.ApprovedRoutes, pfx.String())
				}
			}
		}
		info.RoutesApproved = c.Route != "" && approved[canon(c.Route)]
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
