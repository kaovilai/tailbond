package agent

// Living next to other VPNs (docs/macos.md "Using tailbond with NetBird",
// docs/architecture.md "Living next to other VPNs"): on a Mac or a Linux
// client, tailbond leaves NetBird's, the Tailscale app's, any VPN's routes
// and DNS alone (internal/coexist decides, this file feeds it and applies
// the plan). Not on OpenWrt: the router's tailnet routes, DNS and bypass are
// its hooks' (and phones' VPN apps never run next to it).

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/coexist"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

const (
	coexistEvery   = 5 * time.Second
	coexistRefresh = 30 * time.Second // the NetBird CLI, name lookups
)

// coexistState is guarded by Agent.mu (in clientState).
type coexistState struct {
	plan       coexist.Plan
	nb         coexist.NetBird
	detectedAt time.Time
	nbIPs      []netip.Addr
	bypassIPs  []netip.Prefix
	applied    bool
	err        string
	// what the running mqvpn was started with
	mqvpnSplit bool
	mqvpnMTU   int
}

func (a *Agent) coexistOn() bool {
	if a.role != config.RoleClient || a.client == nil {
		return false
	}
	if goruntime.GOOS != "darwin" && (goruntime.GOOS != "linux" || isOpenWrt()) {
		return false
	}
	return a.store.Get().Client.Coexist != config.CoexistOff
}

func (a *Agent) coexistApplier() *coexist.Applier {
	return &coexist.Applier{OS: goruntime.GOOS, RecordPath: filepath.Join(a.store.Get().StateDir, "coexist.json"),
		ResolverDir: a.cxResolverDir, Run: a.cxRun, Logf: a.Logf}
}

// coexistRestore takes away what an earlier run left (crash), or what this
// run added (stop). Before mqvpn starts, like the phone hub's proxy restore.
func (a *Agent) coexistRestore() {
	if !a.coexistOn() {
		return
	}
	ap := a.coexistApplier()
	if ap.Current().OS == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ap.Restore(ctx); err != nil {
		a.Logf("coexist: restore: %v", err)
	} else {
		a.Logf("coexist: took away the routes and resolver files tailbond had added")
	}
	a.mu.Lock()
	if a.client.cx != nil {
		a.client.cx.applied = false
	}
	a.mu.Unlock()
}

func (a *Agent) coexistWatcher(ctx context.Context) {
	if !a.coexistOn() {
		return
	}
	t := time.NewTicker(coexistEvery)
	defer t.Stop()
	for {
		a.coexistRound(ctx)
		select {
		case <-ctx.Done():
			return // Run restores on the way out
		case <-t.C:
		}
	}
}

func (a *Agent) cxCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	if a.cxRun != nil {
		return a.cxRun(ctx, name, args...)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// netbirdBin finds the NetBird CLI (launchd's PATH is short).
func netbirdBin() string {
	for _, p := range []string{"netbird", "/usr/local/bin/netbird", "/opt/homebrew/bin/netbird", "/usr/bin/netbird"} {
		if lp, err := exec.LookPath(p); err == nil {
			return lp
		}
	}
	return ""
}

func (a *Agent) detectNetBird(ctx context.Context, ifs map[string][]netip.Prefix) coexist.NetBird {
	bin := netbirdBin()
	if bin == "" {
		return coexist.NetBird{}
	}
	out, err := a.cxCommand(ctx, bin, "status", "--json")
	var nb coexist.NetBird
	if err != nil || !json.Valid(out) {
		nb = coexist.NetBirdFromError(out, err)
	} else if nb, err = coexist.ParseNetBirdStatus(out); err != nil {
		nb = coexist.NetBirdFromError(out, err)
	}
	if p, err := netip.ParsePrefix(nb.Addr); err == nil {
		for ifn, as := range ifs {
			for _, x := range as {
				if x.Addr() == p.Addr() {
					nb.Iface = ifn
				}
			}
		}
		if ifi, err := net.InterfaceByName(nb.Iface); err == nil {
			nb.MTU = ifi.MTU
		}
	}
	return nb
}

func ifaceAddrs() map[string][]netip.Prefix {
	out := map[string][]netip.Prefix{}
	ifs, _ := net.Interfaces()
	for _, ifi := range ifs {
		as, _ := ifi.Addrs()
		for _, x := range as {
			if n, ok := x.(*net.IPNet); ok {
				if p, err := netip.ParsePrefix(n.String()); err == nil {
					a, _ := netip.AddrFromSlice(n.IP)
					out[ifi.Name] = append(out[ifi.Name], netip.PrefixFrom(a.Unmap(), p.Bits()))
				}
			}
		}
	}
	return out
}

// routeTable and the physical uplink (the default route not on a tunnel).
func (a *Agent) routeTable(ctx context.Context) ([]coexist.Route, coexist.Uplink) {
	var up coexist.Uplink
	if goruntime.GOOS == "darwin" {
		out, _ := a.cxCommand(ctx, "netstat", "-rn")
		for _, l := range strings.Split(string(out), "\n") {
			f := strings.Fields(l)
			if len(f) >= 4 && f[0] == "default" && up.Iface == "" && !coexist.TunnelLike(f[3]) && !strings.HasPrefix(f[1], "link#") && !strings.Contains(f[1], ":") {
				up.Gateway, up.Iface = f[1], f[3]
			}
		}
		return coexist.ParseNetstat(string(out)), a.withMTU(up)
	}
	out4, _ := a.cxCommand(ctx, "ip", "-4", "route", "show", "table", "main")
	out6, _ := a.cxCommand(ctx, "ip", "-6", "route", "show", "table", "main")
	for _, l := range strings.Split(string(out4), "\n") {
		f := strings.Fields(l)
		if len(f) >= 5 && f[0] == "default" && f[1] == "via" && up.Iface == "" {
			for i := 2; i+1 < len(f); i++ {
				if f[i] == "dev" && !coexist.TunnelLike(f[i+1]) {
					up.Gateway, up.Iface = f[2], f[i+1]
				}
			}
		}
	}
	return append(coexist.ParseIPRoute(string(out4)), coexist.ParseIPRoute(string(out6))...), a.withMTU(up)
}

func (a *Agent) withMTU(up coexist.Uplink) coexist.Uplink {
	if ifi, err := net.InterfaceByName(up.Iface); err == nil {
		up.MTU = ifi.MTU
	}
	return up
}

// tailnetPeers: the tailnet machines' addresses from the server's machine
// list (GET /api/peers, cached by the peers fetch).
func (a *Agent) tailnetPeers() ([]netip.Addr, string) {
	a.mu.Lock()
	raw := a.client.peers
	a.mu.Unlock()
	var v peersView
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return nil, ""
	}
	var out []netip.Addr
	nodes := v.Peers
	if v.Self != nil {
		nodes = append(nodes, *v.Self)
	}
	for _, n := range nodes {
		for _, s := range n.TailscaleIPs {
			if ip, err := netip.ParseAddr(s); err == nil {
				out = append(out, ip)
			}
		}
	}
	return out, v.MagicDNSSuffix
}

func resolveHosts(ctx context.Context, hosts []string) []netip.Addr {
	var mu sync.Mutex
	var out []netip.Addr
	var wg sync.WaitGroup
	for _, h := range hosts {
		if ip, err := netip.ParseAddr(h); err == nil {
			out = append(out, ip)
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			ips, err := net.DefaultResolver.LookupNetIP(c, "ip4", h)
			if err != nil {
				return
			}
			mu.Lock()
			out = append(out, ips...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

// bypassDirect: the bypass rules that apply on this machine (IP ranges and
// domains, target direct); domains resolved.
func (a *Agent) bypassDirect(ctx context.Context, cfg *config.Config) []netip.Prefix {
	var out []netip.Prefix
	var domains []string
	for _, r := range cfg.Client.Bypass {
		if !r.On() || r.Target != config.TargetDirect {
			continue
		}
		switch r.Kind {
		case config.BypassIP:
			if p, err := netip.ParsePrefix(r.Match); err == nil {
				out = append(out, p)
			} else if ip, err := netip.ParseAddr(r.Match); err == nil {
				out = append(out, netip.PrefixFrom(ip, ip.BitLen()))
			}
		case config.BypassDomain:
			domains = append(domains, strings.TrimPrefix(r.Match, "*."))
		}
	}
	for _, ip := range resolveHosts(ctx, domains) {
		out = append(out, netip.PrefixFrom(ip, ip.BitLen()))
	}
	return out
}

func (a *Agent) coexistRound(ctx context.Context) {
	cfg := a.store.Get()
	ifs := ifaceAddrs()
	if a.cxIfaces != nil {
		ifs = a.cxIfaces()
	}
	routes, up := a.routeTable(ctx)
	a.mu.Lock()
	cx := a.client.cx
	if cx == nil {
		cx = &coexistState{}
		a.client.cx = cx
	}
	stale := time.Since(cx.detectedAt) > coexistRefresh
	nb, nbIPs, byIPs := cx.nb, cx.nbIPs, cx.bypassIPs
	tun, ready, p := a.rt.tunName, a.rt.ready, a.client.payload
	a.mu.Unlock()
	if stale {
		nb = a.detectNetBird(ctx, ifs)
		nbIPs = resolveHosts(ctx, nb.Endpoints)
		byIPs = a.bypassDirect(ctx, cfg)
	}
	if ready && p != nil {
		a.mu.Lock()
		old := time.Since(a.client.peersAt) > time.Minute
		a.mu.Unlock()
		if old {
			if b, err := a.serverAPI(ctx, http.MethodGet, "/api/peers", nil); err == nil {
				a.mu.Lock()
				a.client.peers, a.client.peersAt = b, time.Now()
				a.mu.Unlock()
			}
		}
	}
	peers, suffix := a.tailnetPeers()
	tsRunning := false
	if !cfg.DisableTailscale && a.ts != nil {
		tsRunning = a.ts.Get(ctx).Running
	}
	in := coexist.Input{
		OurTun: tun, Peers: peers, Routes: routes, IfaceAddrs: ifs, TailscaleRunning: tsRunning,
		NetBird: nb, NetBirdIPs: nbIPs, WorkMode: cfg.Client.WorkVPNOrDefault(), Uplink: up,
		OurMTU: clientMTU(cfg), Bypass: byIPs, ForceSplit: cfg.Client.SplitDNS,
	}
	if p != nil {
		in.TunnelSubnet = p.TunnelSubnet()
	}
	plan := coexist.Make(in)
	a.mu.Lock()
	if stale {
		cx.nb, cx.nbIPs, cx.bypassIPs, cx.detectedAt = nb, nbIPs, byIPs, time.Now()
	}
	cx.plan = plan
	running := a.sup.Active()
	restart := running && ready && (plan.SplitDNS != cx.mqvpnSplit || (plan.WantMTU > 0 && plan.WantMTU != cx.mqvpnMTU))
	a.mu.Unlock()
	if restart {
		a.Logf("coexist: restarting mqvpn (split DNS %v, MTU %d)", plan.SplitDNS, max(plan.WantMTU, clientMTU(cfg)))
		a.sup.Restart()
		return
	}
	ap := a.coexistApplier()
	if !ready || tun == "" || p == nil {
		// the tunnel is down: nothing of ours should point at it
		if ap.Current().OS != "" {
			a.coexistRestore()
		}
		return
	}
	dnsIP := ""
	if !cfg.Client.NoDNS && len(p.DNS) > 0 {
		dnsIP = p.DNS[0]
	}
	want := coexist.Want(goruntime.GOOS, tun, plan, up, dnsIP, coexist.DNSDomains(suffix), cfg.Client.KillSwitch)
	err := ap.Apply(ctx, want)
	a.mu.Lock()
	cx.applied, cx.err = true, ""
	if err != nil {
		cx.err = err.Error()
	}
	a.mu.Unlock()
}

// coexistPreStart: what mqvpn starts with: global DNS off when another VPN
// is there (split DNS instead), and Bonded's MTU.
func (a *Agent) coexistPreStart(ctx context.Context, cfg *config.Config) (split bool, mtu int) {
	mtu = clientMTU(cfg)
	if !a.coexistOn() {
		return false, mtu
	}
	a.mu.Lock()
	cx := a.client.cx
	a.mu.Unlock()
	if cx == nil {
		a.coexistRound(ctx) // detect before the first start
		a.mu.Lock()
		cx = a.client.cx
		a.mu.Unlock()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if cx == nil {
		return false, mtu
	}
	split = cx.plan.SplitDNS
	if cx.plan.WantMTU > mtu {
		mtu = cx.plan.WantMTU
	}
	cx.mqvpnSplit, cx.mqvpnMTU = split, mtu
	if cx.plan.WantMTU == 0 {
		cx.mqvpnMTU = 0
	}
	return split, mtu
}

func (a *Agent) coexistStatus() *status.Coexist {
	if !a.coexistOn() {
		return nil
	}
	cfg := a.store.Get()
	a.mu.Lock()
	defer a.mu.Unlock()
	cx := a.client.cx
	if cx == nil {
		return nil
	}
	pl := cx.plan
	st := &status.Coexist{
		NetBird: cx.nb, Foreign: []status.CoexistForeign{}, SplitDNS: pl.SplitDNS, TailnetViaTailscaleApp: pl.TailnetDirect,
		WorkVPN: cfg.Client.WorkVPNOrDefault(), WorkVPNEffective: pl.WorkEffective, WantMTU: pl.WantMTU,
		TailnetRoutes: len(pl.TunRoutes), DirectRoutes: len(pl.DirectRoutes), Warnings: pl.Warnings, BondedBlocked: pl.BondedBlocked, Conflicts: pl.Conflicts,
		Error: cx.err,
	}
	for _, f := range pl.Foreign {
		st.Foreign = append(st.Foreign, status.CoexistForeign{Name: f.Name, Kind: f.Kind, Iface: f.Iface, Addrs: f.Addrs, Prefixes: f.Prefixes})
	}
	for _, s := range pl.Skipped {
		st.Skipped = append(st.Skipped, s.Prefix+": "+s.Reason)
	}
	if pl.SplitDNS {
		st.DNSDomains = []string{"ts.net"}
	}
	return st
}

// hCoexist: GET the other VPNs found and what tailbond does about them;
// POST {"work_vpn": "direct"|"bonded", "split_dns": bool}.
func (a *Agent) hCoexist(w http.ResponseWriter, r *http.Request) {
	if !a.coexistOn() {
		httpErr(w, http.StatusNotFound, errors.New("only on a Mac or a Linux client"))
		return
	}
	if r.Method == http.MethodPost {
		var in struct {
			WorkVPN  *string `json:"work_vpn"`
			SplitDNS *bool   `json:"split_dns"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&in); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
		if _, err := a.store.Update(func(c *config.Config) error {
			if in.WorkVPN != nil {
				c.Client.WorkVPN = *in.WorkVPN
			}
			if in.SplitDNS != nil {
				c.Client.SplitDNS = *in.SplitDNS
			}
			return nil
		}); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
		a.coexistRound(r.Context())
	}
	writeJSON(w, http.StatusOK, a.coexistStatus())
}
