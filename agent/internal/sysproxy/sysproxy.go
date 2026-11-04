// Package sysproxy points the operating system's SOCKS proxy at a phone
// hub and puts the previous setting back (docs/phone-hub.md). macOS only:
// networksetup -setsocksfirewallproxy on the active network service.
//
// Before it changes anything it writes a restore record (the service and its
// previous SOCKS settings) to a file, so a crash, a kill or a reboot can't
// leave the Mac pointing at a phone that's gone: tailbondd restores from it
// at start (its LaunchDaemon starts at boot).
package sysproxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Settings are one service's SOCKS proxy settings.
type Settings struct {
	Enabled bool   `json:"enabled"`
	Server  string `json:"server"`
	Port    int    `json:"port"`
	Auth    bool   `json:"auth"`
}

// PAC is a service's automatic proxy configuration (PAC URL) setting.
type PAC struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

// Record is what Restore puts back.
type Record struct {
	Service string `json:"service"`
	// Mode is what we set: "socks" (the SOCKS proxy) or "pac" (the
	// automatic proxy configuration URL).
	Mode     string    `json:"mode"`
	Previous Settings  `json:"previous"`
	PrevPAC  PAC       `json:"previous_pac"`
	Set      Settings  `json:"set"`
	SetPAC   string    `json:"set_pac,omitempty"`
	At       time.Time `json:"at"`
}

// Runner runs a command and returns its output (tests replace it).
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Mac sets and restores the macOS SOCKS proxy.
type Mac struct {
	RecordPath string
	Run        Runner
}

func (m *Mac) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.Run != nil {
		return m.Run(ctx, name, args...)
	}
	return execRunner(ctx, name, args...)
}

// ActiveService is the network service ("Wi-Fi", "Ethernet", "iPhone USB")
// that carries the default route: the first "default" row of the IPv4
// routing table that isn't a tunnel (a VPN, tailbond's own tunnel included,
// routes 0/1 and 128/1 or adds its own default on a utun).
func (m *Mac) ActiveService(ctx context.Context) (string, error) {
	out, err := m.run(ctx, "netstat", "-rn", "-f", "inet")
	if err != nil {
		return "", fmt.Errorf("netstat -rn: %v", err)
	}
	dev := defaultDevice(string(out))
	if dev == "" {
		return "", errors.New("no default route")
	}
	order, err := m.run(ctx, "networksetup", "-listnetworkserviceorder")
	if err != nil {
		return "", err
	}
	if s := serviceFor(string(order), dev); s != "" {
		return s, nil
	}
	return "", fmt.Errorf("no network service for %s", dev)
}

func defaultDevice(netstat string) string {
	for _, l := range strings.Split(netstat, "\n") {
		f := strings.Fields(l)
		if len(f) < 4 || f[0] != "default" {
			continue
		}
		dev := f[len(f)-1]
		tunnel := false
		for _, p := range []string{"utun", "ipsec", "ppp", "gif", "stf", "lo"} {
			if strings.HasPrefix(dev, p) {
				tunnel = true
			}
		}
		if !tunnel {
			return dev
		}
	}
	return ""
}

var reService = regexp.MustCompile(`^\((?:\d+|\*)\)\s+(.+)$`)
var reDevice = regexp.MustCompile(`Device:\s*([^)\s]+)`)

// serviceFor finds dev's service in `networksetup -listnetworkserviceorder`:
//
//	(1) Wi-Fi
//	(Hardware Port: Wi-Fi, Device: en0)
func serviceFor(order, dev string) string {
	name := ""
	sc := bufio.NewScanner(strings.NewReader(order))
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if m := reService.FindStringSubmatch(l); m != nil {
			name = m[1] // "(*)" marks a disabled service
			continue
		}
		if m := reDevice.FindStringSubmatch(l); m != nil && m[1] == dev && name != "" {
			return name
		}
	}
	return ""
}

// Get reads a service's SOCKS settings (networksetup -getsocksfirewallproxy).
func (m *Mac) Get(ctx context.Context, service string) (Settings, error) {
	out, err := m.run(ctx, "networksetup", "-getsocksfirewallproxy", service)
	if err != nil {
		return Settings{}, fmt.Errorf("networksetup: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return parseGet(string(out)), nil
}

func parseGet(out string) Settings {
	var s Settings
	for _, l := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "Enabled":
			s.Enabled = v == "Yes"
		case "Server":
			s.Server = v
		case "Port":
			s.Port, _ = strconv.Atoi(v)
		case "Authenticated Proxy Enabled":
			s.Auth = v == "1"
		}
	}
	return s
}

// GetPAC reads a service's automatic proxy URL (networksetup -getautoproxyurl).
func (m *Mac) GetPAC(ctx context.Context, service string) (PAC, error) {
	out, err := m.run(ctx, "networksetup", "-getautoproxyurl", service)
	if err != nil {
		return PAC{}, fmt.Errorf("networksetup: %v: %s", err, strings.TrimSpace(string(out)))
	}
	var p PAC
	for _, l := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "URL":
			if v != "(null)" {
				p.URL = v
			}
		case "Enabled":
			p.Enabled = v == "Yes"
		}
	}
	return p, nil
}

// begin writes the record for a change to service (mode "socks" or "pac")
// before the change is made, keeping the original "previous" when the
// record is already ours for that service.
func (m *Mac) begin(ctx context.Context, service, mode string, set func(*Record)) error {
	var rec Record
	if old, err := m.load(); err == nil && old.Service == service && old.Mode == mode {
		rec = old // already ours: keep the original "previous"
	} else {
		if err == nil {
			// another service (the default route moved) or mode: put that back first
			if rerr := m.Restore(ctx); rerr != nil {
				return rerr
			}
		}
		prev, err := m.Get(ctx, service)
		if err != nil {
			return err
		}
		pac, err := m.GetPAC(ctx, service)
		if err != nil {
			return err
		}
		rec = Record{Service: service, Mode: mode, Previous: prev, PrevPAC: pac}
	}
	set(&rec)
	rec.At = time.Now()
	return m.save(rec)
}

// UsePAC sets service's automatic proxy configuration URL, saving the
// previous settings first.
func (m *Mac) UsePAC(ctx context.Context, service, url string) error {
	if err := m.begin(ctx, service, "pac", func(r *Record) { r.SetPAC = url }); err != nil {
		return err
	}
	if out, err := m.run(ctx, "networksetup", "-setautoproxyurl", service, url); err != nil {
		return fmt.Errorf("networksetup -setautoproxyurl: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := m.run(ctx, "networksetup", "-setautoproxystate", service, "on"); err != nil {
		return fmt.Errorf("networksetup -setautoproxystate: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Use points service's SOCKS proxy at host:port (user/pass optional),
// saving the previous settings first (the record is written before the
// change).
func (m *Mac) Use(ctx context.Context, service, host string, port int, user, pass string) error {
	if err := m.begin(ctx, service, "socks", func(r *Record) {
		r.Set = Settings{Enabled: true, Server: host, Port: port, Auth: user != ""}
	}); err != nil {
		return err
	}
	args := []string{"-setsocksfirewallproxy", service, host, strconv.Itoa(port)}
	if user != "" {
		args = append(args, "on", user, pass)
	} else {
		args = append(args, "off")
	}
	if out, err := m.run(ctx, "networksetup", args...); err != nil {
		return fmt.Errorf("networksetup -setsocksfirewallproxy: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := m.run(ctx, "networksetup", "-setsocksfirewallproxystate", service, "on"); err != nil {
		return fmt.Errorf("networksetup -setsocksfirewallproxystate: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Restore puts back what the record says and removes it; no record, nothing
// to do.
func (m *Mac) Restore(ctx context.Context) error {
	rec, err := m.load()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := m.PutBack(ctx, rec); err != nil {
		return err
	}
	if err := os.Remove(m.RecordPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Matches reports whether rec's service has its previous setting again.
func (m *Mac) Matches(ctx context.Context, rec Record) (bool, error) {
	if rec.Mode == "pac" {
		p, err := m.GetPAC(ctx, rec.Service)
		return err == nil && p == rec.PrevPAC, err
	}
	s, err := m.Get(ctx, rec.Service)
	if err != nil {
		return false, err
	}
	want := rec.Previous
	if want.Server == "" { // nothing to put back but the state
		return s.Enabled == want.Enabled, nil
	}
	return s.Enabled == want.Enabled && s.Server == want.Server && s.Port == want.Port, nil
}

// PutBack applies rec's previous setting (Restore without the record file).
func (m *Mac) PutBack(ctx context.Context, rec Record) error {
	onOff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	var cmds [][]string
	if rec.Mode == "pac" {
		if p := rec.PrevPAC; p.URL != "" {
			cmds = append(cmds, []string{"-setautoproxyurl", rec.Service, p.URL})
		}
		cmds = append(cmds, []string{"-setautoproxystate", rec.Service, onOff(rec.PrevPAC.Enabled)})
	} else {
		// the server we set goes, whatever was there before comes back
		// (networksetup can't clear the server; an empty previous one keeps
		// ours, switched off)
		if p := rec.Previous; p.Server != "" && p.Port > 0 {
			cmds = append(cmds, []string{"-setsocksfirewallproxy", rec.Service, p.Server, strconv.Itoa(p.Port), "off"})
		}
		cmds = append(cmds, []string{"-setsocksfirewallproxystate", rec.Service, onOff(rec.Previous.Enabled)})
	}
	for _, c := range cmds {
		if out, err := m.run(ctx, "networksetup", c...); err != nil {
			return fmt.Errorf("networksetup %s: %v: %s", c[0], err, strings.TrimSpace(string(out)))
		}
	}
	// check it took: setting the server switches the proxy on, and the
	// state change right after it has been seen not to stick
	state := cmds[len(cmds)-1]
	for try := 0; try < 3; try++ {
		var on, want bool
		if rec.Mode == "pac" {
			p, err := m.GetPAC(ctx, rec.Service)
			if err != nil {
				return err
			}
			on, want = p.Enabled, rec.PrevPAC.Enabled
		} else {
			s, err := m.Get(ctx, rec.Service)
			if err != nil {
				return err
			}
			on, want = s.Enabled, rec.Previous.Enabled
		}
		if on == want {
			break
		}
		time.Sleep(time.Duration(try+1) * 300 * time.Millisecond)
		if out, err := m.run(ctx, "networksetup", state...); err != nil {
			return fmt.Errorf("networksetup %s: %v: %s", state[0], err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// Active is the record of a proxy we set, if any.
func (m *Mac) Active() (Record, bool) {
	r, err := m.load()
	return r, err == nil
}

func (m *Mac) load() (Record, error) {
	b, err := os.ReadFile(m.RecordPath)
	if err != nil {
		return Record{}, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return Record{}, fmt.Errorf("restore record %s: %v", m.RecordPath, err)
	}
	return r, nil
}

func (m *Mac) save(r Record) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	tmp := m.RecordPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, m.RecordPath)
}
