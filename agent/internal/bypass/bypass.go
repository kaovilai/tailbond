// Package bypass renders bypass rules (docs/architecture.md, "Bypass
// rules") into the files the OpenWrt bypass-apply hook loads:
//
//   - an nftables table `inet tailbond` (firewall4) with one set per domain
//     rule for dnsmasq's nftset= and a named counter per rule;
//   - the same for firewall3: an ipset restore file and iptables /
//     ip6tables restore files for a mangle chain `tailbond_bypass`;
//   - two dnsmasq drop-ins (nftset= or ipset= lines, plus server= lines so
//     bypassed domains resolve through the WAN's own DNS);
//   - the WAN list: which mark and routing table each "only this WAN" target
//     uses.
//
// The hook decides which variant the router can run; the agent never
// touches the firewall itself.
package bypass

import (
	"fmt"
	"math/bits"
	"net/netip"
	"slices"
	"sort"
	"strings"

	"github.com/kaovilai/tailbond/agent/internal/config"
)

const (
	DefaultMask      = 0x07000000
	DefaultTableBase = 80
	MaxWANs          = 5
	Table            = "tailbond"         // nft table (inet family)
	Chain            = "tailbond_bypass"  // iptables mangle chain, LAN input
	RestoreChain     = "tailbond_restore" // every input: the connection's mark
)

// Marks within the mask: 1 = direct, 2 = tunnel, 3.. = one WAN each.
const (
	markDirect = 1
	markTunnel = 2
	markWAN0   = 3
)

type Options struct {
	Mask      uint32 // 0 = DefaultMask
	TableBase int    // 0 = DefaultTableBase
	// WAN DNS servers by OpenWrt network name, and each WAN's device (for
	// server=ip@dev). LocalDNS off: no server= lines.
	WANDNS   map[string][]string
	WANDev   map[string]string
	LocalDNS bool
	// HostMACs: DHCP host name (lower case) -> the MACs leased under it now;
	// a device rule with a host also matches those
	HostMACs map[string][]string
}

// WAN is one "only this WAN" target.
type WAN struct {
	Name  string
	Mark  uint32
	Table int
}

type Files struct {
	Nft           string // nft -f
	IPSet         string // ipset restore
	IPTables      string // iptables-restore --noflush
	IP6Tables     string // ip6tables-restore --noflush
	DnsmasqNftset string
	DnsmasqIPSet  string
	WANs          []WAN
	Domains       []string // domains to look up once so the sets fill
	Skipped       []string // rule ids left out (too many WANs)
}

// Marks returns the direct, tunnel and first-WAN mark values for a mask.
func Marks(mask uint32) (direct, tunnel uint32, wan func(i int) uint32) {
	shift := uint(bits.TrailingZeros32(mask))
	return markDirect << shift, markTunnel << shift, func(i int) uint32 { return uint32(markWAN0+i) << shift }
}

// MaxWANsFor is how many WAN marks fit in mask (values up to the mask).
func MaxWANsFor(mask uint32) int {
	shift := uint(bits.TrailingZeros32(mask))
	n := int(mask>>shift) - markWAN0 + 1
	if n > MaxWANs {
		n = MaxWANs
	}
	if n < 0 {
		n = 0
	}
	return n
}

// order: device rules first, then port, IP and domain rules, each in list
// order. The first rule that matches a new connection decides.
func order(rules []config.BypassRule) []config.BypassRule {
	rank := map[string]int{config.BypassDevice: 0, config.BypassPort: 1, config.BypassIP: 2, config.BypassDomain: 3}
	out := append([]config.BypassRule(nil), rules...)
	sort.SliceStable(out, func(i, j int) bool { return rank[out[i].Kind] < rank[out[j].Kind] })
	return out
}

// Render builds every variant for the enabled rules (rules must be
// normalized; disabled ones are skipped).
func Render(rules []config.BypassRule, opt Options) Files {
	mask := opt.Mask
	if mask == 0 {
		mask = DefaultMask
	}
	base := opt.TableBase
	if base == 0 {
		base = DefaultTableBase
	}
	direct, tunnel, wanMark := Marks(mask)
	maxW := MaxWANsFor(mask)
	var f Files
	wanIdx := map[string]int{}
	markOf := map[string]uint32{}
	var live []config.BypassRule
	for _, r := range order(rules) {
		if !r.On() {
			continue
		}
		switch r.Target {
		case config.TargetDirect:
			markOf[r.ID] = direct
		case config.TargetTunnel:
			markOf[r.ID] = tunnel
		case config.TargetWAN:
			i, ok := wanIdx[r.WAN]
			if !ok {
				if len(wanIdx) >= maxW {
					f.Skipped = append(f.Skipped, r.ID)
					continue
				}
				i = len(wanIdx)
				wanIdx[r.WAN] = i
				f.WANs = append(f.WANs, WAN{Name: r.WAN, Mark: wanMark(i), Table: base + i})
			}
			markOf[r.ID] = wanMark(i)
		default:
			continue
		}
		live = append(live, r)
	}
	macs := deviceMACs(live, opt.HostMACs)
	f.Nft = renderNft(live, markOf, mask, macs)
	f.IPSet, f.IPTables, f.IP6Tables = renderIPT(live, markOf, mask, macs)
	f.DnsmasqNftset, f.DnsmasqIPSet, f.Domains = renderDnsmasq(live, opt)
	return f
}

func hex(v uint32) string { return fmt.Sprintf("0x%08x", v) }

func setName(id string, v6 bool) string {
	if v6 {
		return "bp_" + id + "_6"
	}
	return "bp_" + id + "_4"
}

func ipsetName(id string, v6 bool) string {
	if v6 {
		return "tb_" + id + "_6"
	}
	return "tb_" + id + "_4"
}

// deviceMACs: every MAC each device rule matches: its own, plus the ones
// leased now under its host name that no other device rule claims.
func deviceMACs(rules []config.BypassRule, hostMACs map[string][]string) map[string][]string {
	claimed := map[string]bool{}
	for _, r := range rules {
		if r.Kind == config.BypassDevice && r.Match != "" {
			claimed[r.Match] = true
		}
	}
	out := map[string][]string{}
	for _, r := range rules {
		if r.Kind != config.BypassDevice {
			continue
		}
		var list []string
		if r.Match != "" {
			list = append(list, r.Match)
		}
		if r.Host != "" {
			for _, m := range hostMACs[strings.ToLower(r.Host)] {
				if !claimed[m] && !slices.Contains(list, m) {
					list = append(list, m)
				}
			}
		}
		sort.Strings(list[min(1, len(list)):]) // the rule's own MAC first
		out[r.ID] = list
	}
	return out
}

// nftMatches returns the match expressions of a rule: one nft rule each.
func nftMatches(r config.BypassRule, macs map[string][]string) []string {
	switch r.Kind {
	case config.BypassDevice:
		if m := macs[r.ID]; len(m) == 1 {
			return []string{"ether saddr " + m[0]}
		} else if len(m) > 1 {
			return []string{"ether saddr { " + strings.Join(m, ", ") + " }"}
		}
		return []string{"ip saddr " + r.IP}
	case config.BypassIP:
		p := netip.MustParsePrefix(r.Match)
		if p.Addr().Is4() {
			return []string{"ip daddr " + p.String()}
		}
		return []string{"ip6 daddr " + p.String()}
	case config.BypassPort:
		proto, ports, _ := strings.Cut(r.Match, "/")
		return []string{proto + " dport " + ports}
	case config.BypassDomain:
		return []string{"ip daddr @" + setName(r.ID, false), "ip6 daddr @" + setName(r.ID, true)}
	}
	return nil
}

func renderNft(rules []config.BypassRule, markOf map[string]uint32, mask uint32, macs map[string][]string) string {
	keep := ^mask
	var b strings.Builder
	// recreate atomically: declare, delete, define (nft -f is one transaction)
	fmt.Fprintf(&b, "table inet %s\ndelete table inet %s\n", Table, Table)
	fmt.Fprintf(&b, "table inet %s {\n", Table)
	b.WriteString("\t# LAN devices, filled in by the bypass-apply hook\n")
	b.WriteString("\tset lan_ifs {\n\t\ttype ifname\n\t}\n")
	for _, r := range rules {
		if r.Kind == config.BypassDomain {
			fmt.Fprintf(&b, "\t# %s (dnsmasq nftset=)\n", r.Match)
			fmt.Fprintf(&b, "\tset %s {\n\t\ttype ipv4_addr\n\t}\n", setName(r.ID, false))
			fmt.Fprintf(&b, "\tset %s {\n\t\ttype ipv6_addr\n\t}\n", setName(r.ID, true))
		}
	}
	for _, r := range rules {
		fmt.Fprintf(&b, "\tcounter c_%s {\n\t}\n", r.ID)
	}
	// before GL's own policy chains (priority mangle); only our bits change
	b.WriteString("\tchain prerouting {\n\t\ttype filter hook prerouting priority mangle - 5; policy accept;\n")
	// a connection keeps the route it started with, and its replies carry
	// the mark too: with src_valid_mark the kernel's reverse-path check
	// then uses the same rules, instead of the kill switch's blackhole.
	// (Nothing else has marked the packet yet at this priority; GL's own
	// chains restore their bits later and keep ours.)
	fmt.Fprintf(&b, "\t\tct mark & %s != 0 meta mark set ct mark & %s return\n", hex(mask), hex(mask))
	b.WriteString("\t\tiifname != @lan_ifs return\n")
	for _, r := range rules {
		m := markOf[r.ID]
		for _, match := range nftMatches(r, macs) {
			fmt.Fprintf(&b, "\t\t%s counter name \"c_%s\" meta mark set meta mark & %s | %s ct mark set ct mark & %s | %s accept comment \"%s\"\n",
				match, r.ID, hex(keep), hex(m), hex(keep), hex(m), r.ID)
		}
	}
	b.WriteString("\t}\n}\n")
	return b.String()
}

// iptMatch returns the iptables match for one family ("" = not this family).
func iptMatch(r config.BypassRule, v6 bool, macs map[string][]string) []string {
	switch r.Kind {
	case config.BypassDevice:
		if m := macs[r.ID]; len(m) > 0 {
			var out []string
			for _, x := range m {
				out = append(out, "-m mac --mac-source "+x)
			}
			return out
		}
		if v6 {
			return nil
		}
		return []string{"-s " + r.IP}
	case config.BypassIP:
		p := netip.MustParsePrefix(r.Match)
		if p.Addr().Is4() == v6 {
			return nil
		}
		return []string{"-d " + p.String()}
	case config.BypassPort:
		proto, ports, _ := strings.Cut(r.Match, "/")
		return []string{"-p " + proto + " --dport " + strings.ReplaceAll(ports, "-", ":")}
	case config.BypassDomain:
		return []string{"-m set --match-set " + ipsetName(r.ID, v6) + " dst"}
	}
	return nil
}

func renderIPT(rules []config.BypassRule, markOf map[string]uint32, mask uint32, macs map[string][]string) (ipset, v4, v6 string) {
	var s strings.Builder
	for _, r := range rules {
		if r.Kind == config.BypassDomain {
			fmt.Fprintf(&s, "create %s hash:ip family inet -exist\nflush %s\n", ipsetName(r.ID, false), ipsetName(r.ID, false))
			fmt.Fprintf(&s, "create %s hash:ip family inet6 -exist\nflush %s\n", ipsetName(r.ID, true), ipsetName(r.ID, true))
		}
	}
	chain := func(v6 bool) string {
		var b strings.Builder
		b.WriteString("*mangle\n")
		// replies and later packets: every interface (see renderNft)
		fmt.Fprintf(&b, ":%s - [0:0]\n-F %s\n", RestoreChain, RestoreChain)
		fmt.Fprintf(&b, "-A %s -m connmark ! --mark 0x0/%s -j CONNMARK --restore-mark --nfmask %s --ctmask %s\n", RestoreChain, hex(mask), hex(mask), hex(mask))
		fmt.Fprintf(&b, ":%s - [0:0]\n-F %s\n", Chain, Chain)
		fmt.Fprintf(&b, "-A %s -m mark ! --mark 0x0/%s -j RETURN\n", Chain, hex(mask))
		for _, r := range rules {
			for _, m := range iptMatch(r, v6, macs) {
				fmt.Fprintf(&b, "-A %s %s -m mark --mark 0x0/%s -m comment --comment \"tb:%s\" -j MARK --set-xmark %s/%s\n",
					Chain, m, hex(mask), r.ID, hex(markOf[r.ID]), hex(mask))
			}
		}
		fmt.Fprintf(&b, "-A %s -m mark ! --mark 0x0/%s -j CONNMARK --save-mark --nfmask %s --ctmask %s\n", Chain, hex(mask), hex(mask), hex(mask))
		b.WriteString("COMMIT\n")
		return b.String()
	}
	return s.String(), chain(false), chain(true)
}

func renderDnsmasq(rules []config.BypassRule, opt Options) (nftset, ipset string, domains []string) {
	var n, i strings.Builder
	head := "# tailbond bypass rules (generated; see docs/architecture.md)\n"
	n.WriteString(head)
	i.WriteString(head)
	for _, r := range rules {
		if r.Kind != config.BypassDomain {
			continue
		}
		domains = append(domains, r.Match)
		fmt.Fprintf(&n, "nftset=/%s/4#inet#%s#%s,6#inet#%s#%s\n", r.Match, Table, setName(r.ID, false), Table, setName(r.ID, true))
		fmt.Fprintf(&i, "ipset=/%s/%s,%s\n", r.Match, ipsetName(r.ID, false), ipsetName(r.ID, true))
		for _, s := range localServers(r, opt) {
			line := fmt.Sprintf("server=/%s/%s\n", r.Match, s)
			n.WriteString(line)
			i.WriteString(line)
		}
	}
	return n.String(), i.String(), domains
}

// localServers: the WAN DNS a bypassed domain resolves through, so it works
// while the tunnel (and its DNS) is down and CDNs answer for where the
// router is. "Only <WAN>" uses that WAN's servers bound to its device.
func localServers(r config.BypassRule, opt Options) []string {
	if !opt.LocalDNS || r.Target == config.TargetTunnel {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if r.Target == config.TargetWAN {
		for _, ip := range opt.WANDNS[r.WAN] {
			if dev := opt.WANDev[r.WAN]; dev != "" {
				add(ip + "@" + dev)
			} else {
				add(ip)
			}
		}
		return out
	}
	nets := make([]string, 0, len(opt.WANDNS))
	for k := range opt.WANDNS {
		nets = append(nets, k)
	}
	sort.Strings(nets)
	for _, k := range nets {
		for _, ip := range opt.WANDNS[k] {
			add(ip)
		}
	}
	return out
}

// MarksLine is the hook's marks file: "<mask> <direct> <tunnel>".
func MarksLine(mask uint32) string {
	if mask == 0 {
		mask = DefaultMask
	}
	d, t, _ := Marks(mask)
	return fmt.Sprintf("%s %s %s\n", hex(mask), hex(d), hex(t))
}

// WANList is the hook's WAN file: "<network> <mark> <table>" per line.
func (f Files) WANList() string {
	var b strings.Builder
	for _, w := range f.WANs {
		fmt.Fprintf(&b, "%s %s %d\n", w.Name, hex(w.Mark), w.Table)
	}
	return b.String()
}
