// Package pairing encodes and decodes tailbond://pair URIs and renders them as QR SVG.
package pairing

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
