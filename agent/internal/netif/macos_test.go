package netif

import (
	"encoding/binary"
	"testing"
)

const hwPortsOut = `
Hardware Port: Ethernet
Device: en0
Ethernet Address: 00:11:22:33:44:01

Hardware Port: Wi-Fi
Device: en1
Ethernet Address: 3c:22:fb:00:00:01

Hardware Port: Thunderbolt Bridge
Device: bridge0
Ethernet Address: 82:0a:00:00:00:01

Hardware Port: Thunderbolt 1
Device: en2
Ethernet Address: 82:0a:00:00:00:02

Hardware Port: iPhone USB
Device: en8
Ethernet Address: 6a:00:00:00:00:01

Hardware Port: USB 10/100/1000 LAN
Device: en7
Ethernet Address: 00:e0:4c:00:00:01

Hardware Port: Thunderbolt Ethernet Slot 1
Device: en9
Ethernet Address: 00:e0:4c:00:00:02

Hardware Port: Bluetooth PAN
Device: en6
Ethernet Address: 3c:22:fb:00:00:02

VLAN Configurations
===================
`

func TestMacHardwarePorts(t *testing.T) {
	ports := ParseHardwarePorts(hwPortsOut)
	want := []struct {
		dev, kind, label string
		skip             bool
	}{
		{"en0", KindEthernet, "Ethernet", false},
		{"en1", KindWifi, "Wi-Fi", false},
		{"bridge0", "", "", true},
		{"en2", "", "", true},
		{"en8", KindUSB, "iPhone (USB)", false},
		{"en7", KindEthernet, "USB 10/100/1000 LAN", false},
		{"en9", KindEthernet, "Thunderbolt Ethernet Slot 1", false},
		{"en6", KindUSB, "Bluetooth tethering", false},
	}
	if len(ports) != len(want) {
		t.Fatalf("parsed %d ports, want %d: %v", len(ports), len(want), ports)
	}
	for _, w := range want {
		hp, ok := ports[w.dev]
		if !ok {
			t.Fatalf("%s missing", w.dev)
		}
		kind, label, skip := ClassifyPort(hp.Port, w.dev)
		if kind != w.kind || label != w.label || skip != w.skip {
			t.Errorf("%s (%q): got %q %q %v, want %q %q %v", w.dev, hp.Port, kind, label, skip, w.kind, w.label, w.skip)
		}
	}
	// no hardware port: the name heuristic
	if k, l, s := ClassifyPort("", "en5"); k != KindEthernet || l != "" || s {
		t.Errorf("unlisted en5: %q %q %v", k, l, s)
	}
}

func TestMacSkipped(t *testing.T) {
	for _, n := range []string{"lo0", "utun3", "awdl0", "llw0", "bridge100", "ap1", "gif0", "stf0", "anpi0", "feth0", "tb0"} {
		if !darwinSkipped(n) {
			t.Errorf("%s should be skipped", n)
		}
	}
	for _, n := range []string{"en0", "en8"} {
		if darwinSkipped(n) {
			t.Errorf("%s should not be skipped", n)
		}
	}
}

func TestMacIfconfigStatus(t *testing.T) {
	out := "lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384\n\tinet 127.0.0.1 netmask 0xff000000\n" +
		"en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500\n\tether 00:11:22:33:44:01\n\tmedia: autoselect\n\tstatus: active\n" +
		"en7: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500\n\tmedia: autoselect (none)\n\tstatus: inactive\n" +
		"utun0: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1380\n"
	st := ParseIfconfigStatus(out)
	if !st["en0"] || st["en7"] || len(st) != 2 {
		t.Fatalf("got %v", st)
	}
}

// A NET_RT_IFLIST2 dump: an RTM_NEWMADDR2-ish message, then two
// RTM_IFINFO2 messages; the counters of the asked-for index come back.
func TestMacIflist2Counters(t *testing.T) {
	le := binary.LittleEndian
	msg := func(typ byte, index uint16, ib, ob uint64) []byte {
		m := make([]byte, ifm2MinLen+8)
		le.PutUint16(m[0:], uint16(len(m)))
		m[2], m[3] = 5, typ
		le.PutUint16(m[ifm2IndexOff:], index)
		le.PutUint64(m[ifm2DataOff+ifData64IBytes:], ib)
		le.PutUint64(m[ifm2DataOff+ifData64OBytes:], ob)
		return m
	}
	var b []byte
	b = append(b, msg(0x13, 4, 1, 1)...)
	b = append(b, msg(rtmIfinfo2, 1, 100, 200)...)
	b = append(b, msg(rtmIfinfo2, 4, 1<<40, 7)...)
	if tx, rx, ok := countersFromIflist2(b, 4); !ok || rx != 1<<40 || tx != 7 {
		t.Fatalf("index 4: %d %d %v", tx, rx, ok)
	}
	if _, _, ok := countersFromIflist2(b, 9); ok {
		t.Fatal("index 9 is not in the dump")
	}
	if _, _, ok := countersFromIflist2(b[:10], 1); ok {
		t.Fatal("a truncated dump must not parse")
	}
}
