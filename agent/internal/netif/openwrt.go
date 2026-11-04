package netif

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// OpenWrt / GL.iNet: WANs come from netifd (`ubus call network.interface
// dump`) and the firewall's wan zone, so paths can be keyed by network name
// (wan, wwan, tethering, modem_0001...) whatever device a phone or modem
// brings up this time.

// WAN is one OpenWrt network that can carry a path.
type WAN struct {
	Network string
	Device  string
	Proto   string
	Up      bool
	Kind    string
	Label   string
	DNS     []string // the network's own DNS servers (DHCP), for captive probes
}

// run executes a command (tests replace it).
var run = func(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// lookPath reports whether a command exists (tests replace it).
var lookPath = func(name string) bool { _, err := exec.LookPath(name); return err == nil }

type ifDump struct {
	Interface []struct {
		Interface string   `json:"interface"`
		Up        bool     `json:"up"`
		Proto     string   `json:"proto"`
		L3Device  string   `json:"l3_device"`
		Device    string   `json:"device"`
		DNSServer []string `json:"dns-server"`
	} `json:"interface"`
}

var cellularProtos = map[string]bool{
	"qmi": true, "mbim": true, "ncm": true, "3g": true, "qcm": true, "modemmanager": true,
	"wwan": true, "directip": true, "quectel": true, "xmm": true,
}

// reWANName matches GL.iNet's WAN network names (lib/functions/kmwan.sh):
// wan, secondwan, usbwan, wwan, tethering, modem_*.
var reWANName = regexp.MustCompile(`^(wan|secondwan|usbwan|wwan|tethering|modem.*)$`)

// classifyNetwork returns kind and label for a network by its name and proto.
func classifyNetwork(name, proto string) (kind, label string) {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "modem") || cellularProtos[proto]:
		label = "Cellular"
		if i := strings.LastIndex(n, "_s"); i > 0 && len(n) == i+3 && n[i+2] >= '1' && n[i+2] <= '9' {
			label = "Cellular (SIM " + n[i+2:] + ")"
		}
		return KindCellular, label
	case strings.HasPrefix(n, "tethering"):
		return KindUSB, "Tethering"
	case strings.HasPrefix(n, "wwan"):
		return KindWifi, "Repeater"
	case n == "secondwan":
		return KindEthernet, "Ethernet 2"
	case n == "usbwan":
		return KindEthernet, "USB Ethernet"
	case n == "wan":
		return KindEthernet, "Ethernet"
	}
	return "", ""
}

// wanZoneNetworks lists the networks in the firewall zone named "wan".
func wanZoneNetworks() map[string]bool {
	out, err := run("uci", "-q", "show", "firewall")
	if err != nil {
		return nil
	}
	return parseWANZone(string(out))
}

func parseWANZone(show string) map[string]bool {
	names := map[string]string{} // section -> zone name
	nets := map[string][]string{}
	for _, line := range strings.Split(show, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		parts := strings.Split(k, ".")
		if len(parts) != 3 || parts[0] != "firewall" {
			continue
		}
		switch parts[2] {
		case "name":
			names[parts[1]] = strings.Trim(v, "'")
		case "network":
			for _, f := range strings.Fields(v) {
				nets[parts[1]] = append(nets[parts[1]], strings.Trim(f, "'"))
			}
		}
	}
	res := map[string]bool{}
	for sec, name := range names {
		if name == "wan" {
			for _, n := range nets[sec] {
				res[n] = true
			}
		}
	}
	return res
}

// OpenWrtWANs returns the WAN networks with their current devices; ok is
// false when this isn't OpenWrt (no ubus).
func OpenWrtWANs() (wans []WAN, ok bool) {
	if !lookPath("ubus") {
		return nil, false
	}
	out, err := run("ubus", "call", "network.interface", "dump")
	if err != nil {
		return nil, false
	}
	var d ifDump
	if json.Unmarshal(out, &d) != nil {
		return nil, false
	}
	zone := wanZoneNetworks()
	byDev := map[string]int{}
	for _, it := range d.Interface {
		name := it.Interface
		if name == "loopback" || name == "lan" || name == "tailbond" {
			continue
		}
		if !zone[name] && !reWANName.MatchString(name) {
			continue
		}
		// IPv6 twins (wan6, wwan6, modem_..._6) share the IPv4 network's device
		if it.Proto == "dhcpv6" || it.Proto == "6in4" || it.Proto == "6to4" {
			continue
		}
		dev := it.L3Device
		if dev == "" {
			dev = it.Device
		}
		if dev == "" || strings.HasPrefix(dev, "/") { // a modem's /dev node: not up yet
			continue
		}
		// GL gives a QMI/MBIM modem an IPv4 child network "<modem>_4" on
		// the same device; key both as the modem so settings stick.
		if strings.HasPrefix(name, "modem") && strings.HasSuffix(name, "_4") {
			name = strings.TrimSuffix(name, "_4")
		}
		kind, label := classifyNetwork(name, it.Proto)
		if kind == "" {
			kind, label = KindOf(dev), name
		}
		w := WAN{Network: name, Device: dev, Proto: it.Proto, Up: it.Up, Kind: kind, Label: label, DNS: it.DNSServer}
		if i, dup := byDev[dev]; dup {
			// keep the one that's up (e.g. modem_0001 vs modem_0001_4)
			if !wans[i].Up && w.Up {
				wans[i] = w
			}
			continue
		}
		byDev[dev] = len(wans)
		wans = append(wans, w)
	}
	return wans, true
}

// annotate fills Network/Label/Kind/WAN on discovered interfaces.
func annotate(ifs []Iface, wans []WAN) []Iface {
	byDev := map[string]WAN{}
	for _, w := range wans {
		byDev[w.Device] = w
	}
	for i := range ifs {
		if w, ok := byDev[ifs[i].Name]; ok {
			label := w.Label
			if t := TetherLabel(ifs[i].Driver); t != "" {
				label = t // the tethering WAN is an iPhone
			}
			ifs[i].Network, ifs[i].Label, ifs[i].Kind, ifs[i].WAN, ifs[i].DNS = w.Network, label, w.Kind, true, w.DNS
		}
	}
	return ifs
}
