// Package config loads, validates and persists tailbond.json
// (docs/architecture.md, "Agent config file").
package config

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
)

const (
	RoleServer = "server"
	RoleClient = "client"

	ModeSpeed     = "speed"
	ModeStreaming = "streaming"
	ModeRedundant = "redundant"
	ModeFailover  = "failover"

	PriorityPrimary   = "primary"
	PrioritySecondary = "secondary"
	PriorityBackup    = "backup"

	DefaultPath     = "/etc/tailbond/tailbond.json"
	DefaultTunName  = "tb0"
	DefaultCtlAddr  = "127.0.0.1:9077"
	DefaultListen   = ":8077"
	DefaultPort     = 443
	DefaultSubnet   = "10.77.0.0/24"
	DefaultSubnet6  = "fd77:77::/112"
	DefaultStateDir = "/var/lib/tailbond"
	DefaultHooksDir = "/usr/lib/tailbond/hooks"
	DefaultMqvpnBin = "/usr/bin/mqvpn"
)

// TailnetRoutes are the tailnet prefixes bonded clients reach through the server.
var TailnetRoutes = []string{"100.64.0.0/10", "fd7a:115c:a1e0::/48"}

var Modes = []string{ModeSpeed, ModeStreaming, ModeRedundant, ModeFailover}

func ValidMode(m string) bool {
	for _, x := range Modes {
		if x == m {
			return true
		}
	}
	return false
}

func ValidPriority(p string) bool {
	return p == PriorityPrimary || p == PrioritySecondary || p == PriorityBackup
}

type User struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	Created int64  `json:"created,omitempty"`
	// Additive: the Tailscale stable node ID of the device that asked to
	// pair (POST /api/pair-requests); approving that device again re-issues
	// this credential instead of adding a second one.
	TSNode string `json:"ts_node,omitempty"`
}

type Server struct {
	Port             int    `json:"port"`
	PublicAddr       string `json:"public_addr"`
	Subnet           string `json:"subnet"`
	Subnet6          string `json:"subnet6"`
	UseTailscaleCert bool   `json:"use_tailscale_cert"`
	// Additive to the contract: registered clients (mqvpn users).
	Users []User `json:"users,omitempty"`
	// Additive: disable the DNS forwarder on the tunnel IP.
	DisableDNS bool `json:"disable_dns,omitempty"`
}

type PathSetting struct {
	Iface     string `json:"iface"`
	Enabled   bool   `json:"enabled"`
	Priority  string `json:"priority"`
	DataCapMB int64  `json:"data_cap_mb"`
	// Additive: override the kind heuristic.
	Kind string `json:"kind,omitempty"`
}

type Client struct {
	PairingURI string        `json:"pairing_uri"`
	Paths      []PathSetting `json:"paths"`
	// Additive: false on OpenWrt, where hooks do the routing.
	ManageRoutes *bool `json:"manage_routes,omitempty"`
	// Additive: keep mqvpn from changing the system resolver (OpenWrt, tests).
	NoDNS bool `json:"no_dns,omitempty"`
	// Additive: user pressed "disconnect"; persisted so a reboot keeps it off.
	Disconnected bool `json:"disconnected,omitempty"`
}

func (c *Client) RoutesManaged() bool { return c.ManageRoutes == nil || *c.ManageRoutes }

type Config struct {
	Role       string `json:"role"`
	MqvpnBin   string `json:"mqvpn_bin"`
	StateDir   string `json:"state_dir"`
	HooksDir   string `json:"hooks_dir"`
	Listen     string `json:"listen"`
	UIPassword string `json:"ui_password"`
	Mode       string `json:"mode"`
	Server     Server `json:"server"`
	Client     Client `json:"client"`
	// Additive: skip every tailscale CLI call (e2e tests, hosts without it).
	DisableTailscale bool `json:"disable_tailscale,omitempty"`
	// Additive: mqvpn control API endpoint (contract fixes 127.0.0.1:9077).
	ControlAddr string `json:"control_addr,omitempty"`
	// Additive: mqvpn log level.
	LogLevel string `json:"log_level,omitempty"`
	// Additive: Android APK the web UI offers as GET /tailbond.apk
	// (default DefaultAndroidAPK; served only if the file exists).
	AndroidAPK string `json:"android_apk,omitempty"`
	// Additive: GL.iNet router package offered as GET /tailbond-glinet.ipk.
	GLiNetIPK string `json:"glinet_ipk,omitempty"`
}

// DefaultAndroidAPK is where packages may install the Android app for download.
const DefaultAndroidAPK = "/usr/share/tailbond/tailbond.apk"

// DefaultGLiNetIPK is where packages may put the GL.iNet router package
// (the _neon-vfpv4 .ipk) so the home server can offer it for download.
const DefaultGLiNetIPK = "/usr/share/tailbond/tailbond-glinet.ipk"

// GLiNetIPKPath is the router package to offer as GET /tailbond-glinet.ipk.
func (c *Config) GLiNetIPKPath() string {
	if c.GLiNetIPK != "" {
		return c.GLiNetIPK
	}
	return DefaultGLiNetIPK
}

// APKPath is the Android APK to offer for download.
func (c *Config) APKPath() string {
	if c.AndroidAPK != "" {
		return c.AndroidAPK
	}
	return DefaultAndroidAPK
}

func Default() *Config {
	return &Config{
		Role:     RoleServer,
		MqvpnBin: DefaultMqvpnBin,
		StateDir: DefaultStateDir,
		HooksDir: DefaultHooksDir,
		Listen:   DefaultListen,
		Mode:     ModeSpeed,
		Server: Server{
			Port:             DefaultPort,
			Subnet:           DefaultSubnet,
			Subnet6:          DefaultSubnet6,
			UseTailscaleCert: true,
		},
	}
}

// ApplyDefaults fills zero values so older or hand-written files keep working.
func (c *Config) ApplyDefaults() {
	d := Default()
	if c.Role == "" {
		c.Role = d.Role
	}
	if c.MqvpnBin == "" {
		c.MqvpnBin = d.MqvpnBin
	}
	if c.StateDir == "" {
		c.StateDir = d.StateDir
	}
	if c.HooksDir == "" {
		c.HooksDir = d.HooksDir
	}
	if c.Listen == "" {
		c.Listen = d.Listen
	}
	if c.Mode == "" {
		c.Mode = d.Mode
	}
	if c.Server.Port == 0 {
		c.Server.Port = d.Server.Port
	}
	if c.Server.Subnet == "" {
		c.Server.Subnet = d.Server.Subnet
	}
	if c.ControlAddr == "" {
		c.ControlAddr = DefaultCtlAddr
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	for i := range c.Client.Paths {
		if c.Client.Paths[i].Priority == "" {
			c.Client.Paths[i].Priority = PriorityPrimary
		}
	}
}

func (c *Config) Validate() error {
	if c.Role != RoleServer && c.Role != RoleClient {
		return fmt.Errorf("role must be %q or %q, got %q", RoleServer, RoleClient, c.Role)
	}
	if !ValidMode(c.Mode) {
		return fmt.Errorf("unknown mode %q", c.Mode)
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port out of range: %d", c.Server.Port)
	}
	if c.Role == RoleServer {
		p, err := netip.ParsePrefix(c.Server.Subnet)
		if err != nil || !p.Addr().Is4() {
			return fmt.Errorf("server.subnet must be an IPv4 CIDR: %q", c.Server.Subnet)
		}
		if c.Server.Subnet6 != "" {
			if p6, err := netip.ParsePrefix(c.Server.Subnet6); err != nil || !p6.Addr().Is6() {
				return fmt.Errorf("server.subnet6 must be an IPv6 CIDR: %q", c.Server.Subnet6)
			}
		}
	}
	for _, p := range c.Client.Paths {
		if p.Iface == "" {
			return fmt.Errorf("client.paths: empty iface")
		}
		if !ValidPriority(p.Priority) {
			return fmt.Errorf("client.paths[%s]: bad priority %q", p.Iface, p.Priority)
		}
	}
	return nil
}

// ServerTunnelIP is the first host of the IPv4 subnet (the server's tun address).
func (c *Config) ServerTunnelIP() string {
	p, err := netip.ParsePrefix(c.Server.Subnet)
	if err != nil {
		return ""
	}
	return p.Masked().Addr().Next().String()
}

func (c *Config) PathSetting(iface string) (PathSetting, bool) {
	for _, p := range c.Client.Paths {
		if p.Iface == iface {
			return p, true
		}
	}
	return PathSetting{}, false
}

// Store is the on-disk config with a mutex; every mutation is persisted atomically.
type Store struct {
	mu   sync.Mutex
	path string
	cfg  *Config
	// command-line overrides: applied on Get, never persisted
	role, listen string
}

func Load(path string) (*Store, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		cfg = &Config{}
		if err := json.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	case os.IsNotExist(err):
		// first boot: defaults
	default:
		return nil, err
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &Store{path: path, cfg: cfg}, nil
}

func NewStore(path string, cfg *Config) *Store { return &Store{path: path, cfg: cfg} }

func (s *Store) Path() string { return s.path }

// SetOverrides applies --role / --listen without writing them to disk.
func (s *Store) SetOverrides(role, listen string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.role, s.listen = role, listen
	return s.view(s.cfg).Validate()
}

func (s *Store) view(c *Config) *Config {
	out := clone(c)
	if s.role != "" {
		out.Role = s.role
	}
	if s.listen != "" {
		out.Listen = s.listen
	}
	return out
}

// Get returns a deep copy of the current config.
func (s *Store) Get() *Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view(s.cfg)
}

// Update applies fn to a copy, validates, persists, then swaps it in.
func (s *Store) Update(fn func(*Config) error) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.cfg)
	if err := fn(next); err != nil {
		return nil, err
	}
	next.ApplyDefaults()
	if err := s.view(next).Validate(); err != nil {
		return nil, err
	}
	if err := WriteFile(s.path, next); err != nil {
		return nil, err
	}
	s.cfg = next
	return s.view(next), nil
}

func WriteFile(path string, cfg *Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	// 0600: the file holds client PSKs and the UI password.
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func clone(c *Config) *Config {
	b, _ := json.Marshal(c)
	out := &Config{}
	_ = json.Unmarshal(b, out)
	return out
}
