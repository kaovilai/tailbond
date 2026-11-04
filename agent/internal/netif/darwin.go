//go:build darwin

package netif

import (
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// hwPorts caches `networksetup -listallhardwareports` (it changes only
// when an adapter or an iPhone is plugged in; a minute is soon enough,
// and an unknown device triggers a refresh at once).
var hwPorts struct {
	mu    sync.Mutex
	at    time.Time
	ports map[string]HardwarePort
}

func hardwarePorts(want string) map[string]HardwarePort {
	hwPorts.mu.Lock()
	defer hwPorts.mu.Unlock()
	_, known := hwPorts.ports[want]
	if hwPorts.ports == nil || time.Since(hwPorts.at) > time.Minute || (want != "" && !known && time.Since(hwPorts.at) > 5*time.Second) {
		if out, err := run("networksetup", "-listallhardwareports"); err == nil {
			hwPorts.ports, hwPorts.at = ParseHardwarePorts(string(out)), time.Now()
		} else if hwPorts.ports == nil {
			hwPorts.ports = map[string]HardwarePort{}
		}
	}
	return hwPorts.ports
}

// KindOf classifies one interface by its hardware port.
func KindOf(name string) string {
	kind, _, _ := ClassifyPort(hardwarePorts(name)[name].Port, name)
	if kind == "" {
		return KindUnknown
	}
	return kind
}

// Discover lists this Mac's candidate WAN interfaces: Wi-Fi, Ethernet and
// USB-C adapters, an iPhone over USB, labelled by hardware port.
func Discover() ([]Iface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ports := hardwarePorts("")
	media := map[string]bool{}
	if out, err := run("ifconfig"); err == nil {
		media = ParseIfconfigStatus(string(out))
	}
	out := []Iface{}
	for _, ni := range ifs {
		if ni.Flags&net.FlagLoopback != 0 || ni.Flags&net.FlagPointToPoint != 0 || darwinSkipped(ni.Name) {
			continue
		}
		hp, listed := ports[ni.Name]
		if !listed {
			// a just-plugged adapter: ask networksetup again
			hp, listed = hardwarePorts(ni.Name)[ni.Name]
		}
		it, skip := describeMac(ni, hp, media)
		if skip || (!listed && len(ni.HardwareAddr) == 0) {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}

func describeMac(ni net.Interface, hp HardwarePort, media map[string]bool) (Iface, bool) {
	kind, label, skip := ClassifyPort(hp.Port, ni.Name)
	it := Iface{Name: ni.Name, Kind: kind, Label: label, Up: ni.Flags&net.FlagUp != 0,
		MAC: ni.HardwareAddr.String(), Addrs: []string{}, Driver: hp.Port}
	if st, ok := media[ni.Name]; ok {
		it.Carrier = st
	} else {
		it.Carrier = it.Up
	}
	if addrs, err := ni.Addrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLinkLocalUnicast() {
				it.Addrs = append(it.Addrs, ipn.String())
			}
		}
	}
	if it.Kind == "" {
		it.Kind = KindUnknown
	}
	return it, skip
}

// Lookup describes one interface by name, whatever Discover makes of it
// (a path the user names explicitly); tunnels are never paths.
func Lookup(name string) (Iface, bool) {
	ni, err := net.InterfaceByName(name)
	if err != nil || ni.Flags&net.FlagPointToPoint != 0 {
		return Iface{}, false
	}
	media := map[string]bool{}
	if out, err := run("ifconfig", name); err == nil {
		media = ParseIfconfigStatus(string(out))
	}
	it, _ := describeMac(*ni, hardwarePorts(name)[name], media)
	return it, true
}

// Counters reads an interface's byte counters (64-bit, if_data64) from
// the kernel's interface list (sysctl NET_RT_IFLIST2).
func Counters(name string) (tx, rx uint64, ok bool) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return 0, 0, false
	}
	b, err := unix.SysctlRaw("net.route", 0, syscall.AF_UNSPEC, unix.NET_RT_IFLIST2, 0)
	if err != nil {
		return 0, 0, false
	}
	return countersFromIflist2(b, ifi.Index)
}
