package backup

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
)

// StateFiles are the state_dir files a backup carries: the self-signed TLS
// certificate and key (clients pin it) and the private global mqvpn key.
// Tailscale certificates are left out: Tailscale reissues them.
var StateFiles = []string{"server.crt", "server.key", "global.key"}

// A server's backup also carries the backups its clients pushed to it
// (client-backups/<client>/<file>.tbbackup, each still locked with its own
// passphrase), so they move with the server.
var clientBackupRE = regexp.MustCompile(`^client-backups/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\.tbbackup$`)

const maxClientBackup = 256 << 10

// Payload is a backup's plaintext.
type Payload struct {
	Format int               `json:"format"`
	Config json.RawMessage   `json:"config"`
	Files  map[string]string `json:"files,omitempty"` // name -> base64
	UCI    string            `json:"uci,omitempty"`   // `uci export tailbond` (OpenWrt)
}

// Collect reads what a backup of this machine contains: the config file
// as saved (or, before it was ever saved, the config in memory).
func Collect(cfgPath, stateDir, uci string, fallback *config.Config) (*Payload, error) {
	b, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) && fallback != nil {
		b, err = json.Marshal(fallback)
	}
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("%s isn't valid JSON", cfgPath)
	}
	p := &Payload{Format: 1, Config: b, Files: map[string]string{}, UCI: uci}
	for _, f := range StateFiles {
		if data, err := os.ReadFile(filepath.Join(stateDir, f)); err == nil {
			p.Files[f] = base64.StdEncoding.EncodeToString(data)
		}
	}
	clients, _ := os.ReadDir(filepath.Join(stateDir, "client-backups"))
	for _, c := range clients {
		files, _ := os.ReadDir(filepath.Join(stateDir, "client-backups", c.Name()))
		for _, f := range files {
			name := "client-backups/" + c.Name() + "/" + f.Name()
			if info, err := f.Info(); err != nil || !info.Mode().IsRegular() || info.Size() > maxClientBackup || !isStateFile(name) {
				continue
			}
			if data, err := os.ReadFile(filepath.Join(stateDir, filepath.FromSlash(name))); err == nil {
				p.Files[name] = base64.StdEncoding.EncodeToString(data)
			}
		}
	}
	return p, nil
}

// Decode parses and checks a payload.
func Decode(pt []byte) (*Payload, *config.Config, error) {
	var p Payload
	if err := json.Unmarshal(pt, &p); err != nil || p.Format != 1 || len(p.Config) == 0 {
		return nil, nil, errors.New("the backup's contents aren't readable")
	}
	cfg := &config.Config{}
	if err := json.Unmarshal(p.Config, cfg); err != nil {
		return nil, nil, fmt.Errorf("the backup's settings: %w", err)
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("the backup's settings: %w", err)
	}
	for name := range p.Files {
		if !isStateFile(name) {
			return nil, nil, fmt.Errorf("unexpected file %q in the backup", name)
		}
	}
	return &p, cfg, nil
}

func isStateFile(n string) bool {
	for _, f := range StateFiles {
		if f == n {
			return true
		}
	}
	return clientBackupRE.MatchString(n) && !strings.Contains(n, "..")
}

// Apply writes the payload's state files (the config is written by the
// caller through config.WriteFile, so paths that differ on this machine can
// be kept).
func Apply(p *Payload, stateDir string) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}
	for name, b64 := range p.Files {
		if !isStateFile(name) {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		dst := filepath.Join(stateDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(dst+".tmp", data, 0o600); err != nil {
			return err
		}
		if err := os.Rename(dst+".tmp", dst); err != nil {
			return err
		}
	}
	return nil
}

// Change is one line of a restore preview.
type Change struct {
	What string `json:"what"`
	From string `json:"from"`
	To   string `json:"to"`
}

// Changes describes what restoring next over cur replaces (no secrets).
func Changes(cur, next *config.Config) []Change {
	var out []Change
	add := func(what, from, to string) {
		if from != to {
			out = append(out, Change{What: what, From: from, To: to})
		}
	}
	add("Role", cur.Role, next.Role)
	add("Mode", modeName(cur.Mode), modeName(next.Mode))
	if cur.Role == config.RoleServer || next.Role == config.RoleServer {
		add("Devices (clients)", names(cur.Server.Users), names(next.Server.Users))
		add("Public address", orNone(cur.Server.PublicAddr), orNone(next.Server.PublicAddr))
		add("Tunnel port", fmt.Sprint(cur.Server.Port), fmt.Sprint(next.Server.Port))
		add("Tunnel addresses", cur.Server.Subnet, next.Server.Subnet)
		add("Approvers", orNone(strings.Join(cur.Approvers, ", ")), orNone(strings.Join(next.Approvers, ", ")))
	}
	if cur.Role == config.RoleClient || next.Role == config.RoleClient {
		add("Servers", serverNames(cur.Client.Servers), serverNames(next.Client.Servers))
		add("Bypass rules", fmt.Sprint(len(cur.Client.Bypass)), fmt.Sprint(len(next.Client.Bypass)))
		add("Connection settings", fmt.Sprint(len(cur.Client.Paths)), fmt.Sprint(len(next.Client.Paths)))
	}
	add("Dashboard password", set(cur.UIPassword != ""), set(next.UIPassword != ""))
	add("Update policy", policy(cur), policy(next))
	return out
}

// Summary is a backup's contents without keys or passwords
// (`tailbondd backup --plain`, GET /api/backup/summary).
func Summary(cfg *config.Config, p *Payload) map[string]any {
	s := map[string]any{"role": cfg.Role, "mode": cfg.Mode, "password_set": cfg.UIPassword != "", "update_policy": policy(cfg)}
	if cfg.Role == config.RoleServer {
		var users []string
		for _, u := range cfg.Server.Users {
			users = append(users, u.Name)
		}
		s["clients"] = users
		s["public_addr"], s["port"], s["subnet"], s["approvers"] = cfg.Server.PublicAddr, cfg.Server.Port, cfg.Server.Subnet, cfg.Approvers
	} else {
		var servers []string
		for _, e := range cfg.Client.Servers {
			label := e.Name
			if pl, err := pairing.Decode(e.PairingURI); err == nil {
				label += " (" + pl.Server + ")"
			}
			servers = append(servers, label)
		}
		s["servers"], s["bypass_rules"], s["paths"] = servers, len(cfg.Client.Bypass), len(cfg.Client.Paths)
	}
	if p != nil {
		var files []string
		for f := range p.Files {
			files = append(files, f)
		}
		sort.Strings(files)
		s["files"], s["uci"] = files, p.UCI != ""
	}
	return s
}

func names(us []config.User) string {
	var n []string
	for _, u := range us {
		n = append(n, u.Name)
	}
	return orNone(strings.Join(n, ", "))
}

func serverNames(es []config.ServerEntry) string {
	var n []string
	for _, e := range es {
		n = append(n, e.Name)
	}
	return orNone(strings.Join(n, ", "))
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func set(b bool) string {
	if b {
		return "set"
	}
	return "not set"
}

// policy and modeName: the words the dashboards use.
func policy(c *config.Config) string {
	switch c.UpdateOrDefault().Policy {
	case config.UpdateImmediate:
		return "Right away"
	case config.UpdateAuto:
		return "Automatic (night)"
	case config.UpdateOff:
		return "Off"
	}
	return "Tell me"
}

func modeName(m string) string {
	switch m {
	case "":
		return "(default)"
	case config.ModeSpeed:
		return "Speed"
	}
	return strings.ToUpper(m[:1]) + m[1:]
}

// KeepOnRestore applies what a restore keeps from this machine (cur) to the
// backup's settings (next): its own paths and binding, and for a server its
// public address and tunnel port (the address devices reach it at, and the
// port its firewall allows); devices learn them (docs/backup-restore.md).
func KeepOnRestore(cur, next *config.Config) *config.Config {
	c := *next
	c.MqvpnBin, c.StateDir, c.HooksDir, c.Listen, c.ControlAddr = cur.MqvpnBin, cur.StateDir, cur.HooksDir, cur.Listen, cur.ControlAddr
	c.AndroidAPK, c.GLiNetIPK, c.MacOSBundle = cur.AndroidAPK, cur.GLiNetIPK, cur.MacOSBundle
	if c.Role == config.RoleServer && cur.Role == config.RoleServer {
		if cur.Server.PublicAddr != "" {
			c.Server.PublicAddr = cur.Server.PublicAddr
		}
		if cur.Server.Port != 0 {
			c.Server.Port = cur.Server.Port
		}
	}
	return &c
}

// Encode is the payload's plaintext bytes.
func (p *Payload) Encode() ([]byte, error) { return json.Marshal(p) }
