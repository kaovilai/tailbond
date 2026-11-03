package config

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/kaovilai/tailbond/agent/internal/pairing"
)

// MigrateServers keeps client.servers and client.pairing_uri in step
// (docs/architecture.md, "Agent config file" → Migration):
//   - a pairing_uri that isn't in the list (an old single-server config, or
//     one set by hand or through UCI) becomes the primary;
//   - pairing_uri is then always the primary's URI ("" with no servers).
func (c *Client) MigrateServers() {
	uri := strings.TrimSpace(c.PairingURI)
	if uri != "" && !c.hasURI(uri) {
		name := "home"
		if p, err := pairing.Decode(uri); err == nil {
			name = p.Label()
		}
		entry := ServerEntry{Name: name, PairingURI: uri}
		if len(c.Servers) == 0 {
			c.Servers = []ServerEntry{entry}
		} else {
			rest := []ServerEntry{entry}
			for _, e := range c.Servers[1:] {
				if e.Name != name {
					rest = append(rest, e)
				}
			}
			c.Servers = rest
		}
		c.Servers[0].Name = UniqueServerName(c.Servers[1:], c.Servers[0].Name)
	}
	if len(c.Servers) > 0 {
		c.PairingURI = c.Servers[0].PairingURI
	} else {
		c.PairingURI = ""
	}
}

func (c *Client) hasURI(uri string) bool {
	for _, e := range c.Servers {
		if strings.TrimSpace(e.PairingURI) == uri {
			return true
		}
	}
	return false
}

// UniqueServerName returns name, or name-2, name-3... if a server in list
// already has it.
func UniqueServerName(list []ServerEntry, name string) string {
	if name == "" {
		name = "server"
	}
	taken := map[string]bool{}
	for _, e := range list {
		taken[e.Name] = true
	}
	out := name
	for n := 2; taken[out]; n++ {
		out = fmt.Sprintf("%s-%d", name, n)
	}
	return out
}

// ServerIndex returns the position of a server by name, or -1.
func (c *Client) ServerIndex(name string) int {
	for i, e := range c.Servers {
		if e.Name == name {
			return i
		}
	}
	return -1
}

// Overlaps reports whether two prefixes share any address.
func Overlaps(a, b netip.Prefix) bool {
	return a.IsValid() && b.IsValid() && a.Overlaps(b)
}

// NextFreeSubnet is the first /24 from 10.78.0.0/24 up (to 10.99.255.0/24)
// that overlaps nothing in avoid, for a server that has to move out of a
// subnet a client already uses (docs/architecture.md, "Distinct subnets").
func NextFreeSubnet(avoid []netip.Prefix) (netip.Prefix, bool) {
	for b := 78; b <= 99; b++ {
		for c := 0; c < 256; c++ {
			p := netip.PrefixFrom(netip.AddrFrom4([4]byte{10, byte(b), byte(c), 0}), 24)
			free := true
			for _, x := range avoid {
				if Overlaps(p, x) {
					free = false
					break
				}
			}
			if free {
				return p, true
			}
		}
	}
	return netip.Prefix{}, false
}
