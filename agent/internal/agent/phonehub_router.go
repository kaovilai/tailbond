package agent

import (
	"context"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/proxy"
)

// routerHubs are the hotspot router's addresses on a Mac, for when mDNS
// and the server's list find no phone hub (docs/phone-hub.md):
//   - the IPv6 link-local routers from `ndp -rn`, with their zone
//     (networksetup's "IPv6 Router" is often empty or stale)
//   - the IPv4 default gateways, not 192.0.0.x (an iPhone hotspot's CLAT
//     address on IPv6-only carriers: nothing listens there)
//   - 172.20.10.1 when an interface is on 172.20.10.0/28 (an iPhone
//     hotspot whose gateway isn't the default route right now)
func routerHubs(ctx context.Context) []hubCand {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var addrs []string
	if out, err := exec.CommandContext(ctx, "ndp", "-rn").Output(); err == nil {
		addrs = append(addrs, ndpRouters(string(out))...)
	}
	if out, err := exec.CommandContext(ctx, "netstat", "-rn", "-f", "inet").Output(); err == nil {
		addrs = append(addrs, ipv4Gateways(string(out))...)
	}
	if onIPhoneHotspot() {
		addrs = append(addrs, "172.20.10.1")
	}
	return routerCands(addrs)
}

func routerCands(addrs []string) []hubCand {
	var out []hubCand
	seen := map[string]bool{}
	for _, a := range addrs {
		if seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, hubCand{ID: "router:" + a, Label: "Your phone", Via: "router",
			addrs: []string{net.JoinHostPort(a, strconv.Itoa(proxy.DefaultPort))}})
	}
	// one phone, several addresses: one entry that tries them in order
	if len(out) > 1 {
		all := hubCand{ID: "router", Label: "Your phone", Via: "router"}
		for _, c := range out {
			all.addrs = append(all.addrs, c.addrs...)
		}
		return []hubCand{all}
	}
	if len(out) == 1 {
		out[0].ID = "router"
	}
	return out
}

// ndpRouters parses `ndp -rn`:
//
//	fe80::1c3e:84ff:fe9b:3c64%en0 if=en0, flags=, pref=medium, expire=21m55s
func ndpRouters(out string) []string {
	var r []string
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 || !strings.HasPrefix(f[0], "fe80:") || !strings.Contains(f[0], "%") {
			continue
		}
		if strings.HasPrefix(f[0][strings.Index(f[0], "%")+1:], "utun") {
			continue
		}
		r = append(r, f[0])
	}
	return r
}

func ipv4Gateways(out string) []string {
	cgnatCLAT := netip.MustParsePrefix("192.0.0.0/29")
	var r []string
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) < 4 || f[0] != "default" {
			continue
		}
		ip, err := netip.ParseAddr(f[1])
		if err != nil || !ip.Is4() || cgnatCLAT.Contains(ip) {
			continue
		}
		r = append(r, ip.String())
	}
	return r
}

func onIPhoneHotspot() bool {
	hot := netip.MustParsePrefix("172.20.10.0/28")
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip, ok := netip.AddrFromSlice(n.IP.To4()); ok && hot.Contains(ip) {
				return true
			}
		}
	}
	return false
}
