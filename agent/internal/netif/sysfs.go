//go:build !darwin

package netif

// Linux (and OpenWrt): interfaces from sysfs.

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SysfsRoot is /sys/class/net, overridable in tests.
var SysfsRoot = "/sys/class/net"

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
	it.Label = TetherLabel(f.Driver)
	return it
}

// Counters reads kernel byte counters for an interface (fallback stats when
// mqvpn's client control API is absent).
func Counters(name string) (tx, rx uint64, ok bool) {
	dir := filepath.Join(SysfsRoot, name, "statistics")
	t, err1 := strconv.ParseUint(readTrim(filepath.Join(dir, "tx_bytes")), 10, 64)
	r, err2 := strconv.ParseUint(readTrim(filepath.Join(dir, "rx_bytes")), 10, 64)
	return t, r, err1 == nil && err2 == nil
}

// Lookup describes one interface by name, whatever Discover makes of it
// (a path the user names explicitly); tunnels are never paths.
func Lookup(name string) (Iface, bool) {
	ni, err := net.InterfaceByName(name)
	if err != nil || ni.Flags&net.FlagPointToPoint != 0 {
		return Iface{}, false
	}
	return describe(*ni), true
}
