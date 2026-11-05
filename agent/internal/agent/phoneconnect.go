package agent

// One click from the Mac (docs/phone-hub.md, "One click from the Mac"):
// the menu bar's "Connect via phone".
//
// The menu bar app does the Bluetooth half (CoreBluetooth; the protocol is
// internal/bleproto): it logs in to the phone, has it start its hotspot
// (Shizuku) and its phone hub, joins the hotspot with CoreWLAN when it can,
// and then hands what happened to tailbondd, POST /api/phone-connect. From
// there tailbondd goes on by itself, step by step, and every failure names
// its step:
//
//	bluetooth  the app's Bluetooth result. Off, denied, not supported or no
//	           phone in range: fall back to a phone hotspot this Mac is on
//	           already. login, hotspot, phone_hub: the phone refused; stop.
//	wifi       join the phone's hotspot (networksetup, as root), unless the
//	           app did
//	find       look for the phone's hub (the phone-hub discovery: mDNS, the
//	           server's list, the hotspot's router), the phone the app named
//	           first
//	use        "Use phone hub" (useHub: the forwarder, the system proxy)
//	internet   fetch a small URL through the forwarder; nothing comes back:
//	           stop using the hub (the system proxy goes back) and say so
//
// Click again: stop using the hub (the system proxy goes back). The app
// tells the phone hub.off; the hotspot stays on unless the user asks.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	xproxy "golang.org/x/net/proxy"

	"github.com/kaovilai/tailbond/agent/internal/bleproto"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// Connect via phone: states and steps (status.PhoneConnect).
const (
	pcIdle    = "idle"
	pcWorking = "working"
	pcBonded  = "bonded"
	pcFailed  = "failed"

	stepBluetooth = "bluetooth"
	stepLogin     = "login"
	stepHotspot   = "hotspot"
	stepPhoneHub  = "phone_hub"
	stepWiFi      = "wifi"
	stepFind      = "find"
	stepUse       = "use"
	stepInternet  = "internet"
)

// pcDefaultCheckURL is fetched through the phone to prove internet
// (TB_PHONE_CHECK_URL overrides it; the e2e test fetches a token only the
// tunnel reaches).
const pcDefaultCheckURL = "http://connectivitycheck.gstatic.com/generate_204"

// PhoneDisconnectedNote is said after the user clicks to disconnect.
const PhoneDisconnectedNote = "Stopped using the phone: the proxy settings are back. The phone's hotspot stays on, and this Mac may still be on it."

// pcPhone is the phone, as the app's Bluetooth session saw it.
type pcPhone struct {
	ID      string `json:"id"` // its phone hub's id (the proxy's mDNS id)
	Label   string `json:"label"`
	KeyID   string `json:"kid"`
	Network string `json:"network"` // 5G, LTE, Wi-Fi
	Battery int    `json:"battery"`
	Paths   int    `json:"paths"` // its bonded paths
}

// phoneConnectReq is POST /api/phone-connect {"connect": ...}.
type phoneConnectReq struct {
	// BLE: "ok", or why the app has no phone over Bluetooth: "off",
	// "denied", "unsupported", "no_phone" (fall back), "auth", "hotspot",
	// "hub" (the phone refused).
	BLE      string   `json:"ble"`
	BLEError string   `json:"ble_error"`
	Phone    *pcPhone `json:"phone"`
	SSID     string   `json:"ssid"`
	Pass     string   `json:"pass"`
	Joined   bool     `json:"joined"` // the app joined the hotspot itself (CoreWLAN)
	PAC      *bool    `json:"pac"`    // default true, as the dashboard
}

// pcState is guarded by Agent.mu (phoneHubState.conn).
type pcState struct {
	state, step, via string
	phone            pcPhone
	hubID            string
	err, note        string
	cancel           context.CancelFunc
	done             chan struct{}
}

func pcFallbackWhy(ble, msg string) (string, bool) {
	var why string
	switch ble {
	case "off":
		why = "Bluetooth is off on this Mac"
	case "denied":
		why = "tailbond may not use Bluetooth (System Settings → Privacy & Security → Bluetooth)"
	case "unsupported":
		why = "this Mac has no Bluetooth LE"
	case "no_phone":
		why = "no phone with tailbond answered over Bluetooth"
	default:
		return "", false
	}
	if msg != "" && ble != "no_phone" {
		why += " (" + msg + ")"
	}
	return why, true
}

// phoneConnectStart runs the steps in the background (wait: until done).
func (a *Agent) phoneConnectStart(in phoneConnectReq, wait bool) error {
	hs := a.phoneHub()
	a.mu.Lock()
	if hs.conn.state == pcWorking {
		a.mu.Unlock()
		return errors.New("already connecting")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	done := make(chan struct{})
	hs.conn = pcState{state: pcWorking, step: stepBluetooth, cancel: cancel, done: done}
	if in.Phone != nil {
		hs.conn.phone = *in.Phone
	}
	a.mu.Unlock()
	go func() {
		defer close(done)
		defer cancel()
		a.phoneConnectRun(ctx, in)
	}()
	if wait {
		<-done
	}
	return nil
}

func (a *Agent) phoneConnectRun(ctx context.Context, in phoneConnectReq) {
	hs := a.phoneHub()
	set := func(f func(c *pcState)) {
		a.mu.Lock()
		f(&hs.conn)
		a.mu.Unlock()
	}
	step := func(s string) { set(func(c *pcState) { c.step = s }) }
	fail := func(s, msg string) {
		set(func(c *pcState) { c.state, c.step, c.err = pcFailed, s, msg })
		a.Logf("connect via phone: %s: %s", s, msg)
	}
	or := func(s, d string) string {
		if s == "" {
			return d
		}
		return s
	}

	// bluetooth: what the app found
	fallback := false
	note := ""
	switch in.BLE {
	case "ok":
		if in.Phone == nil {
			fail(stepBluetooth, "The Bluetooth session gave no phone.")
			return
		}
	case "auth":
		fail(stepLogin, "The phone refused this Mac's login: are both paired with the same home server? "+in.BLEError)
		return
	case "hotspot":
		fail(stepHotspot, "The phone couldn't start its hotspot: "+or(in.BLEError, "no reason given")+".")
		return
	case "hub":
		fail(stepPhoneHub, "The phone couldn't start sharing its bonded connection: "+or(in.BLEError, "no reason given")+".")
		return
	default:
		why, ok := pcFallbackWhy(in.BLE, in.BLEError)
		if !ok {
			fail(stepBluetooth, fmt.Sprintf("Unknown Bluetooth result %q.", in.BLE))
			return
		}
		fallback = true
		note = "No Bluetooth: " + why + ". Used the phone hotspot this Mac is on."
		set(func(c *pcState) { c.via = "hotspot" })
	}
	if !fallback {
		set(func(c *pcState) { c.via = "bluetooth" })
	}

	// wifi: the app joined, or we do
	if !fallback && !in.Joined && in.SSID != "" {
		step(stepWiFi)
		if a.joinWiFi == nil {
			fail(stepWiFi, fmt.Sprintf("Join the phone's hotspot %q yourself, then click again.", in.SSID))
			return
		}
		if err := a.joinWiFi(ctx, in.SSID, in.Pass); err != nil {
			fail(stepWiFi, fmt.Sprintf("Couldn't join the phone's hotspot %q: %v.", in.SSID, err))
			return
		}
		a.Logf("connect via phone: joined the phone's hotspot %q", in.SSID)
	}

	// find: the phone's hub, by its id; on the fallback any phone hub
	step(stepFind)
	logins := a.hubLogins()
	if len(logins) == 0 {
		fail(stepFind, "Connect to your home server once first: the phone's password comes from it.")
		return
	}
	want := ""
	if in.Phone != nil {
		want = in.Phone.ID
	}
	wait := a.pcFindWait
	if wait == 0 {
		wait = 45 * time.Second
	}
	// found through the hotspot's router, a hub has no id of its own: it is
	// the phone if it took the login for the phone's key id
	var phoneLogin *hubLogin
	if in.Phone != nil {
		if l, ok := logins[in.Phone.KeyID]; ok {
			phoneLogin = &l
		}
	}
	isPhone := func(c hubCand) bool {
		if want != "" && c.ID == want {
			return true
		}
		return c.Via == "router" && phoneLogin != nil && c.user == phoneLogin.user && c.pass == phoneLogin.pass
	}
	deadline := time.Now().Add(wait)
	var cand *hubCand
	unbonded := false
	for {
		found := a.findHubs(ctx, logins)
		a.mu.Lock()
		hs.found, hs.asked = found, time.Now()
		a.mu.Unlock()
		for i := range found {
			// a hub that says it isn't bonded (an iPhone: mobile data only)
			// isn't what this click promises
			if found[i].Bonded == "0" {
				unbonded = true
				continue
			}
			if isPhone(found[i]) || (fallback && cand == nil) {
				cand = &found[i]
				if isPhone(found[i]) {
					break
				}
			}
		}
		if cand != nil || time.Now().After(deadline) || ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(min(3*time.Second, wait/4)):
		}
	}
	if cand == nil {
		switch {
		case unbonded && fallback:
			fail(stepFind, "Only a phone sharing its mobile data, not a bonded connection, answered (an iPhone). To use it anyway, pick it in the dashboard's \"Use a phone's connection\".")
		case fallback:
			fail(stepFind, "No phone hub found on this network. "+note+" Turn on the phone's hotspot and Share bonded connection in tailbond, join the hotspot, and click again.")
		default:
			fail(stepFind, "On the phone's hotspot, but its phone hub didn't answer. Is Share bonded connection on, and tailbond connected on the phone?")
		}
		return
	}
	set(func(c *pcState) {
		c.hubID = cand.ID
		if c.phone.Label == "" {
			c.phone.Label = cand.Label
		}
	})

	// use: as the dashboard's "Use"
	step(stepUse)
	pac := in.PAC == nil || *in.PAC
	if err := a.useHub(ctx, cand.ID, pac); err != nil {
		fail(stepUse, "Couldn't use the phone hub: "+err.Error()+".")
		return
	}

	// internet: a small fetch through the forwarder
	step(stepInternet)
	a.mu.Lock()
	mine := hs.use
	local := ""
	if mine != nil {
		local = mine.ln.Addr().String()
	}
	a.mu.Unlock()
	if err := a.fetchThrough(ctx, local); err != nil {
		// only the hub this click set up: the dashboard may have picked
		// another meanwhile
		hubID := cand.ID
		a.stopHubIf("", func(u *hubUse) bool { return u == mine && u.cand.ID == hubID })
		fail(stepInternet, "The phone hub answers, but nothing reaches the internet through it ("+err.Error()+"), so the proxy is off again. Is tailbond connected on the phone?")
		return
	}
	set(func(c *pcState) { c.state, c.step, c.err, c.note = pcBonded, "", "", note })
	a.Logf("connect via phone: bonded via %q (%s)", cand.Label, map[bool]string{true: "a hotspot this Mac was on", false: "Bluetooth"}[fallback])
}

// fetchThrough GETs the check URL through the forwarder at local.
func (a *Agent) fetchThrough(ctx context.Context, local string) error {
	if local == "" {
		return errors.New("no forwarder")
	}
	u := a.pcCheckURL
	if u == "" {
		u = os.Getenv("TB_PHONE_CHECK_URL")
	}
	if u == "" {
		u = pcDefaultCheckURL
	}
	d, err := xproxy.SOCKS5("tcp", local, nil, &net.Dialer{Timeout: 5 * time.Second})
	if err != nil {
		return err
	}
	cl := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{
		DialContext: d.(xproxy.ContextDialer).DialContext,
		Proxy:       nil,
	},
		// a captive portal answers with a redirect or its own page: not internet
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	// generate_204 answers 204 and nothing else (as captive.ProbesFor);
	// another check URL (the e2e's token) any 2xx
	if (strings.HasSuffix(u, "generate_204") && resp.StatusCode != http.StatusNoContent) || resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s answered %s (a sign-in page?)", u, resp.Status)
	}
	return nil
}

// phoneDisconnect: the second click. Stops a run that's still going, stops
// using the hub (the proxy settings go back).
func (a *Agent) phoneDisconnect() {
	hs := a.phoneHub()
	a.mu.Lock()
	c := hs.conn
	a.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
		<-c.done
	}
	a.mu.Lock()
	inUse := hs.use != nil
	a.mu.Unlock()
	note := ""
	if inUse {
		note = PhoneDisconnectedNote
		a.stopHub(note)
	}
	a.mu.Lock()
	hs.conn = pcState{state: pcIdle, note: note}
	a.mu.Unlock()
}

// phoneConnectStatusLocked: a.mu held.
func (a *Agent) phoneConnectStatusLocked(hs *phoneHubState) *status.PhoneConnect {
	c := hs.conn
	out := &status.PhoneConnect{State: c.state, Step: c.step, Via: c.via, Phone: c.phone.Label, Network: c.phone.Network,
		Paths: c.phone.Paths, Battery: c.phone.Battery, Error: c.err, Note: c.note}
	if out.State == "" {
		out.State = pcIdle
	}
	// bonded only while that hub is in use (the dashboard may have stopped
	// it, or the phone went away)
	if c.state == pcBonded && (hs.use == nil || hs.use.cand.ID != c.hubID) {
		*out = status.PhoneConnect{State: pcIdle, Note: hs.note}
	}
	if out.State != pcWorking && out.State != pcFailed {
		out.Step = ""
	}
	return out
}

// hPhoneConnect: GET where "Connect via phone" is; POST {"connect": {...}}
// to start (with "wait": true, answer when done), {"disconnect": true} to
// stop.
func (a *Agent) hPhoneConnect(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient || a.phoneHub() == nil {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	a.hubAsked()
	if r.Method == http.MethodPost {
		var in struct {
			Connect    *phoneConnectReq `json:"connect"`
			Disconnect bool             `json:"disconnect"`
			Wait       bool             `json:"wait"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in); err != nil {
			httpErr(w, http.StatusBadRequest, err)
			return
		}
		switch {
		case in.Disconnect:
			a.phoneDisconnect()
		case in.Connect != nil:
			if err := a.phoneConnectStart(*in.Connect, in.Wait); err != nil {
				httpErr(w, http.StatusConflict, err)
				return
			}
		default:
			httpErr(w, http.StatusBadRequest, errors.New(`want {"connect": {...}} or {"disconnect": true}`))
			return
		}
	}
	hs := a.phoneHub()
	a.mu.Lock()
	st := a.phoneConnectStatusLocked(hs)
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, st)
}

// bleLogin is the menu bar app's Bluetooth login for one phone.
type bleLogin struct {
	KeyID  string `json:"kid"`
	User   string `json:"user"`
	Secret string `json:"secret"`
}

// hPhoneConnectKeys gives the menu bar app (this Mac only) its Bluetooth
// logins: one per phone the home server gave this Mac a pass for.
func (a *Agent) hPhoneConnectKeys(w http.ResponseWriter, r *http.Request) {
	if a.role != config.RoleClient || a.phoneHub() == nil {
		httpErr(w, http.StatusNotFound, errNotFound)
		return
	}
	if !remoteLoopback(r) {
		httpErr(w, http.StatusForbidden, errors.New("only from this machine"))
		return
	}
	a.mu.Lock()
	p := a.client.payload
	a.mu.Unlock()
	out := []bleLogin{}
	if p != nil {
		for kid, hp := range a.hubPasses(p) {
			u, s := bleproto.Credentials(hp.key, p.User, hp.exp)
			out = append(out, bleLogin{kid, u, s})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"logins": out})
}

func remoteLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// macJoinWiFi joins a Wi-Fi network with networksetup (tailbondd runs as
// root, so this needs no Location permission, unlike CoreWLAN in the app).
func macJoinWiFi(ctx context.Context, ssid, pass string) error {
	out, err := exec.CommandContext(ctx, "networksetup", "-listallhardwareports").Output()
	if err != nil {
		return fmt.Errorf("networksetup: %v", err)
	}
	dev := wifiDevice(string(out))
	if dev == "" {
		return errors.New("this Mac has no Wi-Fi")
	}
	// networksetup says why it failed on stdout, with exit status 0
	b, err := exec.CommandContext(ctx, "networksetup", "-setairportnetwork", dev, ssid, pass).CombinedOutput()
	if msg := strings.TrimSpace(string(b)); err != nil || msg != "" {
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(strings.ReplaceAll(msg, pass, "…"))
	}
	return nil
}

// wifiDevice finds the Wi-Fi device (en0) in networksetup -listallhardwareports.
func wifiDevice(out string) string {
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == "Hardware Port: Wi-Fi" && i+1 < len(lines) {
			if d, ok := strings.CutPrefix(strings.TrimSpace(lines[i+1]), "Device: "); ok {
				return strings.TrimSpace(d)
			}
		}
	}
	return ""
}
