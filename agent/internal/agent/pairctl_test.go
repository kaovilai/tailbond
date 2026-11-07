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
	isFirst := len(f.got) == 1
	if isFirst {
		delay = f.delayFirst
	}
	f.mu.Unlock()
	if delay > 0 {
		// the request is served (a registration is made) but the caller
		// has not seen the answer yet
		time.Sleep(delay)
	}
	if noRespond && isFirst {
		return // the answer never comes: the caller reads EOF or a timeout
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

// A delete issued while the retry waits for the control socket no longer
// races the registration -- it drains it: the per-name lock holds the
// delete until the pairing finishes, then removes the credential from
// both the store and mqvpn.
func TestPairingDeletedDuringRetry(t *testing.T) {
	f := newFakeCtl(t)
	f.startAfter(700 * time.Millisecond)
	srv := pairServerCtl(t, f.addr)
	addUserRetry = 5 * time.Second
	defer func() { addUserRetry = 30 * time.Millisecond }()

	type pairOut struct {
		uri string
		err error
	}
	addCh := make(chan pairOut, 1)
	go func() {
		r, err := srv.addClient(context.Background(), "pixel")
		out := pairOut{err: err}
		if r != nil {
			out.uri = r.URI
		}
		addCh <- out
	}()
	// the delete lands while the pairing still waits for the socket
	time.Sleep(200 * time.Millisecond)
	rmCh := make(chan error, 1)
	go func() { rmCh <- srv.removeClient(context.Background(), "pixel") }()
	select {
	case r := <-addCh:
		if r.err != nil || r.uri == "" {
			t.Fatalf("pairing: %v", r.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("pairing never returned")
	}
	select {
	case err := <-rmCh:
		if err != nil {
			t.Fatalf("removeClient: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("removeClient never returned (did it drain the pairing?)")
	}
	cmds := f.commands()
	if len(cmds) != 2 || cmds[0] != "add_user" || cmds[1] != "remove_user" {
		t.Fatalf("control API saw %v, want add_user then remove_user", cmds)
	}
	if got := srv.store.Get().Server.Users; len(got) != 0 {
		t.Fatalf("store users %+v", got)
	}
}

// The stale-pairing guard also covers raw store edits, which don't take
// the per-name lock (a config restore can swap credentials underneath an
// in-flight pairing): the cleanup must not revoke a replacement that a
// direct store edit installed.
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
	// while the pairing waits for the control API's answer, a restore
	// swaps the credential for a different key
	time.Sleep(150 * time.Millisecond)
	srv.store.Update(func(c *config.Config) error {
		for i := range c.Server.Users {
			if c.Server.Users[i].Name == "pixel" {
				c.Server.Users[i].Key = "replacement-key"
			}
		}
		return nil
	})
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "removed while pairing") {
			t.Fatalf("stale pairing: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stale pairing never returned")
	}
	cmds := f.commands()
	if len(cmds) != 1 || cmds[0] != "add_user" {
		t.Fatalf("control API saw %v, want add_user only (no remove_user)", cmds)
	}
	if got := srv.store.Get().Server.Users; len(got) != 1 || got[0].Key != "replacement-key" {
		t.Fatalf("store users %+v", got)
	}
}

// A cancel after the add_user was sent and served must still clean the
// registration: mqctl uses the context only for dialing, so the answer
// failing (here: a read timeout past mqctl's own deadline) does not mean
// mqvpn never registered the key.
func TestPairingCancelAfterSentRequestCleansRegistration(t *testing.T) {
	f := newFakeCtl(t)
	f.noRespond = true
	f.delayFirst = 2500 * time.Millisecond // past mqctl's 2 s read deadline
	f.startAfter(0)
	f.waitBound(t)
	srv := pairServerCtl(t, f.addr)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := srv.addClient(ctx, "pixel")
	if err == nil || !strings.Contains(err.Error(), "abandoned") {
		t.Fatalf("addClient: %v", err)
	}
	// the rollback takes the key back out of mqvpn even though the caller
	// is long gone
	deadline := time.Now().Add(5 * time.Second)
	var cmds []string
	for time.Now().Before(deadline) {
		if cmds = f.commands(); len(cmds) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(cmds) < 2 || cmds[0] != "add_user" || cmds[1] != "remove_user" {
		t.Fatalf("control API saw %v, want add_user then remove_user", cmds)
	}
	if got := srv.store.Get().Server.Users; len(got) != 0 {
		t.Fatalf("store users %+v", got)
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
