package coexist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// Runner runs a command (tests fake it).
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// State is what is in place: the target of Apply, and the restore record.
type State struct {
	OS     string   `json:"os"` // darwin | linux
	Tun    string   `json:"tun,omitempty"`
	TunRts []string `json:"tun_routes,omitempty"`
	// Direct routes and the gateway/interface they went via.
	Direct    []string `json:"direct_routes,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	UplinkDev string   `json:"uplink_dev,omitempty"`
	// DNS: the split domains and the server they point at.
	DNSServer  string   `json:"dns_server,omitempty"`
	DNSDomains []string `json:"dns_domains,omitempty"`
	// Kill switch allowances.
	PassIfaces []string `json:"pass_ifaces,omitempty"`
	PassHosts  []string `json:"pass_hosts,omitempty"`
}

// Want turns a plan into the state to have (dnsServer "" = no split DNS).
func Want(osName, tun string, pl Plan, up Uplink, dnsServer string, domains []string, killSwitch bool) State {
	s := State{OS: osName, Tun: tun, Gateway: up.Gateway, UplinkDev: up.Iface}
	for _, p := range pl.TunRoutes {
		s.TunRts = append(s.TunRts, p.String())
	}
	for _, p := range pl.DirectRoutes {
		s.Direct = append(s.Direct, p.String())
	}
	if pl.SplitDNS && dnsServer != "" {
		s.DNSServer, s.DNSDomains = dnsServer, domains
	}
	if killSwitch {
		s.PassIfaces = slices.Clone(pl.PassIfaces)
		for _, a := range pl.PassHosts {
			s.PassHosts = append(s.PassHosts, a.String())
		}
		sort.Strings(s.PassHosts)
	}
	return s
}

// Applier makes the OS match a State, recording it (RecordPath) before
// anything changes so Restore (at stop, or at the next start after a crash)
// takes it all away.
type Applier struct {
	OS          string // "darwin" | "linux"
	RecordPath  string
	ResolverDir string // darwin: /etc/resolver
	Run         Runner
	Logf        func(string, ...any)
}

func (a *Applier) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r := a.Run
	if r == nil {
		r = execRunner
	}
	out, err := r(ctx, name, args...)
	if err != nil && a.Logf != nil {
		a.Logf("coexist: %s %s: %v: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, err
}

func (a *Applier) resolverDir() string {
	if a.ResolverDir != "" {
		return a.ResolverDir
	}
	return "/etc/resolver"
}

// Current is the recorded state (zero when nothing is in place).
func (a *Applier) Current() State {
	var s State
	if b, err := os.ReadFile(a.RecordPath); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func (a *Applier) save(s State) error {
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := a.RecordPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.RecordPath)
}

// Apply moves from the recorded state to want: removes what's gone, adds
// what's new. The record is the union while it works, want afterwards.
func (a *Applier) Apply(ctx context.Context, want State) error {
	cur := a.Current()
	union := want
	union.TunRts = unionStr(cur.TunRts, want.TunRts)
	union.Direct = unionStr(cur.Direct, want.Direct)
	union.DNSDomains = unionStr(cur.DNSDomains, want.DNSDomains)
	if union.DNSServer == "" {
		union.DNSServer = cur.DNSServer
	}
	union.PassIfaces = unionStr(cur.PassIfaces, want.PassIfaces)
	union.PassHosts = unionStr(cur.PassHosts, want.PassHosts)
	if err := a.save(union); err != nil {
		return err
	}
	var errs []error
	// tunnel interface changed (mqvpn restarted): all tunnel routes again
	tunMoved := cur.Tun != "" && cur.Tun != want.Tun
	for _, r := range cur.TunRts {
		if tunMoved || !slices.Contains(want.TunRts, r) {
			a.delRoute(ctx, cur.OS, r, cur.Tun, "")
		}
	}
	gwMoved := cur.Gateway != want.Gateway || cur.UplinkDev != want.UplinkDev
	for _, r := range cur.Direct {
		if gwMoved || !slices.Contains(want.Direct, r) {
			a.delRoute(ctx, cur.OS, r, cur.UplinkDev, cur.Gateway)
		}
	}
	for _, r := range want.TunRts {
		if tunMoved || !slices.Contains(cur.TunRts, r) {
			if err := a.addRoute(ctx, want.OS, r, want.Tun, ""); err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, r := range want.Direct {
		if gwMoved || !slices.Contains(cur.Direct, r) {
			if err := a.addRoute(ctx, want.OS, r, want.UplinkDev, want.Gateway); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if !sameDNS(cur, want) {
		a.clearDNS(ctx, cur)
		if want.DNSServer != "" {
			if err := a.setDNS(ctx, want); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if !slices.Equal(cur.PassIfaces, want.PassIfaces) || !slices.Equal(cur.PassHosts, want.PassHosts) || cur.Tun != want.Tun {
		a.clearPass(ctx, cur)
		if len(want.PassIfaces)+len(want.PassHosts) > 0 {
			if err := a.setPass(ctx, want); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if err := a.save(want); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Restore takes away everything recorded.
func (a *Applier) Restore(ctx context.Context) error {
	cur := a.Current()
	if cur.OS == "" {
		return nil
	}
	for _, r := range cur.TunRts {
		a.delRoute(ctx, cur.OS, r, cur.Tun, "")
	}
	for _, r := range cur.Direct {
		a.delRoute(ctx, cur.OS, r, cur.UplinkDev, cur.Gateway)
	}
	a.clearDNS(ctx, cur)
	a.clearPass(ctx, cur)
	if err := os.Remove(a.RecordPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func unionStr(a, b []string) []string {
	out := slices.Clone(a)
	for _, s := range b {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func sameDNS(a, b State) bool {
	return a.DNSServer == b.DNSServer && slices.Equal(a.DNSDomains, b.DNSDomains) && (a.OS != "linux" || a.Tun == b.Tun)
}

// ---- routes ----

func (a *Applier) addRoute(ctx context.Context, osName, prefix, dev, gw string) error {
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return err
	}
	if osName == "linux" {
		args := []string{"route", "replace", p.String()}
		if gw != "" {
			args = append(args, "via", gw)
		}
		if dev != "" {
			args = append(args, "dev", dev)
		}
		if p.Addr().Is6() {
			args = append([]string{"-6"}, args...)
		}
		_, err = a.run(ctx, "ip", args...)
		return err
	}
	args := darwinRouteArgs("add", p, dev, gw)
	out, err := a.run(ctx, "route", args...)
	if err != nil && strings.Contains(string(out), "File exists") {
		// already there (another run, or the interface's own route): replace
		_, _ = a.run(ctx, "route", darwinRouteArgs("delete", p, dev, gw)...)
		_, err = a.run(ctx, "route", args...)
	}
	return err
}

func (a *Applier) delRoute(ctx context.Context, osName, prefix, dev, gw string) {
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return
	}
	if osName == "linux" {
		args := []string{"route", "del", p.String()}
		if dev != "" {
			args = append(args, "dev", dev)
		}
		if p.Addr().Is6() {
			args = append([]string{"-6"}, args...)
		}
		_, _ = a.run(ctx, "ip", args...)
		return
	}
	_, _ = a.run(ctx, "route", darwinRouteArgs("delete", p, dev, gw)...)
}

// darwinRouteArgs: route -n add -inet[6] -host|-net P (-interface DEV | GW)
func darwinRouteArgs(verb string, p netip.Prefix, dev, gw string) []string {
	fam := "-inet"
	if p.Addr().Is6() {
		fam = "-inet6"
	}
	args := []string{"-n", verb, fam}
	if p.IsSingleIP() {
		args = append(args, "-host", p.Addr().String())
	} else {
		args = append(args, "-net", p.String())
	}
	if gw != "" {
		args = append(args, gw)
	} else if dev != "" {
		args = append(args, "-interface", dev)
	}
	return args
}

// ---- split DNS ----

// ResolverMarker is the first line of the /etc/resolver files tailbond
// writes; a file without it isn't touched.
const ResolverMarker = "# tailbond: split DNS for the tailnet (docs/macos.md)"

func (a *Applier) setDNS(ctx context.Context, s State) error {
	if s.OS == "linux" {
		var errs []error
		if _, err := a.run(ctx, "resolvectl", "dns", s.Tun, s.DNSServer); err != nil {
			errs = append(errs, err)
		}
		args := []string{"domain", s.Tun}
		for _, d := range s.DNSDomains {
			args = append(args, "~"+d)
		}
		if _, err := a.run(ctx, "resolvectl", args...); err != nil {
			errs = append(errs, err)
		}
		// only these domains: never the default route for every name
		if _, err := a.run(ctx, "resolvectl", "default-route", s.Tun, "false"); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	if err := os.MkdirAll(a.resolverDir(), 0o755); err != nil {
		return err
	}
	var errs []error
	for _, d := range s.DNSDomains {
		p := filepath.Join(a.resolverDir(), d)
		if b, err := os.ReadFile(p); err == nil && !strings.HasPrefix(string(b), ResolverMarker) {
			errs = append(errs, fmt.Errorf("%s exists and isn't tailbond's: left alone", p))
			continue
		}
		body := ResolverMarker + "\nnameserver " + s.DNSServer + "\n"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			errs = append(errs, err)
		}
	}
	a.flushDNS(ctx)
	return errors.Join(errs...)
}

func (a *Applier) clearDNS(ctx context.Context, s State) {
	if s.DNSServer == "" && len(s.DNSDomains) == 0 {
		return
	}
	if s.OS == "linux" {
		if s.Tun != "" {
			_, _ = a.run(ctx, "resolvectl", "revert", s.Tun)
		}
		return
	}
	for _, d := range s.DNSDomains {
		p := filepath.Join(a.resolverDir(), d)
		if b, err := os.ReadFile(p); err == nil && strings.HasPrefix(string(b), ResolverMarker) {
			_ = os.Remove(p)
		}
	}
	a.flushDNS(ctx)
}

func (a *Applier) flushDNS(ctx context.Context) {
	_, _ = a.run(ctx, "dscacheutil", "-flushcache")
	_, _ = a.run(ctx, "killall", "-HUP", "mDNSResponder")
}

// ---- kill switch allowances ----

// PFAnchor sorts before mqvpn's com.apple/250.mqvpn: pf evaluates the
// com.apple anchors in name order, and these are "quick" passes.
const PFAnchor = "com.apple/249.tailbond"

const iptComment = "tailbond-coexist"

func (a *Applier) setPass(ctx context.Context, s State) error {
	if s.OS == "linux" {
		var errs []error
		for _, spec := range iptSpecs(s) {
			if _, err := a.run(ctx, spec[0], append([]string{"-I", "OUTPUT"}, spec[1:]...)...); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	var b strings.Builder
	for _, i := range s.PassIfaces {
		fmt.Fprintf(&b, "pass out quick on %s all\npass in quick on %s all\n", i, i)
	}
	for _, h := range s.PassHosts {
		fam := "inet"
		if strings.Contains(h, ":") {
			fam = "inet6"
		}
		fmt.Fprintf(&b, "pass out quick %s to %s\n", fam, h)
	}
	r := a.Run
	if r == nil {
		r = pfLoad
	}
	_, err := r(ctx, "pfctl", "-a", PFAnchor, "-f", "-", "\x00"+b.String())
	return err
}

// pfLoad runs pfctl with the last argument ("\x00rules") on stdin.
func pfLoad(ctx context.Context, name string, args ...string) ([]byte, error) {
	if n := len(args); n > 0 && strings.HasPrefix(args[n-1], "\x00") {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args[:n-1]...)
		cmd.Stdin = strings.NewReader(args[n-1][1:])
		return cmd.CombinedOutput()
	}
	return execRunner(ctx, name, args...)
}

func (a *Applier) clearPass(ctx context.Context, s State) {
	if len(s.PassIfaces)+len(s.PassHosts) == 0 {
		return
	}
	if s.OS == "linux" {
		for _, spec := range iptSpecs(s) {
			for range 4 { // duplicates from a crash
				if _, err := a.run(ctx, spec[0], append([]string{"-D", "OUTPUT"}, spec[1:]...)...); err != nil {
					break
				}
			}
		}
		return
	}
	_, _ = a.run(ctx, "pfctl", "-a", PFAnchor, "-F", "all")
}

// iptSpecs: [binary, match..., -j ACCEPT] per allowance.
func iptSpecs(s State) [][]string {
	var out [][]string
	for _, i := range s.PassIfaces {
		for _, bin := range []string{"iptables", "ip6tables"} {
			out = append(out, []string{bin, "-o", i, "-m", "comment", "--comment", iptComment, "-j", "ACCEPT"})
		}
	}
	for _, h := range s.PassHosts {
		bin := "iptables"
		if strings.Contains(h, ":") {
			bin = "ip6tables"
		}
		out = append(out, []string{bin, "-d", h, "-m", "comment", "--comment", iptComment, "-j", "ACCEPT"})
	}
	return out
}
