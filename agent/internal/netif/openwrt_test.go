package netif

import (
	"errors"
	"strings"
	"testing"
)

// A Slate 7 at a hotel: repeater Wi-Fi up, phone USB tethering up, the
// ethernet WAN unplugged, a cellular dongle (GL names modem_<bus>, with an
// IPv4 twin) and the usual IPv6 twins.
const slateDump = `{"interface":[
 {"interface":"lan","up":true,"proto":"static","l3_device":"br-lan","device":"br-lan"},
 {"interface":"loopback","up":true,"proto":"static","l3_device":"lo"},
 {"interface":"wan","up":false,"proto":"dhcp","device":"eth0"},
 {"interface":"wan6","up":false,"proto":"dhcpv6","device":"eth0"},
 {"interface":"wwan","up":true,"proto":"dhcp","l3_device":"phy0-sta0","device":"phy0-sta0"},
 {"interface":"wwan6","up":true,"proto":"dhcpv6","l3_device":"phy0-sta0"},
 {"interface":"tethering","up":true,"proto":"dhcp","l3_device":"usb0","device":"usb0"},
 {"interface":"tethering6","up":true,"proto":"dhcpv6","l3_device":"usb0"},
 {"interface":"modem_0001","up":false,"proto":"qmi","device":"/dev/cdc-wdm0"},
 {"interface":"modem_0001_4","up":true,"proto":"dhcp","l3_device":"wwan0","device":"wwan0"},
 {"interface":"guest","up":true,"proto":"static","l3_device":"br-guest"},
 {"interface":"tailbond","up":true,"proto":"none","l3_device":"tb0"}
]}`

const slateFirewall = `firewall.cfg01e63d=defaults
firewall.cfg02dc81=zone
firewall.cfg02dc81.name='lan'
firewall.cfg02dc81.network='lan'
firewall.cfg03dc81=zone
firewall.cfg03dc81.name='wan'
firewall.cfg03dc81.network='wan' 'wan6' 'wwan' 'wwan6' 'tethering' 'tethering6' 'modem_0001' 'modem_0001_4'
firewall.guest=zone
firewall.guest.name='guest'
firewall.guest.network='guest'`

func stubOpenWrt(t *testing.T, dump, fw string) {
	t.Helper()
	oldRun, oldLook := run, lookPath
	t.Cleanup(func() { run, lookPath = oldRun, oldLook })
	lookPath = func(string) bool { return true }
	run = func(name string, args ...string) ([]byte, error) {
		switch name + " " + strings.Join(args, " ") {
		case "ubus call network.interface dump":
			return []byte(dump), nil
		case "uci -q show firewall":
			return []byte(fw), nil
		}
		return nil, errors.New("unexpected " + name)
	}
}

func TestOpenWrtWANs(t *testing.T) {
	stubOpenWrt(t, slateDump, slateFirewall)
	wans, ok := OpenWrtWANs()
	if !ok {
		t.Fatal("not detected as OpenWrt")
	}
	got := map[string]WAN{}
	for _, w := range wans {
		got[w.Network] = w
	}
	if len(wans) != 4 {
		t.Fatalf("wans = %+v", wans)
	}
	for net, want := range map[string][3]string{
		"wan":          {"eth0", KindEthernet, "Ethernet"},
		"wwan":         {"phy0-sta0", KindWifi, "Repeater"},
		"tethering":    {"usb0", KindUSB, "Tethering"},
		"modem_0001":   {"wwan0", KindCellular, "Cellular"},
	} {
		w := got[net]
		if w.Device != want[0] || w.Kind != want[1] || w.Label != want[2] {
			t.Errorf("%s = %+v, want %v", net, w, want)
		}
	}
	ifs := annotate([]Iface{{Name: "usb0", Kind: KindEthernet}, {Name: "br-lan"}, {Name: "wwan0"}}, wans)
	if ifs[0].Network != "tethering" || ifs[0].Kind != KindUSB || !ifs[0].WAN || ifs[0].Key() != "tethering" {
		t.Fatalf("usb0 = %+v", ifs[0])
	}
	if ifs[1].WAN || ifs[1].Key() != "br-lan" {
		t.Fatalf("br-lan = %+v", ifs[1])
	}
}

func TestOpenWrtWANsWithoutZoneUsesGLNames(t *testing.T) {
	stubOpenWrt(t, slateDump, "")
	wans, _ := OpenWrtWANs()
	if len(wans) != 4 {
		t.Fatalf("wans = %+v", wans)
	}
}

func TestNotOpenWrt(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })
	lookPath = func(string) bool { return false }
	if _, ok := OpenWrtWANs(); ok {
		t.Fatal("detected OpenWrt without ubus")
	}
}

func TestClassifyNetworkSIM(t *testing.T) {
	if k, l := classifyNetwork("modem_0001_s2", "qmi"); k != KindCellular || l != "Cellular (SIM 2)" {
		t.Fatalf("%s %s", k, l)
	}
	if k, l := classifyNetwork("secondwan", "dhcp"); k != KindEthernet || l != "Ethernet 2" {
		t.Fatalf("%s %s", k, l)
	}
	if k, _ := classifyNetwork("lte", "mbim"); k != KindCellular {
		t.Fatalf("mbim proto: %s", k)
	}
}
