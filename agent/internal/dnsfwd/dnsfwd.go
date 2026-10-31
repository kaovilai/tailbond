// Package dnsfwd serves DNS on the tunnel IP: tailnet names go to MagicDNS
// (100.100.100.100), everything else to the system resolver.
package dnsfwd

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const MagicDNS = "100.100.100.100:53"

type Forwarder struct {
	Addr string // e.g. 10.77.0.1:53
	// Suffixes returns the zones routed to MagicDNS (called per query, cheap).
	Suffixes func() []string
	// TailscaleUp reports whether MagicDNS is reachable at all.
	TailscaleUp func() bool
	Log         func(string)
	// ResolvConf overrides /etc/resolv.conf (tests).
	ResolvConf string

	mu      sync.Mutex
	servers []*dns.Server
}

func (f *Forwarder) logf(s string) {
	if f.Log != nil {
		f.Log(s)
	}
}

// Run binds UDP+TCP, retrying until ctx ends (the tun address appears only
// after mqvpn starts).
func (f *Forwarder) Run(ctx context.Context) {
	h := dns.HandlerFunc(f.serve)
	for ctx.Err() == nil {
		pc, err := net.ListenPacket("udp", f.Addr)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}
		ln, err := net.Listen("tcp", f.Addr)
		if err != nil {
			pc.Close()
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}
		udp := &dns.Server{PacketConn: pc, Handler: h}
		tcp := &dns.Server{Listener: ln, Handler: h}
		f.mu.Lock()
		f.servers = []*dns.Server{udp, tcp}
		f.mu.Unlock()
		f.logf("dns forwarder listening on " + f.Addr)
		errc := make(chan error, 2)
		go func() { errc <- udp.ActivateAndServe() }()
		go func() { errc <- tcp.ActivateAndServe() }()
		select {
		case <-ctx.Done():
		case err := <-errc:
			f.logf("dns forwarder stopped: " + errString(err))
		}
		_ = udp.Shutdown()
		_ = tcp.Shutdown()
	}
}

func errString(err error) string {
	if err == nil {
		return "closed"
	}
	return err.Error()
}

// Route decides the upstream for a query name.
func (f *Forwarder) Route(name string) []string {
	name = strings.ToLower(dns.Fqdn(name))
	tsUp := f.TailscaleUp == nil || f.TailscaleUp()
	if tsUp {
		zones := []string{"ts.net."}
		if f.Suffixes != nil {
			zones = append(zones, f.Suffixes()...)
		}
		for _, z := range zones {
			z = strings.ToLower(dns.Fqdn(z))
			if z != "." && (name == z || strings.HasSuffix(name, "."+z)) {
				return []string{MagicDNS}
			}
		}
	}
	return f.system()
}

func (f *Forwarder) system() []string {
	path := f.ResolvConf
	if path == "" {
		path = "/etc/resolv.conf"
	}
	cc, err := dns.ClientConfigFromFile(path)
	var out []string
	if err == nil {
		for _, s := range cc.Servers {
			out = append(out, net.JoinHostPort(s, cc.Port))
		}
	}
	if len(out) == 0 {
		out = []string{"127.0.0.53:53"}
	}
	return out
}

func (f *Forwarder) serve(w dns.ResponseWriter, req *dns.Msg) {
	if len(req.Question) == 0 {
		m := new(dns.Msg)
		m.SetRcode(req, dns.RcodeFormatError)
		_ = w.WriteMsg(m)
		return
	}
	proto := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		proto = "tcp"
	}
	c := &dns.Client{Net: proto, Timeout: 4 * time.Second}
	for _, up := range f.Route(req.Question[0].Name) {
		resp, _, err := c.Exchange(req, up)
		if err == nil && resp != nil {
			_ = w.WriteMsg(resp)
			return
		}
	}
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeServerFailure)
	_ = w.WriteMsg(m)
}
