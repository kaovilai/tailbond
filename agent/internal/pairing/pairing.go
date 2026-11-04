// Package pairing encodes and decodes tailbond://pair URIs and renders them as QR SVG.
package pairing

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

const (
	Scheme  = "tailbond"
	Version = 1
)

type Payload struct {
	V             int      `json:"v"`
	Name          string   `json:"name"`
	Server        string   `json:"server"`
	SNI           string   `json:"sni,omitempty"`
	User          string   `json:"user"`
	Key           string   `json:"key"`
	Insecure      bool     `json:"insecure"`
	CertSHA256    string   `json:"cert_sha256,omitempty"`
	DNS           []string `json:"dns,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	TailnetRoutes []string `json:"tailnet_routes,omitempty"`
	// Additive: the server's own name and IPv4 tunnel subnet, so a client
	// can keep several servers apart (docs/architecture.md).
	ServerName string `json:"server_name,omitempty"`
	Subnet     string `json:"subnet,omitempty"`
	// Additive: the server's share key (Pair & Share): devices paired with
	// the same server use each other's shared mobile data with it.
	ShareKey string `json:"share_key,omitempty"`
}

// DefaultSubnet is what a server without the subnet key uses.
const DefaultSubnet = "10.77.0.0/24"

// Label is the server's name in a client's list: server_name, else the
// first label of sni, else the host of server.
func (p *Payload) Label() string {
	if n := strings.TrimSpace(p.ServerName); n != "" {
		return n
	}
	if p.SNI != "" {
		if net.ParseIP(p.SNI) == nil {
			first, _, _ := strings.Cut(p.SNI, ".")
			if first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(p.Server)
	if err != nil {
		host = p.Server
	}
	return host
}

// TunnelSubnet is the server's IPv4 tunnel subnet: subnet, else the /24
// around dns[0], else DefaultSubnet.
func (p *Payload) TunnelSubnet() netip.Prefix {
	if pfx, err := netip.ParsePrefix(p.Subnet); err == nil && pfx.Addr().Is4() {
		return pfx.Masked()
	}
	for _, d := range p.DNS {
		if a, err := netip.ParseAddr(d); err == nil && a.Is4() {
			pfx, _ := a.Prefix(24)
			return pfx
		}
	}
	return netip.MustParsePrefix(DefaultSubnet)
}

func (p *Payload) Validate() error {
	if p.V != Version {
		return fmt.Errorf("unsupported pairing version %d", p.V)
	}
	if p.Server == "" || p.Key == "" {
		return errors.New("pairing payload needs server and key")
	}
	if _, _, err := net.SplitHostPort(p.Server); err != nil {
		return fmt.Errorf("server %q: %w", p.Server, err)
	}
	if p.Insecure && p.CertSHA256 == "" {
		return errors.New("insecure pairing without cert_sha256 pin")
	}
	return nil
}

// Encode returns tailbond://pair?d=<base64url(JSON)>.
func Encode(p *Payload) (string, error) {
	if p.V == 0 {
		p.V = Version
	}
	if err := p.Validate(); err != nil {
		return "", err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return Scheme + "://pair?d=" + base64.RawURLEncoding.EncodeToString(b), nil
}

// Decode accepts the URI, tolerating padding and surrounding whitespace.
func Decode(uri string) (*Payload, error) {
	uri = strings.TrimSpace(uri)
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("pairing uri: %w", err)
	}
	if u.Scheme != Scheme || (u.Host != "pair" && u.Opaque != "//pair") {
		return nil, errors.New("not a tailbond://pair URI")
	}
	d := u.Query().Get("d")
	if d == "" {
		return nil, errors.New("pairing uri: missing d parameter")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(d, "="))
	if err != nil {
		return nil, fmt.Errorf("pairing uri: bad base64url: %w", err)
	}
	p := &Payload{}
	if err := json.Unmarshal(raw, p); err != nil {
		return nil, fmt.Errorf("pairing uri: bad JSON: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}
