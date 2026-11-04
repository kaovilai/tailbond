package share

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestMDNSAnswerParses(t *testing.T) {
	a := Advert{ID: "3fa2c1d0e9", Label: "Pixel's 5G", KeyID: KeyID(testKey), Port: DefaultPort}
	q := new(dns.Msg)
	q.SetQuestion(serviceFQDN(""), dns.TypePTR)
	for _, legacy := range []bool{true, false} {
		r := answer(q, a, net.IPv4(192, 168, 8, 23), legacy)
		if r == nil {
			t.Fatal("no answer to a sharer query")
		}
		if legacy && r.Id != q.Id {
			t.Fatal("legacy unicast must echo the query id")
		}
		b, err := r.Pack()
		if err != nil {
			t.Fatal(err)
		}
		got := parseAnswer(b, &net.UDPAddr{IP: net.IPv4(10, 9, 9, 9), Port: 5353})
		if len(got) != 1 {
			t.Fatalf("found %+v", got)
		}
		f := got[0]
		if f.ID != a.ID || f.Label != a.Label || f.KeyID != a.KeyID || f.Via != "mdns" ||
			f.Addr.String() != "192.168.8.23:47077" {
			t.Fatalf("found %+v %v", f, f.Addr)
		}
	}
	// other services are ignored
	q2 := new(dns.Msg)
	q2.SetQuestion("_airplay._tcp.local.", dns.TypePTR)
	if answer(q2, a, net.IPv4(1, 2, 3, 4), true) != nil {
		t.Fatal("answered someone else's service")
	}
}

func TestMDNSParseFallsBackToSourceAddress(t *testing.T) {
	a := Advert{ID: "abc", Label: "iPhone", KeyID: "k", Port: 47077}
	q := new(dns.Msg)
	q.SetQuestion(serviceFQDN(""), dns.TypePTR)
	r := answer(q, a, nil, true) // no A record
	b, _ := r.Pack()
	got := parseAnswer(b, &net.UDPAddr{IP: net.IPv4(192, 168, 1, 50), Port: 5353})
	if len(got) != 1 || got[0].Addr.String() != "192.168.1.50:47077" {
		t.Fatalf("found %+v", got)
	}
}

// The phone hub's proxy uses the same records under _tailbond-proxy._tcp,
// and a Pair & Share browse doesn't pick it up (nor the other way round).
func TestMDNSProxyService(t *testing.T) {
	a := Advert{ID: "px-1", Label: "Pixel's hub", KeyID: "abcd", Port: 1888, Service: "_tailbond-proxy._tcp"}
	q := new(dns.Msg)
	q.SetQuestion("_tailbond-proxy._tcp.local.", dns.TypePTR)
	r := answer(q, a, net.IPv4(192, 168, 43, 1), true)
	if r == nil {
		t.Fatal("no answer to a proxy query")
	}
	b, _ := r.Pack()
	got := parseAnswerFor("_tailbond-proxy._tcp", b, nil)
	if len(got) != 1 || got[0].Addr.Port != 1888 || !got[0].Addr.IP.Equal(net.IPv4(192, 168, 43, 1)) || got[0].Label != "Pixel's hub" {
		t.Fatalf("proxy answer: %+v", got)
	}
	if n := len(parseAnswer(b, nil)); n != 0 {
		t.Errorf("a share browse found %d proxies", n)
	}
	qs := new(dns.Msg)
	qs.SetQuestion(serviceFQDN(""), dns.TypePTR)
	if answer(qs, a, net.IPv4(1, 2, 3, 4), true) != nil {
		t.Error("a proxy answered a share query")
	}
}
