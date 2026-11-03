// Package config loads, validates and persists tailbond.json
// (docs/architecture.md, "Agent config file").
package config

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

	DefaultPath    = "/etc/tailbond/tailbond.json"
	DefaultTunName = "tb0"
	// DefaultClientMTU is safe on any IPv4/IPv6 uplink (the Android app uses it too).
	DefaultClientMTU = 1280
	DefaultCtlAddr   = "127.0.0.1:9077"
	DefaultListen    = ":8077"
	DefaultPort      = 443
	DefaultSubnet    = "10.77.0.0/24"
	DefaultSubnet6   = "fd77:77::/112"
	DefaultStateDir  = "/var/lib/tailbond"
	DefaultHooksDir  = "/usr/lib/tailbond/hooks"
	DefaultMqvpnBin  = "/usr/bin/mqvpn"
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
	// Additive: don't advertise the tunnel subnet to the tailnet (default:
	// advertise it; hooks get TB_ADVERTISE_ROUTE=0).
	NoAdvertiseRoute bool `json:"no_advertise_route,omitempty"`
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
	// MTU of the tunnel device. 0 = DefaultClientMTU. mqvpn's own "auto"
	// assumes a 1500-byte uplink; cellular, hotel and WSL links are often
	// smaller, and then every packet above ~1250 bytes is lost.
	MTU int `json:"mtu,omitempty"`
	// Additive: user pressed "disconnect"; persisted so a reboot keeps it off.
	Disconnected bool `json:"disconnected,omitempty"`
	// Additive: probe each WAN for a captive portal and keep portal WANs
	// out of bonding until signed in (default on); probe URLs (default:
	// captive.DefaultProbes).
	CaptivePortalDetect *bool    `json:"captive_portal_detect,omitempty"`
	CaptiveProbeURLs    []string `json:"captive_probe_urls,omitempty"`
	// Additive: the ordered server list (first = primary) and how the
	// client fails over between them (docs/architecture.md, "Multiple
	// servers and failover"). PairingURI mirrors the primary's URI.
	Servers  []ServerEntry `json:"servers,omitempty"`
	Failover *Failover     `json:"failover,omitempty"`
	// Additive: LAN traffic that skips the tunnel ("Bypass rules").
	Bypass []BypassRule `json:"bypass,omitempty"`
	// Additive: DHCP leases for GET /api/lan-clients (default /tmp/dhcp.leases).
	LeasesFile string `json:"leases_file,omitempty"`
	// Additive: fwmark bits the bypass rules use (default 0x07000000), and
	// whether bypassed domains resolve through the WAN's DNS (default on).
	BypassMarkMask string `json:"bypass_mark_mask,omitempty"`
	BypassLocalDNS *bool  `json:"bypass_local_dns,omitempty"`
}

// ServerEntry is one paired server in the client's list.
type ServerEntry struct {
	Name       string `json:"name"`
	PairingURI string `json:"pairing_uri"`
	// nil = enabled
	Enabled *bool `json:"enabled,omitempty"`
}

func (e ServerEntry) On() bool { return e.Enabled == nil || *e.Enabled }

const (
	PolicyPrefer = "prefer"
	PolicyAlways = "always"

	DefaultSwitchAfterSec   = 15
	DefaultFailbackAfterSec = 120
	DefaultProbeEverySec    = 10
)

// Failover is client.failover; zero values mean the defaults.
type Failover struct {
	Policy           string `json:"policy,omitempty"`
	Server           string `json:"server,omitempty"` // policy "always"
	SwitchAfterSec   int    `json:"switch_after_sec,omitempty"`
	FailbackAfterSec int    `json:"failback_after_sec,omitempty"`
	ProbeEverySec    int    `json:"probe_every_sec,omitempty"`
}

// FailoverOrDefault returns client.failover with defaults filled in.
func (c *Client) FailoverOrDefault() Failover {
	f := Failover{}
	if c.Failover != nil {
		f = *c.Failover
	}
	if f.Policy == "" {
		f.Policy = PolicyPrefer
	}
	if f.SwitchAfterSec <= 0 {
		f.SwitchAfterSec = DefaultSwitchAfterSec
	}
	if f.FailbackAfterSec <= 0 {
		f.FailbackAfterSec = DefaultFailbackAfterSec
	}
	if f.ProbeEverySec <= 0 {
		f.ProbeEverySec = DefaultProbeEverySec
	}
	return f
}

// MarkMask parses client.bypass_mark_mask (0 = the default). It must be
// one run of at least 3 bits (direct, tunnel, one WAN).
func (c *Client) MarkMask() (uint32, error) {
	if c.BypassMarkMask == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(c.BypassMarkMask), "0x"), 16, 32)
	m := uint32(v)
	low := m & -m
	if err != nil || m == 0 || (m+low)&m != 0 || m/low < 3 {
		return 0, fmt.Errorf("client.bypass_mark_mask %q: one run of at least 2 bits, e.g. 0x07000000", c.BypassMarkMask)
	}
	return m, nil
}

func (c *Client) LocalDNSForBypass() bool { return c.BypassLocalDNS == nil || *c.BypassLocalDNS }

// DefaultLeasesFile is OpenWrt dnsmasq's lease file.
const DefaultLeasesFile = "/tmp/dhcp.leases"

func (c *Client) LeasesPath() string {
	if c.LeasesFile != "" {
		return c.LeasesFile
	}
	return DefaultLeasesFile
}

// CaptiveDetect reports whether captive-portal detection is on (default).
func (c *Client) CaptiveDetect() bool { return c.CaptivePortalDetect == nil || *c.CaptivePortalDetect }

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
	// Additive (server role): the phone landing page ("Get tailbond") on
	// the Tailscale IPs. LandingPort is the port it binds (0 = 80, -1 =
	// off); LandingPublicPort the port tailnet devices use (0 = the same;
	// OpenWrt binds 8078 and redirects tailnet tcp/80 to it, since GL's
	// nginx owns port 80).
	LandingPort int `json:"landing_port,omitempty"`
	// Additive (server): Tailscale logins who may use the dashboard from
	// the tailnet without a password, besides the server's owner.
	Approvers         []string `json:"approvers,omitempty"`
	LandingPublicPort int      `json:"landing_public_port,omitempty"`
	// Additive: auto-updates and the server's update mirror, scheduled
	// encrypted backups (docs/architecture.md, "Updates", "Backup and restore").
	Update *UpdateSettings `json:"update,omitempty"`
	Backup *BackupSettings `json:"backup,omitempty"`
}

// UpdateSettings: the update check (docs/architecture.md, "Updates").
// Keys of older versions (channel, window, pin, source, mirror, …) are
// ignored.
type UpdateSettings struct {
	Policy          string `json:"policy,omitempty"` // notify (default) | auto | off
	Repo            string `json:"repo,omitempty"`   // owner/name or https://github.com/owner/name
	CheckEveryHours int    `json:"check_every_hours,omitempty"`
	APIURL          string `json:"api_url,omitempty"` // default https://api.github.com
}

const (
	UpdateNotify = "notify"
	UpdateAuto   = "auto"
	UpdateOff    = "off"
)

type BackupSettings struct {
	Passphrase string `json:"passphrase,omitempty"`
	EveryHours int    `json:"every_hours,omitempty"`
	Keep       int    `json:"keep,omitempty"`
	Push       *bool  `json:"push,omitempty"`
}

var updateRepoRE = regexp.MustCompile(`^([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+|https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(\.git)?/?)$`)

// UpdateOrDefault returns the update settings with defaults filled in.
func (c *Config) UpdateOrDefault() UpdateSettings {
	u := UpdateSettings{}
	if c.Update != nil {
		u = *c.Update
	}
	if u.Policy == "" {
		u.Policy = UpdateNotify
	}
	if u.CheckEveryHours <= 0 {
		u.CheckEveryHours = 24
	}
	return u
}

// BackupOrDefault returns the backup settings with defaults filled in.
func (c *Config) BackupOrDefault() BackupSettings {
	b := BackupSettings{}
	if c.Backup != nil {
		b = *c.Backup
	}
	if b.EveryHours <= 0 {
		b.EveryHours = 24
	}
	if b.Keep <= 0 {
		b.Keep = 7
	}
	if b.Push == nil {
		on := true
		b.Push = &on
	}
	return b
}

// DefaultLandingPort serves the landing page at http://<server>/.
const DefaultLandingPort = 80

// LandingPorts returns the bind and public port of the landing page, or
// ok=false when it is turned off.
func (c *Config) LandingPorts() (bind, public int, ok bool) {
	if c.LandingPort < 0 {
		return 0, 0, false
	}
	bind = c.LandingPort
	if bind == 0 {
		bind = DefaultLandingPort
	}
	public = c.LandingPublicPort
	if public <= 0 {
		public = bind
	}
	return bind, public, true
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
	c.Client.MigrateServers()
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
	if c.LandingPort > 65535 || c.LandingPublicPort < 0 || c.LandingPublicPort > 65535 {
		return fmt.Errorf("landing_port / landing_public_port out of range")
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
	seen := map[string]bool{}
	for _, e := range c.Client.Servers {
		if e.Name == "" || e.PairingURI == "" {
			return fmt.Errorf("client.servers: every server needs a name and a pairing_uri")
		}
		if seen[e.Name] {
			return fmt.Errorf("client.servers: %q listed twice", e.Name)
		}
		seen[e.Name] = true
	}
	if f := c.Client.Failover; f != nil {
		if f.Policy != "" && f.Policy != PolicyPrefer && f.Policy != PolicyAlways {
			return fmt.Errorf("client.failover.policy must be %q or %q", PolicyPrefer, PolicyAlways)
		}
		if f.SwitchAfterSec < 0 || f.FailbackAfterSec < 0 || f.ProbeEverySec < 0 {
			return fmt.Errorf("client.failover: negative timer")
		}
	}
	if err := ValidateBypass(c.Client.Bypass); err != nil {
		return err
	}
	if _, err := c.Client.MarkMask(); err != nil {
		return err
	}
	if u := c.Update; u != nil {
		switch u.Policy {
		case "", UpdateNotify, UpdateAuto, UpdateOff:
		default:
			return fmt.Errorf("update.policy must be notify, auto or off, not %q", u.Policy)
		}
		if u.Repo != "" && !updateRepoRE.MatchString(u.Repo) {
			return fmt.Errorf("update.repo must be owner/name or https://github.com/owner/name, not %q", u.Repo)
		}
		if u.APIURL != "" && !strings.HasPrefix(u.APIURL, "https://") && !strings.HasPrefix(u.APIURL, "http://") {
			return fmt.Errorf("update.api_url must be an http(s) URL, not %q", u.APIURL)
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
