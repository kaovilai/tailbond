package netif

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		f    Facts
		want string
	}{
		{Facts{Name: "wlan0"}, KindWifi},
		{Facts{Name: "phy1-sta0"}, KindWifi},
		{Facts{Name: "wwan"}, KindWifi},
		{Facts{Name: "wlp3s0"}, KindWifi},
		{Facts{Name: "foo", Wireless: true}, KindWifi},
		{Facts{Name: "wwan0"}, KindCellular},
		{Facts{Name: "rmnet_data0"}, KindCellular},
		{Facts{Name: "usb0", Driver: "qmi_wwan"}, KindCellular},
		{Facts{Name: "eth2", Driver: "cdc_mbim"}, KindCellular},
		{Facts{Name: "eth1", DevType: "wwan"}, KindCellular},
		{Facts{Name: "usb0", Driver: "rndis_host"}, KindUSB},
		{Facts{Name: "enx001122334455", USB: true}, KindUSB},
		{Facts{Name: "eth0"}, KindEthernet},
		{Facts{Name: "enp1s0"}, KindEthernet},
		{Facts{Name: "wan"}, KindEthernet},
		{Facts{Name: "xyz0"}, KindUnknown},
	}
	for _, c := range cases {
		if got := Classify(c.f); got != c.want {
			t.Errorf("Classify(%+v) = %s want %s", c.f, got, c.want)
		}
	}
}

func TestExcluded(t *testing.T) {
	for _, n := range []string{"lo", "tb0", "tailscale0", "docker0", "br-lan", "veth12", "podman0"} {
		if !excluded(n) {
			t.Errorf("%s not excluded", n)
		}
	}
	for _, n := range []string{"eth0", "wlan0", "wwan0", "usb0"} {
		if excluded(n) {
			t.Errorf("%s excluded", n)
		}
	}
}
