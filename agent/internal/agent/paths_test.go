package agent

import (
	"slices"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/netif"
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

	if got := desiredPaths(cfg, ifs, none); !slices.Equal(got, []string{"eth1", "phy0-sta0"}) {
		t.Fatalf("unconfigured WANs: %v", got)
	}
	// Save mobile data: the phone (secondary by default) stands by
	cfg.Mode = config.ModeFailover
	if got := desiredPaths(cfg, ifs, none); !slices.Equal(got, []string{"phy0-sta0"}) {
		t.Fatalf("failover default: %v", got)
	}
	cfg.Mode = config.ModeSpeed
	// "Backup only" on the phone: stays out while the hotel Wi-Fi works
	cfg.Client.Paths = []config.PathSetting{{Iface: "tethering", Enabled: true, Priority: config.PriorityBackup}}
	if got := desiredPaths(cfg, ifs, none); !slices.Equal(got, []string{"phy0-sta0"}) {
		t.Fatalf("backup tethering: %v", got)
	}
	ifs[0].Carrier = false // hotel Wi-Fi drops
	if got := desiredPaths(cfg, ifs, none); !slices.Equal(got, []string{"eth1", "phy0-sta0"}) {
		t.Fatalf("backup steps in: %v", got)
	}
	// data cap reached: out even as the only backup
	capped := func(p config.PathSetting) bool { return p.Iface == "tethering" }
	cfg.Client.Paths[0].DataCapMB = 100
	if got := desiredPaths(cfg, ifs, capped); slices.Contains(got, "eth1") {
		t.Fatalf("capped path used: %v", got)
	}
	// "Use" off
	ifs[0].Carrier = true
	cfg.Client.Paths = []config.PathSetting{{Iface: "wwan", Enabled: false, Priority: config.PriorityPrimary}}
	if got := desiredPaths(cfg, ifs, none); !slices.Equal(got, []string{"eth1"}) {
		t.Fatalf("wwan off: %v", got)
	}
}
