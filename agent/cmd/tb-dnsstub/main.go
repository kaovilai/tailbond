// Command tb-dnsstub is a tiny DNS server for tests: it answers A queries
// for the zones it's given and refuses the rest. scripts/e2e-macos.sh runs
// it as the stand-in for a work VPN's DNS (an /etc/resolver/corp.example
// that points at it), to check tailbond leaves that resolver alone.
//
//	tb-dnsstub -listen 127.0.0.1:5390 -a corp.example=10.92.0.5
package main

import (
	"flag"
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
)

type answers map[string]net.IP

func (a answers) String() string { return "" }

func (a answers) Set(s string) error {
	zone, ip, ok := strings.Cut(s, "=")
	if !ok || net.ParseIP(ip) == nil {
		return &net.ParseError{Type: "zone=IPv4", Text: s}
	}
	a[dns.Fqdn(strings.ToLower(zone))] = net.ParseIP(ip).To4()
	return nil
}

func main() {
	listen := flag.String("listen", "127.0.0.1:5390", "UDP address to answer on")
	zones := answers{}
	flag.Var(zones, "a", "zone=IPv4: A answer for the zone and every name under it (repeatable)")
	flag.Parse()
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		for _, q := range r.Question {
			name := strings.ToLower(q.Name)
			for zone, ip := range zones {
				if name == zone || strings.HasSuffix(name, "."+zone) {
					if q.Qtype == dns.TypeA {
						m.Answer = append(m.Answer, &dns.A{Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 5}, A: ip})
					}
				}
			}
		}
		if len(m.Answer) == 0 && len(r.Question) > 0 {
			m.Rcode = dns.RcodeNameError
		}
		log.Printf("%s %s -> %d answer(s)", w.RemoteAddr(), r.Question[0].Name, len(m.Answer))
		_ = w.WriteMsg(m)
	})
	log.Printf("answering %d zone(s) on %s", len(zones), *listen)
	log.Fatal((&dns.Server{Addr: *listen, Net: "udp"}).ListenAndServe())
}
