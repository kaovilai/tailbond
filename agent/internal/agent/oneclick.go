package agent

// The menu bar's one click, "Connect via phone" (docs/phone-hub.md, "One
// click from the Mac"): from a click to working bonded internet through the
// phone — the phone starts its hotspot and its proxy, the Mac joins the
// hotspot and points the system proxy at the phone's tunnel.
//
// The halves live where their libraries are: the Bluetooth (CoreBluetooth)
// and Wi-Fi join (CoreWLAN) halves run in the Swift menu bar app, this Mac's
// user session; the privileged half — finding the phone's proxy, pointing
// the system proxy at it, checking the internet through it, and putting
// everything back — is tailbondd's, here. The menu app reports how its
// halves went and this state machine does the rest, keeping what the menu
// bar shows (status.PhoneHubClick) whatever happened:
//
//   - ble ok, wifi ok → use the phone's proxy, check the internet through
//     it, "Bonded via phone · <network type>";
//   - ble failed (no phone found, Bluetooth not allowed, the phone refused
//     the password, the hotspot couldn't start) → the same click falls back
//     to an already-joined phone hotspot and "Use phone hub", and says so;
//   - the Wi-Fi join failed → the click fails at "wifi";
//   - no proxy found → fails at "proxy";
//   - the proxy is up but the internet through it doesn't answer → fails at
//     "internet", with the system proxy put back how it was;
//   - off → stop using the hub, put the settings back, leave the hotspot on.
//
// Not persisted, like the hub choice itself: after tailbondd stops or
// crashes, the system proxy is put back (sysproxy's restore record) and the
// user clicks again.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	xproxy "golang.org/x/net/proxy"

	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

const (
	// clickCheckURL is what the internet check fetches through the phone's
	// proxy: something small that any internet answers. The e2e test points
	// it at its tunnel-only token server.
	clickCheckURL = "http://connectivitycheck.gstatic.com/generate_204"
	// the phone's proxy may take a moment to appear after hub.on (its mDNS
	// advert, its registration): keep looking this long
	clickFindFor = 15 * time.Second
	// how long the internet check may take
	clickCheckTimeout = 20 * time.Second
	// ClickOffNote is said after a click-off: like the dashboard's stop, the
	// Mac may still be on the phone's hotspot.
	ClickOffNote = HubStoppedNote
)

// clickIn is what the menu bar app reports about its halves of the click.
type clickIn struct {
	On   bool        `json:"on"`
	BLE  *bleReport  `json:"ble,omitempty"`
	WiFi *wifiReport `json:"wifi,omitempty"`
	// CheckURL overrides what the internet check fetches (tests, the e2e).
	CheckURL string `json:"check_url,omitempty"`
}

// bleReport is how the Bluetooth half went (the menu app's own words for
// the errors; the state machine only sorts them by step).
type bleReport struct {
	// State: "ok" the phone answered, "failed" a step failed.
	State string `json:"state"`
	// Step that failed: scan (no phone, Bluetooth off or not allowed), auth
	// (the phone refused the password), hotspot (it couldn't start).
	Step  string `json:"step,omitempty"`
	Error string `json:"error,omitempty"`
	// KeyID: the phone's share key id, from its BLE challenge; the hub to
	// use is matched by it.
	KeyID string `json:"key_id,omitempty"`
	// Label: the phone's name as the phone says it.
	Label string `json:"label,omitempty"`
	// SSID and Pass of the hotspot the phone started (not kept).
	SSID string `json:"ssid,omitempty"`
	Pass string `json:"pass,omitempty"`
	// Net, Battery, Charging: the phone's status reply.
	Net      string `json:"net,omitempty"`
	Battery  int    `json:"battery,omitempty"`
	Charging bool   `json:"charging,omitempty"`
}

// wifiReport is how joining the hotspot's Wi-Fi went (CoreWLAN, the menu
// app): ok with the SSID joined, or the error.
type wifiReport struct {
	OK    bool   `json:"ok"`
	SSID  string `json:"ssid,omitempty"`
	Error string `json:"error,omitempty"`
}

// clickState is what a click did; guarded by Agent.mu, on phoneHubState.
type clickState struct {
	on       bool
	via      string // ble, fallback
	label    string
	net      string
	battery  int
	charging bool
	step     string // the failed step
	err      string
	bleStep  string // why Bluetooth itself failed, when it fell back
	bleErr   string
	since    time.Time
	hubID    string // the hub in use
}

// hPhoneClick: the menu bar's one click. POST {"on": true, "ble": {...},
// "wifi": {...}} to connect (the menu app's Bluetooth and Wi-Fi halves are
// reported in ble and wifi; both optional — without them this is the
// no-Bluetooth fallback), {"on": false} to disconnect.
func (a *Agent) hPhoneClick(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient || a.phoneHub() == nil {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.hubAsked()
	var in clickIn
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	if in.On {
		a.clickOn(r.Context(), &in)
	} else {
		a.clickOff()
	}
	writeJSON(w, http.StatusOK, a.phoneHubStatusFor(r.RemoteAddr))
}

// clickOff: stop using the phone and put the settings back. The hotspot is
// left on (the phone may serve others); the menu app sends hub.off over
// Bluetooth itself when it can.
func (a *Agent) clickOff() {
	hs := a.phoneHub()
	hs.op.Lock()
	defer hs.op.Unlock()
	a.mu.Lock()
	was := (hs.click != nil && hs.click.on) || hs.use != nil
	hs.click = nil
	a.mu.Unlock()
	note := ""
	if was {
		note = ClickOffNote
	}
	a.stopHubLocked(hs, note)
}

// clickOn runs the click: record how the menu app's halves went, find the
// phone's proxy, point the system proxy at it, check the internet through
// it, and keep what happened for the menu bar.
func (a *Agent) clickOn(ctx context.Context, in *clickIn) {
	hs := a.phoneHub()
	hs.op.Lock()
	defer hs.op.Unlock()

	// what a click did so far; published (and only then read by the status)
	// under a.mu, always as a whole — while the click runs, on stays false
	// so nothing claims a bond that hasn't happened
	c := &clickState{via: "fallback", since: time.Now()}
	mutate := func(f func(*clickState)) {
		a.mu.Lock()
		f(c)
		hs.click = c
		a.mu.Unlock()
	}
	fail := func(step, err string) {
		mutate(func(c *clickState) { c.on, c.step, c.err = false, step, err })
		a.Logf("phone click: %s: %s", step, err)
	}
	logins := a.hubLogins()
	if len(logins) == 0 {
		fail("proxy", "connect to your home server once first: the phone's password comes from it")
		return
	}

	// the menu app's halves
	if in.BLE != nil && in.BLE.State == "ok" {
		mutate(func(c *clickState) {
			c.via = "ble"
			c.label, c.net = in.BLE.Label, in.BLE.Net
			c.battery, c.charging = in.BLE.Battery, in.BLE.Charging
		})
		if in.WiFi != nil && !in.WiFi.OK {
			why := in.WiFi.Error
			if why == "" {
				why = "the hotspot's Wi-Fi didn't answer"
			}
			fail("wifi", why)
			return
		}
		a.Logf("phone click: the phone %q answered over Bluetooth (network %s, battery %d%%)", in.BLE.Label, in.BLE.Net, in.BLE.Battery)
	} else if in.BLE != nil {
		// Bluetooth didn't work (no phone found, not allowed, refused, no
		// hotspot): the same click falls back to an already-joined hotspot
		// and "Use phone hub", saying so.
		mutate(func(c *clickState) { c.bleStep, c.bleErr = in.BLE.Step, in.BLE.Error })
		a.Logf("phone click: Bluetooth didn't work (%s: %s); trying the phone's hotspot as it is", in.BLE.Step, in.BLE.Error)
	}

	// find the phone's proxy; it may take a moment to appear after hub.on
	var pick *hubCand
	findFor := clickFindFor
	if a.clickFindFor > 0 {
		findFor = a.clickFindFor
	}
	deadline := time.Now().Add(findFor)
	for {
		found := a.findHubs(ctx, logins)
		a.mu.Lock()
		hs.found = found
		hs.asked = time.Now()
		a.mu.Unlock()
		pick = pickHub(found, in.BLE)
		if pick != nil || time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		a.clickPause(ctx)
	}
	if ctx.Err() != nil {
		// the caller went away mid-click: not a bond
		fail("proxy", "the click was cancelled before the phone's proxy was found")
		return
	}
	if pick == nil {
		fail("proxy", "no phone hub was found: check the phone is sharing its bonded connection (and that this Mac is on its hotspot)")
		return
	}

	// use it (the existing dashboard path: forwarder + system proxy)
	if err := a.useHubLocked(ctx, hs, pick.ID, true); err != nil {
		fail("proxy", err.Error())
		return
	}
	a.mu.Lock()
	local := ""
	if u := hs.use; u != nil {
		local = u.ln.Addr().String()
	}
	a.mu.Unlock()

	// the internet through it (a proxy that answers but carries nothing is
	// worse than none: undo and say so)
	check := in.CheckURL
	if check == "" {
		check = clickCheckURL
	}
	if err := a.clickCheck(ctx, local, check); err != nil {
		a.stopHubLocked(hs, "")
		fail("internet", fmt.Sprintf("the phone's proxy is up, but the internet through it didn't answer (%v)", err))
		return
	}
	mutate(func(c *clickState) {
		c.on = true
		c.hubID, c.label = pick.ID, pick.Label
	})
	a.Logf("phone click: bonded via %s (%s): the internet answers through %s", pick.Label, c.via, local)
}

// pickHub chooses the hub for a click: the phone the menu app talked to
// over Bluetooth (by share key id), else the first found.
func pickHub(found []hubCand, ble *bleReport) *hubCand {
	if ble != nil && ble.KeyID != "" {
		for i := range found {
			if found[i].KeyID == ble.KeyID {
				return &found[i]
			}
		}
		return nil // the phone we want isn't among them; don't take another's
	}
	if len(found) > 0 {
		return &found[0]
	}
	return nil
}

// clickPause waits between hub lookups (tests make it instant).
func (a *Agent) clickPause(ctx context.Context) {
	d := 3 * time.Second
	if a.clickWait != nil {
		d = a.clickWait(ctx)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// clickCheck fetches one small URL through the forwarder (SOCKS5, loopback,
// no password — like the apps that follow the system proxy) and wants any
// answer: the point is that the phone's tunnel carries traffic, not which
// page came back.
func (a *Agent) clickCheck(ctx context.Context, local, url string) error {
	if local == "" {
		return errors.New("no forwarder")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("check URL %q isn't http(s)", url)
	}
	d, err := xproxy.SOCKS5("tcp", local, nil, &net.Dialer{Timeout: 5 * time.Second})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, clickCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Transport: &http.Transport{DialContext: d.(xproxy.ContextDialer).DialContext, DisableKeepAlives: true}}).Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	resp.Body.Close()
	return nil
}

// clickStatus is the menu bar's view of the click.
func (s *clickState) clickStatus() *status.PhoneHubClick {
	if s == nil {
		return nil
	}
	return &status.PhoneHubClick{
		On: s.on, Via: s.via, Label: s.label, Net: s.net, Battery: s.battery, Charging: s.charging,
		Step: s.step, Error: s.err, BLEStep: s.bleStep, BLEError: s.bleErr, Since: s.since,
	}
}
