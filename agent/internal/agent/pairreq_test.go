package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/tailscale"
)

// whoisStub knows two tailnet devices.
func whoisStub(_ context.Context, ip string) (*tailscale.Whois, error) {
	switch ip {
	case "100.101.9.9":
		return &tailscale.Whois{NodeID: "nSLATE", Node: "slate7", DNSName: "slate7.tail1234.ts.net", Hostname: "GL-BE3600", OS: "linux", User: "alex@example.com"}, nil
	case "100.101.8.8":
		return &tailscale.Whois{NodeID: "nPIXEL", Node: "pixel-8", DNSName: "pixel-8.tail1234.ts.net", OS: "android", User: "sam@example.com"}, nil
	}
	return nil, errors.New("no such peer")
}

func pairServer(t *testing.T) *Agent {
	t.Helper()
	cfg := config.Default()
	cfg.Role = config.RoleServer
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.Server.PublicAddr = "home.example.net"
	cfg.ApplyDefaults()
	a := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{Version: "test"})
	a.whois = whoisStub
	return a
}

// req serves one request from remote address remote.
func req(t *testing.T, a *Agent, method, path, remote, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(method, path, rd)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	r.RemoteAddr = remote
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode %q: %v", w.Body, err)
	}
}

type created struct {
	ID, Token, Status, Name string
}

func TestHelloIsPublicAndMinimal(t *testing.T) {
	a := pairServer(t)
	a.store.Update(func(c *config.Config) error { c.UIPassword = "secret"; return nil })
	w := req(t, a, "GET", "/api/hello", "100.101.9.9:4000", "", nil)
	if w.Code != 200 {
		t.Fatalf("hello: %d %s", w.Code, w.Body)
	}
	var h map[string]any
	decode(t, w, &h)
	if h["tailbond"] != true || h["role"] != "server" || h["version"] != "test" || len(h) != 4 {
		t.Fatalf("hello = %v", h)
	}
	// everything else still needs the password
	if w := req(t, a, "GET", "/api/pair-requests", "100.101.9.9:4000", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("list without login: %d", w.Code)
	}
}

func TestPairRequestOnlyFromTailnet(t *testing.T) {
	a := pairServer(t)
	for _, remote := range []string{"192.168.8.20:5000", "10.77.0.2:5000", "[2001:db8::1]:5000"} {
		if w := req(t, a, "POST", "/api/pair-requests", remote, "", map[string]string{"name": "x"}); w.Code != http.StatusForbidden {
			t.Fatalf("%s: %d, want 403", remote, w.Code)
		}
	}
	// a tailnet address that whois doesn't know (spoofed on the LAN)
	if w := req(t, a, "POST", "/api/pair-requests", "100.99.99.99:5000", "", map[string]string{"name": "x"}); w.Code != http.StatusForbidden {
		t.Fatalf("unknown peer: %d, want 403", w.Code)
	}
}

func TestPairRequestApproveFlow(t *testing.T) {
	a := pairServer(t)
	const dev = "100.101.9.9:4000"
	w := req(t, a, "POST", "/api/pair-requests", dev, "", map[string]string{"name": "My Slate 7!", "kind": "router"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var c created
	decode(t, w, &c)
	if c.ID == "" || c.Token == "" || c.Status != "pending" || c.Name != "my-slate-7" {
		t.Fatalf("created = %+v", c)
	}
	// dedup: asking again returns the same request
	var c2 created
	decode(t, req(t, a, "POST", "/api/pair-requests", "100.101.9.9:4001", "", map[string]string{"name": "slate"}), &c2)
	if c2.ID != c.ID || c2.Token != c.Token {
		t.Fatalf("dedup: %+v vs %+v", c2, c)
	}
	// admin list shows who asked
	var list []map[string]any
	decode(t, req(t, a, "GET", "/api/pair-requests", "127.0.0.1:9000", "", nil), &list)
	if len(list) != 1 || list[0]["device"] != "slate7" || list[0]["user"] != "alex@example.com" || list[0]["kind"] != "router" {
		t.Fatalf("list = %v", list)
	}
	if _, leaked := list[0]["token"]; leaked {
		t.Fatal("token leaked in the admin list")
	}
	a.poll(context.Background())
	if pend, _ := a.Snapshot().PairRequests.([]pairRequest); len(pend) != 1 {
		t.Fatalf("status pair_requests = %#v", a.Snapshot().PairRequests)
	}
	// polling: token and source address required
	if w := req(t, a, "GET", "/api/pair-requests/"+c.ID, dev, "", nil); w.Code != 404 {
		t.Fatalf("poll without token: %d", w.Code)
	}
	if w := req(t, a, "GET", "/api/pair-requests/"+c.ID, "100.101.8.8:1", c.Token, nil); w.Code != 404 {
		t.Fatalf("poll from another device: %d", w.Code)
	}
	var st map[string]any
	decode(t, req(t, a, "GET", "/api/pair-requests/"+c.ID, dev, c.Token, nil), &st)
	if st["status"] != "pending" || st["uri"] != nil {
		t.Fatalf("pending poll = %v", st)
	}
	// the asking device can't approve itself
	if w := req(t, a, "POST", "/api/pair-requests/"+c.ID+"/approve", dev, "", map[string]string{}); w.Code != http.StatusForbidden {
		t.Fatalf("self-approve: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/pair-requests/"+c.ID+"/approve", "127.0.0.1:9000", "", map[string]string{"name": "slate7"}); w.Code != 200 {
		t.Fatalf("approve: %d %s", w.Code, w.Body)
	}
	decode(t, req(t, a, "GET", "/api/pair-requests/"+c.ID, dev, c.Token, nil), &st)
	uri, _ := st["uri"].(string)
	p, err := pairing.Decode(uri)
	if st["status"] != "approved" || err != nil || p.User != "slate7" || p.Server != "home.example.net:443" {
		t.Fatalf("approved poll = %v (%v)", st, err)
	}
	users := a.store.Get().Server.Users
	if len(users) != 1 || users[0].Name != "slate7" || users[0].TSNode != "nSLATE" || users[0].Key != p.Key {
		t.Fatalf("users = %+v", users)
	}
	// twice is a conflict
	if w := req(t, a, "POST", "/api/pair-requests/"+c.ID+"/approve", "127.0.0.1:9000", "", map[string]string{}); w.Code != http.StatusConflict {
		t.Fatalf("approve twice: %d", w.Code)
	}

	// the same router asks again later (reset): same credential, no duplicate
	a.mu.Lock()
	a.pairReqs.byID[c.ID].Expires = time.Now().Add(-time.Second)
	a.mu.Unlock()
	var c3 created
	decode(t, req(t, a, "POST", "/api/pair-requests", dev, "", map[string]string{"name": "other"}), &c3)
	if c3.ID == c.ID || c3.Name != "slate7" {
		t.Fatalf("re-pair request = %+v", c3)
	}
	req(t, a, "POST", "/api/pair-requests/"+c3.ID+"/approve", "127.0.0.1:9000", "", map[string]string{})
	decode(t, req(t, a, "GET", "/api/pair-requests/"+c3.ID, dev, c3.Token, nil), &st)
	p2, _ := pairing.Decode(st["uri"].(string))
	if p2 == nil || p2.Key != p.Key || len(a.store.Get().Server.Users) != 1 {
		t.Fatalf("re-pair issued a new credential: %v users=%d", st, len(a.store.Get().Server.Users))
	}
}

func TestPairRequestDeclineExpireRateLimit(t *testing.T) {
	a := pairServer(t)
	const phone = "100.101.8.8:1"
	var c created
	decode(t, req(t, a, "POST", "/api/pair-requests", phone, "", map[string]string{}), &c)
	if c.Name != "pixel-8" {
		t.Fatalf("name from whois: %q", c.Name)
	}
	req(t, a, "POST", "/api/pair-requests/"+c.ID+"/decline", "127.0.0.1:1", "", map[string]string{})
	var st map[string]any
	decode(t, req(t, a, "GET", "/api/pair-requests/"+c.ID, phone, c.Token, nil), &st)
	if st["status"] != "declined" || st["uri"] != nil {
		t.Fatalf("declined poll = %v", st)
	}
	// two more allowed in the window, then 429
	for i := 0; i < 2; i++ {
		var ci created
		decode(t, req(t, a, "POST", "/api/pair-requests", phone, "", map[string]string{}), &ci)
		a.mu.Lock()
		a.pairReqs.byID[ci.ID].Status = reqDeclined
		a.mu.Unlock()
	}
	if w := req(t, a, "POST", "/api/pair-requests", phone, "", map[string]string{}); w.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit: %d", w.Code)
	}
	// pending requests expire
	a2 := pairServer(t)
	decode(t, req(t, a2, "POST", "/api/pair-requests", phone, "", map[string]string{}), &c)
	a2.mu.Lock()
	a2.pairReqs.byID[c.ID].Expires = time.Now().Add(-time.Second)
	a2.mu.Unlock()
	decode(t, req(t, a2, "GET", "/api/pair-requests/"+c.ID, phone, c.Token, nil), &st)
	if st["status"] != "expired" {
		t.Fatalf("expired poll = %v", st)
	}
	if w := req(t, a2, "POST", "/api/pair-requests/"+c.ID+"/approve", "127.0.0.1:1", "", map[string]string{}); w.Code != http.StatusConflict {
		t.Fatalf("approve expired: %d", w.Code)
	}
}

func TestClientName(t *testing.T) {
	for in, want := range map[string]string{"My Slate 7!": "my-slate-7", "": "device", "GL-BE3600": "gl-be3600", "..x..": "x"} {
		if got := clientName(in); got != want {
			t.Errorf("clientName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := clientName("", "  ", "nas"); got != "nas" {
		t.Errorf("fallback = %q", got)
	}
}

// TestJoinFlow: a client-role agent discovers the server among its tailnet
// peers, asks to pair, and imports the pairing once the admin approves.
func TestJoinFlow(t *testing.T) {
	srv := pairServer(t)
	// the server sees the router's tailnet address
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "100.101.9.9:40000"
		srv.Handler().ServeHTTP(w, r)
	}))
	defer ts.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))

	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.MqvpnBin = "/bin/false"
	cfg.ApplyDefaults()
	cli := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	cli.apiPort = port
	defer cli.sup.Stop()

	// discovery with Tailscale off: nothing found, says so
	var d discoveryResult
	decode(t, req(t, cli, "GET", "/api/discover", "192.168.8.100:1", "", nil), &d)
	if d.TailscaleRunning || len(d.Servers) != 0 {
		t.Fatalf("discover = %+v", d)
	}

	w := req(t, cli, "POST", "/api/join", "192.168.8.100:1", "", map[string]string{"server": "127.0.0.1", "name": "slate7"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("join: %d %s", w.Code, w.Body)
	}
	waitFor(t, func() bool { return cli.joinView().State == joinWaiting })
	srv.mu.Lock()
	pend := srv.pairReqs.pending()
	srv.mu.Unlock()
	if len(pend) != 1 || pend[0].Name != "slate7" || pend[0].Kind != "router" {
		t.Fatalf("server pending = %+v", pend)
	}
	cli.poll(context.Background())
	if j, ok := cli.Snapshot().Join.(*joinState); !ok || j.State != joinWaiting {
		t.Fatalf("status join = %#v", cli.Snapshot().Join)
	}
	req(t, srv, "POST", "/api/pair-requests/"+pend[0].ID+"/approve", "127.0.0.1:1", "", map[string]string{})
	waitFor(t, func() bool { return cli.joinView().State == joinApproved })
	if !cli.paired() || !strings.HasPrefix(cli.store.Get().Client.PairingURI, "tailbond://pair") {
		t.Fatal("pairing not imported")
	}

	// unpair forgets it
	if w := req(t, cli, "DELETE", "/api/pair", "192.168.8.100:1", "", nil); w.Code != 200 || cli.paired() || cli.store.Get().Client.PairingURI != "" {
		t.Fatalf("unpair: %d paired=%v", w.Code, cli.paired())
	}
}

func TestJoinDeclinedAndUnreachable(t *testing.T) {
	srv := pairServer(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "100.101.9.9:40000"
		srv.Handler().ServeHTTP(w, r)
	}))
	defer ts.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.ApplyDefaults()
	cli := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	cli.apiPort = port
	req(t, cli, "POST", "/api/join", "", "", map[string]string{"server": "127.0.0.1"})
	waitFor(t, func() bool { return cli.joinView().State == joinWaiting })
	srv.mu.Lock()
	id := srv.pairReqs.pending()[0].ID
	srv.mu.Unlock()
	req(t, srv, "POST", "/api/pair-requests/"+id+"/decline", "127.0.0.1:1", "", map[string]string{})
	waitFor(t, func() bool { return cli.joinView().State == joinDeclined })

	cli.apiPort = "1" // nothing listens there
	req(t, cli, "POST", "/api/join", "", "", map[string]string{"server": "127.0.0.1"})
	waitFor(t, func() bool { return cli.joinView().State == joinError })
	if cli.joinView().Error != "unreachable" {
		t.Fatalf("join = %+v", cli.joinView())
	}
	// cancel resets
	req(t, cli, "DELETE", "/api/join", "", "", nil)
	if cli.joinView().State != joinIdle {
		t.Fatal("cancel did not reset")
	}
}

func TestDiscoverProbesPeers(t *testing.T) {
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.StateDir = t.TempDir()
	cfg.ApplyDefaults()
	cli := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	cli.ts = tailscale.NewFake(&tailscale.Info{Installed: true, Running: true, Self: "slate7", PeerList: []tailscale.Node{
		{HostName: "homebox", DNSName: "homebox.tail1234.ts.net", TailscaleIPs: []string{"100.101.1.1", "fd7a:115c:a1e0::1"}, Online: true},
		{HostName: "nas", DNSName: "nas.tail1234.ts.net", TailscaleIPs: []string{"100.101.2.2"}, Online: true},
		{HostName: "old", DNSName: "old.tail1234.ts.net", TailscaleIPs: []string{"100.101.3.3"}, Online: false},
	}})
	probed := map[string]bool{}
	var mu sync.Mutex
	cli.hello = func(_ context.Context, addr string) (*helloInfo, error) {
		mu.Lock()
		probed[addr] = true
		mu.Unlock()
		if addr == "100.101.1.1:8077" {
			return &helloInfo{Tailbond: true, Name: "homebox", Role: "server", Version: "1"}, nil
		}
		return nil, errors.New("refused")
	}
	d := cli.discover(context.Background(), true)
	if len(d.Servers) != 1 || d.Servers[0].Name != "homebox" || d.Servers[0].IP != "100.101.1.1" || d.Checked != 2 || probed["100.101.3.3:8077"] {
		t.Fatalf("discover = %+v probed=%v", d, probed)
	}
	if addr, label := cli.resolveServer(context.Background(), "HomeBox"); addr != "100.101.1.1:8077" || label != "homebox" {
		t.Fatalf("resolve = %s %s", addr, label)
	}
	if addr, _ := cli.resolveServer(context.Background(), "homebox.tail1234.ts.net"); addr != "100.101.1.1:8077" {
		t.Fatalf("resolve dns = %s", addr)
	}
}

func init() { joinPollEvery = 50 * time.Millisecond }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out")
}
