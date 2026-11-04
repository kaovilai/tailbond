package share

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
)

var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

// serviceFQDN is the browse name.
const serviceFQDN = Service + ".local."

// Found is a sharer a hub can use.
type Found struct {
	ID    string       `json:"id"`
	Label string       `json:"label"`
	KeyID string       `json:"key_id"`
	Addr  *net.UDPAddr `json:"-"`
	// How the hub learned of it: "mdns" or "server" (the home server's list).
	Via string `json:"via"`
}

// Advert is what a sharer announces over mDNS.
type Advert struct {
	ID, Label, KeyID string
	Port             int
}

func txt(a Advert) []string {
	return []string{"v=1", "id=" + a.ID, "k=" + a.KeyID, "n=" + a.Label}
}

func instanceName(a Advert) string {
	// DNS-SD instance names are free text; keep them unique per device
	n := a.Label
	if n == "" {
		n = "tailbond share"
	}
	return n + " (" + shortID(a.ID) + ")"
}

func shortID(id string) string {
	if len(id) > 6 {
		return id[:6]
	}
	return id
}

// Browse asks for sharers on each interface (one mDNS query, legacy
// unicast: answers come straight back to our port) and collects answers
// for wait.
func Browse(ctx context.Context, ifaces []net.Interface, wait time.Duration) ([]Found, error) {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	defer c.Close()
	p := ipv4.NewPacketConn(c)
	q := new(dns.Msg)
	q.SetQuestion(serviceFQDN, dns.TypePTR)
	q.RecursionDesired = false
	b, err := q.Pack()
	if err != nil {
		return nil, err
	}
	sent := 0
	for i := range ifaces {
		if p.SetMulticastInterface(&ifaces[i]) != nil {
			continue
		}
		if _, err := p.WriteTo(b, nil, mdnsGroup); err == nil {
			sent++
		}
	}
	if sent == 0 {
		return nil, errors.New("share: no interface to ask on")
	}
	deadline := time.Now().Add(wait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = c.SetReadDeadline(deadline)
	byID := map[string]Found{}
	buf := make([]byte, 9000)
	for {
		n, from, err := c.ReadFromUDP(buf)
		if err != nil {
			break
		}
		for _, f := range parseAnswer(buf[:n], from) {
			byID[f.ID] = f
		}
	}
	out := make([]Found, 0, len(byID))
	for _, f := range byID {
		out = append(out, f)
	}
	return out, nil
}

// parseAnswer reads sharers from an mDNS response. The address is the A
// record when there is one, else the responder's source address.
func parseAnswer(b []byte, from *net.UDPAddr) []Found {
	var m dns.Msg
	if m.Unpack(b) != nil || !m.Response {
		return nil
	}
	rrs := append(append([]dns.RR{}, m.Answer...), m.Extra...)
	type inst struct {
		port   int
		target string
		kv     map[string]string
	}
	insts := map[string]*inst{}
	addrs := map[string]net.IP{}
	for _, rr := range rrs {
		switch r := rr.(type) {
		case *dns.PTR:
			if strings.EqualFold(r.Hdr.Name, serviceFQDN) {
				if insts[strings.ToLower(r.Ptr)] == nil {
					insts[strings.ToLower(r.Ptr)] = &inst{kv: map[string]string{}}
				}
			}
		case *dns.A:
			addrs[strings.ToLower(r.Hdr.Name)] = r.A
		}
	}
	for _, rr := range rrs {
		in := insts[strings.ToLower(rr.Header().Name)]
		if in == nil {
			continue
		}
		switch r := rr.(type) {
		case *dns.SRV:
			in.port, in.target = int(r.Port), strings.ToLower(r.Target)
		case *dns.TXT:
			for _, kv := range r.Txt {
				k, v, _ := strings.Cut(kv, "=")
				in.kv[strings.ToLower(k)] = v
			}
		}
	}
	var out []Found
	for _, in := range insts {
		if in.port == 0 || in.kv["id"] == "" || in.kv["v"] != "1" {
			continue
		}
		ip := addrs[in.target]
		if ip == nil && from != nil {
			ip = from.IP
		}
		if ip == nil {
			continue
		}
		out = append(out, Found{ID: in.kv["id"], Label: in.kv["n"], KeyID: in.kv["k"], Addr: &net.UDPAddr{IP: ip, Port: in.port}, Via: "mdns"})
	}
	return out
}

// answer builds the response to q for a, with ip as the host address;
// nil when q doesn't ask for sharers.
func answer(q *dns.Msg, a Advert, ip net.IP, legacy bool) *dns.Msg {
	asks := false
	for _, x := range q.Question {
		if strings.EqualFold(x.Name, serviceFQDN) && (x.Qtype == dns.TypePTR || x.Qtype == dns.TypeANY) {
			asks = true
		}
	}
	if !asks {
		return nil
	}
	inst := instanceName(a) + "." + serviceFQDN
	host := "tailbond-share-" + shortID(a.ID) + ".local."
	r := new(dns.Msg)
	r.Response, r.Authoritative = true, true
	if legacy {
		// legacy unicast: echo the id and the question (RFC 6762 §6.7)
		r.Id = q.Id
		r.Question = q.Question
	}
	ttl := uint32(120)
	if legacy {
		ttl = 10
	}
	r.Answer = []dns.RR{&dns.PTR{Hdr: dns.RR_Header{Name: serviceFQDN, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl}, Ptr: inst}}
	r.Extra = []dns.RR{
		&dns.SRV{Hdr: dns.RR_Header{Name: inst, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl}, Port: uint16(a.Port), Target: host},
		&dns.TXT{Hdr: dns.RR_Header{Name: inst, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl}, Txt: txt(a)},
	}
	if ip4 := ip.To4(); ip4 != nil {
		r.Extra = append(r.Extra, &dns.A{Hdr: dns.RR_Header{Name: host, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}, A: ip4})
	}
	return r
}

// Advertise answers mDNS queries for sharers on ifaces (joined to the mDNS
// group on each) until ctx ends. Other mDNS responders on the host (avahi)
// keep working: the socket shares port 5353.
func Advertise(ctx context.Context, ifaces []net.Interface, a Advert) error {
	lc := net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error {
		var serr error
		err := c.Control(func(fd uintptr) { serr = reusePort(int(fd)) })
		if err != nil {
			return err
		}
		return serr
	}}
	pc, err := lc.ListenPacket(ctx, "udp4", "0.0.0.0:5353")
	if err != nil {
		return err
	}
	defer pc.Close()
	p := ipv4.NewPacketConn(pc)
	_ = p.SetControlMessage(ipv4.FlagInterface, true)
	joined := 0
	for i := range ifaces {
		if p.JoinGroup(&ifaces[i], mdnsGroup) == nil {
			joined++
		}
	}
	if joined == 0 {
		return errors.New("share: couldn't join mDNS on any interface")
	}
	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()
	buf := make([]byte, 9000)
	for {
		n, cm, from, err := p.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		var q dns.Msg
		if q.Unpack(buf[:n]) != nil || q.Response {
			continue
		}
		src, _ := from.(*net.UDPAddr)
		if src == nil || cm == nil {
			continue
		}
		ifi, err := net.InterfaceByIndex(cm.IfIndex)
		if err != nil {
			continue
		}
		ip := ifaceIPv4In(ifi, src.IP)
		if ip == nil {
			continue
		}
		legacy := src.Port != 5353
		unicast := legacy
		for _, x := range q.Question {
			if x.Qclass&(1<<15) != 0 { // QU bit
				unicast = true
			}
		}
		r := answer(&q, a, ip, legacy)
		if r == nil {
			continue
		}
		out, err := r.Pack()
		if err != nil {
			continue
		}
		if unicast {
			_, _ = p.WriteTo(out, nil, src)
		} else {
			_, _ = p.WriteTo(out, &ipv4.ControlMessage{IfIndex: ifi.Index}, mdnsGroup)
		}
	}
}

// ifaceIPv4In is ifi's IPv4 address on peer's subnet, else its first IPv4.
func ifaceIPv4In(ifi *net.Interface, peer net.IP) net.IP {
	addrs, _ := ifi.Addrs()
	var first net.IP
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.To4() == nil {
			continue
		}
		if first == nil {
			first = n.IP.To4()
		}
		if n.Contains(peer) {
			return n.IP.To4()
		}
	}
	return first
}

// MulticastIfaces are the up, multicast-capable, non-loopback interfaces
// with an IPv4 address, minus skip (the tunnel, say).
func MulticastIfaces(skip ...string) []net.Interface {
	all, _ := net.Interfaces()
	var out []net.Interface
	for _, ifi := range all {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagMulticast == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		skipped := false
		for _, s := range skip {
			if ifi.Name == s {
				skipped = true
			}
		}
		if skipped {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
				out = append(out, ifi)
				break
			}
		}
	}
	return out
}
