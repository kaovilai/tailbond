package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/bypass"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/hooks"
)

// Bypass rules (client role): LAN traffic that skips the tunnel. The agent
// keeps the rules and hands them to the platform's bypass-apply hook as a
// JSON file; the hook does the nftables/ipset/dnsmasq/ip-rule work
// (docs/architecture.md, "Bypass rules").

// bypassState is guarded by Agent.mu (inside clientState).
type bypassState struct {
	applied    string // the rule file last handed to the hook
	counters   map[string]bypassCounter
	countersAt time.Time
}

// bypassCounter: connections a rule sent past the tunnel (its first
// packet; later packets follow the connection's saved mark).
type bypassCounter struct {
	Connections uint64 `json:"connections"`
}

// bypassSupport is what the hook found on this router (TB_BYPASS_STATUS).
type bypassSupport struct {
	Domains  string `json:"domains"`            // nftset | ipset | none | unknown
	Firewall string `json:"firewall,omitempty"` // nft | iptables
	Reason   string `json:"reason,omitempty"`
}

// arpPath is the neighbour table (tests point it elsewhere).
var arpPath = "/proc/net/arp"

func (a *Agent) bypassFile() string {
	return filepath.Join(a.store.Get().StateDir, "bypass.json")
}

func (a *Agent) bypassStatusFile() string {
	return filepath.Join(a.store.Get().StateDir, "bypass-status.json")
}

func (a *Agent) bypassCountersFile() string {
	return filepath.Join(a.store.Get().StateDir, "bypass-counters.json")
}

func (a *Agent) bypassEnv(action, statusFile string) map[string]string {
	a.mu.Lock()
	env := a.hookEnvLocked()
	a.mu.Unlock()
	env["TB_BYPASS_FILE"] = a.bypassFile()
	env["TB_BYPASS_STATUS"] = statusFile
	env["TB_BYPASS_ACTION"] = action
	return env
}

// bypassWatcher applies the rules at start, after changes (applyBypass)
// and on WAN changes (SIGHUP).
func (a *Agent) bypassWatcher(ctx context.Context) {
	a.applyBypass(true)
	t := time.NewTicker(5 * time.Minute) // re-assert (e.g. after a firewall reload)
	defer t.Stop()
	// a device with a rule may come back with a new private address: follow
	// its DHCP name (the hook runs only when the rendered rules change)
	leases := time.NewTicker(30 * time.Second)
	defer leases.Stop()
	for {
		select {
		case <-ctx.Done():
			return // Run clears the rules synchronously (clearBypass)
		case <-a.bypassNow:
			a.applyBypass(true)
		case <-t.C:
			a.applyBypass(true)
		case <-leases.C:
			a.applyBypass(false)
		}
	}
}

// clearBypass removes the rules when the agent stops. An apply still
// queued from before is dropped (bypassGen), one running is waited for.
func (a *Agent) clearBypass() {
	a.bypassGen.Add(1)
	a.mu.Lock()
	a.client.bypass.applied = ""
	a.mu.Unlock()
	a.bypassMu.Lock()
	defer a.bypassMu.Unlock()
	a.hooks.Run(hooks.BypassApply, a.bypassEnv("clear", a.bypassStatusFile()))
}

// pokeBypass re-applies the rules soon (WAN up/down).
func (a *Agent) pokeBypass() {
	select {
	case a.bypassNow <- struct{}{}:
	default:
	}
}

// applyBypass writes the enabled rules to the rule file and runs the hook
// (queued, like the other hooks). With force it runs even if nothing
// changed, since WAN gateways and firewall reloads change underneath.
func (a *Agent) applyBypass(force bool) {
	cfg := a.store.Get()
	rules := []config.BypassRule{}
	for _, r := range cfg.Client.Bypass {
		if r.On() {
			rules = append(rules, r)
		}
	}
	mask, _ := cfg.Client.MarkMask()
	opt := bypass.Options{Mask: mask, LocalDNS: cfg.Client.LocalDNSForBypass(), WANDNS: map[string][]string{}, WANDev: map[string]string{}}
	opt.HostMACs = map[string][]string{}
	for _, c := range lanClients(cfg.Client.LeasesPath(), "", nil) {
		if c.Name != "" {
			k := strings.ToLower(c.Name)
			opt.HostMACs[k] = append(opt.HostMACs[k], c.MAC)
		}
	}
	a.mu.Lock()
	for dev, it := range a.client.ifs {
		if it.WAN && it.Usable() && len(it.DNS) > 0 {
			opt.WANDNS[it.Key()] = it.DNS
			opt.WANDev[it.Key()] = dev
		}
	}
	a.mu.Unlock()
	f := bypass.Render(rules, opt)
	for _, id := range f.Skipped {
		a.Logf("bypass rule %s: at most %d \"only this connection\" choices; rule not applied", id, bypass.MaxWANs)
	}
	files := map[string]string{
		"bypass.json":        mustJSON(map[string]any{"rules": rules}),
		"bypass.nft":         f.Nft,
		"bypass.ipset":       f.IPSet,
		"bypass.iptables":    f.IPTables,
		"bypass.ip6tables":   f.IP6Tables,
		"bypass-nftset.conf": f.DnsmasqNftset,
		"bypass-ipset.conf":  f.DnsmasqIPSet,
		"bypass-wans":        f.WANList(),
		"bypass-domains":     strings.Join(f.Domains, "\n"),
		"bypass-marks":       bypass.MarksLine(mask),
	}
	var all strings.Builder
	for _, k := range slices.Sorted(maps.Keys(files)) {
		all.WriteString(k + "\x00" + files[k] + "\x00")
	}
	a.mu.Lock()
	same := a.client.bypass.applied == all.String()
	a.client.bypass.applied = all.String()
	a.mu.Unlock()
	if same && !force {
		return
	}
	dir := filepath.Dir(a.bypassFile())
	for name, body := range files {
		if err := writeFileAtomic(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			a.Logf("bypass rules: %v", err)
			return
		}
	}
	gen, env := a.bypassGen.Load(), a.bypassEnv("apply", a.bypassStatusFile())
	select {
	case a.hookq <- func() {
		a.bypassMu.Lock()
		defer a.bypassMu.Unlock()
		if a.bypassGen.Load() != gen {
			return // cleared since: don't put the rules back
		}
		a.hooks.Run(hooks.BypassApply, env)
	}:
	default:
		a.Logf("hook queue full, dropping %s", hooks.BypassApply)
	}
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func (a *Agent) readBypassSupport() bypassSupport {
	s := bypassSupport{Domains: "unknown"}
	if b, err := os.ReadFile(a.bypassStatusFile()); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

// bypassCounters asks the hook for the per-rule counters, at most every 5 s.
func (a *Agent) bypassCounters() map[string]bypassCounter {
	a.mu.Lock()
	cached, at := a.client.bypass.counters, a.client.bypass.countersAt
	a.mu.Unlock()
	if cached != nil && time.Since(at) < 5*time.Second {
		return cached
	}
	out := map[string]bypassCounter{}
	f := a.bypassCountersFile()
	_ = os.Remove(f)
	a.hooks.Run(hooks.BypassApply, a.bypassEnv("counters", f))
	if b, err := os.ReadFile(f); err == nil {
		var v struct {
			Counters map[string]bypassCounter `json:"counters"`
		}
		if json.Unmarshal(b, &v) == nil && v.Counters != nil {
			out = v.Counters
		}
	}
	a.mu.Lock()
	a.client.bypass.counters, a.client.bypass.countersAt = out, time.Now()
	a.mu.Unlock()
	return out
}

type wanView struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Up    bool   `json:"up"`
	// Default: carries the router's default route now, i.e. where
	// "Use the internet directly" goes
	Default bool `json:"default,omitempty"`
}

// routePath is the IPv4 routing table (tests point it elsewhere).
var routePath = "/proc/net/route"

// defaultRouteDev is the device of the main table's default route with the
// lowest metric ("" if none).
func defaultRouteDev(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	best, bestMetric := "", -1
	for _, line := range strings.Split(string(b), "\n")[1:] {
		fs := strings.Fields(line)
		// Iface Destination Gateway Flags RefCnt Use Metric Mask ...
		if len(fs) < 8 || fs[1] != "00000000" || fs[7] != "00000000" {
			continue
		}
		var m int
		if _, err := fmt.Sscan(fs[6], &m); err != nil {
			continue
		}
		if bestMetric < 0 || m < bestMetric {
			best, bestMetric = fs[0], m
		}
	}
	return best
}

func (a *Agent) hBypassGet(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	cfg := a.store.Get()
	rules := cfg.Client.Bypass
	if rules == nil {
		rules = []config.BypassRule{}
	}
	defDev := defaultRouteDev(routePath)
	a.mu.Lock()
	wans := []wanView{}
	seen := map[string]bool{}
	for _, dev := range slices.Sorted(maps.Keys(a.client.ifs)) {
		it := a.client.ifs[dev]
		if !it.WAN || seen[it.Key()] {
			continue
		}
		seen[it.Key()] = true
		wans = append(wans, wanView{Name: it.Key(), Label: it.Label, Up: it.Usable(), Default: dev == defDev})
	}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"rules": rules, "support": a.readBypassSupport(), "wans": wans, "counters": a.bypassCounters(),
	})
}

func (a *Agent) hBypassPost(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	var in config.BypassRule
	if err := readJSON(r, &in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	rule, err := config.NormalizeBypass(in)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := a.store.Update(func(c *config.Config) error {
		for i, x := range c.Client.Bypass {
			if rule.ID != "" && x.ID == rule.ID {
				c.Client.Bypass[i] = rule
				return nil
			}
		}
		if len(c.Client.Bypass) >= config.MaxBypassRules {
			return fmt.Errorf("at most %d rules", config.MaxBypassRules)
		}
		if !config.ValidBypassID(rule.ID) {
			rule.ID = randHex(4)
		}
		for _, x := range c.Client.Bypass {
			if bypassSame(x, rule) {
				return fmt.Errorf("there is already a rule for %s", describeMatch(rule))
			}
		}
		c.Client.Bypass = append(c.Client.Bypass, rule)
		return nil
	}); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	a.Logf("bypass rule %s: %s %s -> %s", rule.ID, rule.Kind, describeMatch(rule), rule.Target+suffixIf(rule.WAN))
	a.applyBypass(false)
	writeJSON(w, http.StatusOK, rule)
}

func suffixIf(wan string) string {
	if wan == "" {
		return ""
	}
	return " " + wan
}

func describeMatch(r config.BypassRule) string {
	if r.Match == "" {
		return r.IP
	}
	return r.Match
}

// bypassSame: two rules for the same thing (devices by MAC, else by IP).
func bypassSame(a, b config.BypassRule) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == config.BypassDevice && a.Match == "" && b.Match == "" {
		return a.IP == b.IP
	}
	return a.Match == b.Match
}

func (a *Agent) hBypassDelete(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	id := r.PathValue("id")
	found := false
	if _, err := a.store.Update(func(c *config.Config) error {
		c.Client.Bypass = slices.DeleteFunc(c.Client.Bypass, func(x config.BypassRule) bool {
			if x.ID == id {
				found = true
				return true
			}
			return false
		})
		return nil
	}); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	if !found {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.Logf("bypass rule %s removed", id)
	a.applyBypass(false)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// lanClient is one entry of GET /api/lan-clients.
type lanClient struct {
	MAC    string `json:"mac"`
	IP     string `json:"ip,omitempty"`
	Name   string `json:"name,omitempty"`
	Online bool   `json:"online"`
}

func (a *Agent) hLANClients(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient {
		httpErr(w, http.StatusBadRequest, errClientOnly)
		return
	}
	skip := map[string]bool{config.DefaultTunName: true, "lo": true}
	a.mu.Lock()
	for dev, it := range a.client.ifs {
		if it.WAN {
			skip[dev] = true
		}
	}
	for _, dev := range a.client.active {
		skip[dev] = true
	}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, lanClients(a.store.Get().Client.LeasesPath(), arpPath, skip))
}

// lanClients merges the DHCP leases (dnsmasq format: expiry mac ip name
// clientid) with the neighbour table (complete entries on non-WAN devices).
func lanClients(leases, arp string, skipDev map[string]bool) []lanClient {
	byMAC := map[string]*lanClient{}
	var order []string
	add := func(mac string) *lanClient {
		if c, ok := byMAC[mac]; ok {
			return c
		}
		c := &lanClient{MAC: mac}
		byMAC[mac] = c
		order = append(order, mac)
		return c
	}
	if f, err := os.Open(leases); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			fs := strings.Fields(sc.Text())
			if len(fs) < 4 {
				continue
			}
			mac, ok := normMAC(fs[1])
			if !ok {
				continue
			}
			c := add(mac)
			if ip, err := netip.ParseAddr(fs[2]); err == nil && ip.Is4() {
				c.IP = ip.String()
			}
			if fs[3] != "*" {
				c.Name = fs[3]
			}
		}
		f.Close()
	}
	if f, err := os.Open(arp); err == nil {
		sc := bufio.NewScanner(f)
		sc.Scan() // header
		for sc.Scan() {
			fs := strings.Fields(sc.Text())
			// IP address, HW type, Flags, HW address, Mask, Device
			if len(fs) < 6 || fs[2] == "0x0" || skipDev[fs[5]] {
				continue
			}
			mac, ok := normMAC(fs[3])
			if !ok || mac == "00:00:00:00:00:00" {
				continue
			}
			c := add(mac)
			c.Online = true
			if c.IP == "" {
				c.IP = fs[0]
			}
		}
		f.Close()
	}
	out := make([]lanClient, 0, len(order))
	for _, m := range order {
		out = append(out, *byMAC[m])
	}
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.Online != y.Online {
			return x.Online
		}
		if (x.Name == "") != (y.Name == "") {
			return x.Name != ""
		}
		if x.Name != y.Name {
			return strings.ToLower(x.Name) < strings.ToLower(y.Name)
		}
		return x.IP < y.IP
	})
	return out
}

func normMAC(s string) (string, bool) {
	r, err := config.NormalizeBypass(config.BypassRule{Kind: config.BypassDevice, Match: s, Target: config.TargetDirect})
	if err != nil {
		return "", false
	}
	return r.Match, true
}
