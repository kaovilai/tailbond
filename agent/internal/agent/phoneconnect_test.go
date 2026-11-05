package agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/bleproto"
	"github.com/kaovilai/tailbond/agent/internal/proxy"
	"github.com/kaovilai/tailbond/agent/internal/share"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// pcAgent: hubAgent plus the one-click hooks: a fake Wi-Fi join (it
// records what it was asked), a check URL on this machine, a short search.
func pcAgent(t *testing.T) (*Agent, *fakeSysProxy, *[]hubCand, *sync.Mutex, *[]string) {
	t.Helper()
	a, fs, cands, mu := hubAgent(t)
	var joined []string
	a.joinWiFi = func(_ context.Context, ssid, pass string) error {
		joined = append(joined, ssid+"/"+pass)
		return nil
	}
	check := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	t.Cleanup(check.Close)
	a.pcCheckURL = check.URL + "/generate_204"
	a.pcFindWait = 300 * time.Millisecond
	a.apiPort = "8077"
	return a, fs, cands, mu, &joined
}

func pcPixel() *pcPhone {
	return &pcPhone{ID: "pixel-hub", Label: "Pixel 9", KeyID: kid, Network: "5G", Battery: 81, Paths: 2}
}

// connect POSTs a connect and waits for it to finish.
func connect(t *testing.T, a *Agent, in phoneConnectReq) status.PhoneConnect {
	t.Helper()
	w := req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{"connect": in, "wait": true})
	if w.Code != 200 {
		t.Fatalf("connect: %d %s", w.Code, w.Body)
	}
	var st status.PhoneConnect
	decode(t, w, &st)
	return st
}

func setCands(mu *sync.Mutex, cands *[]hubCand, c ...hubCand) {
	mu.Lock()
	defer mu.Unlock()
	*cands = c
}

func TestPhoneConnectBluetooth(t *testing.T) {
	a, fs, cands, mu, joined := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands, hubCand{ID: "pixel-hub", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}})
	st := connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), SSID: "Pixel 9", Pass: "correct horse"})
	if st.State != pcBonded || st.Via != "bluetooth" || st.Phone != "Pixel 9" || st.Network != "5G" || st.Paths != 2 || st.Error != "" {
		t.Fatalf("bonded: %+v", st)
	}
	if len(*joined) != 1 || (*joined)[0] != "Pixel 9/correct horse" {
		t.Fatalf("joined: %v", *joined)
	}
	if !strings.Contains(fs.log(), "pac Wi-Fi http://127.0.0.1:8077/proxy.pac") {
		t.Fatalf("system proxy (PAC, as the dashboard's default): %s", fs.log())
	}
	// the menu bar reads it from the status's phone_hub (the snapshot
	// loop copies phoneHubStatus)
	ph := a.phoneHubStatus()
	if ph.Connect == nil || ph.Connect.State != pcBonded || ph.Using == nil || ph.Using.ID != "pixel-hub" {
		t.Fatalf("status: %+v", ph)
	}
	// the app joined itself (CoreWLAN): no second join
	*joined = nil
	connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), SSID: "Pixel 9", Pass: "x", Joined: true})
	if len(*joined) != 0 {
		t.Fatalf("joined again: %v", *joined)
	}

	// the second click: the proxy settings go back
	w := req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{"disconnect": true})
	decode(t, w, &st)
	if st.State != pcIdle || st.Note != PhoneDisconnectedNote {
		t.Fatalf("disconnect: %+v", st)
	}
	if !strings.HasSuffix(fs.log(), "restore Wi-Fi") {
		t.Fatalf("not restored: %s", fs.log())
	}
	if _, ok := fs.Active(); ok {
		t.Fatal("a restore record is left")
	}
	a.mu.Lock()
	inUse := a.client.hub.use != nil
	a.mu.Unlock()
	if inUse {
		t.Fatal("hub still in use")
	}
}

// Bluetooth off, denied, not supported, or no phone in range: the same
// click uses a phone hotspot the Mac is on already, and says so.
func TestPhoneConnectFallback(t *testing.T) {
	a, _, cands, mu, joined := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands, hubCand{ID: "some-hub", Label: "Pixel", Via: "router", addrs: []string{hubAddr}})
	st := connect(t, a, phoneConnectReq{BLE: "denied"})
	if st.State != pcBonded || st.Via != "hotspot" || st.Phone != "Pixel" || !strings.Contains(st.Note, "may not use Bluetooth") {
		t.Fatalf("fallback: %+v", st)
	}
	if len(*joined) != 0 {
		t.Fatal("joined a network without Bluetooth")
	}
	a.phoneDisconnect()

	// no phone over Bluetooth, and none on this network
	setCands(mu, cands)
	st = connect(t, a, phoneConnectReq{BLE: "no_phone"})
	if st.State != pcFailed || st.Step != stepFind || !strings.Contains(st.Error, "no phone with tailbond answered over Bluetooth") ||
		!strings.Contains(st.Error, "Turn on the phone's hotspot") {
		t.Fatalf("no phone: %+v", st)
	}
}

// Every failure names its step, and leaves the system proxy as it was.
func TestPhoneConnectFailures(t *testing.T) {
	a, fs, cands, mu, _ := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")

	st := connect(t, a, phoneConnectReq{BLE: "auth", BLEError: "wrong login"})
	if st.State != pcFailed || st.Step != stepLogin || !strings.Contains(st.Error, "same home server") {
		t.Fatalf("bad auth: %+v", st)
	}
	st = connect(t, a, phoneConnectReq{BLE: "hotspot", BLEError: "Shizuku isn't running on the phone", Phone: pcPixel()})
	if st.State != pcFailed || st.Step != stepHotspot || !strings.Contains(st.Error, "Shizuku isn't running") {
		t.Fatalf("hotspot refused: %+v", st)
	}
	st = connect(t, a, phoneConnectReq{BLE: "hub", BLEError: "not paired", Phone: pcPixel()})
	if st.State != pcFailed || st.Step != stepPhoneHub {
		t.Fatalf("hub refused: %+v", st)
	}
	st = connect(t, a, phoneConnectReq{BLE: "maybe"})
	if st.State != pcFailed || st.Step != stepBluetooth {
		t.Fatalf("unknown result: %+v", st)
	}

	// Wi-Fi join fails
	a.joinWiFi = func(context.Context, string, string) error { return errors.New("Could not find network Pixel 9") }
	st = connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), SSID: "Pixel 9", Pass: "p"})
	if st.State != pcFailed || st.Step != stepWiFi || !strings.Contains(st.Error, "Could not find network") {
		t.Fatalf("wifi: %+v", st)
	}
	// no way to join on this OS
	a.joinWiFi = nil
	st = connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), SSID: "Pixel 9", Pass: "p"})
	if st.State != pcFailed || st.Step != stepWiFi || !strings.Contains(st.Error, "yourself") {
		t.Fatalf("wifi, no joiner: %+v", st)
	}

	// joined, but no phone hub answers (another hub doesn't count)
	setCands(mu, cands, hubCand{ID: "someone-else", Label: "Other", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}})
	st = connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true})
	if st.State != pcFailed || st.Step != stepFind || !strings.Contains(st.Error, "didn't answer") {
		t.Fatalf("no proxy found: %+v", st)
	}
	if fs.log() != "" {
		t.Fatalf("the system proxy was touched: %s", fs.log())
	}

	// the hub answers, but nothing gets through it: the proxy goes back
	dead := &proxy.Server{Auth: proxy.Verifier([]byte("home-share-key"), time.Now, nil),
		Dial: func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("tunnel down") }}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	t.Cleanup(func() { ln.Close() })
	go dead.Serve(ln)
	setCands(mu, cands, hubCand{ID: "pixel-hub", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{ln.Addr().String()}})
	st = connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true})
	if st.State != pcFailed || st.Step != stepInternet || !strings.Contains(st.Error, "proxy is off again") {
		t.Fatalf("no internet: %+v", st)
	}
	if !strings.HasPrefix(fs.log(), "pac Wi-Fi") || !strings.HasSuffix(fs.log(), "restore Wi-Fi") {
		t.Fatalf("not put back: %s", fs.log())
	}
	if _, ok := fs.Active(); ok {
		t.Fatal("a restore record is left")
	}
	a.mu.Lock()
	inUse := a.client.hub.use != nil
	a.mu.Unlock()
	if inUse {
		t.Fatal("hub still in use after the internet check failed")
	}
}

// Bonded only while that hub is in use: stopped in the dashboard, or the
// phone gone, the menu says idle (with the hub's note).
func TestPhoneConnectFollowsTheHub(t *testing.T) {
	a, _, cands, mu, _ := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands, hubCand{ID: "pixel-hub", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}})
	if st := connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true}); st.State != pcBonded {
		t.Fatalf("connect: %+v", st)
	}
	a.stopHub(HubGoneNote)
	var st status.PhoneConnect
	decode(t, req(t, a, "GET", "/api/phone-connect", lan, "", nil), &st)
	if st.State != pcIdle || st.Note != HubGoneNote {
		t.Fatalf("after the hub went: %+v", st)
	}
	// a second click while one runs is refused
	setCands(mu, cands)
	a.pcFindWait = 2 * time.Second
	if w := req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{"connect": phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true}}); w.Code != 200 {
		t.Fatalf("start: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{"connect": phoneConnectReq{BLE: "denied"}}); w.Code != 409 {
		t.Fatalf("second connect: %d", w.Code)
	}
	// and disconnect stops it
	start := time.Now()
	req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{"disconnect": true})
	if time.Since(start) > time.Second {
		t.Fatal("disconnect waited for the search")
	}
	if w := req(t, a, "POST", "/api/phone-connect", lan, "", map[string]any{}); w.Code != 400 {
		t.Fatalf("empty body: %d", w.Code)
	}
}

// The menu bar app's Bluetooth logins: this Mac only, the BLE secret (not
// the proxy password).
func TestPhoneConnectKeys(t *testing.T) {
	a, _, _, _, _ := pcAgent(t)
	if w := req(t, a, "GET", "/api/phone-connect/keys", lan, "", nil); w.Code != http.StatusForbidden {
		t.Fatalf("from the LAN: %d", w.Code)
	}
	var out struct{ Logins []bleLogin }
	decode(t, req(t, a, "GET", "/api/phone-connect/keys", "127.0.0.1:5000", "", nil), &out)
	if len(out.Logins) != 1 || out.Logins[0].KeyID != kid {
		t.Fatalf("logins: %+v", out)
	}
	l := out.Logins[0]
	hub, e, _ := strings.Cut(l.User, "/")
	var exp uint64
	for _, c := range e {
		exp = exp*10 + uint64(c-'0')
	}
	_, want := bleproto.Credentials(share.PassKey([]byte("home-share-key"), hub, exp), hub, exp)
	_, proxyPass := proxy.Credentials(share.PassKey([]byte("home-share-key"), hub, exp), hub, exp)
	if hub != "mac" || l.Secret != want || l.Secret == proxyPass {
		t.Fatalf("login %+v, want secret %s (not the proxy's %s)", l, want, proxyPass)
	}
	// the phone accepts it
	if _, ok := bleproto.Verifier([]byte("home-share-key"), time.Now, nil)(l.User, "a", "b", bleproto.AuthMAC(l.Secret, "a", "b", l.User)); !ok {
		t.Fatal("the phone refuses the Mac's login")
	}
}

func TestWiFiDevice(t *testing.T) {
	out := "\nHardware Port: Ethernet\nDevice: en1\nEthernet Address: x\n\nHardware Port: Wi-Fi\nDevice: en0\nEthernet Address: y\n"
	if d := wifiDevice(out); d != "en0" {
		t.Fatalf("wifi device: %q", d)
	}
	if wifiDevice("Hardware Port: Ethernet\nDevice: en1\n") != "" {
		t.Fatal("found Wi-Fi on a Mac without it")
	}
}

// Found only through the hotspot's router (no mDNS, no server list), the
// hub has no id of its own: it is the phone when it took the login for the
// phone's key id.
func TestPhoneConnectRouterHub(t *testing.T) {
	a, _, cands, mu, _ := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands)
	a.hubRouters = func(context.Context) []hubCand {
		return []hubCand{{ID: "router", Label: "Your phone", Via: "router", addrs: []string{hubAddr}}}
	}
	st := connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true})
	if st.State != pcBonded || st.Phone != "Pixel 9" {
		t.Fatalf("router hub: %+v", st)
	}
	a.phoneDisconnect()
	// the phone's key id doesn't match the login the router took: not it
	p := pcPixel()
	p.KeyID = "deadbeef"
	if st := connect(t, a, phoneConnectReq{BLE: "ok", Phone: p, Joined: true}); st.State != pcFailed || st.Step != stepFind {
		t.Fatalf("another key: %+v", st)
	}
}

// A hub that says it isn't bonded (an iPhone: mobile data only) isn't
// "Bonded via phone".
func TestPhoneConnectSkipsUnbonded(t *testing.T) {
	a, fs, cands, mu, _ := pcAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands, hubCand{ID: "iphone", Label: "iPhone", Via: "mdns", Bonded: "0", KeyID: kid, addrs: []string{hubAddr}})
	st := connect(t, a, phoneConnectReq{BLE: "off"})
	if st.State != pcFailed || st.Step != stepFind || !strings.Contains(st.Error, "not a bonded connection") {
		t.Fatalf("unbonded: %+v", st)
	}
	if fs.log() != "" {
		t.Fatalf("proxy touched: %s", fs.log())
	}
}

// A captive portal's page or redirect isn't internet: generate_204 must
// answer 204.
func TestPhoneConnectCaptivePortal(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusFound} {
		a, fs, cands, mu, _ := pcAgent(t)
		_, hubAddr := fakeHub(t, "home-share-key")
		setCands(mu, cands, hubCand{ID: "pixel-hub", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}})
		portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if code == http.StatusFound {
				http.Redirect(w, r, "/login", code)
				return
			}
			w.Write([]byte("<html>sign in</html>"))
		}))
		t.Cleanup(portal.Close)
		a.pcCheckURL = portal.URL + "/generate_204"
		st := connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true})
		if st.State != pcFailed || st.Step != stepInternet || !strings.HasSuffix(fs.log(), "restore Wi-Fi") {
			t.Fatalf("portal %d: %+v / %s", code, st, fs.log())
		}
	}
}

// The dashboard picks another phone while the internet check runs: a
// failed check must not tear that one down.
func TestPhoneConnectCheckDoesntStopAnotherHub(t *testing.T) {
	a, _, cands, mu, _ := pcAgent(t)
	dialing := make(chan struct{}, 1)
	release := make(chan struct{})
	slow := &proxy.Server{Auth: proxy.Verifier([]byte("home-share-key"), time.Now, nil),
		Dial: func(context.Context, string, string) (net.Conn, error) {
			select {
			case dialing <- struct{}{}:
			default:
			}
			<-release
			return nil, errors.New("tunnel down")
		}}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	t.Cleanup(func() { ln.Close() })
	go slow.Serve(ln)
	_, otherAddr := fakeHub(t, "home-share-key")
	setCands(mu, cands,
		hubCand{ID: "pixel-hub", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{ln.Addr().String()}},
		hubCand{ID: "tablet-hub", Label: "Tablet", Via: "mdns", KeyID: kid, addrs: []string{otherAddr}})
	done := make(chan status.PhoneConnect)
	go func() { done <- connect(t, a, phoneConnectReq{BLE: "ok", Phone: pcPixel(), Joined: true}) }()
	select {
	case <-dialing:
	case <-time.After(5 * time.Second):
		t.Fatal("the check never reached the hub")
	}
	if w := req(t, a, "POST", "/api/phone-hub", lan, "", map[string]any{"use": "tablet-hub"}); w.Code != 200 {
		t.Fatalf("dashboard switch: %d %s", w.Code, w.Body)
	}
	close(release)
	st := <-done
	if st.State != pcFailed || st.Step != stepInternet {
		t.Fatalf("check: %+v", st)
	}
	a.mu.Lock()
	u := a.client.hub.use
	a.mu.Unlock()
	if u == nil || u.cand.ID != "tablet-hub" {
		t.Fatalf("the dashboard's hub was torn down: %+v", u)
	}
	a.stopHub("")
}
