package agent

import (
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

// Sharing turned off (or the sharer gone): the share's link and its
// interface are gone before the next check, yet mqvpn still has the relay
// path and must be restarted without it.
func TestPathSetChangedDropsGoneRelayPaths(t *testing.T) {
	active := []string{"eth0", "sh3fa2c1"}
	want := []string{"eth0"}
	present := map[string]bool{"eth0": true} // the share is no longer listed
	if !pathSetChanged(active, want, present, map[string]bool{"sh3fa2c1": true}) {
		t.Fatal("a removed relay path left in mqvpn")
	}
	// a plain interface that vanished is left to mqvpn's own path handling
	if pathSetChanged([]string{"eth0", "wwan0"}, want, present, nil) {
		t.Fatal("restart for a vanished interface")
	}
	if !pathSetChanged(want, active, map[string]bool{"eth0": true, "sh3fa2c1": true}, nil) {
		t.Fatal("a new path not added")
	}
}
