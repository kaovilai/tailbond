// Package netif discovers WAN candidate interfaces and guesses their kind.
package netif

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	KindWifi     = "wifi"
	KindCellular = "cellular"
	KindEthernet = "ethernet"
	KindUSB      = "usb"
	KindUnknown  = "unknown"
	// KindShared is a nearby device's mobile data, shared over the LAN (Pair & Share).
	KindShared = "shared-cellular"
)

// SysfsRoot is /sys/class/net, overridable in tests.
var SysfsRoot = "/sys/class/net"

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

func readTrim(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func facts(name string) Facts {
	dir := filepath.Join(SysfsRoot, name)
	f := Facts{Name: name}
	if _, err := os.Stat(filepath.Join(dir, "wireless")); err == nil {
		f.Wireless = true
	}
	if _, err := os.Stat(filepath.Join(dir, "phy80211")); err == nil {
		f.Wireless = true
	}
	for _, line := range strings.Split(readTrim(filepath.Join(dir, "uevent")), "\n") {
		if v, ok := strings.CutPrefix(line, "DEVTYPE="); ok {
			f.DevType = v
		}
	}
	if t, err := os.Readlink(filepath.Join(dir, "device", "driver")); err == nil {
		f.Driver = filepath.Base(t)
	}
	if t, err := filepath.EvalSymlinks(filepath.Join(dir, "device")); err == nil && strings.Contains(t, "/usb") {
		f.USB = true
	}
	return f
}

// KindOf classifies one interface by name using sysfs.
func KindOf(name string) string { return Classify(facts(name)) }

// excluded interfaces are never WAN paths.
func excluded(name string) bool {
	for _, p := range []string{"lo", "tb", "tailscale", "docker", "br-", "virbr", "podman", "cni", "flannel", "veth", "tun", "tap", "wg", "mqvpn", "dummy", "ifb", "sit", "ip6tnl", "gre", "bond", "team"} {
		if name == p || strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// Discover lists candidate WAN interfaces: ARP-capable (ethernet-type) links
// that are not bridges, tunnels or container plumbing.
func Discover() ([]Iface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := []Iface{}
	for _, ni := range ifs {
		if ni.Flags&net.FlagLoopback != 0 || excluded(ni.Name) {
			continue
		}
		dir := filepath.Join(SysfsRoot, ni.Name)
		if _, err := os.Stat(filepath.Join(dir, "bridge")); err == nil {
			continue
		}
		// type 1 = ether; 519/530 = rmnet/raw-ip modems (ARPHRD_NONE=65534 is tun).
		typ, _ := strconv.Atoi(readTrim(filepath.Join(dir, "type")))
		if typ != 1 && typ != 519 && typ != 530 && !strings.HasPrefix(ni.Name, "rmnet") && !strings.HasPrefix(ni.Name, "wwan") {
			continue
		}
		out = append(out, describe(ni))
	}
	if wans, ok := OpenWrtWANs(); ok {
		// WAN devices the generic filter skipped (e.g. a bridged WAN port)
		have := map[string]bool{}
		for _, it := range out {
			have[it.Name] = true
		}
		for _, w := range wans {
			if have[w.Device] {
				continue
			}
			if ni, err := net.InterfaceByName(w.Device); err == nil {
				out = append(out, describe(*ni))
				have[w.Device] = true
			}
		}
		out = annotate(out, wans)
	}
	return out, nil
}

func describe(ni net.Interface) Iface {
	dir := filepath.Join(SysfsRoot, ni.Name)
	f := facts(ni.Name)
	it := Iface{
		Name: ni.Name, Kind: Classify(f), Up: ni.Flags&net.FlagUp != 0,
		Carrier: readTrim(filepath.Join(dir, "carrier")) == "1", Driver: f.Driver,
		MAC: ni.HardwareAddr.String(), Addrs: []string{},
	}
	if addrs, err := ni.Addrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLinkLocalUnicast() {
				it.Addrs = append(it.Addrs, ipn.String())
			}
		}
	}
	return it
}

// Usable reports whether a path can carry traffic right now.
func (i Iface) Usable() bool { return i.Up && i.Carrier && len(i.Addrs) > 0 }

// Counters reads kernel byte counters for an interface (fallback stats when
// mqvpn's client control API is absent).
func Counters(name string) (tx, rx uint64, ok bool) {
	dir := filepath.Join(SysfsRoot, name, "statistics")
	t, err1 := strconv.ParseUint(readTrim(filepath.Join(dir, "tx_bytes")), 10, 64)
	r, err2 := strconv.ParseUint(readTrim(filepath.Join(dir, "rx_bytes")), 10, 64)
	return t, r, err1 == nil && err2 == nil
}
