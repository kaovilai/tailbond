package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/dnsfwd"
	"github.com/kaovilai/tailbond/agent/internal/hooks"
	"github.com/kaovilai/tailbond/agent/internal/mqconf"
	"github.com/kaovilai/tailbond/agent/internal/mqctl"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/supervisor"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
	"github.com/kaovilai/tailbond/agent/internal/tlsutil"
)

func (a *Agent) startServer(ctx context.Context) {
	cfg := a.store.Get()
	if !cfg.Server.DisableDNS {
		go a.runDNS(ctx)
	}
	go a.certRenewer(ctx)
	a.netcheck() // warm the cache: the public address comes from it
	if goruntime.GOOS == "darwin" {
		go a.tailscaleWatch(ctx)
	}
	a.sup.Start()
}

// runServerUp queues the server-up hook and records the Tailscale state it
// ran with (tailscaleWatch compares against it).
func (a *Agent) runServerUp(env map[string]string) {
	a.queueHook(func() {
		sig := a.tailscaleSig(context.Background())
		a.hooks.Run(hooks.ServerUp, env)
		a.mu.Lock()
		if a.rt.upHookFired {
			a.rt.tsApplied = sig
		}
		a.mu.Unlock()
	})
}

// tailscaleSig: Tailscale's IPv4 address and the interface holding it, as
// the server's network rules depend on them ("off", "none" without one).
func (a *Agent) tailscaleSig(ctx context.Context) string {
	if a.store.Get().DisableTailscale {
		return "off"
	}
	ti := a.ts.Get(ctx)
	if !ti.Running {
		return "none"
	}
	for _, s := range ti.IPs {
		ip, err := netip.ParseAddr(s)
		if err != nil || !ip.Is4() {
			continue
		}
		return ip.String() + "@" + ifaceOf(ip)
	}
	return "none"
}

// ifaceOf: the interface with this address ("" if none here).
func ifaceOf(ip netip.Addr) string {
	ifs, _ := net.Interfaces()
	for _, ni := range ifs {
		addrs, _ := ni.Addrs()
		for _, ad := range addrs {
			if p, err := netip.ParsePrefix(ad.String()); err == nil && p.Addr() == ip {
				return ni.Name
			}
		}
	}
	return ""
}

// tailscaleWatch (macOS): when Tailscale starts after the server (the app
// starts at login, tailbondd at boot) or its utun is recreated, the
// server-up hook runs again (TB_REFRESH=1), so the tailnet NAT, the web
// rules and the route advertisement follow it.
func (a *Agent) tailscaleWatch(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.mu.Lock()
		applied, up := a.rt.tsApplied, a.rt.upHookFired
		a.mu.Unlock()
		if !up || applied == "" {
			continue
		}
		cur := a.tailscaleSig(ctx)
		if cur == applied {
			continue
		}
		a.Logf("tailscale changed (%s -> %s): applying the server's network rules again", applied, cur)
		a.mu.Lock()
		if !a.rt.upHookFired {
			a.mu.Unlock()
			continue
		}
		a.rt.tsApplied = cur
		env := a.hookEnvLocked()
		a.mu.Unlock()
		env["TB_REFRESH"] = "1"
		a.runServerUp(env)
	}
}

// runDNS serves the tunnel DNS on the server's tunnel IP, and moves with
// it when the subnet changes (renumberSubnet).
func (a *Agent) runDNS(ctx context.Context) {
	for {
		fctx, cancel := context.WithCancel(ctx)
		f := &dnsfwd.Forwarder{
			Addr: net.JoinHostPort(a.store.Get().ServerTunnelIP(), "53"),
			Suffixes: func() []string {
				ti := a.ts.Get(context.Background())
				if ti.MagicDNSSuffix != "" {
					return []string{ti.MagicDNSSuffix}
				}
				return nil
			},
			TailscaleUp: func() bool { return a.ts.Get(context.Background()).Running },
			Log:         func(s string) { a.Logf("%s", s) },
		}
		go f.Run(fctx)
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-a.subnetMoved:
			cancel()
		}
	}
}

// renumberSubnet moves a server that has no clients yet out of subnets an
// asking client already uses (docs/architecture.md, "Distinct subnets"):
// the first free /24 from 10.78.0.0/24 up that avoids them and every local
// route. mqvpn restarts; its down/up hooks run with the old/new subnet.
func (a *Agent) renumberSubnet(avoid []netip.Prefix) (string, error) {
	cfg := a.store.Get()
	cur, err := netip.ParsePrefix(cfg.Server.Subnet)
	if err != nil {
		return "", err
	}
	clash := false
	for _, x := range avoid {
		if config.Overlaps(cur, x) {
			clash = true
		}
	}
	if !clash {
		return "", nil
	}
	used := append(slices.Clone(avoid), localPrefixes(cur)...)
	next, ok := config.NextFreeSubnet(used)
	if !ok {
		return "", errors.New("no free /24 for the tunnel")
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		if len(c.Server.Users) > 0 {
			return errHasClients
		}
		c.Server.Subnet = next.String()
		return nil
	}); err != nil {
		return "", err
	}
	a.Logf("tunnel subnet %s -> %s (the asking client already uses %s)", cur, next, cur)
	a.ts.SetRoute(next.String())
	select {
	case a.subnetMoved <- struct{}{}:
	default:
	}
	if a.sup.Active() {
		a.sup.Restart()
	}
	return next.String(), nil
}

var errHasClients = errors.New("the server already has clients")

// localPrefixes are this machine's interface subnets and routes, except
// skip (the tunnel's own).
func localPrefixes(skip netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	ifs, _ := net.InterfaceAddrs()
	for _, ad := range ifs {
		if p, err := netip.ParsePrefix(ad.String()); err == nil && p.Addr().Is4() && !p.Addr().IsLoopback() {
			if p = p.Masked(); p != skip {
				out = append(out, p)
			}
		}
	}
	if b, err := os.ReadFile("/proc/net/route"); err == nil {
		for _, line := range strings.Split(string(b), "\n")[1:] {
			fs := strings.Fields(line)
			if len(fs) < 8 || fs[1] == "00000000" {
				continue
			}
			dst, err1 := strconv.ParseUint(fs[1], 16, 32)
			mask, err2 := strconv.ParseUint(fs[7], 16, 32)
			if err1 != nil || err2 != nil {
				continue
			}
			// little-endian hex
			d := [4]byte{byte(dst), byte(dst >> 8), byte(dst >> 16), byte(dst >> 24)}
			m := uint32(mask)
			bits := 0
			for m != 0 {
				bits += int(m & 1)
				m >>= 1
			}
			p := netip.PrefixFrom(netip.AddrFrom4(d), bits).Masked()
			if p != skip {
				out = append(out, p)
			}
		}
	}
	return out
}

// certRenewer refreshes the tailscale cert daily; a changed cert restarts mqvpn.
func (a *Agent) certRenewer(ctx context.Context) {
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.mu.Lock()
		cur := a.tlsInfo
		a.mu.Unlock()
		if cur.source != "tailscale" {
			continue
		}
		next, err := a.prepareTLS(ctx, true)
		if err == nil && next.sha256 != cur.sha256 && a.sup.Active() {
			a.Logf("tailscale certificate renewed, restarting mqvpn")
			a.sup.Restart()
		}
	}
}

// prepareTLS picks the tailscale cert when available, else a self-signed one.
func (a *Agent) prepareTLS(ctx context.Context, forceRefresh bool) (tlsInfo, error) {
	cfg := a.store.Get()
	var info tlsInfo
	if cfg.Server.UseTailscaleCert && !cfg.DisableTailscale {
		ti := a.ts.Get(ctx)
		if ti.Running && ti.DNSName != "" {
			certFile := filepath.Join(cfg.StateDir, "ts-cert.pem")
			keyFile := filepath.Join(cfg.StateDir, "ts-key.pem")
			fresh := false
			if fi, err := os.Stat(certFile); err == nil && time.Since(fi.ModTime()) < 24*time.Hour && !forceRefresh {
				fresh = true
			}
			var err error
			if !fresh {
				err = a.ts.Cert(ctx, ti.DNSName, certFile, keyFile)
				if err != nil {
					a.Logf("tailscale cert for %s unavailable (%v); using self-signed certificate", ti.DNSName, err)
				}
			}
			if err == nil {
				if sha, ferr := tlsutil.FingerprintFile(certFile); ferr == nil {
					info = tlsInfo{certFile: certFile, keyFile: keyFile, sni: ti.DNSName, sha256: sha, source: "tailscale"}
				}
			}
		}
	}
	if info.certFile == "" {
		certFile := filepath.Join(cfg.StateDir, "server.crt")
		keyFile := filepath.Join(cfg.StateDir, "server.key")
		host, _ := os.Hostname()
		names := []string{host}
		if h := hostOnly(cfg.Server.PublicAddr); h != "" {
			names = append(names, h)
		}
		if err := tlsutil.EnsureSelfSigned(certFile, keyFile, names); err != nil {
			return info, fmt.Errorf("self-signed cert: %w", err)
		}
		sha, err := tlsutil.FingerprintFile(certFile)
		if err != nil {
			return info, err
		}
		info = tlsInfo{certFile: certFile, keyFile: keyFile, sni: host, insecure: true, sha256: sha, source: "self-signed"}
	}
	a.mu.Lock()
	a.tlsInfo = info
	a.mu.Unlock()
	return info, nil
}

func hostOnly(addr string) string {
	if addr == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return strings.Trim(addr, "[]")
}

func (a *Agent) serverPreStart(ctx context.Context) error {
	cfg := a.store.Get()
	tls, err := a.prepareTLS(ctx, false)
	if err != nil {
		return err
	}
	users := make([]mqconf.User, 0, len(cfg.Server.Users))
	for _, u := range cfg.Server.Users {
		users = append(users, mqconf.User{Name: u.Name, Key: u.Key})
	}
	global := ""
	if len(users) == 0 {
		// mqvpn refuses to start without a credential; this one is never shared.
		global, err = a.stateSecret("global.key")
		if err != nil {
			return err
		}
	}
	ini, err := mqconf.Server(mqconf.ServerParams{
		Port: cfg.Server.Port, Subnet: cfg.Server.Subnet, Subnet6: cfg.Server.Subnet6,
		TunName: config.DefaultTunName, CertFile: tls.certFile, KeyFile: tls.keyFile,
		Users: users, GlobalKey: global, Mode: cfg.Mode, ControlAddr: cfg.ControlAddr,
		LogLevel: cfg.LogLevel, EgressAllow: config.TailnetRoutes,
	})
	if err != nil {
		return err
	}
	return writeFileAtomic(a.confPath(), []byte(ini), 0o600)
}

// stateSecret is a random secret in the state dir, created on first use.
// Concurrent first callers get the same one.
func (a *Agent) stateSecret(name string) (string, error) {
	a.secretMu.Lock()
	defer a.secretMu.Unlock()
	p := filepath.Join(a.store.Get().StateDir, name)
	if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) > 0 {
		return strings.TrimSpace(string(b)), nil
	}
	k, err := genKey()
	if err != nil {
		return "", err
	}
	return k, writeFileAtomic(p, []byte(k+"\n"), 0o600)
}

func (a *Agent) serverStatus(ctx context.Context) *status.Status {
	cfg := a.store.Get()
	a.mu.Lock()
	ready := a.rt.ready
	tls := a.tlsInfo
	a.mu.Unlock()

	st := &status.Status{
		Role: cfg.Role, Mode: cfg.Mode, Paths: []status.Path{}, TunnelIP: cfg.ServerTunnelIP(),
	}
	var mst *mqctl.Status
	var mstats *mqctl.Stats
	if a.sup.Info().State == supervisor.Running {
		var err error
		if mst, err = a.ctl.Status(ctx); err == nil {
			ready = true
			st.ControlAPI = true
			mstats, _ = a.ctl.Stats(ctx)
		} else {
			mst = nil
		}
	}
	st.State, st.Error = a.supState(ready)
	names := make([]string, 0, len(cfg.Server.Users))
	for _, u := range cfg.Server.Users {
		names = append(names, u.Name)
	}
	a.rater.Begin()
	st.Clients, st.Totals, st.UptimeSec = status.Server(a.rater, time.Now(), mst, mstats, names)
	vers := a.clientVersions()
	for i := range st.Clients {
		if v, ok := vers[st.Clients[i].Name]; ok {
			st.Clients[i].Version, st.Clients[i].Kind = v.Version, v.Kind
		}
	}
	a.rater.End()
	a.noteOutside(st.Clients)
	st.Tailscale = a.tsStatus(ctx)
	a.mu.Lock()
	a.pairReqs.gc(time.Now())
	if pend := a.pairReqs.pending(); len(pend) > 0 {
		st.PairRequests = pend
	}
	a.mu.Unlock()
	st.Server = a.publicAddr()
	if tls.source == "tailscale" {
		st.Server = tls.sni
	}
	return st
}

// publicAddr is what clients dial (cached, never blocks): see publicAddrFor.
func (a *Agent) publicAddr() string {
	addr, _ := a.publicAddrFor(false)
	return addr
}

// publicAddrFor is what clients dial: server.public_addr; else the public
// IPv4 Tailscale sees this machine at (netcheck) with the tunnel port, so
// no Dynamic DNS is needed: clients fetch the refreshed pairing over
// Tailscale when it changes; else a guess (the source address of the
// default route), usually a LAN address. auto reports the second case.
// fresh re-runs netcheck first when its result is over 2 minutes old
// (pairing requests; may wait a few seconds).
func (a *Agent) publicAddrFor(fresh bool) (addr string, auto bool) {
	cfg := a.store.Get()
	port := strconv.Itoa(cfg.Server.Port)
	if pa := strings.TrimSpace(cfg.Server.PublicAddr); pa != "" {
		if _, _, err := net.SplitHostPort(pa); err == nil {
			return pa, false
		}
		return net.JoinHostPort(strings.Trim(pa, "[]"), port), false
	}
	var n *tailscale.Netcheck
	if fresh {
		n = a.netcheckMaxAge(2 * time.Minute)
	} else {
		n = a.netcheck()
	}
	// IPv4 first. Not when the WAN address itself is carrier-grade NAT: the
	// public IPv4 is then shared and nothing reaches this machine on it.
	v4 := publicIPv4(n)
	if w, err := netip.ParseAddr(wanAddr()); err == nil && cgnat.Contains(w.Unmap()) {
		v4 = ""
	}
	if v4 != "" {
		return net.JoinHostPort(v4, port), true
	}
	// public IPv6: reachable with no port forward (the firewall still has to
	// allow the tunnel port); clients need IPv6 on a path to use it
	if v6 := publicIPv6(n); v6 != "" {
		return net.JoinHostPort(v6, port), true
	}
	if ip := outboundIP(); ip != "" {
		return net.JoinHostPort(ip, port), false
	}
	return "", false
}

// publicIPv6 is the global unicast IPv6 address from a netcheck result, or
// "" when there is none (ULA, link-local and tailnet addresses don't count).
func publicIPv6(n *tailscale.Netcheck) string {
	if n == nil || n.GlobalV6 == "" {
		return ""
	}
	ap, err := netip.ParseAddrPort(n.GlobalV6)
	if err != nil {
		return ""
	}
	ip := ap.Addr()
	if !ip.Is6() || ip.Is4In6() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return ""
	}
	return ip.String()
}

// publicIPv4 is the internet-facing IPv4 from a netcheck result, or "" when
// there is none or it can't be reached from outside (private, carrier-grade NAT).
func publicIPv4(n *tailscale.Netcheck) string {
	if n == nil || n.GlobalV4 == "" {
		return ""
	}
	ap, err := netip.ParseAddrPort(n.GlobalV4)
	if err != nil {
		return ""
	}
	ip := ap.Addr().Unmap()
	if !ip.Is4() || isPrivate(ip) || cgnat.Contains(ip) || ip.IsUnspecified() || ip.IsLoopback() {
		return ""
	}
	return ip.String()
}

// wanAddr is outboundIP, a variable so tests don't depend on the host's routes.
var wanAddr = outboundIP

func outboundIP() string {
	c, err := net.Dial("udp", "192.0.2.1:9")
	if err != nil {
		return ""
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}

type pairResult struct {
	Name  string `json:"name"`
	URI   string `json:"uri"`
	QRSVG string `json:"qr_svg"`
}

func (a *Agent) pairingFor(u config.User) (*pairResult, error) {
	cfg := a.store.Get()
	a.mu.Lock()
	tls := a.tlsInfo
	a.mu.Unlock()
	if tls.certFile == "" {
		var err error
		if tls, err = a.prepareTLS(context.Background(), false); err != nil {
			return nil, err
		}
	}
	server, _ := a.publicAddrFor(true)
	if server == "" {
		return nil, errors.New("no public address: set server.public_addr")
	}
	p := &pairing.Payload{
		V: pairing.Version, Name: u.Name, Server: server, User: u.Name, Key: u.Key,
		Mode: cfg.Mode, TailnetRoutes: config.TailnetRoutes, SNI: tls.sni,
		ServerName: a.serverName(), Subnet: cfg.Server.Subnet, ShareKey: a.shareKeyFor(u),
	}
	if tls.insecure {
		p.Insecure, p.CertSHA256 = true, tls.sha256
	}
	if !cfg.Server.DisableDNS {
		p.DNS = []string{cfg.ServerTunnelIP()}
	}
	uri, err := pairing.Encode(p)
	if err != nil {
		return nil, err
	}
	svg, err := pairing.QRSVG(uri)
	if err != nil {
		return nil, err
	}
	return &pairResult{Name: u.Name, URI: uri, QRSVG: svg}, nil
}

func (a *Agent) addClient(ctx context.Context, name string) (*pairResult, error) {
	return a.addClientNode(ctx, name, "")
}

// addClientNode registers a client; nodeID is the Tailscale node that asked
// for it (pair requests), "" otherwise.
func (a *Agent) addClientNode(ctx context.Context, name, nodeID string) (*pairResult, error) {
	if !mqconf.ValidUserName(name) {
		return nil, errors.New("name must be 1-63 characters: letters, digits, '.', '_' or '-'")
	}
	key, err := genKey()
	if err != nil {
		return nil, err
	}
	u := config.User{Name: name, Key: key, Created: time.Now().Unix(), TSNode: nodeID}
	if _, err := a.store.Update(func(c *config.Config) error {
		for _, x := range c.Server.Users {
			if x.Name == name {
				return fmt.Errorf("client %q already exists", name)
			}
		}
		c.Server.Users = append(c.Server.Users, u)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := a.addUserToMqvpn(ctx, name, key); err != nil {
		var apiErr *mqctl.APIError
		switch {
		case ctx.Err() != nil:
			// The caller is gone (request canceled, phone gave up): the
			// URI would never be delivered, so free the name instead of
			// leaving a credential nothing can authenticate with. First
			// case: a cancel that coincides with another failure is still
			// a cancel.
			a.Logf("add_user %s: caller gone during the retry (%v)", name, ctx.Err())
			rolled := a.rollbackClient(name, key)
			// ctx.Err() alone does not prove the request was never sent:
			// mqctl uses the context only for dialing, and an answer lost
			// after mqvpn served the add leaves a live registration for a
			// credential nobody holds. Take the key back out unless a
			// replacement owns the name now. WithoutCancel: the cleanup
			// must not die with the caller it is cleaning up after.
			if !storeHasName(a.store.Get(), name) {
				if rerr := a.ctl.RemoveUser(context.WithoutCancel(ctx), name); rerr != nil && !errors.Is(rerr, mqctl.ErrUnavailable) {
					a.Logf("remove_user %s (caller gone): %v", name, rerr)
				}
			}
			return nil, fmt.Errorf("pairing for client %q was abandoned: %w%s", name, ctx.Err(), rolled)
		case errors.Is(err, mqctl.ErrUnavailable):
			// mqvpn is down or predates the control API: the config render
			// carries the user when it (re)starts, and the tunnel port is
			// down with it, so nothing connects in the meantime. Say so.
			a.Logf("add_user %s: mqvpn control API unavailable; client takes effect when mqvpn (re)starts", name)
		case errors.As(err, &apiErr):
			// mqvpn heard us and refused: the URI would never authenticate.
			// Roll the entry back rather than hand it out (and block the
			// name).
			a.Logf("add_user %s: %v", name, err)
			return nil, fmt.Errorf("mqvpn did not accept client %q: %v%s", name, err, a.rollbackClient(name, key))
		default:
			// A transport failure the control API never answered: mqvpn may
			// have registered the user, so keep the credential rather than
			// delete a possibly-working one. If it didn't, removing the
			// client and pairing again fixes it.
			a.Logf("add_user %s: control API answered unclearly (%v); keeping the client", name, err)
		}
	} else {
		a.Logf("client %s added", name)
	}
	// A delete may have landed while the retry waited for the control
	// socket: removeClient then cleared the store but could not reach
	// mqvpn either, and the retry just registered the revoked key. Check
	// the credential is still ours before handing out its URI, and take
	// the key back out of mqvpn if it isn't.
	if !storeHasUser(a.store.Get(), name, key) {
		// Deleted while pairing. Clean mqvpn only when no user of this
		// name is left: a delete plus re-pair may have replaced the
		// credential, remove_user works by name, and removing now would
		// take the replacement's registration out with this one's.
		if !storeHasName(a.store.Get(), name) {
			// WithoutCancel: the pairing's own caller may be gone too
			if err := a.ctl.RemoveUser(context.WithoutCancel(ctx), name); err != nil && !errors.Is(err, mqctl.ErrUnavailable) {
				a.Logf("remove_user %s (deleted during pairing): %v", name, err)
			}
		}
		return nil, fmt.Errorf("client %q was removed while pairing", name)
	}
	return a.pairingFor(u)
}

// storeHasName: any user of this name is in the config.
func storeHasName(c *config.Config, name string) bool {
	for _, x := range c.Server.Users {
		if x.Name == name {
			return true
		}
	}
	return false
}

// storeHasUser: the exact credential (name and key) is in the config.
func storeHasUser(c *config.Config, name, key string) bool {
	for _, x := range c.Server.Users {
		if x.Name == name && x.Key == key {
			return true
		}
	}
	return false
}

// rollbackClient removes this request's credential — name and key, so a
// concurrent re-pair of the same name keeps its own — and returns an
// error suffix describing the outcome ("" when the rollback held).
func (a *Agent) rollbackClient(name, key string) string {
	if _, uerr := a.store.Update(func(c *config.Config) error {
		users := c.Server.Users[:0]
		for _, x := range c.Server.Users {
			if x.Name != name || x.Key != key {
				users = append(users, x)
			}
		}
		c.Server.Users = users
		return nil
	}); uerr != nil {
		a.Logf("add_user %s: rollback failed, the client stays in the store: %v", name, uerr)
		return " (rollback failed: " + uerr.Error() + ")"
	}
	return ""
}

// addUserToMqvpn registers the user with the running mqvpn. The HTTP API
// can accept a pairing before mqvpn's control socket has bound (the
// supervisor starts mqvpn asynchronously), so an "unavailable" control API
// is retried for a few seconds first. Found by the CGNAT relay e2e: a
// pairing that landed in that window issued a URI whose every connection
// the server rejected with "authentication failed: invalid or missing PSK".
var addUserRetry = 5 * time.Second

func (a *Agent) addUserToMqvpn(ctx context.Context, name, key string) error {
	deadline := time.Now().Add(addUserRetry)
	for {
		err := a.ctl.AddUser(ctx, name, key)
		if err == nil || !errors.Is(err, mqctl.ErrUnavailable) {
			return err
		}
		if time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// rmUser returns users without name, in place where possible.
func rmUser(users []config.User, name string) []config.User {
	out := users[:0]
	for _, x := range users {
		if x.Name != name {
			out = append(out, x)
		}
	}
	return out
}

func (a *Agent) removeClient(ctx context.Context, name string) error {
	found := false
	if _, err := a.store.Update(func(c *config.Config) error {
		before := len(c.Server.Users)
		c.Server.Users = rmUser(c.Server.Users, name)
		found = len(c.Server.Users) != before
		return nil
	}); err != nil {
		return err
	}
	if !found {
		return errNotFound
	}
	if err := a.ctl.RemoveUser(ctx, name); err != nil && !errors.Is(err, mqctl.ErrUnavailable) {
		a.Logf("remove_user %s: %v", name, err)
	}
	a.dropSharers(name)
	a.Logf("client %s revoked", name)
	// With no users left mqvpn falls back to the private global key, which
	// needs a fresh config.
	if len(a.store.Get().Server.Users) == 0 && a.sup.Active() {
		a.sup.Restart()
	}
	return nil
}

var errNotFound = errors.New("not found")
