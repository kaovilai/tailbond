// Package proxy is the phone hub's proxy (docs/phone-hub.md): a SOCKS5
// server (RFC 1928, username/password RFC 1929) and an HTTP CONNECT proxy
// on one port. A phone runs it so that a laptop's apps reach the internet
// through the phone's tailbond tunnel: the proxy's upstream sockets are the
// phone's own app traffic, so they go into its VPN.
//
// This Go implementation is the reference: tb-sharesim runs it as the
// stand-in hub in the macOS end-to-end test, and the Android (ProxyServer.kt)
// and iPhone (TailbondShareKit) servers follow it, pinned by
// testdata/vectors.json (the credentials).
//
// Auth (Pair & Share's keys, internal/share): every paired device has its
// own share key, and a hub (the Mac) gets from the home server a pass for
// each other device, PassKey(that device's key, hub name, expiry). The
// proxy's user is "<hub name>/<expiry>" and its password is derived from
// the pass (Credentials); the phone derives the same pass from its own key
// and checks it offline (Verifier). So a Mac paired with the same server
// can use a phone's hub with no setup, a removed device's access ends when
// its last pass expires, and nobody else gets in. Only private, link-local
// and loopback sources are accepted at all.
package proxy

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/share"
)

// Service is the mDNS service type a hub advertises.
const Service = "_tailbond-proxy._tcp"

// DefaultPort is the proxy's port (the phone's).
const DefaultPort = 1888

// Credentials is a hub's proxy login for one phone, from the pass the home
// server gave the hub for that phone: user "<hubID>/<exp>", password the
// first 16 characters of base32(HMAC-SHA256(pass, "tbp1-proxy")), lower
// case, no padding.
func Credentials(pass []byte, hubID string, exp uint64) (user, password string) {
	m := hmac.New(sha256.New, pass)
	m.Write([]byte("tbp1-proxy"))
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(m.Sum(nil))
	return hubID + "/" + strconv.FormatUint(exp, 10), strings.ToLower(enc[:16])
}

// BLELogin is the Mac's login for a phone's BLE control channel
// (docs/phone-hub.md, "One click from the Mac"), from the same pass as
// Credentials but with its own label, so a leak of one credential is not
// the other: user "<hubID>/<exp>", key the first 16 characters of
// base32(HMAC-SHA256(pass, "tbp1-ble")), lower case, no padding. The phone
// checks a challenge-response HMAC over it (BleVerify on the Android side);
// pinned by testdata/vectors.json like the proxy credentials.
func BLELogin(pass []byte, hubID string, exp uint64) (user, key string) {
	m := hmac.New(sha256.New, pass)
	m.Write([]byte("tbp1-ble"))
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(m.Sum(nil))
	return hubID + "/" + strconv.FormatUint(exp, 10), strings.ToLower(enc[:16])
}

// Verifier checks a login on the phone: the pass derived from the phone's
// own share key (share.PassKey) for the hub and expiry in user, not
// expired. allowed (optional) limits the hubs, by name.
func Verifier(ownKey []byte, now func() time.Time, allowed func(hub string) bool) func(user, pass string) bool {
	return func(user, pass string) bool {
		hub, e, ok := strings.Cut(user, "/")
		if !ok || hub == "" {
			return false
		}
		exp, err := strconv.ParseUint(e, 10, 64)
		if err != nil || int64(exp) < now().Unix() {
			return false
		}
		if allowed != nil && !allowed(hub) {
			return false
		}
		_, want := Credentials(share.PassKey(ownKey, hub, exp), hub, exp)
		return hmac.Equal([]byte(pass), []byte(want))
	}
}

// Server is one proxy. Zero values: no auth (tests), any private source,
// net.Dialer.
type Server struct {
	// Auth, when set, is required (SOCKS5 method 0x02, HTTP
	// Proxy-Authorization: Basic) and checks a login (Verifier).
	Auth func(user, pass string) bool
	// Dial opens the upstream connection; nil = net.Dialer. On a phone the
	// default (the app's own sockets) is what puts the traffic into the
	// VPN tunnel.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
	// Allow says whether a client may connect at all; nil = private,
	// link-local and loopback addresses only.
	Allow func(remote netip.Addr) bool
	Logf  func(format string, args ...any)

	up, down atomic.Uint64
	mu       sync.Mutex
	clients  map[string]*Client
}

// Client is one connected device, by source address.
type Client struct {
	Addr     string    `json:"addr"`
	Conns    int       `json:"conns"`
	Up       uint64    `json:"up_bytes"`
	Down     uint64    `json:"down_bytes"`
	LastSeen time.Time `json:"last_seen"`
}

// Stats is the total and per-client byte counts.
func (s *Server) Stats() (up, down uint64, clients []Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clients {
		clients = append(clients, *c)
	}
	return s.up.Load(), s.down.Load(), clients
}

// AllowPrivate is the default Allow: RFC 1918, CGNAT, unique-local,
// link-local and loopback sources.
func AllowPrivate(a netip.Addr) bool {
	a = a.Unmap()
	return a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() ||
		(a.Is4() && netip.MustParsePrefix("100.64.0.0/10").Contains(a))
}

func (s *Server) logf(f string, a ...any) {
	if s.Logf != nil {
		s.Logf(f, a...)
	}
}

// Serve accepts connections until ln is closed.
func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handle(c)
	}
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	ap, err := netip.ParseAddrPort(c.RemoteAddr().String())
	allow := s.Allow
	if allow == nil {
		allow = AllowPrivate
	}
	if err != nil || !allow(ap.Addr()) {
		return
	}
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	br := bufio.NewReader(c)
	first, err := br.Peek(1)
	if err != nil {
		return
	}
	var target string
	var reply func(ok bool) error
	if first[0] == 5 {
		target, reply, err = s.socks(br, c)
	} else {
		target, reply, err = s.httpConnect(br, c)
	}
	if err != nil {
		// a client that only checks the password (a hub probe) hangs up here
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			s.logf("proxy: %s: %v", ap, err)
		}
		return
	}
	up, err := s.dial(target)
	if err != nil {
		s.logf("proxy: %s → %s: %v", ap, target, err)
		_ = reply(false)
		return
	}
	defer up.Close()
	if reply(true) != nil {
		return
	}
	_ = c.SetDeadline(time.Time{})
	cl := s.track(ap.Addr().String(), 1)
	defer s.track(ap.Addr().String(), -1)
	done := make(chan struct{}, 2)
	// counted as they go: the phone's notification shows live totals
	go func() {
		_, _ = io.Copy(counter{up, func(n int) { s.count(cl, uint64(n), 0) }}, br)
		closeWrite(up)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(counter{c, func(n int) { s.count(cl, 0, uint64(n)) }}, up)
		closeWrite(c)
		done <- struct{}{}
	}()
	<-done
	<-done
}

// counter counts what is written through it.
type counter struct {
	w   io.Writer
	add func(int)
}

func (c counter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.add(n)
	return n, err
}

func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		_ = cw.CloseWrite()
	}
}

func (s *Server) dial(target string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if s.Dial != nil {
		return s.Dial(ctx, "tcp", target)
	}
	var d net.Dialer
	return d.DialContext(ctx, "tcp", target)
}

func (s *Server) track(addr string, d int) *Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clients == nil {
		s.clients = map[string]*Client{}
	}
	c := s.clients[addr]
	if c == nil {
		c = &Client{Addr: addr}
		s.clients[addr] = c
	}
	c.Conns += d
	c.LastSeen = time.Now()
	return c
}

func (s *Server) count(c *Client, up, down uint64) {
	s.up.Add(up)
	s.down.Add(down)
	s.mu.Lock()
	c.Up += up
	c.Down += down
	s.mu.Unlock()
}

// ---- SOCKS5 ----

var errAuth = errors.New("bad credentials")

func (s *Server) socks(br *bufio.Reader, w io.Writer) (string, func(bool) error, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return "", nil, err
	}
	methods := make([]byte, hdr[1])
	if _, err := io.ReadFull(br, methods); err != nil {
		return "", nil, err
	}
	want := byte(0x00)
	if s.Auth != nil {
		want = 0x02
	}
	if !contains(methods, want) {
		_, _ = w.Write([]byte{5, 0xff})
		return "", nil, errors.New("no acceptable auth method")
	}
	if _, err := w.Write([]byte{5, want}); err != nil {
		return "", nil, err
	}
	if want == 0x02 {
		u, p, err := readUserPass(br)
		if err != nil {
			return "", nil, err
		}
		ok := s.Auth(u, p)
		st := byte(0)
		if !ok {
			st = 1
		}
		if _, err := w.Write([]byte{1, st}); err != nil || !ok {
			return "", nil, errAuth
		}
	}
	var req [4]byte
	if _, err := io.ReadFull(br, req[:]); err != nil {
		return "", nil, err
	}
	if req[0] != 5 {
		return "", nil, errors.New("bad version")
	}
	host, err := readAddr(br, req[3])
	if err != nil {
		return "", nil, err
	}
	var pb [2]byte
	if _, err := io.ReadFull(br, pb[:]); err != nil {
		return "", nil, err
	}
	target := net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(pb[:]))))
	if req[1] != 1 { // CONNECT only (UDP ASSOCIATE: docs/phone-hub.md)
		_, _ = w.Write([]byte{5, 7, 0, 1, 0, 0, 0, 0, 0, 0})
		return "", nil, fmt.Errorf("command %d not supported", req[1])
	}
	return target, func(ok bool) error {
		code := byte(0)
		if !ok {
			code = 5 // connection refused
		}
		_, err := w.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
		return err
	}, nil
}

func contains(b []byte, x byte) bool {
	for _, v := range b {
		if v == x {
			return true
		}
	}
	return false
}

func readUserPass(br *bufio.Reader) (string, string, error) {
	ver, err := br.ReadByte()
	if err != nil || ver != 1 {
		return "", "", errors.New("bad auth version")
	}
	read := func() (string, error) {
		n, err := br.ReadByte()
		if err != nil {
			return "", err
		}
		b := make([]byte, n)
		_, err = io.ReadFull(br, b)
		return string(b), err
	}
	u, err := read()
	if err != nil {
		return "", "", err
	}
	p, err := read()
	return u, p, err
}

func readAddr(br *bufio.Reader, atyp byte) (string, error) {
	switch atyp {
	case 1:
		var b [4]byte
		if _, err := io.ReadFull(br, b[:]); err != nil {
			return "", err
		}
		return netip.AddrFrom4(b).String(), nil
	case 4:
		var b [16]byte
		if _, err := io.ReadFull(br, b[:]); err != nil {
			return "", err
		}
		return netip.AddrFrom16(b).String(), nil
	case 3:
		n, err := br.ReadByte()
		if err != nil {
			return "", err
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(br, b); err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", fmt.Errorf("address type %d", atyp)
}

// ---- HTTP CONNECT ----

func (s *Server) httpConnect(br *bufio.Reader, w io.Writer) (string, func(bool) error, error) {
	req, err := http.ReadRequest(br)
	if err != nil {
		return "", nil, err
	}
	if req.Method != http.MethodConnect {
		_, _ = io.WriteString(w, "HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\n\r\n")
		return "", nil, errors.New("only CONNECT")
	}
	if s.Auth != nil {
		u, p, ok := proxyBasicAuth(req.Header.Get("Proxy-Authorization"))
		if !ok || !s.Auth(u, p) {
			_, _ = io.WriteString(w, "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"tailbond\"\r\nContent-Length: 0\r\n\r\n")
			return "", nil, errAuth
		}
	}
	return req.Host, func(ok bool) error {
		st := "200 Connection established"
		if !ok {
			st = "502 Bad Gateway"
		}
		_, err := io.WriteString(w, "HTTP/1.1 "+st+"\r\n\r\n")
		return err
	}, nil
}

func proxyBasicAuth(h string) (string, string, bool) {
	r := http.Request{Header: http.Header{"Authorization": {h}}}
	return r.BasicAuth()
}
