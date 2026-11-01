package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Bypass rules: LAN traffic that skips the tunnel (client role, applied by
// the OpenWrt bypass-apply hook; docs/architecture.md, "Bypass rules").

const (
	BypassDomain = "domain"
	BypassDevice = "device"
	BypassIP     = "ip"
	BypassPort   = "port"

	TargetDirect = "direct" // main table: GL's normal multi-WAN
	TargetWAN    = "wan"    // one WAN only
	TargetTunnel = "tunnel" // always through tailbond

	MaxBypassRules = 64
)

type BypassRule struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Match  string `json:"match"`
	IP     string `json:"ip,omitempty"` // device: used when match (the MAC) is empty
	Target string `json:"target"`
	WAN    string `json:"wan,omitempty"`
	// nil = enabled
	Enabled *bool  `json:"enabled,omitempty"`
	Note    string `json:"note,omitempty"`
}

func (r BypassRule) On() bool { return r.Enabled == nil || *r.Enabled }

var (
	reRuleID  = regexp.MustCompile(`^[a-z0-9]{1,16}$`)
	reLabel   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	reWANName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,32}$`)
	rePort    = regexp.MustCompile(`^(tcp|udp)/([0-9]{1,5})(-([0-9]{1,5}))?$`)
)

// NormalizeBypass checks a rule and returns it in canonical form: domains
// lower-case without scheme, path, "*." or trailing dot; MACs lower-case
// with colons; IPs as prefixes. The ID is left as is.
func NormalizeBypass(r BypassRule) (BypassRule, error) {
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.Match = strings.TrimSpace(r.Match)
	r.Target = strings.ToLower(strings.TrimSpace(r.Target))
	r.WAN = strings.TrimSpace(r.WAN)
	r.Note = strings.TrimSpace(r.Note)
	r.IP = strings.TrimSpace(r.IP)
	switch r.Kind {
	case BypassDomain:
		d, err := normalizeDomain(r.Match)
		if err != nil {
			return r, err
		}
		r.Match, r.IP = d, ""
	case BypassDevice:
		if r.Match != "" {
			hw, err := net.ParseMAC(r.Match)
			if err != nil || len(hw) != 6 {
				return r, fmt.Errorf("%q isn't a device address (MAC)", r.Match)
			}
			r.Match = hw.String()
		}
		if r.IP != "" {
			a, err := netip.ParseAddr(r.IP)
			if err != nil {
				return r, fmt.Errorf("%q isn't an IP address", r.IP)
			}
			r.IP = a.String()
		}
		if r.Match == "" && r.IP == "" {
			return r, errors.New("pick a device")
		}
	case BypassIP:
		p, err := parsePrefix(r.Match)
		if err != nil {
			return r, err
		}
		r.Match, r.IP = p.String(), ""
	case BypassPort:
		m := rePort.FindStringSubmatch(strings.ToLower(strings.ReplaceAll(r.Match, " ", "")))
		if m == nil {
			return r, fmt.Errorf("%q: use tcp/<port>, udp/<port> or udp/<from>-<to>", r.Match)
		}
		lo, _ := strconv.Atoi(m[2])
		hi := lo
		if m[4] != "" {
			hi, _ = strconv.Atoi(m[4])
		}
		if lo < 1 || hi > 65535 || hi < lo {
			return r, fmt.Errorf("%q: ports are 1-65535", r.Match)
		}
		r.Match = m[1] + "/" + strconv.Itoa(lo)
		if hi != lo {
			r.Match += "-" + strconv.Itoa(hi)
		}
		r.IP = ""
	default:
		return r, fmt.Errorf("kind must be domain, device, ip or port, not %q", r.Kind)
	}
	switch r.Target {
	case TargetDirect, TargetTunnel:
		r.WAN = ""
	case TargetWAN:
		if !reWANName.MatchString(r.WAN) {
			return r, errors.New("pick which internet connection to use")
		}
	default:
		return r, fmt.Errorf("target must be direct, wan or tunnel, not %q", r.Target)
	}
	if len(r.Note) > 64 {
		return r, errors.New("the note is longer than 64 characters")
	}
	for _, c := range r.Note {
		if unicode.IsControl(c) {
			return r, errors.New("the note has control characters")
		}
	}
	return r, nil
}

// ValidateBypass checks a whole list (config load and save).
func ValidateBypass(rules []BypassRule) error {
	if len(rules) > MaxBypassRules {
		return fmt.Errorf("client.bypass: at most %d rules", MaxBypassRules)
	}
	ids := map[string]bool{}
	for _, r := range rules {
		if !reRuleID.MatchString(r.ID) {
			return fmt.Errorf("client.bypass: bad id %q", r.ID)
		}
		if ids[r.ID] {
			return fmt.Errorf("client.bypass: id %q twice", r.ID)
		}
		ids[r.ID] = true
		if _, err := NormalizeBypass(r); err != nil {
			return fmt.Errorf("client.bypass[%s]: %w", r.ID, err)
		}
	}
	return nil
}

func ValidBypassID(id string) bool { return reRuleID.MatchString(id) }

func normalizeDomain(s string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(s))
	if i := strings.Index(d, "://"); i >= 0 {
		d = d[i+3:]
	}
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	if h, _, err := net.SplitHostPort(d); err == nil {
		d = h
	}
	d = strings.TrimPrefix(d, "*.")
	d = strings.TrimPrefix(d, ".")
	d = strings.TrimSuffix(d, ".")
	if d == "" {
		return "", errors.New("type a website, e.g. netflix.com")
	}
	if net.ParseIP(d) != nil {
		return "", fmt.Errorf("%q is an IP address: add it as an IP rule", d)
	}
	if len(d) > 253 || !strings.Contains(d, ".") {
		return "", fmt.Errorf("%q isn't a website name", s)
	}
	for _, l := range strings.Split(d, ".") {
		if !reLabel.MatchString(l) {
			return "", fmt.Errorf("%q isn't a website name", s)
		}
	}
	return d, nil
}

func parsePrefix(s string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%q isn't an IP address or range (CIDR)", s)
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}
