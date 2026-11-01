// Package captive tells whether a WAN sits behind a captive portal (hotel
// or airport Wi-Fi, a hotspot that wants a sign-in first) by probing a
// known URL over that WAN only (docs/architecture.md, "Captive portals").
package captive

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	Open    = "open"    // the probe got the expected answer
	Portal  = "portal"  // something answered in its place: sign-in needed
	Offline = "offline" // no answer (DNS failure, timeout, refused)
)

// Probe is one check URL and the answer it gives on an open network.
type Probe struct {
	URL      string
	WantCode int    // expected status code
	WantBody string // if set, the body must contain it
}

// DefaultProbes are tried in order; the first conclusive answer wins.
var DefaultProbes = []Probe{
	{URL: "http://connectivitycheck.gstatic.com/generate_204", WantCode: http.StatusNoContent},
	{URL: "http://detectportal.firefox.com/success.txt", WantCode: http.StatusOK, WantBody: "success"},
}

// ProbesFor turns configured URLs into probes: ".../generate_204" expects
// 204, anything else 200 with "success" in the body (success.txt style).
func ProbesFor(urls []string) []Probe {
	if len(urls) == 0 {
		return DefaultProbes
	}
	out := make([]Probe, 0, len(urls))
	for _, u := range urls {
		if strings.HasSuffix(u, "generate_204") || strings.HasSuffix(u, "/gen_204") {
			out = append(out, Probe{URL: u, WantCode: http.StatusNoContent})
		} else {
			out = append(out, Probe{URL: u, WantCode: http.StatusOK, WantBody: "success"})
		}
	}
	return out
}

// Result of probing one WAN.
type Result struct {
	State     string    `json:"state"`
	PortalURL string    `json:"portal_url,omitempty"`
	Checked   time.Time `json:"checked"`
}

var (
	reMetaRefresh = regexp.MustCompile(`(?i)<meta[^>]+http-equiv=["']?refresh["']?[^>]*content=["']?\s*\d*\s*;\s*url=([^"'>\s]+)`)
	reJSLocation  = regexp.MustCompile(`(?i)(?:window\.)?location(?:\.href)?\s*=\s*["']([^"']+)["']`)
)

// Classify maps one probe answer to a state and, for a portal, its URL.
func Classify(p Probe, code int, location string, body []byte) (string, string) {
	if code == p.WantCode && (p.WantBody == "" || strings.Contains(string(body), p.WantBody)) {
		return Open, ""
	}
	if code >= 300 && code < 400 && location != "" {
		return Portal, resolve(p.URL, location)
	}
	if code == http.StatusNetworkAuthenticationRequired || code == http.StatusOK || code == http.StatusNoContent ||
		(code >= 300 && code < 400) || code == http.StatusForbidden || code == http.StatusUnauthorized ||
		code == http.StatusProxyAuthRequired {
		if m := reMetaRefresh.FindSubmatch(body); m != nil {
			return Portal, resolve(p.URL, string(m[1]))
		}
		if m := reJSLocation.FindSubmatch(body); m != nil {
			return Portal, resolve(p.URL, string(m[1]))
		}
		// the probe URL itself is plain http: opening it shows the portal
		return Portal, p.URL
	}
	// 5xx and friends from something in the path: treat as not open, but
	// not a portal we can name either
	return Offline, ""
}

func resolve(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

// Prober probes over one network device.
type Prober struct {
	Probes  []Probe
	Timeout time.Duration
	// Dial opens a TCP connection over dev; DNS lists the WAN's own DNS
	// servers (empty: the system resolver). Replaced in tests.
	Dial func(ctx context.Context, dev, addr string) (net.Conn, error)
	DNS  func(dev string) []string
}

// Check probes over dev. Conclusive answers (open or portal) stop early;
// all probes failing means offline.
func (p *Prober) Check(ctx context.Context, dev string) Result {
	probes := p.Probes
	if len(probes) == 0 {
		probes = DefaultProbes
	}
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 6 * time.Second
	}
	for _, pr := range probes {
		st, u, err := p.one(ctx, dev, pr, timeout)
		if err != nil {
			continue
		}
		if st != Offline {
			return Result{State: st, PortalURL: u, Checked: time.Now()}
		}
	}
	return Result{State: Offline, Checked: time.Now()}
}

func (p *Prober) one(ctx context.Context, dev string, pr Probe, timeout time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dial := p.Dial
	if dial == nil {
		dial = DialDevice
	}
	var servers []string
	if p.DNS != nil {
		servers = p.DNS(dev)
	}
	tr := &http.Transport{
		Proxy:             nil,
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if net.ParseIP(host) == nil {
				ips, err := lookup(ctx, dev, host, servers, dial)
				if err != nil || len(ips) == 0 {
					return nil, errors.Join(errors.New("dns"), err)
				}
				addr = net.JoinHostPort(ips[0], port)
			}
			return dial(ctx, dev, addr)
		},
	}
	defer tr.CloseIdleConnections()
	c := &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pr.URL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "tailbond-captive-check")
	resp, err := c.Do(req)
	if err != nil {
		return Offline, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	st, u := Classify(pr, resp.StatusCode, resp.Header.Get("Location"), body)
	return st, u, nil
}

// lookup resolves host through the WAN's own DNS servers (over dev), or the
// system resolver when none are known.
func lookup(ctx context.Context, dev, host string, servers []string, dial func(context.Context, string, string) (net.Conn, error)) ([]string, error) {
	if len(servers) == 0 {
		return net.DefaultResolver.LookupHost(ctx, host)
	}
	r := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		var last error
		for _, s := range servers {
			c, err := dialNet(ctx, dev, network, net.JoinHostPort(s, "53"))
			if err == nil {
				return c, nil
			}
			last = err
		}
		return nil, last
	}}
	ips, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			out = append([]string{ip.IP.String()}, out...)
		} else {
			out = append(out, ip.IP.String())
		}
	}
	return out, nil
}

// DialDevice opens a TCP connection that leaves through dev only.
func DialDevice(ctx context.Context, dev, addr string) (net.Conn, error) {
	return dialNet(ctx, dev, "tcp", addr)
}
