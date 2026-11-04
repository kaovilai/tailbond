package agent

import (
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	xproxy "golang.org/x/net/proxy"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/proxy"
	"github.com/kaovilai/tailbond/agent/internal/status"
	"github.com/kaovilai/tailbond/agent/internal/sysproxy"
)

type fakeSysProxy struct {
	mu      sync.Mutex
	service string
	rec     *sysproxy.Record
	calls   []string
	// another writer undid the restore of this service
	clobbered string
}

func (f *fakeSysProxy) ActiveService(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.service, nil
}

func (f *fakeSysProxy) Use(_ context.Context, svc, host string, port int, user, pass string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "use "+svc+" "+net.JoinHostPort(host, strconv.Itoa(port))+" "+user)
	f.rec = &sysproxy.Record{Service: svc, Mode: "socks"}
	return nil
}

func (f *fakeSysProxy) UsePAC(_ context.Context, svc, url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "pac "+svc+" "+url)
	f.rec = &sysproxy.Record{Service: svc, Mode: "pac"}
	return nil
}

func (f *fakeSysProxy) Restore(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rec != nil {
		f.calls = append(f.calls, "restore "+f.rec.Service)
	}
	f.rec = nil
	return nil
}

func (f *fakeSysProxy) Active() (sysproxy.Record, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rec == nil {
		return sysproxy.Record{}, false
	}
	return *f.rec, true
}

func (f *fakeSysProxy) Matches(_ context.Context, r sysproxy.Record) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clobbered != r.Service, nil
}

func (f *fakeSysProxy) PutBack(_ context.Context, r sysproxy.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "putback "+r.Service)
	f.clobbered = ""
	return nil
}

func (f *fakeSysProxy) log() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, "; ")
}

// echoServer answers "re:" + what it got.
func echoServer(t *testing.T) string {
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

// fakeHub is a phone's proxy with the credentials of key.
func fakeHub(t *testing.T, key string) (*proxy.Server, string) {
	t.Helper()
	u, p := proxy.Credentials([]byte(key))
	s := &proxy.Server{User: u, Pass: p}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go s.Serve(ln)
	return s, ln.Addr().String()
}

func hubAgent(t *testing.T) (*Agent, *fakeSysProxy, *[]hubCand, *sync.Mutex) {
	t.Helper()
	a := clientAgent(t, func(c *config.Config) { c.Client.ShareKey = "home-share-key" })
	fs := &fakeSysProxy{service: "Wi-Fi"}
	a.sysProxy = fs
	a.hubLocal = "127.0.0.1:0"
	a.hubRecheck = []time.Duration{50 * time.Millisecond}
	var mu sync.Mutex
	cands := &[]hubCand{}
	a.hubBrowse = func(context.Context) []hubCand {
		mu.Lock()
		defer mu.Unlock()
		return append([]hubCand(nil), *cands...)
	}
	a.hubRouters = func(context.Context) []hubCand { return nil }
	a.hubAsked()
	return a, fs, cands, &mu
}

func throughForwarder(t *testing.T, local, target string) string {
	t.Helper()
	d, err := xproxy.SOCKS5("tcp", local, nil, &net.Dialer{Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	c, err := d.Dial("tcp", target)
	if err != nil {
		t.Fatalf("through %s to %s: %v", local, target, err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	c.Write([]byte("hello"))
	b, _ := io.ReadAll(c)
	return string(b)
}

func TestPhoneHubUseMoveGone(t *testing.T) {
	a, fs, cands, mu := hubAgent(t)
	hub, hubAddr := fakeHub(t, "home-share-key")
	_, otherAddr := fakeHub(t, "someone-elses-key")
	target := echoServer(t)
	mu.Lock()
	*cands = []hubCand{
		{ID: "pixel", Label: "Pixel", Via: "mdns", addrs: []string{"127.0.0.1:1", hubAddr}},
		{ID: "stranger", Label: "Not ours", Via: "mdns", addrs: []string{otherAddr}},
	}
	mu.Unlock()
	a.hubRound(context.Background())
	var st status.PhoneHub
	w := req(t, a, "GET", "/api/phone-hub", lan, "", nil)
	decode(t, w, &st)
	if len(st.Found) != 1 || st.Found[0].ID != "pixel" || st.Found[0].Addr != hubAddr {
		t.Fatalf("found: %+v (a hub with another password must not be listed)", st.Found)
	}
	w = req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"use": "pixel"})
	decode(t, w, &st)
	if w.Code != 200 || st.Using == nil || st.Using.Mode != "socks" || st.Using.Service != "Wi-Fi" {
		t.Fatalf("use: %d %s", w.Code, w.Body)
	}
	local := st.Using.Local
	if !strings.Contains(fs.log(), "use Wi-Fi "+local+" ") {
		t.Fatalf("system proxy not pointed at the forwarder %s: %s", local, fs.log())
	}
	if got := throughForwarder(t, local, target); got != "re:hello" {
		t.Fatalf("through the hub: %q", got)
	}
	if up, _, clients := hub.Stats(); up < 5 || len(clients) != 1 {
		t.Fatalf("the hub didn't carry it: up %d %+v", up, clients)
	}

	// the phone's address changes (a new hotspot session): the forwarder follows
	hub2, hub2Addr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", addrs: []string{hub2Addr}}}
	mu.Unlock()
	a.hubRound(context.Background())
	if got := throughForwarder(t, local, target); got != "re:hello" {
		t.Fatalf("after the move: %q", got)
	}
	if up, _, _ := hub2.Stats(); up < 5 {
		t.Fatal("the new address didn't carry it")
	}
	if strings.Count(fs.log(), "use ") != 1 {
		t.Fatalf("the system setting changed on a move: %s", fs.log())
	}

	// gone for hubMissMax rounds: proxy off, settings back, note
	mu.Lock()
	*cands = nil
	mu.Unlock()
	for range hubMissMax {
		a.hubRound(context.Background())
	}
	w = req(t, a, "GET", "/api/phone-hub", lan, "", nil)
	st = status.PhoneHub{}
	decode(t, w, &st)
	if st.Using != nil || st.Note != HubGoneNote {
		t.Fatalf("after the hub went: %s", w.Body)
	}
	if !strings.HasSuffix(fs.log(), "restore Wi-Fi") {
		t.Fatalf("not restored: %s", fs.log())
	}
	if _, err := net.Dial("tcp", local); err == nil {
		t.Fatal("forwarder still listening")
	}
}

func TestPhoneHubStopPACAndStartupRestore(t *testing.T) {
	a, fs, cands, mu := hubAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "server", addrs: []string{hubAddr}}}
	mu.Unlock()
	a.apiPort = "8077"
	w := req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"refresh": true, "use": "pixel", "pac": true})
	var st status.PhoneHub
	decode(t, w, &st)
	if w.Code != 200 || st.Using == nil || st.Using.Mode != "pac" {
		t.Fatalf("use with PAC: %d %s", w.Code, w.Body)
	}
	if !strings.Contains(fs.log(), "pac Wi-Fi http://127.0.0.1:8077/proxy.pac") {
		t.Fatalf("PAC: %s", fs.log())
	}
	w = req(t, a, "GET", "/proxy.pac", lan, "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "SOCKS5 127.0.0.1:") || !strings.Contains(w.Body.String(), `".ts.net"`) {
		t.Fatalf("pac file: %d %s", w.Code, w.Body)
	}
	// the user stops it
	w = req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"use": ""})
	st = status.PhoneHub{}
	decode(t, w, &st)
	if st.Using != nil || st.Note != "" || !strings.HasSuffix(fs.log(), "restore Wi-Fi") {
		t.Fatalf("stop: %s / %s", w.Body, fs.log())
	}
	// unknown hub
	if w := req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"use": "nope"}); w.Code != 409 {
		t.Fatalf("unknown hub: %d", w.Code)
	}
	// a record left by a crash is put back at start, and again when
	// another writer undoes it
	fs.rec = &sysproxy.Record{Service: "iPhone USB", Mode: "socks"}
	fs.mu.Lock()
	fs.clobbered = "iPhone USB"
	fs.mu.Unlock()
	a.restoreSysProxy()
	if !strings.Contains(fs.log(), "restore iPhone USB") {
		t.Fatalf("startup restore: %s", fs.log())
	}
	time.Sleep(300 * time.Millisecond)
	if !strings.HasSuffix(fs.log(), "putback iPhone USB") {
		t.Fatalf("recheck: %s", fs.log())
	}
}

// Without a system proxy to set (Linux), the forwarder still runs: the
// user points apps at it.
func TestPhoneHubManual(t *testing.T) {
	a, _, cands, mu := hubAgent(t)
	a.sysProxy = nil
	_, hubAddr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", addrs: []string{hubAddr}}}
	mu.Unlock()
	w := req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"refresh": true, "use": "pixel"})
	var st status.PhoneHub
	decode(t, w, &st)
	if w.Code != 200 || st.Using == nil || st.Using.Mode != "manual" || !st.Manual {
		t.Fatalf("manual: %d %s", w.Code, w.Body)
	}
	if got := throughForwarder(t, st.Using.Local, echoServer(t)); got != "re:hello" {
		t.Fatalf("through: %q", got)
	}
	a.stopHub("")
}

// Nobody looking and no hub in use: no discovery at all.
func TestPhoneHubIdle(t *testing.T) {
	a, _, _, _ := hubAgent(t)
	n := 0
	a.hubBrowse = func(context.Context) []hubCand { n++; return nil }
	a.client.hub.asked = time.Now().Add(-hubInterest - time.Second)
	a.hubRound(context.Background())
	if n != 0 {
		t.Fatal("browsed with nobody looking")
	}
	req(t, a, "GET", "/api/status", lan, "", nil)
	a.hubRound(context.Background())
	if n != 1 {
		t.Fatal("didn't browse after a status request")
	}
}

func TestRouterCandidates(t *testing.T) {
	ndp := "Neighbor                        Linklayer Address  Netif Expire    St Flgs Prbs\n" +
		"fe80::1c3e:84ff:fe9b:3c64%en0 if=en0, flags=, pref=medium, expire=21m55s\n" +
		"fe80::1%utun3 if=utun3, flags=, pref=medium, expire=never\n"
	if got := ndpRouters(ndp); len(got) != 1 || got[0] != "fe80::1c3e:84ff:fe9b:3c64%en0" {
		t.Fatalf("ndp: %v", got)
	}
	ns := "Destination        Gateway            Flags               Netif Expire\n" +
		"default            192.0.0.1          UGScg                 en0\n" +
		"default            172.20.10.1        UGScIg                en0\n" +
		"default            link#22            UCSIg               utun4\n"
	if got := ipv4Gateways(ns); len(got) != 1 || got[0] != "172.20.10.1" {
		t.Fatalf("gateways (CLAT 192.0.0.1 skipped): %v", got)
	}
	c := routerCands([]string{"fe80::1c3e:84ff:fe9b:3c64%en0", "172.20.10.1", "172.20.10.1"})
	if len(c) != 1 || len(c[0].addrs) != 2 || c[0].addrs[0] != "[fe80::1c3e:84ff:fe9b:3c64%en0]:1888" {
		t.Fatalf("router candidates: %+v", c)
	}
	if Possessive("Pixel") != "Pixel's" || Possessive("James") != "James'" {
		t.Fatal("possessive")
	}
}
