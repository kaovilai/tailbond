package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/config"
)

// pairing hits mqvpn's control API, which no test runs: keep the
// unavailable-retry window short so the suite doesn't pay for it.
func TestMain(m *testing.M) {
	addUserRetry = 30 * time.Millisecond
	m.Run()
}

// fakeCtl is a stand-in for mqvpn's JSON control API (docs/control-api.md):
// one newline-terminated request per TCP connection, one JSON response.
// The port is reserved at construction and the listener only comes up when
// startAfter fires, modelling mqvpn whose control socket binds after the
// agent's HTTP API.
type fakeCtl struct {
	mu         sync.Mutex
	addr       string
	failNext   int           // respond with an error to the next N requests
	noRespond  bool          // accept, read, close: an answer that never comes
	delayFirst time.Duration // hold the first response, after reading it
	got        []string
	up         bool
	ln         net.Listener
}

func newFakeCtl(t *testing.T) *fakeCtl {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeCtl{addr: ln.Addr().String()}
	ln.Close()
	return f
}

func (f *fakeCtl) startAfter(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		ln, err := net.Listen("tcp", f.addr)
		if err != nil {
			return
		}
		f.mu.Lock()
		f.ln = ln
		f.up = true
		f.mu.Unlock()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c) // one delayed answer must not block the others
		}
	}()
}

// waitBound blocks until the listener is up: a test that wants a refused
// answer must not race the bind, or the first attempt reads as a dial
// failure instead of the refusal it is there to exercise.
func (f *fakeCtl) waitBound(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		up := f.up
		f.mu.Unlock()
		if up {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fake control API never started")
}

func (f *fakeCtl) serve(c net.Conn) {
	defer c.Close()
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return
	}
	var req map[string]any
	_ = json.Unmarshal([]byte(line), &req)
	cmd, _ := req["cmd"].(string)
	f.mu.Lock()
	f.got = append(f.got, cmd)
	fail := false
	if f.failNext > 0 {
		f.failNext--
		fail = true
	}
	noRespond := f.noRespond
	delay := time.Duration(0)
	if len(f.got) == 1 {
		delay = f.delayFirst
	}
	f.mu.Unlock()
	if noRespond {
		return // the answer never comes: the caller reads EOF
	}
	if delay > 0 {
		// the registration is made (this request is served) but the
		// caller has not seen the answer yet
		time.Sleep(delay)
	}
	resp := map[string]any{"ok": !fail}
	if fail {
		resp["error"] = "too many clients"
	}
	b, _ := json.Marshal(resp)
	_, _ = c.Write(append(b, '\n'))
}

func (f *fakeCtl) commands() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.got...)
}

// pairServerCtl is pairServer pointed at a chosen control address.
func pairServerCtl(t *testing.T, ctlAddr string) *Agent {
	t.Helper()
	cfg := config.Default()
	cfg.Role = config.RoleServer
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.Server.PublicAddr = "home.example.net"
	cfg.ControlAddr = ctlAddr
	cfg.ApplyDefaults()
	a := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{Version: "test"})
	a.whois = whoisStub
	return a
}

// Pairing must survive the control socket binding late: the agent's HTTP
// API is up before mqvpn finishes starting, and the retry has to bridge
// that gap or the URI it hands out can never authenticate (the CGNAT relay
// e2e caught this as a CI flake: every connection rejected with "invalid
// or missing PSK").
func TestPairingWaitsForMqvpnControlAPI(t *testing.T) {
	f := newFakeCtl(t)
	f.startAfter(300 * time.Millisecond)
	srv := pairServerCtl(t, f.addr)
	addUserRetry = 5 * time.Second
	defer func() { addUserRetry = 30 * time.Millisecond }()

	res, err := srv.addClient(context.Background(), "pixel")
	if err != nil {
		t.Fatalf("addClient during control-API startup: %v", err)
	}
	if res.Name != "pixel" || res.URI == "" {
		t.Fatalf("pairing result %+v", res)
	}
	cmds := f.commands()
	if len(cmds) == 0 || cmds[0] != "add_user" {
		t.Fatalf("control API saw %v, want add_user", cmds)
	}
}

// A definitive refusal must fail the pairing and roll the entry back,
// instead of handing out a URI mqvpn will never authenticate.
func TestPairingFailsWhenMqvpnRefuses(t *testing.T) {
	f := newFakeCtl(t)
	f.failNext = 1 // refuse the first add, accept the re-pair
	f.startAfter(0)
	f.waitBound(t)
	srv := pairServerCtl(t, f.addr)

	if _, err := srv.addClient(context.Background(), "pixel"); err == nil {
		t.Fatal("addClient succeeded despite mqvpn refusing the user")
	}
	if got := srv.store.Get().Server.Users; len(got) != 0 {
		t.Fatalf("store kept the refused user: %+v", got)
	}
	// the name is free again
	if _, err := srv.addClient(context.Background(), "pixel"); err != nil {
		t.Fatalf("re-pair after rollback: %v", err)
	}
}

// A control API that accepts the request but never answers is not a
// refusal: mqvpn may have registered the user, so the credential stays
// (deleting a possibly-working one is worse than one that may be dead).
func TestPairingKeepsClientOnUnclearControlAnswer(t *testing.T) {
	f := newFakeCtl(t)
	f.noRespond = true
	f.startAfter(0)
	f.waitBound(t)
	srv := pairServerCtl(t, f.addr)

	res, err := srv.addClient(context.Background(), "pixel")
	if err != nil {
		t.Fatalf("addClient with an unclear control answer: %v", err)
	}
	if res.URI == "" {
		t.Fatal("no pairing URI")
	}
	if got := srv.store.Get().Server.Users; len(got) != 1 || got[0].Name != "pixel" {
		t.Fatalf("store users %+v", got)
	}
}

// A delete that lands while the retry waits for the control socket must
// cancel the pairing: no URI for a revoked client, and a key the retry
// registered in mqvpn anyway is taken back out.
func TestPairingDeletedDuringRetry(t *testing.T) {
	f := newFakeCtl(t)
	f.startAfter(700 * time.Millisecond)
	srv := pairServerCtl(t, f.addr)
	addUserRetry = 5 * time.Second
	defer func() { addUserRetry = 30 * time.Millisecond }()

	errCh := make(chan error, 1)
	go func() {
		_, err := srv.addClient(context.Background(), "pixel")
		errCh <- err
	}()
	// the delete lands mid-retry, while the control socket is still down
	time.Sleep(200 * time.Millisecond)
	if err := srv.removeClient(context.Background(), "pixel"); err != nil {
		t.Fatalf("removeClient: %v", err)
	}
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "removed while pairing") {
			t.Fatalf("addClient during delete: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("addClient never returned")
	}
	cmds := f.commands()
	has := func(want string) bool {
		for _, c := range cmds {
			if c == want {
				return true
			}
		}
		return false
	}
	// the retry registered the key once the socket came up; the check
	// then removed it again
	if !has("add_user") || !has("remove_user") {
		t.Fatalf("control API saw %v, want add_user then remove_user", cmds)
	}
	if got := srv.store.Get().Server.Users; len(got) != 0 {
		t.Fatalf("store users %+v", got)
	}
}

// A delete plus re-pair of the same name, landing between the original
// pairing's registration and its staleness check, must not cost the
// replacement its registration: remove_user works by name, so the stale
// cleanup leaves mqvpn alone when a user of that name still exists.
func TestStalePairingSparesReplacement(t *testing.T) {
	f := newFakeCtl(t)
	f.delayFirst = 600 * time.Millisecond
	f.startAfter(0)
	f.waitBound(t)
	srv := pairServerCtl(t, f.addr)

	errCh := make(chan error, 1)
	go func() {
		_, err := srv.addClient(context.Background(), "pixel")
		errCh <- err
	}()
	// The first pairing sits inside AddUser: the fake has already served
	// the registration, its answer is still pending. Delete and re-pair
	// complete inside that window.
	time.Sleep(150 * time.Millisecond)
	if err := srv.removeClient(context.Background(), "pixel"); err != nil {
		t.Fatalf("removeClient: %v", err)
	}
	res, err := srv.addClient(context.Background(), "pixel")
	if err != nil {
		t.Fatalf("re-pair: %v", err)
	}
	if res.URI == "" {
		t.Fatal("re-pair returned no URI")
	}
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "removed while pairing") {
			t.Fatalf("stale pairing: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stale pairing never returned")
	}
	cmds := f.commands()
	// add_user(K1), remove_user, add_user(K2) — the stale cleanup must
	// not append a second remove_user, which would unregister the
	// replacement
	if len(cmds) != 3 || cmds[2] != "add_user" {
		t.Fatalf("control API saw %v, want add_user remove_user add_user", cmds)
	}
	if got := srv.store.Get().Server.Users; len(got) != 1 || got[0].Name != "pixel" {
		t.Fatalf("store users %+v", got)
	}
}

// A caller that gives up while the retry waits for the control socket
// must not leave a credential behind: mqvpn never saw the key, and the
// URI is never delivered, so the name would just be blocked.
func TestPairingCallerGoneDuringRetry(t *testing.T) {
	f := newFakeCtl(t)
	f.startAfter(30 * time.Second) // never up within the test
	srv := pairServerCtl(t, f.addr)
	addUserRetry = 5 * time.Second
	defer func() { addUserRetry = 30 * time.Millisecond }()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := srv.addClient(ctx, "pixel")
	if err == nil || !strings.Contains(err.Error(), "abandoned") {
		t.Fatalf("addClient with a canceled caller: %v", err)
	}
	if got := srv.store.Get().Server.Users; len(got) != 0 {
		t.Fatalf("store kept the abandoned user: %+v", got)
	}
	// the name is free again (the re-pair needn't wait out the window)
	addUserRetry = 30 * time.Millisecond
	if _, err := srv.addClient(context.Background(), "pixel"); err != nil {
		t.Fatalf("re-pair after abandonment: %v", err)
	}
}

// With mqvpn simply absent, pairing proceeds: the config render carries
// the user when mqvpn (re)starts, and the tunnel port is down with it.
func TestPairingProceedsWithoutMqvpn(t *testing.T) {
	srv := pairServerCtl(t, "127.0.0.1:1") // nothing listens
	res, err := srv.addClient(context.Background(), "pixel")
	if err != nil {
		t.Fatalf("addClient without mqvpn: %v", err)
	}
	if res.URI == "" {
		t.Fatal("no pairing URI")
	}
	if got := srv.store.Get().Server.Users; len(got) != 1 || got[0].Name != "pixel" {
		t.Fatalf("store users %+v", got)
	}
}
