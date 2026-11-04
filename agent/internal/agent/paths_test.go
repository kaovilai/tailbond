package agent

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/netif"
)

// OpenWrt: WANs are used without configuration; settings keyed by network
// name follow the network whatever device it is on today.
func TestDesiredPathsOpenWrtKeys(t *testing.T) {
	up := func(name, network string) netif.Iface {
		return netif.Iface{Name: name, Network: network, WAN: network != "", Up: true, Carrier: true, Addrs: []string{"192.0.2.2/24"}}
	}
	ifs := []netif.Iface{up("phy0-sta0", "wwan"), up("eth1", "tethering"), up("br-lan", "")}
	ifs[0].Kind, ifs[1].Kind = netif.KindWifi, netif.KindUSB
	cfg := config.Default()
	cfg.Role = config.RoleClient
	none := func(config.PathSetting) bool { return false }

	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"eth1", "phy0-sta0"}) {
		t.Fatalf("unconfigured WANs: %v", got)
	}
	// Save mobile data: the phone (secondary by default) stands by
	cfg.Mode = config.ModeFailover
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"phy0-sta0"}) {
		t.Fatalf("failover default: %v", got)
	}
	cfg.Mode = config.ModeSpeed
	// "Backup only" on the phone: stays out while the hotel Wi-Fi works
	cfg.Client.Paths = []config.PathSetting{{Iface: "tethering", Enabled: true, Priority: config.PriorityBackup}}
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"phy0-sta0"}) {
		t.Fatalf("backup tethering: %v", got)
	}
	ifs[0].Carrier = false // hotel Wi-Fi drops
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"eth1", "phy0-sta0"}) {
		t.Fatalf("backup steps in: %v", got)
	}
	// data cap reached: out even as the only backup
	capped := func(p config.PathSetting) bool { return p.Iface == "tethering" }
	cfg.Client.Paths[0].DataCapMB = 100
	if got := desiredPaths(cfg, ifs, capped, nil); slices.Contains(got, "eth1") {
		t.Fatalf("capped path used: %v", got)
	}
	// behind a captive portal: held back, and the phone (backup) steps in
	// because the hotel Wi-Fi isn't usable for bonding
	portal := func(i netif.Iface) bool { return i.Network == "wwan" }
	cfg.Client.Paths[0].DataCapMB = 0
	if got := desiredPaths(cfg, ifs, none, portal); !slices.Equal(got, []string{"eth1"}) {
		t.Fatalf("captive wwan: %v", got)
	}
	// "Use" off
	ifs[0].Carrier = true
	cfg.Client.Paths = []config.PathSetting{{Iface: "wwan", Enabled: false, Priority: config.PriorityPrimary}}
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"eth1"}) {
		t.Fatalf("wwan off: %v", got)
	}
}

// macOS: "Use this Mac's connections" off bonds only the phones' shares.
func TestDesiredPathsUseLocalOff(t *testing.T) {
	ifs := []netif.Iface{
		{Name: "en0", Kind: netif.KindWifi, Up: true, Carrier: true, Addrs: []string{"192.0.2.2/24"}},
		{Name: "en8", Kind: netif.KindUSB, Label: "iPhone (USB)", Up: true, Carrier: true, Addrs: []string{"172.20.10.2/28"}},
		{Name: "sh0a1b2c", Kind: netif.KindShared, WAN: true, Up: true, Carrier: true, Addrs: []string{"127.0.0.1/8"}},
	}
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.Client.Paths = []config.PathSetting{{Iface: "en0", Enabled: true, Priority: config.PriorityPrimary}, {Iface: "en8", Enabled: true, Priority: config.PriorityPrimary}}
	none := func(config.PathSetting) bool { return false }
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"en0", "en8", "sh0a1b2c"}) {
		t.Fatalf("all: %v", got)
	}
	off := false
	cfg.Client.UseLocal = &off
	if got := desiredPaths(cfg, ifs, none, nil); !slices.Equal(got, []string{"sh0a1b2c"}) {
		t.Fatalf("Mac's connections off: %v", got)
	}
}

// A path setting can name an interface discovery skips (lo0 in the macOS
// e2e); it is looked up and used, once, and never the tunnel or a share.
func TestNamedIfs(t *testing.T) {
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.Client.Paths = []config.PathSetting{{Iface: "en0"}, {Iface: "lo0"}, {Iface: "nosuch1"}, {Iface: config.DefaultTunName}, {Iface: "sh0a1b2c"}}
	a := &Agent{store: config.NewStore(filepath.Join(t.TempDir(), "tb.json"), cfg)}
	looked := []string{}
	a.lookupIf = func(n string) (netif.Iface, bool) {
		looked = append(looked, n)
		return netif.Iface{Name: n, Up: true}, n == "lo0"
	}
	got := a.namedIfs([]netif.Iface{{Name: "en0"}})
	if len(got) != 1 || got[0].Name != "lo0" {
		t.Fatalf("named: %v", got)
	}
	if !slices.Equal(looked, []string{"lo0", "nosuch1"}) {
		t.Fatalf("looked up %v", looked)
	}
}
