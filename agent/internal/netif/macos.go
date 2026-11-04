package netif

// macOS: what an interface is comes from its hardware port
// (`networksetup -listallhardwareports`), whether it has a link from
// `ifconfig` ("status: active"). The parsers live here, untagged, so they
// are tested on every platform; darwin.go runs the commands.

import (
	"bufio"
	"encoding/binary"
	"strings"
)

// HardwarePort is one entry of `networksetup -listallhardwareports`.
type HardwarePort struct {
	Port   string // "Wi-Fi", "iPhone USB", "USB 10/100/1000 LAN", ...
	Device string // "en0"
}

// ParseHardwarePorts reads `networksetup -listallhardwareports`:
//
//	Hardware Port: Wi-Fi
//	Device: en0
//	Ethernet Address: 3c:22:fb:00:00:01
func ParseHardwarePorts(out string) map[string]HardwarePort {
	ports := map[string]HardwarePort{}
	var cur HardwarePort
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if v, ok := strings.CutPrefix(line, "Hardware Port:"); ok {
			cur = HardwarePort{Port: strings.TrimSpace(v)}
		} else if v, ok := strings.CutPrefix(line, "Device:"); ok {
			cur.Device = strings.TrimSpace(v)
			if cur.Port != "" && cur.Device != "" {
				ports[cur.Device] = cur
			}
		}
	}
	return ports
}

// ClassifyPort gives kind and label for a macOS interface from its
// hardware port; port "" (none listed) falls back to the name heuristic.
// skip is true for ports that never carry a path (bridges, Thunderbolt
// bridge members, VPN-ish virtual ports).
func ClassifyPort(port, dev string) (kind, label string, skip bool) {
	p := strings.ToLower(port)
	switch {
	case port == "":
		return Classify(Facts{Name: dev}), "", false
	case strings.Contains(p, "bridge") || strings.HasPrefix(p, "thunderbolt ") && !strings.Contains(p, "ethernet"):
		// "Thunderbolt Bridge" and its members "Thunderbolt 1", "Thunderbolt 2"
		return "", "", true
	case p == "wi-fi" || p == "airport" || strings.Contains(p, "wi-fi"):
		return KindWifi, "Wi-Fi", false
	case strings.Contains(p, "iphone"):
		return KindUSB, "iPhone (USB)", false
	case strings.Contains(p, "ipad"):
		return KindUSB, "iPad (USB)", false
	case strings.Contains(p, "bluetooth"):
		return KindUSB, "Bluetooth tethering", false
	case strings.Contains(p, "ethernet") || strings.Contains(p, " lan") || strings.HasSuffix(p, "lan") ||
		strings.Contains(p, "adapter") || strings.Contains(p, "usb"):
		// built-in Ethernet, Thunderbolt Ethernet, USB-C / USB 10/100/1000 LAN adapters
		if p == "ethernet" {
			return KindEthernet, "Ethernet", false
		}
		return KindEthernet, port, false
	case strings.Contains(p, "modem") || strings.Contains(p, "wwan") || strings.Contains(p, "lte") || strings.Contains(p, "5g"):
		return KindCellular, port, false
	}
	return Classify(Facts{Name: dev}), port, false
}

// darwinSkipPrefixes are macOS virtual interfaces that are never a WAN
// path (tunnels, AWDL/AirDrop, bridges, Personal Hotspot plumbing, test
// pairs).
var darwinSkipPrefixes = []string{"utun", "awdl", "llw", "bridge", "ap", "gif", "stf", "anpi", "feth", "vmenet", "vmnet", "ipsec", "ppp", "pktap", "XHC", "lo"}

// darwinSkipped reports whether a macOS interface is never a path.
func darwinSkipped(name string) bool {
	for _, p := range darwinSkipPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return excluded(name)
}

// ParseIfconfigStatus reads `ifconfig` (all interfaces) and reports each
// interface's media status: true for "status: active". Interfaces without
// a status line (tunnels, loopback) are absent.
func ParseIfconfigStatus(out string) map[string]bool {
	st := map[string]bool{}
	cur := ""
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if line[0] != '\t' && line[0] != ' ' {
			if i := strings.Index(line, ":"); i > 0 {
				cur = line[:i]
			}
			continue
		}
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "status:"); ok && cur != "" {
			st[cur] = strings.TrimSpace(v) == "active"
		}
	}
	return st
}

// if_msghdr2 (net/if.h): msglen u16, version u8, type u8, addrs i32,
// flags i32, index u16, pad, snd_len, snd_maxlen, snd_drops, timer (i32
// each), then if_data64: 8 bytes of u8 fields, mtu u32, metric u32,
// baudrate, ipackets, ierrors, opackets, oerrors, collisions, ibytes,
// obytes (u64 each).
const (
	rtmIfinfo2     = 0x12
	ifm2IndexOff   = 12
	ifm2DataOff    = 32
	ifData64IBytes = 64
	ifData64OBytes = 72
	ifm2MinLen     = ifm2DataOff + ifData64OBytes + 8
)

// countersFromIflist2 finds interface index in a NET_RT_IFLIST2 dump
// (little-endian, as on every Mac).
func countersFromIflist2(b []byte, index int) (tx, rx uint64, ok bool) {
	le := binary.LittleEndian
	for len(b) >= 4 {
		l := int(le.Uint16(b[0:2]))
		if l < 4 || l > len(b) {
			break
		}
		m := b[:l]
		b = b[l:]
		if m[3] != rtmIfinfo2 || l < ifm2MinLen || int(le.Uint16(m[ifm2IndexOff:])) != index {
			continue
		}
		d := m[ifm2DataOff:]
		return le.Uint64(d[ifData64OBytes:]), le.Uint64(d[ifData64IBytes:]), true
	}
	return 0, 0, false
}
