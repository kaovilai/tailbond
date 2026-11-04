package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/share"
)

var update = flag.Bool("update", false, "rewrite testdata/vectors.json")

// echo: a TCP server that answers "re:" + what it got.
func echo(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				b := make([]byte, 64)
				n, _ := c.Read(b)
				c.Write(append([]byte("re:"), b[:n]...))
			}()
		}
	}()
	return ln.Addr().String()
}

func serve(t *testing.T, s *Server) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go s.Serve(ln)
	return ln.Addr().String()
}

// socksConnect runs a SOCKS5 handshake with user/pass (empty = no auth) to
// target and returns the connection, or the step that failed.
func socksConnect(t *testing.T, proxy, user, pass, target string) (net.Conn, string) {
	t.Helper()
	c, err := net.Dial("tcp", proxy)
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(5 * time.Second))
	m := byte(0)
	if user != "" {
		m = 2
	}
	c.Write([]byte{5, 1, m})
	var r [2]byte
	if _, err := io.ReadFull(c, r[:]); err != nil || r[1] != m {
		c.Close()
		return nil, "method"
	}
	if user != "" {
		b := []byte{1, byte(len(user))}
		b = append(append(b, user...), byte(len(pass)))
		b = append(b, pass...)
		c.Write(b)
		if _, err := io.ReadFull(c, r[:]); err != nil || r[1] != 0 {
			c.Close()
			return nil, "auth"
		}
	}
	host, port, _ := net.SplitHostPort(target)
	req := []byte{5, 1, 0, 3, byte(len(host))}
	req = append(req, host...)
	var p [2]byte
	n, _ := net.LookupPort("tcp", port)
	binary.BigEndian.PutUint16(p[:], uint16(n))
	c.Write(append(req, p[:]...))
	var rep [10]byte
	if _, err := io.ReadFull(c, rep[:]); err != nil || rep[1] != 0 {
		c.Close()
		return nil, "connect"
	}
	return c, ""
}

func roundTrip(t *testing.T, c net.Conn) string {
	t.Helper()
	c.Write([]byte("hello"))
	b := make([]byte, 64)
	n, _ := c.Read(b)
	return string(b[:n])
}

// phoneKey is the phone's own share key; the Mac ("mac") has a pass for it.
var phoneKey = []byte("0123456789abcdef0123456789abcdef")

func macLogin(exp uint64) (string, string) {
	return Credentials(share.PassKey(phoneKey, "mac", exp), "mac", exp)
}

func TestSocksWithCredentials(t *testing.T) {
	exp := uint64(time.Now().Add(24 * time.Hour).Unix())
	user, pass := macLogin(exp)
	s := &Server{Auth: Verifier(phoneKey, time.Now, nil)}
	p := serve(t, s)
	target := echo(t)
	c, step := socksConnect(t, p, user, pass, target)
	if c == nil {
		t.Fatalf("failed at %s", step)
	}
	if got := roundTrip(t, c); got != "re:hello" {
		t.Fatalf("got %q", got)
	}
	c.Close()
	if _, step := socksConnect(t, p, user, "wrong", target); step != "auth" {
		t.Errorf("wrong password: failed at %q, want auth", step)
	}
	if _, step := socksConnect(t, p, "", "", target); step != "method" {
		t.Errorf("no auth offered: failed at %q, want method", step)
	}
	// another hub's name with the Mac's password: the pass is per hub
	if _, step := socksConnect(t, p, "laptop/"+strconv.FormatUint(exp, 10), pass, target); step != "auth" {
		t.Errorf("borrowed password: %q", step)
	}
	// an expired pass
	ou, op := macLogin(uint64(time.Now().Add(-time.Minute).Unix()))
	if _, step := socksConnect(t, p, ou, op, target); step != "auth" {
		t.Errorf("expired pass: %q", step)
	}
	// another server's devices: a pass for a different phone key
	xu, xp := Credentials(share.PassKey([]byte("some other phone"), "mac", exp), "mac", exp)
	if _, step := socksConnect(t, p, xu, xp, target); step != "auth" {
		t.Errorf("other key: %q", step)
	}
	// only allowed hubs
	only := &Server{Auth: Verifier(phoneKey, time.Now, func(h string) bool { return h == "laptop" })}
	if _, step := socksConnect(t, serve(t, only), user, pass, target); step != "auth" {
		t.Errorf("hub not allowed: %q", step)
	}
	time.Sleep(50 * time.Millisecond)
	up, down, clients := s.Stats()
	if up < 5 || down < 8 || len(clients) != 1 || clients[0].Up < 5 {
		t.Errorf("stats: up %d down %d clients %+v", up, down, clients)
	}
}

func TestHTTPConnect(t *testing.T) {
	s := &Server{Auth: func(u, p string) bool { return u == "mac/9" && p == "pw" }}
	p := serve(t, s)
	target := echo(t)
	try := func(auth string) (net.Conn, string) {
		c, err := net.Dial("tcp", p)
		if err != nil {
			t.Fatal(err)
		}
		c.SetDeadline(time.Now().Add(5 * time.Second))
		req := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n"
		if auth != "" {
			req += "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(auth)) + "\r\n"
		}
		c.Write([]byte(req + "\r\n"))
		line, _ := bufio.NewReader(c).ReadString('\n')
		return c, line
	}
	if _, line := try(""); !strings.Contains(line, "407") {
		t.Errorf("no auth: %q", line)
	}
	c, line := try("mac/9:pw")
	if !strings.Contains(line, "200") {
		t.Fatalf("auth: %q", line)
	}
	// the reader above may hold nothing more: the tunnel starts after the blank line
	c.Write([]byte("hello"))
	b := make([]byte, 64)
	time.Sleep(50 * time.Millisecond)
	n, _ := c.Read(b)
	if !strings.Contains(string(b[:n]), "re:hello") {
		t.Errorf("through CONNECT: %q", b[:n])
	}
}

func TestOnlyAllowedSources(t *testing.T) {
	s := &Server{Allow: func(netip.Addr) bool { return false }}
	p := serve(t, s)
	if c, step := socksConnect(t, p, "", "", echo(t)); c != nil || step != "method" {
		t.Errorf("refused source got through (%q)", step)
	}
	for addr, want := range map[string]bool{
		"192.168.43.20": true, "172.20.10.3": true, "100.64.1.2": true, "fe80::1": true, "127.0.0.1": true,
		"10.1.2.3": true, "8.8.8.8": false, "2001:db8::1": false, "203.0.113.9": false,
	} {
		if AllowPrivate(netip.MustParseAddr(addr)) != want {
			t.Errorf("AllowPrivate(%s) != %v", addr, want)
		}
	}
}

func TestUDPAssociateRefused(t *testing.T) {
	p := serve(t, &Server{})
	c, err := net.Dial("tcp", p)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	c.Write([]byte{5, 1, 0})
	var r [2]byte
	io.ReadFull(c, r[:])
	c.Write([]byte{5, 3, 0, 1, 0, 0, 0, 0, 0, 0})
	var rep [10]byte
	if _, err := io.ReadFull(c, rep[:]); err != nil || rep[1] != 7 {
		t.Errorf("UDP ASSOCIATE reply %v (%v), want code 7", rep, err)
	}
}

func TestDialHook(t *testing.T) {
	var dialed string
	s := &Server{Dial: func(ctx context.Context, n, a string) (net.Conn, error) {
		dialed = a
		var d net.Dialer
		return d.DialContext(ctx, n, a)
	}}
	p := serve(t, s)
	target := echo(t)
	c, step := socksConnect(t, p, "", "", target)
	if c == nil {
		t.Fatal(step)
	}
	c.Close()
	if dialed != target {
		t.Errorf("dial hook got %q", dialed)
	}
}

// The Kotlin and Swift servers derive the same login (vectors.json).
func TestCredentialVectors(t *testing.T) {
	type vec struct {
		ShareKey string `json:"share_key"` // the phone's own key
		Hub      string `json:"hub"`
		Exp      uint64 `json:"exp"`
		PassKey  string `json:"pass_key"` // hex: what the server gives the hub
		User     string `json:"user"`
		Pass     string `json:"pass"`
	}
	in := []struct {
		key, hub string
		exp      uint64
	}{{"0123456789abcdef0123456789abcdef", "mac", 1793836800}, {"k", "slate7", 1}, {"a much longer share key, 64 bytes or so, as the server makes it.", "dana-macbook", 4102444800}}
	var vs []vec
	for _, x := range in {
		pk := share.PassKey([]byte(x.key), x.hub, x.exp)
		u, p := Credentials(pk, x.hub, x.exp)
		vs = append(vs, vec{x.key, x.hub, x.exp, hex.EncodeToString(pk), u, p})
		if !Verifier([]byte(x.key), func() time.Time { return time.Unix(0, 0) }, nil)(u, p) {
			t.Errorf("%s: the phone doesn't accept its own derivation", x.hub)
		}
	}
	path := "testdata/vectors.json"
	if *update {
		os.MkdirAll("testdata", 0o755)
		b, _ := json.MarshalIndent(map[string]any{"credentials": vs}, "", "  ")
		os.WriteFile(path, append(b, '\n'), 0o644)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Credentials []vec `json:"credentials"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Credentials) != len(vs) {
		t.Fatalf("vectors: %d, want %d (go test -update)", len(got.Credentials), len(vs))
	}
	for i := range vs {
		if got.Credentials[i] != vs[i] {
			t.Errorf("vector %d: %+v, want %+v (go test ./internal/proxy -update)", i, got.Credentials[i], vs[i])
		}
	}
}
