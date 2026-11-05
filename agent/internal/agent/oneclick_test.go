package agent

// The menu bar's one click (oneclick.go): every step of the ladder, the
// words the menu bar gets, and that a failure leaves nothing behind (the
// system proxy back as it was, no forwarder). The Bluetooth and Wi-Fi
// halves run in the Swift menu app (CoreBluetooth, CoreWLAN); these tests
// feed their reported outcomes into the API, as the menu app does.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/share"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// lo: the menu bar app's loopback origin, the only one the BLE logins go to.
const lo = "127.0.0.1:9"

// clickAgent: a hubAgent whose one-click waits nothing between hub lookups.
func clickAgent(t *testing.T) (*Agent, *fakeSysProxy, *[]hubCand, *sync.Mutex) {
	t.Helper()
	a, fs, cands, mu := hubAgent(t)
	a.apiPort = "8077"
	a.clickWait = func(context.Context) time.Duration { return 0 }
	a.clickFindFor = 10 * time.Millisecond
	return a, fs, cands, mu
}

// click posts a one click and returns the phone hub status it got. The
// menu bar app is a loopback caller (the only one the BLE logins go to).
func click(t *testing.T, a *Agent, body map[string]any) (int, *status.PhoneHub) {
	t.Helper()
	w := req(t, a, "POST", "/api/phone-hub/click", lo, "", body)
	var st status.PhoneHub
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatalf("click: %d %s", w.Code, w.Body)
	}
	return w.Code, &st
}

func clickStatus(t *testing.T, a *Agent, remote ...string) *status.PhoneHub {
	t.Helper()
	w := req(t, a, "GET", "/api/phone-hub", firstRemote(remote), "", nil)
	var st status.PhoneHub
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatalf("status: %d %s", w.Code, w.Body)
	}
	return &st
}

func firstRemote(rs []string) string {
	if len(rs) > 0 {
		return rs[0]
	}
	return lo
}

// okServer answers any HTTP request (the internet through the phone).
func okServer(t *testing.T) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s.Close)
	return s.URL
}

// noServer: a port nothing listens on (the internet is down).
func noServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return "http://" + addr + "/generate_204"
}

func TestPhoneClickConnectsThroughBle(t *testing.T) {
	a, fs, cands, mu := clickAgent(t)
	hub, hubAddr := fakeHub(t, "home-share-key")
	other, otherAddr := fakeHub(t, "someone-elses-key")
	target := okServer(t)
	mu.Lock()
	*cands = []hubCand{
		{ID: "other", Label: "Someone else", Via: "mdns", KeyID: share.KeyID([]byte("someone-elses-key")), addrs: []string{otherAddr}},
		{ID: "pixel", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}},
	}
	mu.Unlock()
	code, st := click(t, a, map[string]any{
		"on": true, "check_url": target,
		"ble": map[string]any{
			"state": "ok", "key_id": kid, "label": "Pixel", "net": "5G",
			"battery": 78, "charging": true, "ssid": "Pixel", "pass": "hotspot-pass",
		},
		"wifi": map[string]any{"ok": true, "ssid": "Pixel"},
	})
	if code != 200 || st.Click == nil || !st.Click.On {
		t.Fatalf("click: %d %+v", code, st.Click)
	}
	c := st.Click
	if c.Via != "ble" || c.Label != "Pixel" || c.Net != "5G" || c.Battery != 78 || !c.Charging {
		t.Fatalf("the click's say: %+v", c)
	}
	if c.Step != "" || c.Error != "" || c.BLEStep != "" {
		t.Fatalf("a connected click with failures: %+v", c)
	}
	// the phone the Mac talked to over Bluetooth, by share key id — not
	// the other one that also takes a password
	if st.Using == nil || st.Using.ID != "pixel" || st.Using.Label != "Pixel" {
		t.Fatalf("using: %+v, want the phone with key id %s", st.Using, kid)
	}
	if up, _, _ := hub.Stats(); up == 0 {
		t.Fatal("no traffic through the phone's proxy (the internet check)")
	}
	if up, _, _ := other.Stats(); up != 0 {
		t.Fatal("traffic through the other phone's proxy")
	}
	// the menu app (loopback) gets the BLE logins for its half; a LAN
	// dashboard gets everything else but not the keys
	if len(st.Logins) != 1 || st.Logins[0].KeyID != kid || st.Logins[0].User == "" || len(st.Logins[0].Key) != 16 {
		t.Fatalf("logins: %+v", st.Logins)
	}
	if lanSt := clickStatus(t, a, lan); len(lanSt.Logins) != 0 {
		t.Fatalf("a LAN caller must not get the BLE logins: %+v", lanSt.Logins)
	}
	if !containsCall(fs.log(), "pac Wi-Fi http://127.0.0.1:8077/proxy.pac") {
		t.Fatalf("the system proxy wasn't set (PAC): %s", fs.log())
	}

	// click off: the setting back, the state gone, the hotspot left alone
	code, st = click(t, a, map[string]any{"on": false})
	if code != 200 || st.Click != nil || st.Using != nil {
		t.Fatalf("click off: %d click=%+v using=%+v", code, st.Click, st.Using)
	}
	if !lastCall(fs.log(), "restore Wi-Fi") {
		t.Fatalf("not restored after off: %s", fs.log())
	}
	if st.Note != ClickOffNote {
		t.Fatalf("note after off: %q", st.Note)
	}
}

func TestPhoneClickFallsBackWhenBleFails(t *testing.T) {
	for _, tc := range []struct{ step, what string }{
		{"scan", "no phone was found"},
		{"auth", "the phone refused"},
		{"hotspot", "Shizuku isn't running"},
	} {
		t.Run(tc.step, func(t *testing.T) {
			a, _, cands, mu := clickAgent(t)
			_, hubAddr := fakeHub(t, "home-share-key")
			mu.Lock()
			*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}}}
			mu.Unlock()
			// the click falls back to an already-joined hotspot and "Use
			// phone hub", saying why Bluetooth didn't work
			code, st := click(t, a, map[string]any{
				"on": true, "check_url": okServer(t),
				"ble": map[string]any{"state": "failed", "step": tc.step, "error": tc.what},
			})
			if code != 200 || st.Click == nil || !st.Click.On {
				t.Fatalf("click: %d %+v", code, st.Click)
			}
			c := st.Click
			if c.Via != "fallback" || c.BLEStep != tc.step || c.BLEError != tc.what {
				t.Fatalf("the fallback's say: %+v", c)
			}
			if st.Using == nil || st.Using.ID != "pixel" {
				t.Fatalf("the fallback didn't use the hub: %+v", st.Using)
			}
		})
	}
}

func TestPhoneClickNoHub(t *testing.T) {
	// no phone hub around at all: the click fails at "proxy", with the
	// Bluetooth failure (a scan that found nothing) alongside
	a, fs, _, _ := clickAgent(t)
	code, st := click(t, a, map[string]any{
		"on": true, "check_url": okServer(t),
		"ble": map[string]any{"state": "failed", "step": "scan", "error": "no phone was found"},
	})
	if code != 200 || st.Click == nil || st.Click.On {
		t.Fatalf("click: %d %+v", code, st.Click)
	}
	if st.Click.Step != "proxy" || st.Click.Error == "" {
		t.Fatalf("the failure's say: %+v", st.Click)
	}
	if st.Using != nil || fs.log() != "" {
		t.Fatalf("nothing should have been set: using=%+v proxy calls=%q", st.Using, fs.log())
	}
}

func TestPhoneClickWifiFailed(t *testing.T) {
	a, fs, cands, mu := clickAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}}}
	mu.Unlock()
	code, st := click(t, a, map[string]any{
		"on": true, "check_url": okServer(t),
		"ble":  map[string]any{"state": "ok", "key_id": kid, "label": "Pixel", "ssid": "Pixel", "pass": "pw"},
		"wifi": map[string]any{"ok": false, "error": "hotspot not visible yet"},
	})
	if code != 200 || st.Click == nil || st.Click.On {
		t.Fatalf("click: %d %+v", code, st.Click)
	}
	if st.Click.Step != "wifi" || st.Click.Error != "hotspot not visible yet" {
		t.Fatalf("the failure's say: %+v", st.Click)
	}
	if st.Using != nil || fs.log() != "" {
		t.Fatalf("the Wi-Fi failure must stop the click before the proxy: %s", fs.log())
	}
}

func TestPhoneClickNoInternetUndoes(t *testing.T) {
	a, fs, cands, mu := clickAgent(t)
	hub, hubAddr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}}}
	mu.Unlock()
	code, st := click(t, a, map[string]any{
		"on": true, "check_url": noServer(t),
		"ble":  map[string]any{"state": "ok", "key_id": kid, "label": "Pixel", "ssid": "Pixel", "pass": "pw"},
		"wifi": map[string]any{"ok": true, "ssid": "Pixel"},
	})
	if code != 200 || st.Click == nil || st.Click.On {
		t.Fatalf("click: %d %+v", code, st.Click)
	}
	if st.Click.Step != "internet" || st.Click.Error == "" {
		t.Fatalf("the failure's say: %+v", st.Click)
	}
	// the proxy was set and is put back: nothing left behind
	if st.Using != nil {
		t.Fatalf("still using the hub after the internet failed: %+v", st.Using)
	}
	if !lastCall(fs.log(), "restore Wi-Fi") {
		t.Fatalf("the system proxy wasn't put back: %s", fs.log())
	}
	if up, _, _ := hub.Stats(); up != 0 {
		t.Fatal("traffic went through the hub anyway")
	}
}

func TestPhoneClickNotPaired(t *testing.T) {
	a := clientAgent(t, nil)
	a.sysProxy = &fakeSysProxy{service: "Wi-Fi"}
	a.clickWait = func(context.Context) time.Duration { return 0 }
	a.hubAsked()
	code, st := click(t, a, map[string]any{"on": true, "check_url": okServer(t)})
	if code != 200 || st.Click == nil || st.Click.On || st.Click.Step != "proxy" {
		t.Fatalf("click: %d %+v", code, st.Click)
	}
	if st.Click.Error == "" {
		t.Fatal("the click must say the password needs the home server")
	}
}

func TestPhoneClickHubGone(t *testing.T) {
	a, fs, cands, mu := clickAgent(t)
	_, hubAddr := fakeHub(t, "home-share-key")
	mu.Lock()
	*cands = []hubCand{{ID: "pixel", Label: "Pixel", Via: "mdns", KeyID: kid, addrs: []string{hubAddr}}}
	mu.Unlock()
	_, st := click(t, a, map[string]any{"on": true, "check_url": okServer(t)})
	if st.Click == nil || !st.Click.On {
		t.Fatalf("click: %+v", st.Click)
	}
	// the phone goes away (the hub watcher gives up on it): the click says
	// so, like the dashboard's card
	mu.Lock()
	*cands = nil
	mu.Unlock()
	for range hubMissMax {
		a.hubRound(context.Background())
	}
	st = clickStatus(t, a)
	if st.Click == nil || st.Click.On || st.Click.Step != "gone" {
		t.Fatalf("after the hub went: %+v", st.Click)
	}
	if st.Using != nil || !lastCall(fs.log(), "restore Wi-Fi") {
		t.Fatalf("not put back: using=%+v calls=%s", st.Using, fs.log())
	}
}

// lastCall: the fake system proxy's calls, ";"-separated; is want the last?
func lastCall(log, want string) bool {
	calls := splitCalls(log)
	return len(calls) > 0 && calls[len(calls)-1] == want
}

func containsCall(log, want string) bool {
	for _, c := range splitCalls(log) {
		if c == want {
			return true
		}
	}
	return false
}

func splitCalls(log string) []string {
	var out []string
	cur := ""
	for _, r := range log {
		switch {
		case r == ';':
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
		case r == ' ' && cur == "":
		default:
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
