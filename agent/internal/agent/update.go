package agent

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
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

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/update"
)

// Updates (docs/architecture.md, "Updates"): check GitHub's latest release
// about daily, show it, and install it with the platform's normal mechanism:
// the static bundle's install.sh --update, opkg on OpenWrt; RPM installs
// update through dnf.

// ErrRestart asks main to exit with ExitRestart (systemd restarts us):
// after a restore.
var ErrRestart = errors.New("restarting")

// ExitRestart is the exit status that asks systemd for a restart
// (RestartForceExitStatus=75 in the units).
const ExitRestart = 75

const (
	methodStatic = "static"
	methodOpkg   = "opkg"
	methodDnf    = "dnf"
	methodBrew   = "brew" // macOS, installed by the Homebrew formula: brew upgrades it
	methodNone   = "none"

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
	mu         sync.Mutex
	view       status.Update
	rel        *update.Release
	lastCheck  time.Time
	busy       bool
	clients    map[string]clientVersion
	backup     backupStatus
	checkNow   chan struct{}
	installNw  chan struct{}
	resultAt   time.Time // mtime of the result file last read
	rolledBack string    // a version that didn't start here: auto skips it
	// tests and the e2e
	method     string
	installEnv string
	pkgDir     string
	ipkCmd     string
	runner     func(name string, args, env []string) error // detached installer
	now        func() time.Time
	dnfAuto    func() string
	// the check cadence and "Right away" (docs/updates.md)
	etag         string    // of the last GitHub answer: If-None-Match
	nextCheck    time.Time // zero: due now (agent start)
	pendingVer   string    // the release waiting to install by itself
	pendingSince time.Time // since when (the 30-minute deferral cap)
	jitter       func() time.Duration
	trafficBusy  func() bool // tests
	// the last version installed by itself, and when: not again within
	// retryInstallAfter (the installer restarts tailbondd; until then the
	// running version is still the old one)
	attemptedVer string
	attemptedAt  time.Time
}

const retryInstallAfter = time.Hour

// testUpdateEvery (TB_TEST_UPDATE_EVERY, e.g. "15s"; tests only:
// packaging/linux/test-update.sh) replaces the hourly cycle of immediate and
// auto, without jitter.
var testUpdateEvery, _ = time.ParseDuration(os.Getenv("TB_TEST_UPDATE_EVERY"))

const (
	// immediate and auto check hourly (plus up to frequentJitter, so a
	// fleet doesn't ask GitHub at the same second); notify every
	// check_every_hours (24)
	frequentEvery  = time.Hour
	frequentJitter = 10 * time.Minute
	// "Right away" waits for a quiet moment at most this long
	immediateDeferCap = 30 * time.Minute
	// a call: traffic both ways at least busyCallBps; heavy traffic: both
	// directions together at least busyHeavyBps (bits per second)
	busyCallBps  = 48_000
	busyHeavyBps = 2_000_000
)

// checkEvery is how often a policy checks for a release.
func checkEvery(u config.UpdateSettings) time.Duration {
	every := time.Duration(u.CheckEveryHours) * time.Hour
	if u.Policy == config.UpdateImmediate || u.Policy == config.UpdateAuto {
		if testUpdateEvery > 0 {
			return testUpdateEvery
		}
		if every > frequentEvery {
			every = frequentEvery
		}
	}
	return every
}

// installVerdict says whether a pending release installs by itself now,
// and if not, why (shown in the dashboard). pendingSince is when this
// release was first found.
func installVerdict(policy string, rel *update.Release, busy, inWindow bool, pendingSince, now time.Time) (bool, string) {
	switch policy {
	case config.UpdateAuto:
		if inWindow {
			return true, ""
		}
		return false, fmt.Sprintf("installs by itself between %02d:00 and %02d:00", autoFromHour, autoToHour)
	case config.UpdateImmediate:
		switch {
		case rel.Security:
			return true, "" // security releases don't wait
		case !busy:
			return true, ""
		case now.Sub(pendingSince) >= immediateDeferCap:
			return true, "" // waited long enough: install anyway
		}
		return false, "installs after the call or download in progress (at most 30 minutes)"
	}
	return false, ""
}

// trafficLooksBusy: a call (traffic both ways) or heavy traffic.
func trafficLooksBusy(t status.Totals) bool {
	return (t.TxBps >= busyCallBps && t.RxBps >= busyCallBps) || t.TxBps+t.RxBps >= busyHeavyBps
}

func (a *Agent) isTrafficBusy() bool {
	if a.upd.trafficBusy != nil {
		return a.upd.trafficBusy()
	}
	a.mu.Lock()
	snap := a.snap
	a.mu.Unlock()
	return snap != nil && trafficLooksBusy(snap.Totals)
}

func (a *Agent) checkJitter() time.Duration {
	if a.upd.jitter != nil {
		return a.upd.jitter()
	}
	if testUpdateEvery > 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(frequentJitter)))
}

// Automatic installs happen at night (local time), when nobody is on a
// call: devices drop for a few seconds while tailbond restarts.
const autoFromHour, autoToHour = 3, 5

func (a *Agent) updNow() time.Time {
	if a.upd.now != nil {
		return a.upd.now()
	}
	return time.Now()
}

func (a *Agent) inAutoWindow() bool {
	h := a.updNow().Hour()
	return h >= autoFromHour && h < autoToHour
}

// dnfAutomatic (Fedora RPM installs): "install" when a dnf-automatic timer
// installs updates, "download" when it only downloads them (Fedora's default
// apply_updates = no), "off" without one. It covers every package, not just
// tailbond.
func (a *Agent) dnfAutomatic() string {
	if a.upd.dnfAuto != nil {
		return a.upd.dnfAuto()
	}
	on := func(t string) bool { return exec.Command("systemctl", "is-enabled", "--quiet", t).Run() == nil }
	switch {
	case on("dnf-automatic-install.timer"):
		return "install"
	case on("dnf5-automatic.timer"), on("dnf-automatic.timer"):
		if applyUpdates("/etc/dnf/automatic.conf") {
			return "install"
		}
		return "download"
	case on("dnf-automatic-download.timer"):
		return "download"
	}
	return "off"
}

// applyUpdates reads apply_updates from a dnf-automatic config.
func applyUpdates(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	on := false
	for _, l := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(l, "=")
		if ok && strings.TrimSpace(k) == "apply_updates" {
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "yes", "true", "1", "on":
				on = true
			default:
				on = false
			}
		}
	}
	return on
}

func newUpdater() *updater {
	return &updater{clients: map[string]clientVersion{}, checkNow: make(chan struct{}, 1), installNw: make(chan struct{}, 1)}
}

func (a *Agent) setUpdate(fn func(v *status.Update)) {
	a.upd.mu.Lock()
	fn(&a.upd.view)
	a.upd.mu.Unlock()
}

func (a *Agent) updateClient() update.Client {
	u := a.store.Get().UpdateOrDefault()
	return update.Client{API: u.APIURL, Token: os.Getenv("TB_GITHUB_TOKEN")}
}

func (a *Agent) updateRepo() string {
	r, err := update.RepoPath(a.store.Get().UpdateOrDefault().Repo)
	if err != nil {
		return a.store.Get().UpdateOrDefault().Repo
	}
	return r
}

// method: how this install updates (docs/architecture.md, "Install methods").
func (a *Agent) method() string {
	if a.upd.method != "" {
		return a.upd.method
	}
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		if _, err := exec.LookPath("opkg"); err == nil {
			return methodOpkg
		}
	}
	if goruntime.GOOS == "darwin" && brewInstalled() {
		return methodBrew
	}
	env := a.upd.installEnv
	if env == "" {
		env = defaultInstallEnv
	}
	if _, err := os.Stat(env); err == nil {
		return methodStatic
	}
	if exe, err := os.Executable(); err == nil && strings.HasPrefix(exe, "/usr/bin/") {
		if exec.Command("rpm", "-qf", exe).Run() == nil {
			return methodDnf
		}
	}
	return methodNone
}

// brewInstalled: this tailbondd is the Homebrew formula's (its opt/ link
// resolves into the Cellar).
func brewInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return strings.Contains(exe, "/Cellar/")
}

func isOpenWrt() bool {
	_, err := os.Stat("/etc/openwrt_release")
	return err == nil
}

// serverAPIBase (client): the active server's API through the tunnel, ""
// while it isn't up.
func (a *Agent) serverAPIBase() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil || a.client.payload == nil || !a.rt.ready {
		return ""
	}
	p := a.client.payload
	host := p.TunnelSubnet().Addr().Next().String()
	if len(p.DNS) > 0 {
		host = p.DNS[0]
	}
	return "http://" + net.JoinHostPort(host, a.apiPort)
}

func installable(m string) bool { return m == methodStatic || m == methodOpkg }

// updateView is the status object (copied).
func (a *Agent) updateView() *status.Update {
	a.loadUpdateResult() // install.sh writes it after the restart's health check
	u := a.store.Get().UpdateOrDefault()
	a.upd.mu.Lock()
	v := a.upd.view
	clients := make(map[string]clientVersion, len(a.upd.clients))
	for k, c := range a.upd.clients {
		clients[k] = c
	}
	a.upd.mu.Unlock()
	v.Current, v.Policy, v.Repo, v.Method = a.opts.Version, u.Policy, a.updateRepo(), a.method()
	v.AutoWindow = fmt.Sprintf("%02d:00-%02d:00", autoFromHour, autoToHour)
	if a.role == config.RoleServer {
		for _, usr := range a.store.Get().Server.Users {
			if c, ok := clients[usr.Name]; ok && update.Newer(a.opts.Version, c.Version) {
				v.Outdated = append(v.Outdated, status.Outdated{Name: usr.Name, Kind: c.Kind, Version: c.Version})
			}
		}
	}
	if u.Policy == config.UpdateOff && (v.State == "idle" || v.State == "") {
		v.State = "off"
	}
	return &v
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

// clientHeaders: what a client sends its server (version reporting).
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

// updateLoop runs for the agent's lifetime: a check a minute after start,
// then every check_every_hours (default 24); Check now / Update now at once.
func (a *Agent) updateLoop(ctx context.Context) {
	a.loadUpdateResult()
	a.setUpdate(func(v *status.Update) {
		if v.State == "" {
			v.State = "idle"
		}
	})
	// check right after start (the tunnel and the API have a moment to come up)
	first := time.NewTimer(5 * time.Second)
	tick := time.Minute
	if testUpdateEvery > 0 && testUpdateEvery/3 < tick {
		tick = max(testUpdateEvery/3, time.Second)
	}
	t := time.NewTicker(tick)
	defer first.Stop()
	defer t.Stop()
	for {
		check, force := false, false
		select {
		case <-ctx.Done():
			return
		case <-first.C:
			check = true
		case <-t.C:
			var install bool
			check, install = a.updateDue()
			if !check && !install {
				continue
			}
		case <-a.upd.checkNow:
			check = true
		case <-a.upd.installNw:
			check, force = true, true
		}
		a.updateTick(ctx, check, force)
	}
}

// updateDue: whether a check is due, and whether a release found earlier
// may install by itself now (a quiet moment, the night window).
func (a *Agent) updateDue() (check, install bool) {
	u := a.store.Get().UpdateOrDefault()
	now := a.updNow()
	a.upd.mu.Lock()
	defer a.upd.mu.Unlock()
	check = !now.Before(a.upd.nextCheck)
	pending := a.upd.rel != nil && update.Newer(a.upd.rel.Version, a.opts.Version) && !update.Same(a.upd.rolledBack, a.upd.rel.Version)
	install = pending && (u.Policy == config.UpdateImmediate || (u.Policy == config.UpdateAuto && a.inAutoWindow()))
	return check, install
}

// updateTick checks (when check, unless off) and installs when forced
// ("Update now") or when the policy says so.
func (a *Agent) updateTick(ctx context.Context, check, force bool) {
	a.upd.mu.Lock()
	if a.upd.busy {
		a.upd.mu.Unlock()
		return
	}
	a.upd.busy = true
	a.upd.mu.Unlock()
	defer func() {
		a.upd.mu.Lock()
		a.upd.busy = false
		a.upd.mu.Unlock()
	}()
	u := a.store.Get().UpdateOrDefault()
	if u.Policy == config.UpdateOff && !force {
		a.setUpdate(func(v *status.Update) { v.State = "off" })
		return
	}
	var rel *update.Release
	if check || force {
		rel = a.check(ctx)
	} else {
		a.upd.mu.Lock()
		rel = a.upd.rel
		a.upd.mu.Unlock()
	}
	// "Update now" on a dev build (not semver) installs the latest release:
	// that's how a build from source moves onto releases.
	if rel == nil || !(update.Newer(rel.Version, a.opts.Version) || (force && !isRelease(a.opts.Version))) {
		a.setUpdate(func(v *status.Update) { v.Waiting = "" })
		return
	}
	if !force {
		if !isRelease(a.opts.Version) {
			return // a dev build shows releases, never installs them by itself
		}
		now := a.updNow()
		a.upd.mu.Lock()
		bad := a.upd.rolledBack
		if a.upd.pendingVer != rel.Version {
			a.upd.pendingVer, a.upd.pendingSince = rel.Version, now
		}
		since := a.upd.pendingSince
		a.upd.mu.Unlock()
		if update.Same(bad, rel.Version) {
			return // it didn't start here: only "Update now" tries it again
		}
		a.upd.mu.Lock()
		tried := a.upd.attemptedVer == rel.Version && now.Sub(a.upd.attemptedAt) < retryInstallAfter
		a.upd.mu.Unlock()
		if tried {
			return // installing (tailbondd restarts when it's done)
		}
		ok, why := installVerdict(u.Policy, rel, a.isTrafficBusy(), a.inAutoWindow(), since, now)
		a.setUpdate(func(v *status.Update) { v.Waiting = why })
		if !ok {
			return
		}
		if u.Policy == config.UpdateImmediate {
			a.Logf("update: %s is out, installing it now (policy immediate%s)", rel.Version, map[bool]string{true: ", security release", false: ""}[rel.Security])
		}
	}
	m := a.method()
	if !installable(m) {
		return
	}
	if !force {
		a.upd.mu.Lock()
		a.upd.attemptedVer, a.upd.attemptedAt = rel.Version, a.updNow()
		a.upd.mu.Unlock()
	}
	if err := a.install(ctx, rel, m); err != nil {
		a.Logf("update to %s: %v", rel.Version, err)
		a.setUpdate(func(v *status.Update) {
			v.State, v.LastResult, v.Error = "failed", "failed", err.Error()
			v.LastResultVersion, v.LastResultAt = rel.Version, time.Now().UTC().Format(time.RFC3339)
		})
	}
}

// check asks GitHub for the latest release.
func (a *Agent) check(ctx context.Context) *update.Release {
	a.setUpdate(func(v *status.Update) {
		if v.State != "failed" && v.State != "rolled_back" {
			v.State = "checking"
		}
	})
	u := a.store.Get().UpdateOrDefault()
	now := a.updNow().UTC()
	a.upd.mu.Lock()
	etag, cached := a.upd.etag, a.upd.rel
	a.upd.mu.Unlock()
	if cached == nil {
		etag = "" // nothing to reuse
	}
	rel, newETag, unchanged, err := a.updateClient().LatestIfChanged(ctx, u.Repo, etag)
	if unchanged {
		rel = cached
	}
	next := now.Add(checkEvery(u))
	if u.Policy == config.UpdateImmediate || u.Policy == config.UpdateAuto {
		next = next.Add(a.checkJitter())
	}
	a.upd.mu.Lock()
	a.upd.lastCheck, a.upd.nextCheck = now, next
	if err == nil {
		a.upd.rel, a.upd.etag = rel, newETag
	} else if errors.Is(err, update.ErrNoRelease) {
		// the release went away (deleted, or the repo changed): forget it,
		// so nothing installs from the stale copy
		a.upd.rel, a.upd.etag = nil, ""
	}
	a.upd.mu.Unlock()
	dnfAuto := ""
	if a.method() == methodDnf {
		dnfAuto = a.dnfAutomatic()
	}
	a.setUpdate(func(v *status.Update) {
		v.LastCheck, v.NextCheck = &now, &next
		if a.method() == methodDnf {
			v.DnfAutomatic = dnfAuto
		}
		if v.State == "checking" {
			v.State = "idle"
		}
		v.NoRelease = errors.Is(err, update.ErrNoRelease)
		if v.NoRelease {
			v.Error, v.Available = "", ""
			return
		}
		if err != nil {
			v.Error = "couldn't check for updates: " + err.Error()
			return
		}
		v.Error, v.Available, v.NotesURL, v.Security = "", "", rel.NotesURL, rel.Security
		// a dev build (not semver) shows the latest release, never installs it
		if update.Newer(rel.Version, a.opts.Version) || !isRelease(a.opts.Version) {
			v.Available = rel.Version
		}
	})
	if err != nil {
		a.Logf("update check: %v", err)
		return nil
	}
	return rel
}

func isRelease(v string) bool {
	_, ok := update.Parse(v)
	return ok
}

func (a *Agent) install(ctx context.Context, rel *update.Release, m string) error {
	a.setUpdate(func(v *status.Update) { v.State, v.Error = "downloading", "" })
	switch m {
	case methodStatic:
		return a.installStatic(ctx, rel)
	case methodOpkg:
		return a.installIPK(ctx, rel)
	}
	return fmt.Errorf("this install updates with %s", m)
}

// bundleName is the static bundle for this machine (release asset
// <name>.tar.gz, unpacking to <name>/).
func bundleName() string {
	if goruntime.GOOS == "darwin" {
		return "tailbond-macos-" + goruntime.GOARCH
	}
	return "tailbond-linux-" + bundleArch()
}

func bundleArch() string {
	if goruntime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

// installStatic: download the bundle (checked against SHA256SUMS), unpack it
// under state_dir/update/<version>/ and run its install.sh --update outside
// our own service (systemd-run), since it restarts tailbond.
func (a *Agent) installStatic(ctx context.Context, rel *update.Release) error {
	name := bundleName() + ".tar.gz"
	root := filepath.Join(a.store.Get().StateDir, "update")
	dir := filepath.Join(root, rel.Version)
	os.RemoveAll(dir)
	tgz := dir + ".tar.gz"
	defer os.Remove(tgz)
	if err := a.updateClient().Download(ctx, rel, name, tgz); err != nil {
		return err
	}
	if err := untar(tgz, dir); err != nil {
		return fmt.Errorf("unpack %s: %w", name, err)
	}
	bundle := filepath.Join(dir, bundleName())
	script := filepath.Join(bundle, "install.sh")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("%s has no install.sh", name)
	}
	pruneUpdates(root, rel.Version)
	a.Logf("update %s -> %s: %s --update (restarts tailbond; goes back to %s if it doesn't come up)", a.opts.Version, rel.Version, script, a.opts.Version)
	a.setUpdate(func(v *status.Update) { v.State = "installing" })
	return a.runDetached(script, []string{"--update", "--bundle-dir", bundle}, nil)
}

// runDetached starts an installer that outlives a restart of tailbond:
// under systemd its own transient unit (systemd-run), else a new session.
func (a *Agent) runDetached(name string, args, env []string) error {
	if a.upd.runner != nil {
		return a.upd.runner(name, args, env)
	}
	if p, err := exec.LookPath("systemd-run"); err == nil && os.Getenv("INVOCATION_ID") != "" {
		sargs := []string{"--unit", "tailbond-update-" + strconv.FormatInt(time.Now().Unix(), 10), "--collect", "--quiet"}
		for _, e := range env {
			sargs = append(sargs, "--setenv", e)
		}
		sargs = append(sargs, "--", name)
		out, err := exec.Command(p, append(sargs, args...)...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemd-run: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// untar unpacks a .tar.gz into dir (regular files and directories only,
// nothing outside dir).
func untar(tgz, dir string) error {
	f, err := os.Open(tgz)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		p := filepath.Join(dir, filepath.Clean("/"+h.Name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o755|0o600)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, io.LimitReader(tr, 512<<20))
			if cerr := w.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
}

// pruneUpdates keeps only the unpacked bundle being installed.
func pruneUpdates(dir, keep string) {
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() && e.Name() != keep {
			if _, ok := update.Parse(e.Name()); ok {
				os.RemoveAll(filepath.Join(dir, e.Name()))
			}
		}
	}
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

// stableIPK is the release asset name of the ipk for an opkg architecture
// that stays the same in every release (docs/glinet.md, Download):
// tailbond-glinet-<arch>.ipk for GL.iNet's stock firmware (neon-vfpv4),
// tailbond-openwrt-<arch>.ipk otherwise.
func stableIPK(arch string) string {
	if strings.HasSuffix(arch, "_neon-vfpv4") {
		return "tailbond-glinet-" + arch + ".ipk"
	}
	return "tailbond-openwrt-" + arch + ".ipk"
}

// ipkFor picks the release's ipk for one of the archs (best first): the
// stable name, else the versioned tailbond_<version>-r<N>_<arch>.ipk.
// local is the package cache's file name, always versioned: update-ipk
// keeps the previous package beside it and reinstall reads the version
// from it (the release has the same file under that name too).
func ipkFor(rel *update.Release, archs []string) (art update.Asset, local string, ok bool) {
	for _, arch := range archs {
		versioned, vok := rel.FindAsset(func(n string) bool {
			return strings.HasPrefix(n, "tailbond_") && strings.HasSuffix(n, "_"+arch+".ipk")
		})
		local = "tailbond_" + rel.Version + "-r1_" + arch + ".ipk"
		if vok {
			local = versioned.Name
		}
		if a, sok := rel.FindAsset(func(n string) bool { return n == stableIPK(arch) }); sok {
			return a, local, true
		}
		if vok {
			return versioned, local, true
		}
	}
	return update.Asset{}, "", false
}

// installIPK: download the ipk (checked against SHA256SUMS) into the
// package cache and run update-ipk detached (opkg install, health check,
// the previous package back on failure).
func (a *Agent) installIPK(ctx context.Context, rel *update.Release) error {
	archs := opkgArchs()
	if len(archs) == 0 && a.upd.method == methodOpkg { // tests: no opkg here
		archs = []string{"aarch64_cortex-a53"}
	}
	art, local, ok := ipkFor(rel, archs)
	if !ok {
		return errors.New("the release has no package for this router's architecture")
	}
	dir := a.upd.pkgDir
	if dir == "" {
		dir = defaultPkgDir
	}
	// the installed package, as /usr/lib/tailbond/pkg-cache records it
	// (update-ipk saves the running one itself when there's none)
	prev := ""
	if b, err := os.ReadFile(filepath.Join(dir, "current")); err == nil {
		if cur := strings.TrimSpace(string(b)); cur != "" && cur != local && !strings.ContainsAny(cur, "/\\") {
			if _, err := os.Stat(filepath.Join(dir, cur)); err == nil {
				prev = filepath.Join(dir, cur)
			}
		}
	}
	dst := filepath.Join(dir, local)
	if err := a.updateClient().Download(ctx, rel, art.Name, dst); err != nil {
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
	a.Logf("update %s -> %s: %s %s (restarts tailbond; the previous package goes back if it doesn't come up)", a.opts.Version, rel.Version, script, art.Name)
	a.setUpdate(func(v *status.Update) { v.State = "installing" })
	return a.runDetached(script, []string{dst, prev}, []string{"TB_WAS_CONNECTED=" + was, "TB_ROLE=" + a.role})
}

// loadUpdateResult shows the installers' last result
// (<state_dir>/update-result.json).
func (a *Agent) loadUpdateResult() {
	p := filepath.Join(a.store.Get().StateDir, resultFileName)
	fi, err := os.Stat(p)
	if err != nil {
		return
	}
	a.upd.mu.Lock()
	seen := fi.ModTime().Equal(a.upd.resultAt)
	a.upd.resultAt = fi.ModTime()
	a.upd.mu.Unlock()
	if seen {
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var r struct {
		Result, Version, From, At, Detail string
	}
	if json.Unmarshal(b, &r) != nil || r.Result == "" {
		return
	}
	if r.Result == "rolled_back" {
		a.upd.mu.Lock()
		a.upd.rolledBack = r.Version
		a.upd.mu.Unlock()
	}
	a.setUpdate(func(v *status.Update) {
		v.LastResult, v.Detail, v.LastResultVersion, v.LastResultAt = r.Result, r.Detail, r.Version, r.At
		if r.Result == "rolled_back" && update.Newer(r.Version, a.opts.Version) {
			v.State = "rolled_back"
		}
	})
	switch {
	case r.Detail == "reinstalled_after_firmware_upgrade":
		a.Logf("tailbond was reinstalled after a firmware upgrade (%s)", r.Version)
	case r.Result == "ok" && update.Same(r.Version, a.opts.Version):
		a.setUpdate(func(v *status.Update) {
			if v.State == "installing" || v.State == "downloading" {
				v.State = "idle"
			}
		})
	}
}

// requestRestart ends Run with ErrRestart.
func (a *Agent) requestRestart() {
	select {
	case a.restart <- struct{}{}:
	default:
	}
}
