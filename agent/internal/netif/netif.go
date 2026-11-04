// Package netif discovers WAN candidate interfaces and guesses their kind.
package netif

import "strings"

const (
	KindWifi     = "wifi"
	KindCellular = "cellular"
	KindEthernet = "ethernet"
	KindUSB      = "usb"
	KindUnknown  = "unknown"
	// KindShared is a nearby device's mobile data, shared over the LAN (Pair & Share).
	KindShared = "shared-cellular"
)

type Iface struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Up      bool     `json:"up"`
	Carrier bool     `json:"carrier"`
	Addrs   []string `json:"addrs"`
	MAC     string   `json:"mac,omitempty"`
	Driver  string   `json:"driver,omitempty"`
	// OpenWrt: the network this device carries and its plain label; WAN
	// means it is in the firewall's wan zone (used without configuration).
	Network string   `json:"network,omitempty"`
	Label   string   `json:"label,omitempty"`
	WAN     bool     `json:"wan,omitempty"`
	DNS     []string `json:"-"`
}

// Key is what path settings and data usage are keyed by: the OpenWrt
// network name when known (stable across devices), else the device.
func (i Iface) Key() string {
	if i.Network != "" {
		return i.Network
	}
	return i.Name
}

// Facts are what Classify looks at; split out so the heuristic is testable.
type Facts struct {
	Name     string
	Wireless bool   // sysfs wireless/ or phy80211/
	DevType  string // uevent DEVTYPE (wlan, wwan, ...)
	Driver   string // device/driver basename
	USB      bool   // the device sits on a USB bus
}

var cellularDrivers = map[string]bool{
	"qmi_wwan": true, "cdc_mbim": true, "huawei_cdc_ncm": true, "option": true,
	"sierra_net": true, "rmnet": true, "ipa": true, "mhi_net": true, "mhi_wwan_mbim": true,
	"t7xx": true, "iosm": true,
}

var usbTetherDrivers = map[string]bool{
	"rndis_host": true, "cdc_ether": true, "cdc_ncm": true, "ipheth": true, "r8152": false,
}

// Classify implements the contract heuristic:
// wlan*/phy*-sta*/wwan → wifi; rmnet*/wwan*/usb*/eth with cellular hints →
// cellular; eth*/en* → ethernet.
func Classify(f Facts) string {
	n := strings.ToLower(f.Name)
	cellHint := f.DevType == "wwan" || cellularDrivers[f.Driver] ||
		strings.Contains(n, "lte") || strings.Contains(n, "5g") || strings.Contains(n, "modem") || strings.Contains(n, "cell")
	switch {
	case f.Wireless || f.DevType == "wlan":
		return KindWifi
	case strings.HasPrefix(n, "wlan") || strings.HasPrefix(n, "wl") || (strings.HasPrefix(n, "phy") && strings.Contains(n, "-sta")):
		return KindWifi
	// OpenWrt names its wifi uplink (repeater) network "wwan"; numbered
	// wwanN devices are modems.
	case n == "wwan":
		return KindWifi
	case strings.HasPrefix(n, "rmnet") || strings.HasPrefix(n, "wwan") || strings.HasPrefix(n, "ccmni"):
		return KindCellular
	case cellHint:
		return KindCellular
	case strings.HasPrefix(n, "usb") || usbTetherDrivers[f.Driver]:
		return KindUSB
	case strings.HasPrefix(n, "eth") || strings.HasPrefix(n, "en") || strings.HasPrefix(n, "wan") || strings.HasPrefix(n, "lan"):
		if f.USB {
			return KindUSB
		}
		return KindEthernet
	}
	return KindUnknown
}

// excluded interfaces are never WAN paths.
func excluded(name string) bool {
	for _, p := range []string{"lo", "tb", "tailscale", "docker", "br-", "virbr", "podman", "cni", "flannel", "veth", "tun", "tap", "wg", "mqvpn", "dummy", "ifb", "sit", "ip6tnl", "gre", "bond", "team"} {
		if name == p || strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// TetherLabel names a USB-tethered phone by its driver: Apple's ipheth is
// always an iPhone (or iPad); "" when the driver says nothing.
func TetherLabel(driver string) string {
	if driver == "ipheth" {
		return "iPhone (USB)"
	}
	return ""
}

// Usable reports whether a path can carry traffic right now.
func (i Iface) Usable() bool { return i.Up && i.Carrier && len(i.Addrs) > 0 }
