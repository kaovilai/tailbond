// Package agent is tailbondd: it owns the config, supervises mqvpn, normalizes
// its status and serves the HTTP API and web UI.
package agent

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/coexist"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/glrepeater"
	"github.com/kaovilai/tailbond/agent/internal/hooks"
	"github.com/kaovilai/tailbond/agent/internal/mqctl"
	"github.com/kaovilai/tailbond/agent/internal/netif"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/supervisor"
	"github.com/kaovilai/tailbond/agent/internal/tailscale"
	"github.com/kaovilai/tailbond/agent/internal/update"
)

type Options struct {
	Version string
	WebDir  string // serve the UI from disk instead of the embedded copy
	Echo    bool   // copy logs to stderr (journald)
}

type Agent struct {
	crash *crashState // Report a problem: the kept crash (reportapi.go)
	opts  Options
	store *config.Store
	logs  *supervisor.Ring
	sup   *supervisor.Supervisor
	ctl   *mqctl.Client
	ts    *tailscale.CLI
	hooks *hooks.Runner
	rater *status.Rater
	role  string

	hookq chan func()
	// hookMu: one hook at a time, also while drainHooks runs the queue at
	// shutdown and hookWorker may still be in the middle of one (a down hook
	// must never overlap the up hook whose changes it undoes)
	hookMu sync.Mutex
	// stopping the hook worker: drainHooks cancels it and waits for it to
	// exit, so a hook it already took off the queue finishes first
	hookStop context.CancelFunc
	hookDone chan struct{}
	pollMu   sync.Mutex // serializes poll (the rater is not concurrency-safe)
	secretMu sync.Mutex // stateSecret: each secret is created once

	mu       sync.Mutex
	snap     *status.Status
	rt       runtime // per-process state parsed from mqvpn output
	tlsInfo  tlsInfo
	sessions map[string]time.Time
	client   *clientState
	pairReqs *pairRequests  // server role
	shareReg *shareRegistry // server role: Pair & Share sharers
	join     *joinState     // client role: asking a server to pair
	userStop bool           // client: the user turned tailbond off or unpaired

	// whois identifies tailnet peers (tests stub it).
	whois func(ctx context.Context, ip string) (*tailscale.Whois, error)
	// hello probes a possible server (tests stub it).
	hello func(ctx context.Context, addr string) (*helloInfo, error)
	// apiPort is the port tailbond servers serve /api/hello and pair requests on.
	apiPort string
	disc    discoveryCache
	rescan  chan struct{}
	nc      netcheckCache
	outside *outsideSeen // server: last client seen from the internet (a.mu)
	// listIfs discovers interfaces (tests stub it).
	listIfs func() ([]netif.Iface, error)
	// lookupIf describes an interface a path setting names (tests leave it nil).
	lookupIf func(string) (netif.Iface, bool)
	// shareIf says whether the interface this machine shares is there and
	// connected (shares.go; tests stub it).
	shareIf func(string) (shareIfFacts, bool)
	wc      whoisCache // admin requests from the tailnet: who sent them
	// phone hub (phonehub.go): the OS proxy setting (nil: not set for
	// you), and test hooks for discovery and the forwarder's address
	sysProxy   systemProxy
	hubBrowse  func(context.Context) []hubCand
	hubRouters func(context.Context) []hubCand
	hubLocal   string
	hubRecheck []time.Duration
	// clickWait: how long the one click waits between hub lookups, and
	// clickFindFor: how long it keeps looking (tests shrink both)
	clickWait    func(context.Context) time.Duration
	clickFindFor time.Duration
	// other VPNs (coexist.go): test hooks for commands and /etc/resolver
	cxRun         coexist.Runner
	cxResolverDir string
	cxIfaces      func() map[string][]netip.Prefix
	dl            dlState // the phone page's downloads: checksums, the macOS bundle fetch
	// landingUp: the phone landing page listens on a Tailscale IP.
	landingUp atomic.Bool
	// glRead reads GL firmware's repeater portal state (tests stub it).
	glRead func(ctx context.Context) glrepeater.State
	// captiveDialer replaces the per-device probe dial (tests).
	captiveDialer func(ctx context.Context, dev, addr string) (net.Conn, error)
	// serverProbe replaces the standby-server probe (tests).
	serverProbe func(ctx context.Context, server, dev string) error
	// subnetMoved tells the DNS forwarder the server's tunnel subnet changed.
	subnetMoved chan struct{}
	// bypassNow re-applies the bypass rules (WAN changes).
	bypassNow chan struct{}
	// bypass hook runs are serialised (bypassMu); a queued apply from
	// before the last clear (bypassGen) is dropped, so rules never come
	// back after the agent cleared them
	bypassMu  sync.Mutex
	bypassGen atomic.Int64
	// auto-updates and backups (update.go, backupapi.go)
	upd     *updater
	restart chan struct{}
}

// runtime is reset every time mqvpn starts.
type runtime struct {
	tunName     string // the tunnel device mqvpn made (macOS: utunN, the kernel's pick)
	tunnelIP    string
	peerIP      string
	ready       bool // server: "server ready" / client: ESTABLISHED
	readyAt     time.Time
	upHookFired bool
	upEnv       map[string]string // what the up hook got; the down hook gets the same
	// tsApplied: the Tailscale state the last server-up hook saw
	// (tailscaleSig; "" = none ran yet), for tailscaleWatch
	tsApplied string
	lastState string
	logPaths  map[string]logPath
}

type logPath struct {
	id      uint64
	srttMs  uint64
	state   string
	updated time.Time
}

type tlsInfo struct {
	certFile, keyFile string
	sni               string
	insecure          bool
	sha256            string
	source            string // "tailscale" | "self-signed"
}

func New(store *config.Store, opts Options) *Agent {
	cfg := store.Get()
	a := &Agent{
		opts:        opts,
		store:       store,
		logs:        supervisor.NewRing(2000),
		ctl:         mqctl.New(cfg.ControlAddr),
		rater:       status.NewRater(),
		role:        cfg.Role,
		hookq:       make(chan func(), 32),
		sessions:    map[string]time.Time{},
		pairReqs:    newPairRequests(),
		shareReg:    &shareRegistry{m: map[string]regSharer{}},
		join:        &joinState{State: joinIdle},
		apiPort:     "8077",
		rescan:      make(chan struct{}, 1),
		subnetMoved: make(chan struct{}, 1),
		bypassNow:   make(chan struct{}, 1),
		upd:         newUpdater(),
		restart:     make(chan struct{}, 1),
		crash:       &crashState{},
	}
	if opts.Echo {
		a.logs.Echo = os.Stderr
	}
	a.ctl.Timeout = 900 * time.Millisecond
	a.ts = &tailscale.CLI{Disabled: cfg.DisableTailscale, Route: cfg.Server.Subnet}
	a.whois = a.ts.Whois
	a.hello = a.fetchHello
	a.glRead = glrepeater.Read
	a.listIfs = netif.Discover
	a.lookupIf = netif.Lookup
	a.shareIf = realShareIf
	a.hooks = &hooks.Runner{Dir: cfg.HooksDir, Log: a.logs.Add}
	mqvpnBin := cfg.MqvpnBin
	a.sup = supervisor.New(supervisor.Spec{
		Bin:      mqvpnBin,
		Args:     []string{"--config", a.confPath()},
		PreStart: a.preStart,
		OnLine:   a.onLine,
		OnStart:  a.onStart,
		OnExit:   a.onExit,
		PidFile:  filepath.Join(cfg.StateDir, "mqvpn.pid"),
	}, a.logs)
	if a.role == config.RoleClient {
		a.client = newClientState()
	}
	return a
}

// Rescan re-checks the WANs now (SIGHUP; OpenWrt's hotplug sends it when
// an interface goes up or down).
func (a *Agent) Rescan() {
	select {
	case a.rescan <- struct{}{}:
	default:
	}
	if a.client != nil {
		a.pokeCaptive("")
		a.pokeBypass()
	}
}

func (a *Agent) Logf(format string, args ...any) { a.logs.Add("agent", fmt.Sprintf(format, args...)) }

func (a *Agent) confPath() string {
	cfg := a.store.Get()
	name := "server.conf"
	if cfg.Role == config.RoleClient {
		name = "client.conf"
	}
	return filepath.Join(cfg.StateDir, name)
}

// Run blocks until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	cfg := a.store.Get()
	if err := os.MkdirAll(cfg.StateDir, 0o700); err != nil {
		return err
	}
	// its own context: drainHooks stops it also on the restart paths, where
	// ctx stays live
	hctx, hcancel := context.WithCancel(ctx)
	a.hookStop, a.hookDone = hcancel, make(chan struct{})
	go func() { a.hookWorker(hctx); close(a.hookDone) }()
	a.Logf("tailbondd %s starting, role=%s mode=%s config=%s", a.opts.Version, cfg.Role, cfg.Mode, a.store.Path())
	a.setupCrash()

	switch cfg.Role {
	case config.RoleServer:
		a.startServer(ctx)
	case config.RoleClient:
		a.startClient(ctx)
	}

	go a.updateLoop(ctx)
	go a.backupLoop(ctx)
	srvErr := make(chan error, 1)
	go func() { srvErr <- a.serveHTTP(ctx) }()
	if cfg.Role == config.RoleServer {
		go a.serveLanding(ctx)
	}

	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		a.poll(ctx)
		select {
		case <-ctx.Done():
			a.Logf("shutting down")
			a.sup.Stop()
			a.drainHooks()
			if a.client != nil {
				a.clearBypass()
				a.stopHub("")
				a.coexistRestore()
			}
			return nil
		case err := <-srvErr:
			a.sup.Stop()
			a.drainHooks()
			return err
		case <-a.restart:
			a.Logf("restarting (update)")
			a.sup.Stop()
			a.drainHooks()
			if a.client != nil {
				a.clearBypass()
				a.stopHub("")
				a.coexistRestore()
			}
			return ErrRestart
		case <-t.C:
		}
	}
}

func (a *Agent) hookWorker(ctx context.Context) {
	for {
		select {
		case f := <-a.hookq:
			a.hookMu.Lock()
			f()
			a.hookMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// drainHooks runs queued hooks synchronously at shutdown (down hooks
// matter), once the worker has stopped: a hook it already dequeued runs to
// completion before this returns.
func (a *Agent) drainHooks() {
	if a.hookStop != nil {
		a.hookStop()
		<-a.hookDone
	}
	a.hookMu.Lock()
	defer a.hookMu.Unlock()
	for {
		select {
		case f := <-a.hookq:
			f()
		default:
			return
		}
	}
}

func (a *Agent) runHook(name string, env map[string]string) {
	a.queueHook(func() { a.hooks.Run(name, env) })
}

// queueHook queues f on the hook worker (one hook at a time).
func (a *Agent) queueHook(f func()) {
	select {
	case a.hookq <- f:
	default:
		a.Logf("hook queue full, dropping one")
	}
}

func (a *Agent) preStart(ctx context.Context) error {
	a.mu.Lock()
	a.rt = runtime{logPaths: map[string]logPath{}}
	a.mu.Unlock()
	if a.role == config.RoleServer {
		return a.serverPreStart(ctx)
	}
	return a.clientPreStart(ctx)
}

func (a *Agent) onStart() {}

func (a *Agent) onExit(err error) {
	a.noteChildCrash(err)
	a.mu.Lock()
	fired := a.rt.upHookFired
	env := a.downEnvLocked()
	a.rt.ready, a.rt.upHookFired = false, false
	a.mu.Unlock()
	if fired {
		if a.role == config.RoleServer {
			a.runHook(hooks.ServerDown, env)
		} else {
			a.runHook(hooks.ClientDown, env)
		}
	}
}

var (
	reState     = regexp.MustCompile(`state: (\w+) → (\w+)`)
	reClientTun = regexp.MustCompile(`TUN (\S+) configured: (\S+) → (\S+)`)
	reServerTun = regexp.MustCompile(`TUN (\S+) configured: (\S+) \(mtu`)
	reReady     = regexp.MustCompile(`mqvpn server ready`)
	reLogPath   = regexp.MustCompile(`\[STATUS\]\s+path(\d+)=(\S+) srtt=(\d+)ms tx=\d+ rx=\d+ (\w+)`)
)

func (a *Agent) onLine(line string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if m := reClientTun.FindStringSubmatch(line); m != nil {
		// a late line from the previous mqvpn (another server's subnet) is not ours
		if ip, err := netip.ParseAddr(m[2]); err == nil && a.client != nil {
			for _, s := range a.client.servers {
				if s.name != a.client.activeServer && s.payload.TunnelSubnet().Contains(ip) {
					return
				}
			}
		}
		if a.rt.tunnelIP != m[2] {
			a.logs.Add("agent", "tunnel address "+m[2])
		}
		a.rt.tunName, a.rt.tunnelIP, a.rt.peerIP = m[1], m[2], m[3]
		return
	}
	if m := reServerTun.FindStringSubmatch(line); m != nil {
		a.rt.tunName, a.rt.tunnelIP = m[1], m[2]
		return
	}
	if reReady.MatchString(line) && a.role == config.RoleServer {
		a.markReadyLocked()
		return
	}
	if m := reState.FindStringSubmatch(line); m != nil && a.role == config.RoleClient {
		a.rt.lastState = m[2]
		switch m[2] {
		case "ESTABLISHED":
			a.markReadyLocked()
		case "RECONNECTING", "CLOSED", "IDLE":
			if a.rt.ready {
				a.rt.ready = false
				if a.rt.upHookFired {
					a.rt.upHookFired = false
					a.runHook(hooks.ClientDown, a.downEnvLocked())
				}
			}
		}
		return
	}
	if m := reLogPath.FindStringSubmatch(line); m != nil {
		var id, srtt uint64
		fmt.Sscan(m[1], &id)
		fmt.Sscan(m[3], &srtt)
		a.rt.logPaths[m[2]] = logPath{id: id, srttMs: srtt, state: m[4], updated: time.Now()}
	}
}

func (a *Agent) markReadyLocked() {
	if a.rt.ready {
		return
	}
	a.rt.ready, a.rt.readyAt = true, time.Now()
	a.rt.upHookFired = true
	a.rt.upEnv = a.hookEnvLocked()
	if a.role == config.RoleServer {
		a.runServerUp(a.rt.upEnv)
	} else {
		a.runHook(hooks.ClientUp, a.rt.upEnv)
	}
}

// downEnvLocked is the env for a down hook: what the matching up hook got
// (a server switch or subnet move must tear down the old routes), with the
// current release/switch flags. Caller holds a.mu.
func (a *Agent) downEnvLocked() map[string]string {
	env := a.hookEnvLocked()
	if a.rt.upEnv != nil {
		cur := env
		env = make(map[string]string, len(a.rt.upEnv)+2)
		for k, v := range a.rt.upEnv {
			env[k] = v
		}
		delete(env, "TB_RELEASE")
		if v, ok := cur["TB_RELEASE"]; ok {
			env["TB_RELEASE"] = v
		}
	}
	if a.client != nil && a.client.switchPending {
		a.client.switchPending = false
		env["TB_SWITCH"] = "1"
	}
	return env
}

// webPorts: the TCP ports the agent serves to the tailnet (dashboard and
// the phone page's bind port), for hooks that open them to tailnet sources.
func webPorts(cfg *config.Config) string {
	_, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		port = strings.TrimPrefix(cfg.Listen, ":")
	}
	if bind, _, ok := cfg.LandingPorts(); ok {
		port += " " + strconv.Itoa(bind)
	}
	return port
}

// tunDevLocked is the tunnel device: the one mqvpn reported, else the name
// it is asked for. Caller holds a.mu.
func (a *Agent) tunDevLocked() string {
	if a.rt.tunName != "" {
		return a.rt.tunName
	}
	return config.DefaultTunName
}

func (a *Agent) hookEnvLocked() map[string]string {
	cfg := a.store.Get()
	env := map[string]string{
		"TB_TUN":       a.tunDevLocked(),
		"TB_ROLE":      cfg.Role,
		"TB_MODE":      cfg.Mode,
		"TB_STATE_DIR": cfg.StateDir,
		"TB_CONFIG":    a.confPath(),
	}
	if cfg.Role == config.RoleServer {
		env["TB_SUBNET"] = cfg.Server.Subnet
		env["TB_SUBNET6"] = cfg.Server.Subnet6
		env["TB_PORT"] = fmt.Sprint(cfg.Server.Port)
		env["TB_WEB_PORTS"] = webPorts(cfg)
		env["TB_ADVERTISE_ROUTE"] = "1"
		if cfg.Server.NoAdvertiseRoute {
			env["TB_ADVERTISE_ROUTE"] = "0"
		}
		// disable_tailscale: the hook leaves Tailscale alone too (no tailnet
		// NAT, no route advertisement)
		env["TB_TAILSCALE"] = "1"
		if cfg.DisableTailscale {
			env["TB_TAILSCALE"], env["TB_ADVERTISE_ROUTE"] = "0", "0"
		}
		if !cfg.Server.DisableDNS {
			env["TB_DNS_IP"] = cfg.ServerTunnelIP()
		} else {
			env["TB_DNS_IP"] = ""
		}
		return env
	}
	if a.userStop {
		// turned off on purpose: the LAN goes back to the WANs, kill switch or not
		env["TB_RELEASE"] = "1"
	}
	env["TB_TUNNEL_IP"] = a.rt.tunnelIP
	env["TB_SERVER_IP"] = ""
	env["TB_TAILNET_ROUTES"] = ""
	env["TB_SERVER_NAME"] = ""
	env["TB_SUBNET"] = ""
	env["TB_DNS_IP"] = ""
	if a.client != nil && a.client.payload != nil {
		p := a.client.payload
		env["TB_SERVER_IP"] = a.client.serverIP
		env["TB_TAILNET_ROUTES"] = strings.Join(p.TailnetRoutes, ",")
		env["TB_SERVER_NAME"] = a.client.activeServer
		env["TB_SUBNET"] = p.TunnelSubnet().String()
		if len(p.DNS) > 0 {
			env["TB_DNS_IP"] = p.DNS[0]
		}
	}
	if a.client != nil {
		env["TB_BYPASS_FILE"] = a.bypassFile()
	}
	return env
}

// poll refreshes the status snapshot once a second.
func (a *Agent) poll(ctx context.Context) {
	a.pollMu.Lock()
	defer a.pollMu.Unlock()
	var st *status.Status
	if a.role == config.RoleServer {
		st = a.serverStatus(ctx)
	} else {
		st = a.clientStatus(ctx)
	}
	st.Version = a.opts.Version
	st.Protocol = update.Protocol
	st.Update = a.updateView()
	st.Crash = a.crashAt()
	a.mu.Lock()
	a.snap = st
	a.mu.Unlock()
}

func (a *Agent) Snapshot() *status.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snap == nil {
		return &status.Status{Role: a.role, State: status.StateDisconnected, Paths: []status.Path{}, Clients: []status.Client{}}
	}
	return a.snap
}

// supState maps the supervisor and readiness to a contract state.
func (a *Agent) supState(ready bool) (string, string) {
	info := a.sup.Info()
	switch info.State {
	case supervisor.Running:
		if ready {
			return status.StateConnected, ""
		}
		return status.StateConnecting, ""
	case supervisor.Starting:
		return status.StateConnecting, ""
	case supervisor.Backoff:
		return status.StateError, info.LastErr
	default:
		return status.StateDisconnected, ""
	}
}

func (a *Agent) tsStatus(ctx context.Context) *status.Tailscale {
	ti := a.ts.Get(ctx)
	return &status.Tailscale{
		Running: ti.Running, Self: ti.Self, RoutesApproved: ti.RoutesApproved,
		Installed: ti.Installed, DNSName: ti.DNSName, IPs: ti.IPs,
	}
}

// genKey returns a random 32-byte base64 PSK (same format as `mqvpn --genkey`).
func genKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// writeFileAtomic replaces path with data through a temp file of its own,
// so concurrent writers never share (or rename away) each other's.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(mode)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
	}
	return err
}

var errClientOnly = errors.New("client role only")
var errServerOnly = errors.New("server role only")
