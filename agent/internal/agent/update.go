package agent

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/status"
	"github.com/tailbond/tailbond/agent/internal/update"
)

// Auto-updates (docs/architecture.md, "Updates"). A server checks the
// public source and mirrors the channel for its clients; a client checks
// its server's mirror first. Linux installs are self-updates under
// state_dir (the packaged binary re-execs them, update.Store.Boot); OpenWrt
// installs go through opkg (/usr/lib/tailbond/update-ipk).

// ErrRestart asks main to exit with update.ExitRestart (systemd restarts us).
var ErrRestart = errors.New("restarting for an update")

const (
	platformLinux   = "linux"
	platformOpenWrt = "openwrt"
	// OpenWrt: kept across firmware upgrades (keep.d), see docs/updates.md
	defaultPkgDir    = "/etc/tailbond/pkg"
	defaultIPKScript = "/usr/lib/tailbond/update-ipk"
	resultFileName   = "update-result.json"
)

type clientVersion struct {
	Version, Kind string
	At            time.Time
}

// updater is the update state (its own lock: checks run long).
type updater struct {
	mu        sync.Mutex
	view      status.Update
	manifest  *update.Manifest
	lastCheck time.Time
	failed    []string
	busy      bool
	traffic   []int64 // client: tunnel bps samples, newest last
	srvProto  int     // client: the active server's protocol
	clients   map[string]clientVersion
	backup    backupStatus
	checkNow  chan struct{}
	installNw chan string
	// tests and the e2e
	platform string
	pkgDir   string
	ipkCmd   string
	now      func() time.Time
}

func newUpdater() *updater {
	return &updater{clients: map[string]clientVersion{}, checkNow: make(chan struct{}, 1), installNw: make(chan string, 1), now: time.Now}
}

func (a *Agent) updateStore() update.Store {
	return update.Store{Dir: filepath.Join(a.store.Get().StateDir, "update")}
}

func (a *Agent) mirror() update.Mirror {
	u := a.store.Get().UpdateOrDefault()
	return update.Mirror{Dir: filepath.Join(a.store.Get().StateDir, "update", "mirror"), MaxBytes: int64(u.MirrorMaxMB) << 20}
}

func (a *Agent) trustedKeys() []ed25519.PublicKey {
	u := a.store.Get().UpdateOrDefault()
	return update.ParseKeys(update.BuildKeys, strings.Join(u.TrustedKeys, ","))
}

func (a *Agent) updatePolicy() update.Policy {
	u := a.store.Get().UpdateOrDefault()
	return update.Policy{Mode: u.Policy, Pin: u.Pin, Window: u.Window, DeferKbps: u.DeferKbps}
}

func (a *Agent) platform() string {
	if a.upd.platform != "" {
		return a.upd.platform
	}
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		if _, err := exec.LookPath("opkg"); err == nil {
			return platformOpenWrt
		}
	}
	return platformLinux
}

// updateView is the status object (copied).
func (a *Agent) updateView() *status.Update {
	u := a.store.Get().UpdateOrDefault()
	a.upd.mu.Lock()
	v := a.upd.view
	clients := make(map[string]clientVersion, len(a.upd.clients))
	for k, c := range a.upd.clients {
		clients[k] = c
	}
	a.upd.mu.Unlock()
	v.Current, v.Policy, v.Channel, v.Pin = a.opts.Version, u.Policy, u.Channel, u.Pin
	if a.role == config.RoleServer {
		v.Window = u.Window
		for _, usr := range a.store.Get().Server.Users {
			if c, ok := clients[usr.Name]; ok && update.Newer(a.opts.Version, c.Version) {
				v.Outdated = append(v.Outdated, status.Outdated{Name: usr.Name, Kind: c.Kind, Version: c.Version})
			}
		}
		if *u.Mirror {
			ms := &status.MirrorStatus{Bytes: a.mirror().Size()}
			ents, _ := os.ReadDir(a.mirror().FilesDir())
			ms.Files = len(ents)
			if raw, _, ok := a.mirror().Manifest(u.Channel); ok {
				var m update.Manifest
				if json.Unmarshal(raw, &m) == nil {
					ms.Version = m.Version
				}
			}
			v.Mirror = ms
		}
	}
	if u.Policy == update.PolicyOff && v.State == "idle" {
		v.State = "off"
	}
	return &v
}

func (a *Agent) setUpdate(fn func(v *status.Update)) {
	a.upd.mu.Lock()
	fn(&a.upd.view)
	a.upd.mu.Unlock()
}

// noteClientVersion records what a paired client reported (server role).
func (a *Agent) noteClientVersion(r *http.Request, client string) {
	v := strings.TrimSpace(r.Header.Get("X-Tailbond-Version"))
	if client == "" || v == "" || len(v) > 40 {
		return
	}
	kind := r.Header.Get("X-Tailbond-Kind")
	if kind != "router" && kind != "android" {
		kind = ""
	}
	a.upd.mu.Lock()
	a.upd.clients[client] = clientVersion{Version: v, Kind: kind, At: time.Now()}
	a.upd.mu.Unlock()
}

// clientVersions for the server status (clients[].version/kind).
func (a *Agent) clientVersions() map[string]clientVersion {
	a.upd.mu.Lock()
	defer a.upd.mu.Unlock()
	out := make(map[string]clientVersion, len(a.upd.clients))
	for k, v := range a.upd.clients {
		out[k] = v
	}
	return out
}

// updateLoop runs for the agent's lifetime.
func (a *Agent) updateLoop(ctx context.Context) {
	a.setUpdate(func(v *status.Update) { v.State = "idle" })
	a.loadUpdateResult()
	if st := a.updateStore().Load(); st.Trial != nil && update.Same(st.Trial.Version, a.opts.Version) {
		go a.trialWatch(ctx, *st.Trial)
	}
	first := time.NewTimer(20 * time.Second)
	t := time.NewTicker(time.Minute)
	sample := time.NewTicker(10 * time.Second)
	defer first.Stop()
	defer t.Stop()
	defer sample.Stop()
	for {
		requested := ""
		select {
		case <-ctx.Done():
			return
		case <-sample.C:
			a.sampleTraffic()
			continue
		case <-first.C:
		case <-t.C:
		case <-a.upd.checkNow:
			a.upd.mu.Lock()
			a.upd.lastCheck = time.Time{}
			a.upd.mu.Unlock()
		case requested = <-a.upd.installNw:
		}
		a.updateTick(ctx, requested)
	}
}

func (a *Agent) sampleTraffic() {
	if a.role != config.RoleClient {
		return
	}
	st := a.Snapshot()
	a.upd.mu.Lock()
	a.upd.traffic = append(a.upd.traffic, st.Totals.TxBps+st.Totals.RxBps)
	if len(a.upd.traffic) > 6 {
		a.upd.traffic = a.upd.traffic[len(a.upd.traffic)-6:]
	}
	a.upd.mu.Unlock()
}

func (a *Agent) trafficKbps() int {
	a.upd.mu.Lock()
	defer a.upd.mu.Unlock()
	if len(a.upd.traffic) == 0 {
		return 0
	}
	var sum int64
	for _, x := range a.upd.traffic {
		sum += x
	}
	return int(sum / int64(len(a.upd.traffic)) / 1000)
}

// updateTick checks when due and installs what the policy allows.
// requested: "Install now" ("latest" or a version).
func (a *Agent) updateTick(ctx context.Context, requested string) {
	u := a.store.Get().UpdateOrDefault()
	if u.Policy == update.PolicyOff && requested == "" {
		a.setUpdate(func(v *status.Update) { v.State = "off" })
		return
	}
	a.upd.mu.Lock()
	if a.upd.busy {
		a.upd.mu.Unlock()
		return
	}
	due := a.upd.lastCheck.IsZero() || time.Since(a.upd.lastCheck) >= time.Duration(u.CheckEveryHours)*time.Hour
	a.upd.busy = true
	a.upd.mu.Unlock()
	defer func() {
		a.upd.mu.Lock()
		a.upd.busy = false
		a.upd.mu.Unlock()
	}()
	if due || requested != "" {
		a.checkManifest(ctx)
	}
	a.upd.mu.Lock()
	m := a.upd.manifest
	a.upd.mu.Unlock()
	if m == nil {
		return
	}
	st := a.updateStore().Load()
	d := update.Decide(a.updatePolicy(), update.Situation{
		Now: time.Now(), Server: a.role == config.RoleServer, Current: a.opts.Version, Manifest: m,
		Failed: st.Failed, ServerProtocol: a.serverProtocol(ctx), TrafficKbps: a.trafficKbps(), Requested: requested != "",
	})
	a.setUpdate(func(v *status.Update) {
		v.Deferred, v.DeferredCode = "", ""
		switch d.Action {
		case update.ActionWait, update.ActionBlocked:
			v.State, v.Deferred, v.DeferredCode = "waiting", d.Reason, d.Code
		case update.ActionNone:
			if v.State != "failed" && v.State != "rolled_back" {
				v.State = "idle"
			}
		}
	})
	if d.Action != update.ActionInstall {
		return
	}
	if err := a.install(ctx, m); err != nil {
		a.Logf("update to %s: %v", m.Version, err)
		a.setUpdate(func(v *status.Update) {
			v.State, v.LastResult, v.Error = "failed", "failed", err.Error()
			v.LastResultVersion, v.LastResultAt = m.Version, time.Now().UTC().Format(time.RFC3339)
		})
	}
}

// sources: the server's own mirror (offline uploads), the home server's
// mirror (clients), then the public source.
func (a *Agent) updateSources(ctx context.Context) []update.Source {
	u := a.store.Get().UpdateOrDefault()
	var out []update.Source
	if a.role == config.RoleClient {
		for _, base := range a.mirrorBases(ctx) {
			out = append(out, update.Source{Base: base, Mirror: true, Header: a.clientHeaders()})
		}
	}
	src := u.Source
	if src == "" {
		src = update.DefaultSource
	}
	if src != "" {
		out = append(out, update.Source{Base: src})
	}
	return out
}

// mirrorBases: the active server's mirror through the tunnel, then over
// Tailscale (its MagicDNS name) when the tunnel is down.
func (a *Agent) mirrorBases(ctx context.Context) []string {
	a.mu.Lock()
	p := a.client.payload
	ready := a.rt.ready
	a.mu.Unlock()
	if p == nil {
		return nil
	}
	var out []string
	if ready {
		host := p.TunnelSubnet().Addr().Next().String()
		if len(p.DNS) > 0 {
			host = p.DNS[0]
		}
		out = append(out, "http://"+net.JoinHostPort(host, a.apiPort)+"/updates")
	}
	if p.SNI != "" && strings.Contains(p.SNI, ".ts.net") && a.ts.Get(ctx).Running {
		out = append(out, "http://"+net.JoinHostPort(p.SNI, a.apiPort)+"/updates")
	}
	return out
}

func (a *Agent) clientHeaders() http.Header {
	h := http.Header{}
	a.mu.Lock()
	if a.client != nil && a.client.payload != nil {
		h.Set("Authorization", "Bearer "+APIToken(a.client.payload.Key))
	}
	a.mu.Unlock()
	h.Set("X-Tailbond-Version", a.opts.Version)
	h.Set("X-Tailbond-Kind", "router")
	return h
}

// serverProtocol: the active server's protocol from GET /api/hello through
// the tunnel (clients only; 0 = unknown).
func (a *Agent) serverProtocol(ctx context.Context) int {
	if a.role != config.RoleClient {
		return 0
	}
	for _, base := range a.mirrorBases(ctx) {
		h, err := a.fetchHello(ctx, strings.TrimSuffix(strings.TrimPrefix(base, "http://"), "/updates"))
		if err == nil {
			a.upd.mu.Lock()
			a.upd.srvProto = h.Protocol
			a.upd.mu.Unlock()
			return h.Protocol
		}
	}
	a.upd.mu.Lock()
	defer a.upd.mu.Unlock()
	return a.upd.srvProto
}

// checkManifest fetches and verifies the channel manifest; servers also
// mirror it and its artifacts.
func (a *Agent) checkManifest(ctx context.Context) {
	u := a.store.Get().UpdateOrDefault()
	keys := a.trustedKeys()
	a.setUpdate(func(v *status.Update) { v.State, v.Error = "checking", "" })
	now := time.Now().UTC()
	a.upd.mu.Lock()
	a.upd.lastCheck = now
	a.upd.mu.Unlock()
	if len(keys) == 0 {
		a.setUpdate(func(v *status.Update) {
			v.State, v.Error, v.LastCheck = "off", update.ErrNoKeys.Error(), &now
		})
		return
	}
	f := &update.Fetcher{}
	got, err := f.Manifest(ctx, a.updateSources(ctx), u.Channel, keys)
	fetchErr := ""
	if err != nil {
		fetchErr = err.Error()
	}
	src := "public"
	if got != nil && got.From.Mirror {
		src = "mirror"
	}
	if a.role == config.RoleServer {
		// the server's own cache: an offline upload newer than what the
		// public source has, or the last good manifest when it's unreachable
		if raw, sig, ok := a.mirror().Manifest(u.Channel); ok {
			if m, verr := update.Verify(raw, sig, keys); verr == nil {
				switch {
				case got == nil:
					got, src = &update.Fetched{Manifest: m, Raw: raw, Sig: sig}, "cache"
					if a.mirror().Origin(u.Channel) == "upload" {
						src = "upload"
					}
					if u.Source == "" && update.DefaultSource == "" {
						fetchErr = "" // offline on purpose: uploads only
					}
				case update.Newer(m.Version, got.Manifest.Version):
					got, src = &update.Fetched{Manifest: m, Raw: raw, Sig: sig}, "upload"
				}
			}
		}
	}
	if got == nil {
		a.setUpdate(func(v *status.Update) { v.State, v.Error, v.LastCheck = "idle", fetchErr, &now })
		return
	}
	a.upd.mu.Lock()
	a.upd.manifest = got.Manifest
	a.upd.mu.Unlock()
	a.setUpdate(func(v *status.Update) {
		v.State, v.LastCheck, v.Source, v.NotesURL, v.Security, v.Error = "idle", &now, src, got.Manifest.NotesURL, got.Manifest.Security, fetchErr
		v.Available = ""
		if update.Newer(got.Manifest.Version, a.opts.Version) || !isRelease(a.opts.Version) {
			v.Available = got.Manifest.Version
		}
	})
	if a.role == config.RoleServer && *u.Mirror && (src == "public" || src == "mirror") {
		if err := a.syncMirror(ctx, u.Channel, got); err != nil {
			a.Logf("update mirror: %v", err)
		}
	}
}

func isRelease(v string) bool {
	_, ok := update.Parse(v)
	return ok
}

// syncMirror stores the manifest and downloads its artifacts (skipping
// ones already cached), keeping the previous version's (keep-last-2).
func (a *Agent) syncMirror(ctx context.Context, channel string, got *update.Fetched) error {
	mir := a.mirror()
	var prev *update.Manifest
	if raw, _, ok := mir.Manifest(channel); ok {
		var m update.Manifest
		if json.Unmarshal(raw, &m) == nil && m.Version != got.Manifest.Version {
			prev = &m
		}
	}
	if err := mir.Store(channel, got.Raw, got.Sig); err != nil {
		return err
	}
	mir.SetOrigin(channel, "public")
	f := &update.Fetcher{}
	src := []update.Source{got.From}
	var total int64
	for _, art := range got.Manifest.Artifacts {
		total += art.Size
		if mir.MaxBytes > 0 && total > mir.MaxBytes {
			a.Logf("update mirror: %s skipped (over mirror_max_mb)", art.Name)
			continue
		}
		dst := mir.FilePath(art.Name)
		if dst == "" || update.CheckFile(dst, art) == nil {
			continue
		}
		if _, err := f.Download(ctx, src, art, dst); err != nil {
			a.Logf("update mirror: %v", err)
		}
	}
	mir.Prune([]*update.Manifest{got.Manifest}, []*update.Manifest{prev})
	return nil
}

// install applies a release on this platform.
func (a *Agent) install(ctx context.Context, m *update.Manifest) error {
	a.setUpdate(func(v *status.Update) { v.State = "downloading" })
	if a.platform() == platformOpenWrt {
		return a.installIPK(ctx, m)
	}
	return a.installLinux(ctx, m)
}

func linuxArch() string {
	switch goruntime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return goruntime.GOARCH
}

func (a *Agent) installLinux(ctx context.Context, m *update.Manifest) error {
	art, ok := m.Find("linux", linuxArch())
	if !ok {
		return fmt.Errorf("the release has no linux/%s build", linuxArch())
	}
	s := a.updateStore()
	dst := filepath.Join(s.DownloadDir(), art.Name)
	if _, err := (&update.Fetcher{}).Download(ctx, a.updateSources(ctx), art, dst); err != nil {
		return err
	}
	defer os.Remove(dst)
	if err := s.Unpack(dst, m.Version); err != nil {
		return err
	}
	tr := update.Trial{}
	st := a.Snapshot()
	if a.role == config.RoleServer {
		for _, c := range st.Clients {
			if c.Connected {
				tr.ClientsBefore++
			}
		}
	} else {
		tr.WasConnected = st.State == status.StateConnected
	}
	if err := s.Install(m.Version, tr); err != nil {
		return err
	}
	a.Logf("update %s -> %s installed; restarting into it (health check within %s, else rollback)", a.opts.Version, m.Version, update.TrialTimeout)
	a.setUpdate(func(v *status.Update) { v.State = "installing" })
	a.requestRestart()
	return nil
}

// opkgArchs lists the package architectures, best first.
func opkgArchs() []string {
	out, err := exec.Command("opkg", "print-architecture").Output()
	if err != nil {
		return nil
	}
	type arch struct {
		name string
		prio int
	}
	var as []arch
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 3 && f[0] == "arch" {
			p, _ := strconv.Atoi(f[2])
			as = append(as, arch{f[1], p})
		}
	}
	sort.Slice(as, func(i, j int) bool { return as[i].prio > as[j].prio })
	var names []string
	for _, x := range as {
		names = append(names, x.name)
	}
	return names
}

func (a *Agent) installIPK(ctx context.Context, m *update.Manifest) error {
	art, ok := m.Find("ipk", opkgArchs()...)
	if !ok {
		return errors.New("the release has no package for this router's architecture")
	}
	dir := a.upd.pkgDir
	if dir == "" {
		dir = defaultPkgDir
	}
	prev := ""
	if ents, err := os.ReadDir(dir); err == nil {
		var newest time.Time
		for _, e := range ents {
			if info, err := e.Info(); err == nil && strings.HasSuffix(e.Name(), ".ipk") && e.Name() != art.Name && info.ModTime().After(newest) {
				newest, prev = info.ModTime(), filepath.Join(dir, e.Name())
			}
		}
	}
	dst := filepath.Join(dir, art.Name)
	if _, err := (&update.Fetcher{}).Download(ctx, a.updateSources(ctx), art, dst); err != nil {
		return err
	}
	script := a.upd.ipkCmd
	if script == "" {
		script = defaultIPKScript
	}
	was := "0"
	if a.Snapshot().State == status.StateConnected {
		was = "1"
	}
	cmd := exec.Command(script, dst, prev)
	cmd.Env = append(os.Environ(), "TB_WAS_CONNECTED="+was, "TB_ROLE="+a.role)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	a.Logf("update %s -> %s: %s %s started (it restarts tailbond; rollback to the previous package if unhealthy)", a.opts.Version, m.Version, script, art.Name)
	a.setUpdate(func(v *status.Update) { v.State = "installing" })
	return nil
}

// loadUpdateResult shows the last result: the Linux store's or the OpenWrt
// updater's result file.
func (a *Agent) loadUpdateResult() {
	var r *update.Result
	if st := a.updateStore().Load(); st.Last != nil {
		r = st.Last
		a.upd.mu.Lock()
		a.upd.failed = st.Failed
		a.upd.mu.Unlock()
	}
	if b, err := os.ReadFile(filepath.Join(a.store.Get().StateDir, resultFileName)); err == nil {
		var x update.Result
		if json.Unmarshal(b, &x) == nil && (r == nil || x.At > r.At) {
			r = &x
		}
	}
	if r == nil {
		return
	}
	a.setUpdate(func(v *status.Update) {
		v.LastResult, v.Detail, v.LastResultVersion, v.LastResultAt = r.Result, r.Detail, r.Version, r.At
		if r.Result == "rolled_back" {
			v.State = "rolled_back"
		}
	})
	if r.Detail == "reinstalled_after_firmware_upgrade" {
		a.Logf("tailbond was reinstalled after a firmware upgrade (%s)", r.Version)
	}
}

// trialWatch: this version is on trial; commit once healthy, else roll
// back and restart into the previous one.
func (a *Agent) trialWatch(ctx context.Context, tr update.Trial) {
	a.setUpdate(func(v *status.Update) { v.State = "trial" })
	a.Logf("update %s on trial until %s", tr.Version, tr.Deadline.Local().Format("15:04:05"))
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		ok, why := a.healthy(tr)
		if ok {
			_ = a.updateStore().Commit("healthy")
			a.Logf("update %s is healthy; keeping it", tr.Version)
			a.setUpdate(func(v *status.Update) {
				v.State, v.LastResult, v.Detail = "idle", "ok", ""
				v.LastResultVersion, v.LastResultAt = tr.Version, time.Now().UTC().Format(time.RFC3339)
			})
			return
		}
		if time.Now().After(tr.Deadline) {
			_ = a.updateStore().Rollback(why)
			a.Logf("update %s failed its health check (%s); rolling back", tr.Version, why)
			a.requestRestart()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// healthy: the API serves (we're answering), mqvpn runs, and the tunnel or
// the clients are back as they were before the update.
func (a *Agent) healthy(tr update.Trial) (bool, string) {
	st := a.Snapshot()
	if a.role == config.RoleServer {
		if st.State != status.StateConnected {
			return false, "mqvpn isn't running"
		}
		if tr.ClientsBefore > 0 {
			for _, c := range st.Clients {
				if c.Connected {
					return true, ""
				}
			}
			return false, "no client reconnected"
		}
		return true, ""
	}
	if tr.WasConnected && st.State != status.StateConnected {
		return false, "the tunnel didn't come back"
	}
	if !tr.WasConnected && st.State == status.StateError {
		return false, "mqvpn fails to start"
	}
	return true, ""
}

// requestRestart ends Run with ErrRestart.
func (a *Agent) requestRestart() {
	select {
	case a.restart <- struct{}{}:
	default:
	}
}

// MqvpnOverride: when this binary was exec'd from an update directory, use
// that version's mqvpn.
func MqvpnOverride() string {
	if os.Getenv(update.EnvExeced) == "" {
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	p := filepath.Join(filepath.Dir(exe), "mqvpn")
	if fi, err := os.Stat(p); err == nil && fi.Mode()&0o111 != 0 {
		return p
	}
	return ""
}
