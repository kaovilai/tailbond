package agent

// Use a phone hub (docs/phone-hub.md): a phone running tailbond can share
// its bonded connection as a proxy (SOCKS5 + HTTP CONNECT, internal/proxy,
// _tailbond-proxy._tcp). This machine finds such phones and, when the user
// picks one, points the system proxy at it.
//
// The system proxy doesn't point at the phone itself but at a forwarder
// on this machine, 127.0.0.1:1889 (no password, loopback only), which
// relays to the phone with the password derived from the share key. That
// way:
//   - apps that ignore SOCKS passwords (Chrome, most of them) still work
//   - the phone's address can be IPv6 link-local with a zone (fe80::1%en0),
//     which networksetup can't take
//   - when the phone's address changes (a new hotspot session, iOS rotating
//     its Private Wi-Fi Address) only the forwarder's upstream changes;
//     the system setting stays as it is
//
// Finding phones, best first: mDNS, the home server's list (kind "proxy",
// on one of our subnets), then, when neither finds one, the hotspot's
// router (the phone): its IPv6 link-local address from `ndp -rn`, zone and
// all; the IPv4 gateway, unless it is 192.0.0.x (the CLAT address an iPhone
// hotspot on an IPv6-only carrier hands out); and 172.20.10.1 when we're on
// an iPhone hotspot subnet. Every address is tried with the proxy password
// before it is listed, and found again every round: nothing is cached
// across sessions.
//
// Not persisted: after tailbondd stops, crashes or the Mac restarts, the
// system proxy is put back how it was (sysproxy's restore record) and the
// user picks the phone again. A phone that's gone for good shouldn't leave
// the Mac without internet.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	xproxy "golang.org/x/net/proxy"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/proxy"
	"github.com/kaovilai/tailbond/agent/internal/share"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/sysproxy"
)

const (
	// LocalProxyAddr is the forwarder the system proxy points at.
	LocalProxyAddr = "127.0.0.1:1889"
	hubEvery       = 15 * time.Second
	hubMissMax     = 3 // rounds without the phone in use before giving up on it
	// look for phone hubs only while someone looks at the status (the
	// dashboard, the menu-bar app) or one is in use: no probes otherwise
	hubInterest = 2 * time.Minute
	// HubGoneNote is said when the phone in use went away.
	HubGoneNote = "Lost the phone hub, so the proxy is off again and this Mac uses its own connection."
	// HubStoppedNote is said after the user stops using a hub: the Mac is
	// likely still on the phone's hotspot (its mobile data, not bonded).
	HubStoppedNote = "Stopped using the phone. This Mac may still be on the phone's hotspot: rejoin your usual Wi-Fi, and turn the hotspot off on the phone."
)

// systemProxy sets and restores the OS proxy setting (sysproxy.Mac; tests
// fake it). nil: this OS has none we set; the user points apps at
// LocalProxyAddr.
type systemProxy interface {
	ActiveService(ctx context.Context) (string, error)
	Use(ctx context.Context, service, host string, port int, user, pass string) error
	UsePAC(ctx context.Context, service, url string) error
	Restore(ctx context.Context) error
	Active() (sysproxy.Record, bool)
	Matches(ctx context.Context, rec sysproxy.Record) (bool, error)
	PutBack(ctx context.Context, rec sysproxy.Record) error
}

// hubCand is a phone hub found nearby.
type hubCand struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Addr  string `json:"addr"` // host:port that took the password
	Via   string `json:"via"`  // mdns, server, router
	// Bonded: "1" the phone's upstream is its bonded tunnel, "0" not (an
	// iPhone: mobile data only), "" unknown
	Bonded string `json:"bonded,omitempty"`
	// KeyID: the phone's share key id (its advert, the server's list); ""
	// for a router found without one (every pass is tried)
	KeyID string `json:"key_id,omitempty"`
	addrs []string
	// the login that worked (Credentials of this hub's pass for the phone)
	user, pass string
}

type hubUse struct {
	cand    hubCand
	since   time.Time
	mode    string // socks, pac, manual
	service string
	missed  int
	user    string
	pass    string
	srv     *proxy.Server
	ln      net.Listener
}

// phoneHubState is guarded by Agent.mu.
type phoneHubState struct {
	found []hubCand
	use   *hubUse
	note  string
	asked time.Time
	// "Connect via phone", the menu bar's one click (phoneconnect.go)
	conn pcState
	// guards useHub/stopHub against each other (not held with a.mu)
	op sync.Mutex
}

func (a *Agent) phoneHub() *phoneHubState {
	if a.client == nil {
		return nil
	}
	return a.client.hub
}

func (a *Agent) initSysProxy() {
	if a.sysProxy == nil && goruntime.GOOS == "darwin" {
		a.sysProxy = &sysproxy.Mac{RecordPath: filepath.Join(a.store.Get().StateDir, "phone-hub-proxy.json")}
	}
	if a.joinWiFi == nil && goruntime.GOOS == "darwin" {
		a.joinWiFi = macJoinWiFi
	}
}

// restoreSysProxy puts back a system proxy left set by an earlier run.
func (a *Agent) restoreSysProxy() {
	if a.sysProxy == nil {
		return
	}
	if r, ok := a.sysProxy.Active(); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := a.sysProxy.Restore(ctx); err != nil {
			a.Logf("phone hub: couldn't put back the proxy settings of %q: %v", r.Service, err)
			return
		}
		a.Logf("phone hub: put back the proxy settings of %q from before tailbondd last stopped", r.Service)
		// at start-up other writers of the same preferences run too (the
		// previous run's mqvpn being stopped puts its DNS back), and a stale
		// commit of theirs can undo this: check again a little later
		go a.recheckRestored(r)
	}
}

func (a *Agent) recheckRestored(r sysproxy.Record) {
	for _, d := range a.recheckDelays() {
		time.Sleep(d)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		ok, err := a.sysProxy.Matches(ctx, r)
		if err == nil && !ok {
			a.mu.Lock()
			inUse := a.client != nil && a.client.hub != nil && a.client.hub.use != nil
			a.mu.Unlock()
			if !inUse { // the user picked a hub meanwhile: theirs now
				a.Logf("phone hub: the proxy settings of %q changed back; putting them back again", r.Service)
				err = a.sysProxy.PutBack(ctx, r)
			}
		}
		cancel()
		if err != nil {
			a.Logf("phone hub: checking the restored proxy settings: %v", err)
		}
	}
}

func (a *Agent) recheckDelays() []time.Duration {
	if a.hubRecheck != nil {
		return a.hubRecheck
	}
	return []time.Duration{3 * time.Second, 10 * time.Second, 30 * time.Second}
}

func (a *Agent) phoneHubWatcher(ctx context.Context) {
	t := time.NewTimer(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.hubRound(ctx)
		t.Reset(hubEvery)
	}
}

// hubLogin is one way to log in to a phone's proxy: the pass the home
// server gave this machine (the hub) for one device, as proxy credentials.
type hubLogin struct {
	keyID, user, pass string
}

// hubLogins: this machine's logins, by the phone's key id (Pair & Share's
// passes, shares.go).
func (a *Agent) hubLogins() map[string]hubLogin {
	a.mu.Lock()
	p := a.client.payload
	a.mu.Unlock()
	if p == nil {
		return nil
	}
	out := map[string]hubLogin{}
	for kid, hp := range a.hubPasses(p) {
		u, pw := proxy.Credentials(hp.key, p.User, hp.exp)
		out[kid] = hubLogin{kid, u, pw}
	}
	return out
}

// hubAsked notes that someone is looking at the status.
func (a *Agent) hubAsked() {
	if hs := a.phoneHub(); hs != nil {
		a.mu.Lock()
		hs.asked = time.Now()
		a.mu.Unlock()
	}
}

func (a *Agent) hubRound(ctx context.Context) {
	hs := a.phoneHub()
	if hs == nil {
		return
	}
	a.mu.Lock()
	idle := hs.use == nil && time.Since(hs.asked) > hubInterest
	if idle {
		hs.found = nil
	}
	a.mu.Unlock()
	if idle {
		return
	}
	logins := a.hubLogins()
	if len(logins) == 0 {
		a.mu.Lock()
		hs.found = nil
		a.mu.Unlock()
		return
	}
	found := a.findHubs(ctx, logins)
	a.mu.Lock()
	hs.found = found
	u := hs.use
	var cur *hubCand
	if u != nil {
		for i := range found {
			if found[i].ID == u.cand.ID {
				cur = &found[i]
			}
		}
		if cur != nil {
			u.missed = 0
			if cur.Addr != u.cand.Addr {
				a.Logf("phone hub: %q is now at %s", cur.Label, cur.Addr)
			}
			u.cand = *cur // the forwarder dials the new address from now on
			u.user, u.pass = cur.user, cur.pass
		} else {
			u.missed++
		}
	}
	a.mu.Unlock()
	if u == nil {
		return
	}
	if cur == nil && u.missed >= hubMissMax {
		a.Logf("phone hub: %q not found for %d rounds", u.cand.Label, u.missed)
		a.stopHub(HubGoneNote)
		return
	}
	// the default route moved to another service (Wi-Fi → iPhone USB):
	// follow it
	if a.sysProxy != nil && u.mode != "manual" {
		if svc, err := a.sysProxy.ActiveService(ctx); err == nil && svc != u.service {
			if err := a.applySysProxy(ctx, svc, u.mode, u.ln.Addr().String()); err != nil {
				a.Logf("phone hub: %v", err)
			} else {
				a.mu.Lock()
				u.service = svc
				a.mu.Unlock()
				a.Logf("phone hub: the system proxy follows the default route to %q", svc)
			}
		}
	}
}

// findHubs lists the phone hubs that take our password.
func (a *Agent) findHubs(ctx context.Context, logins map[string]hubLogin) []hubCand {
	cfg := a.store.Get()
	skip := []string{config.DefaultTunName}
	ifaces := share.MulticastIfaces(skip...)
	var cands []hubCand
	if a.hubBrowse != nil {
		cands = a.hubBrowse(ctx)
	} else {
		if len(ifaces) > 0 {
			if f, err := share.BrowseService(ctx, proxy.Service, ifaces, 1500*time.Millisecond); err == nil {
				for _, x := range f {
					if x.Addr != nil {
						cands = append(cands, hubCand{ID: x.ID, Label: x.Label, Via: "mdns", Bonded: x.Bonded, KeyID: x.KeyID, addrs: []string{x.Addr.String()}})
					}
				}
			}
		}
		cands = append(cands, a.registryHubs(cfg, ifaces)...)
	}
	found := probeAll(ctx, cands, logins)
	if len(found) == 0 {
		var rc []hubCand
		if a.hubRouters != nil {
			rc = a.hubRouters(ctx)
		} else {
			rc = routerHubs(ctx)
		}
		found = probeAll(ctx, rc, logins)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Label+found[i].ID < found[j].Label+found[j].ID })
	return found
}

// registryHubs: the server's list (cached by the share round), kind
// "proxy", on one of our subnets.
func (a *Agent) registryHubs(cfg *config.Config, ifaces []net.Interface) []hubCand {
	a.mu.Lock()
	reg := a.client.share.reg
	p := a.client.payload
	a.mu.Unlock()
	if reg == nil || p == nil {
		return nil
	}
	subnets := localSubnets(ifaces)
	var out []hubCand
	for _, r := range reg.Sharers {
		if r.Kind != "proxy" || r.Client == p.User { // a phone hub's proxy, not our own
			continue
		}
		c := hubCand{ID: r.ID, Label: r.Label, Via: "server", KeyID: r.KeyID}
		for _, ip := range r.LAN {
			if ad, err := netip.ParseAddr(ip); err == nil && inAny(subnets, ad) {
				c.addrs = append(c.addrs, net.JoinHostPort(ip, strconv.Itoa(r.Port)))
			}
		}
		if len(c.addrs) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// probeAll tries each candidate's addresses in parallel, with the login
// for its key id (every login when it has none); the first address and
// login that work are the hub's. Same id twice: the first wins (mDNS
// before the server's list).
func probeAll(ctx context.Context, cands []hubCand, logins map[string]hubLogin) []hubCand {
	type hit struct{ addr, user, pass string }
	res := make([]hit, len(cands))
	var wg sync.WaitGroup
	for i, c := range cands {
		var try []hubLogin
		if l, ok := logins[c.KeyID]; ok {
			try = []hubLogin{l}
		} else if c.KeyID == "" {
			for _, l := range logins {
				try = append(try, l)
			}
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, ad := range c.addrs {
				for _, l := range try {
					if probeHub(ctx, ad, l.user, l.pass) == nil {
						res[i] = hit{ad, l.user, l.pass}
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	seen := map[string]bool{}
	var out []hubCand
	for i, c := range cands {
		if res[i].addr == "" || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		c.Addr, c.user, c.pass = res[i].addr, res[i].user, res[i].pass
		out = append(out, c)
	}
	return out
}

var errHubAuth = errors.New("the phone hub refused the password (paired with another home server?)")

// probeHub runs a SOCKS5 greeting and the password step against addr.
func probeHub(ctx context.Context, addr, user, pass string) error {
	d := net.Dialer{Timeout: 1500 * time.Millisecond}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.Write([]byte{5, 1, 2}); err != nil {
		return err
	}
	var r [2]byte
	if _, err := io.ReadFull(c, r[:]); err != nil {
		return err
	}
	if r[0] != 5 || r[1] != 2 {
		return fmt.Errorf("%s: not a tailbond phone hub", addr)
	}
	b := append([]byte{1, byte(len(user))}, user...)
	b = append(append(b, byte(len(pass))), pass...)
	if _, err := c.Write(b); err != nil {
		return err
	}
	if _, err := io.ReadFull(c, r[:]); err != nil {
		return err
	}
	if r[1] != 0 {
		return errHubAuth
	}
	return nil
}

// useHub starts the forwarder to the hub id and points the system proxy
// at it (pac: through a PAC file that keeps local and tailnet names
// direct).
func (a *Agent) useHub(ctx context.Context, id string, pac bool) error {
	hs := a.phoneHub()
	if hs == nil {
		return errNotFound
	}
	hs.op.Lock()
	defer hs.op.Unlock()
	logins := a.hubLogins()
	if len(logins) == 0 {
		return errors.New("connect to your home server once first: the phone hub's password comes from it")
	}
	lookup := func() (*hubCand, *hubUse) {
		a.mu.Lock()
		defer a.mu.Unlock()
		var cand *hubCand
		for i := range hs.found {
			if hs.found[i].ID == id {
				c := hs.found[i]
				cand = &c
			}
		}
		return cand, hs.use
	}
	cand, cur := lookup()
	if cand == nil {
		// the last round may have missed it (one probe that failed, or the
		// list was dropped while nobody looked): look again before saying no
		found := a.findHubs(ctx, logins)
		a.mu.Lock()
		hs.found = found
		hs.asked = time.Now()
		a.mu.Unlock()
		cand, cur = lookup()
	}
	if cand == nil {
		return errors.New("that phone hub isn't around any more; check the phone is sharing and on the same network")
	}
	mode := "socks"
	if pac {
		mode = "pac"
	}
	if a.sysProxy == nil {
		mode = "manual"
	}
	if cur != nil {
		// another phone, or the other mode: the forwarder stays, the
		// target and the setting change
		a.mu.Lock()
		cur.cand, cur.user, cur.pass, cur.missed = *cand, cand.user, cand.pass, 0
		a.mu.Unlock()
		if mode != cur.mode && mode != "manual" {
			if err := a.applySysProxy(ctx, cur.service, mode, cur.ln.Addr().String()); err != nil {
				return err
			}
			a.mu.Lock()
			cur.mode = mode
			a.mu.Unlock()
		}
		a.Logf("phone hub: using %q at %s", cand.Label, cand.Addr)
		return nil
	}
	if err := probeHub(ctx, cand.Addr, cand.user, cand.pass); err != nil {
		return fmt.Errorf("can't reach %s: %v", cand.Label, err)
	}
	local := a.hubLocal
	if local == "" {
		local = LocalProxyAddr
	}
	ln, err := net.Listen("tcp", local)
	if err != nil {
		return fmt.Errorf("the local proxy port %s: %v", local, err)
	}
	u := &hubUse{cand: *cand, since: time.Now(), mode: mode, user: cand.user, pass: cand.pass, ln: ln}
	u.srv = &proxy.Server{
		Allow: func(ad netip.Addr) bool { return ad.Unmap().IsLoopback() },
		Dial:  a.hubDial,
		Logf:  a.Logf,
	}
	go func() { _ = u.srv.Serve(ln) }()
	if mode != "manual" {
		svc, err := a.sysProxy.ActiveService(ctx)
		if err == nil {
			err = a.applySysProxy(ctx, svc, mode, ln.Addr().String())
		}
		if err != nil {
			ln.Close()
			a.restoreSysProxy()
			return fmt.Errorf("setting the system proxy: %v", err)
		}
		u.service = svc
	}
	a.mu.Lock()
	hs.use, hs.note = u, ""
	a.mu.Unlock()
	where := "point apps' SOCKS5 proxy at " + ln.Addr().String()
	if mode != "manual" {
		where = fmt.Sprintf("system proxy (%s) on %q → %s", mode, u.service, ln.Addr())
	}
	a.Logf("phone hub: using %q at %s; %s", cand.Label, cand.Addr, where)
	return nil
}

// applySysProxy points svc's proxy setting at the forwarder on local.
func (a *Agent) applySysProxy(ctx context.Context, svc, mode, local string) error {
	if mode == "pac" {
		return a.sysProxy.UsePAC(ctx, svc, "http://127.0.0.1:"+a.apiPort+"/proxy.pac")
	}
	host, port, _ := net.SplitHostPort(local)
	pn, _ := strconv.Atoi(port)
	return a.sysProxy.Use(ctx, svc, host, pn, "", "")
}

// hubDial is the forwarder's upstream: SOCKS5 to the hub in use.
func (a *Agent) hubDial(ctx context.Context, network, addr string) (net.Conn, error) {
	a.mu.Lock()
	var u hubUse
	ok := a.client.hub.use != nil
	if ok {
		u = *a.client.hub.use
	}
	a.mu.Unlock()
	if !ok {
		return nil, errors.New("no phone hub in use")
	}
	d, err := xproxy.SOCKS5("tcp", u.cand.Addr, &xproxy.Auth{User: u.user, Password: u.pass}, &net.Dialer{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}
	return d.(xproxy.ContextDialer).DialContext(ctx, network, addr)
}

// stopHub stops using the hub and puts the system proxy back; note says
// why when it wasn't the user.
func (a *Agent) stopHub(note string) { a.stopHubIf(note, nil) }

// stopHubIf is stopHub when ours (nil: any) says the hub in use is the one
// the caller set up; checked under the operation lock, so a hub picked
// meanwhile (the dashboard) isn't torn down.
func (a *Agent) stopHubIf(note string, ours func(*hubUse) bool) {
	hs := a.phoneHub()
	if hs == nil {
		return
	}
	hs.op.Lock()
	defer hs.op.Unlock()
	a.mu.Lock()
	u := hs.use
	if u != nil && ours != nil && !ours(u) {
		a.mu.Unlock()
		return
	}
	hs.use, hs.note = nil, note
	a.mu.Unlock()
	if u == nil {
		return
	}
	u.ln.Close()
	a.restoreSysProxy()
	a.Logf("phone hub: stopped using %q", u.cand.Label)
}

// phoneHubStatus is the dashboard's view.
func (a *Agent) phoneHubStatus() *status.PhoneHub {
	hs := a.phoneHub()
	if hs == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := &status.PhoneHub{Note: hs.note, Found: []status.PhoneHubFound{}, Manual: a.sysProxy == nil, Connect: a.phoneConnectStatusLocked(hs)}
	for _, c := range hs.found {
		out.Found = append(out.Found, status.PhoneHubFound{ID: c.ID, Label: c.Label, Addr: c.Addr, Via: c.Via, Bonded: c.Bonded != "0", Iface: ifaceFor(c.Addr)})
	}
	if u := hs.use; u != nil {
		up, down, clients := u.srv.Stats()
		conns := 0
		for _, c := range clients {
			conns += c.Conns
		}
		out.Using = &status.PhoneHubUse{ID: u.cand.ID, Label: u.cand.Label, Addr: u.cand.Addr, Mode: u.mode, Bonded: u.cand.Bonded != "0", Iface: ifaceFor(u.cand.Addr),
			Service: u.service, Local: u.ln.Addr().String(), Since: u.since, UpBytes: up, DnBytes: down, Conns: conns}
	}
	return out
}

// hPhoneHub: GET the hubs found and the one in use; POST {"use": id,
// "pac": bool} to use one, {"use": ""} to stop.
func (a *Agent) hPhoneHub(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient || a.phoneHub() == nil {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.hubAsked()
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, a.phoneHubStatus())
		return
	}
	var in struct {
		Use     *string `json:"use"`
		PAC     bool    `json:"pac"`
		Refresh bool    `json:"refresh"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if in.Refresh {
		a.hubRound(r.Context())
	}
	if in.Use != nil {
		if *in.Use == "" {
			a.mu.Lock()
			was := a.client.hub.use != nil
			a.mu.Unlock()
			note := ""
			if was {
				note = HubStoppedNote
			}
			a.stopHub(note)
		} else if err := a.useHub(r.Context(), *in.Use, in.PAC); err != nil {
			httpErr(w, http.StatusConflict, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, a.phoneHubStatus())
}

// hProxyPAC is the PAC file for "pac" mode: local, link-local, private and
// tailnet names direct, the rest through the phone hub.
func (a *Agent) hProxyPAC(w http.ResponseWriter, r *http.Request) {
	local := LocalProxyAddr
	if hs := a.phoneHub(); hs != nil {
		a.mu.Lock()
		if hs.use != nil {
			local = hs.use.ln.Addr().String()
		}
		a.mu.Unlock()
	}
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
	_, _ = io.WriteString(w, pacFile(local))
}

func pacFile(local string) string {
	return `// tailbond: use a phone hub (docs/phone-hub.md)
function FindProxyForURL(url, host) {
  if (isPlainHostName(host) || dnsDomainIs(host, ".local") || dnsDomainIs(host, ".ts.net") ||
      dnsDomainIs(host, ".home.arpa") || host == "localhost")
    return "DIRECT";
  var nets = [["10.0.0.0", "255.0.0.0"], ["172.16.0.0", "255.240.0.0"], ["192.168.0.0", "255.255.0.0"],
              ["100.64.0.0", "255.192.0.0"], ["169.254.0.0", "255.255.0.0"], ["127.0.0.0", "255.0.0.0"]];
  if (/^[0-9.]+$/.test(host))
    for (var i = 0; i < nets.length; i++)
      if (isInNet(host, nets[i][0], nets[i][1])) return "DIRECT";
  if (host.indexOf(":") >= 0) return "DIRECT"; // IPv6 literals: link-local and ULA stay local
  return "SOCKS5 ` + local + `; SOCKS ` + local + `";
}
`
}

// ifaceFor is the local interface a hub address is reached on: its zone,
// or the interface whose IPv4 subnet holds it ("" unknown).
func ifaceFor(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return ""
	}
	if i := strings.IndexByte(host, '%'); i >= 0 {
		return host[i+1:]
	}
	ad, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	ifs, _ := net.Interfaces()
	for _, ifi := range ifs {
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok {
				if pf, err := netip.ParsePrefix(n.String()); err == nil && pf.Masked().Contains(ad) && !ad.IsLoopback() {
					return ifi.Name
				}
			}
		}
	}
	return ""
}

// Possessive is "Pixel's" / "James'", for "Using Pixel's bonded connection".
func Possessive(s string) string {
	if strings.HasSuffix(s, "s") {
		return s + "'"
	}
	return s + "'s"
}
