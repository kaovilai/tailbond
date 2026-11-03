package share

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

// echoServer stands in for the home server: it answers every datagram with
// "re:" + the datagram.
func echoServer(t *testing.T) *net.UDPConn {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buf := make([]byte, 65536)
		for {
			n, from, err := c.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = c.WriteToUDP(append([]byte("re:"), buf[:n]...), from)
		}
	}()
	t.Cleanup(func() { c.Close() })
	return c
}

func dialUDP(ctx context.Context, server string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "udp", server)
}

type rig struct {
	sharer *Sharer
	lanAt  *net.UDPAddr
	cancel context.CancelFunc
}

func startSharer(t *testing.T, cfg SharerConfig) *rig {
	t.Helper()
	srv := echoServer(t)
	if cfg.Server == "" {
		cfg.Server = srv.LocalAddr().String()
	}
	if cfg.Key == nil {
		cfg.Key = testKey
	}
	if cfg.Dial == nil {
		cfg.Dial = dialUDP
	}
	cfg.ID, cfg.Label = "pixel-1", "Pixel's 5G"
	s, err := NewSharer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	lan, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = s.Serve(ctx, lan) }()
	t.Cleanup(cancel)
	return &rig{sharer: s, lanAt: lan.LocalAddr().(*net.UDPAddr), cancel: cancel}
}

func startLink(t *testing.T, at *net.UDPAddr, key []byte, hub string) *Link {
	t.Helper()
	l, err := NewLink(LinkConfig{Sharer: at, Key: key, HubID: hub})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go l.Run(ctx)
	t.Cleanup(cancel)
	return l
}

func waitState(t *testing.T, l *Link, want string, within time.Duration) LinkStatus {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if st := l.Status(); st.State == want {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("link state %q, want %q (%+v)", l.Status().State, want, l.Status())
	return LinkStatus{}
}

// mqvpnSide is the hub's mqvpn path socket talking to the link.
func mqvpnSide(t *testing.T, l *Link) *net.UDPConn {
	t.Helper()
	c, err := net.DialUDP("udp4", nil, l.Local())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func roundTrip(t *testing.T, c *net.UDPConn, msg string) {
	t.Helper()
	if _, err := c.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2048)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("no reply through the share: %v", err)
	}
	if got := string(buf[:n]); got != "re:"+msg {
		t.Fatalf("reply %q, want %q", got, "re:"+msg)
	}
}

func TestFramesRoundTrip(t *testing.T) {
	var hn, sn [NonceLen]byte
	hn[0], sn[15] = 1, 2
	h := Hello{HubNonce: hn, HubID: "slate7"}
	var h2 Hello
	if err := h2.Unmarshal(h.Marshal()); err != nil || h2 != h {
		t.Fatalf("hello: %v %+v", err, h2)
	}
	c := Challenge{HubNonce: hn, SharerNonce: sn, SharerID: "id", Label: "Pixel's 5G"}
	var c2 Challenge
	if err := c2.Unmarshal(c.Marshal()); err != nil || c2 != c {
		t.Fatalf("challenge: %v %+v", err, c2)
	}
	w := Welcome{HubNonce: hn, Session: [SessionLen]byte{9}, CapLeft: 12345}
	w.MAC = WelcomeMAC(testKey, hn, sn, w.Session, w.CapLeft)
	var w2 Welcome
	if err := w2.Unmarshal(w.Marshal()); err != nil || w2 != w {
		t.Fatalf("welcome: %v", err)
	}
	a := KAAck{Session: w.Session, Seq: 7, Up: 1, Down: 2, CapLeft: NoCap, Battery: 55, Flags: FlagCharging}
	var a2 KAAck
	if err := a2.Unmarshal(a.Marshal()); err != nil || a2 != a {
		t.Fatalf("ka ack: %v %+v", err, a2)
	}
	if d := Data(nil, w.Session, []byte("xy")); len(d) != DataHeader+2 || d[0] != TypeData {
		t.Fatalf("data frame %x", d)
	}
	// truncated and wrong-version frames are refused
	if (&h2).Unmarshal(h.Marshal()[:5]) == nil {
		t.Fatal("short hello accepted")
	}
	bad := h.Marshal()
	bad[1] = 9
	if (&h2).Unmarshal(bad) == nil {
		t.Fatal("version 9 accepted")
	}
	// ids are capped on the wire
	long := Hello{HubID: strings.Repeat("x", 200)}
	if err := h2.Unmarshal(long.Marshal()); err != nil || len(h2.HubID) != MaxIDLen {
		t.Fatalf("long id: %v %d", err, len(h2.HubID))
	}
}

func TestMACsBindEverything(t *testing.T) {
	var hn, sn [NonceLen]byte
	m := AuthMAC(testKey, hn, sn, "slate7")
	if Equal(m, AuthMAC(testKey, hn, sn, "other")) || Equal(m, AuthMAC([]byte("k2"), hn, sn, "slate7")) {
		t.Fatal("auth MAC ignores the hub id or the key")
	}
	sn[0] = 1
	if Equal(m, AuthMAC(testKey, hn, sn, "slate7")) {
		t.Fatal("auth MAC ignores the sharer nonce")
	}
	if KeyID(testKey) == KeyID([]byte("other")) || len(KeyID(testKey)) != 8 {
		t.Fatal("key id")
	}
}

func TestShareCarriesTrafficBothWays(t *testing.T) {
	r := startSharer(t, SharerConfig{})
	l := startLink(t, r.lanAt, testKey, "slate7")
	st := waitState(t, l, StateReady, 3*time.Second)
	if st.Label != "Pixel's 5G" || st.SharerID != "pixel-1" || st.CapLeft != NoCap {
		t.Fatalf("status %+v", st)
	}
	c := mqvpnSide(t, l)
	// each datagram also (re)teaches the link mqvpn's address
	for i := 0; i < 5; i++ {
		roundTrip(t, c, "quic-"+string(rune('a'+i)))
	}
	ss := r.sharer.Sessions()
	if len(ss) != 1 || ss[0].HubID != "slate7" || ss[0].Up == 0 || ss[0].Down == 0 {
		t.Fatalf("sharer sessions %+v", ss)
	}
	if r.sharer.Used() != ss[0].Up+ss[0].Down {
		t.Fatalf("used %d, sessions %+v", r.sharer.Used(), ss)
	}
	if st := l.Status(); st.Up == 0 || st.Down == 0 {
		t.Fatalf("link counters %+v", st)
	}
}

func TestWrongKeyGetsNoSession(t *testing.T) {
	r := startSharer(t, SharerConfig{})
	l := startLink(t, r.lanAt, []byte("another server's key............."), "slate7")
	st := waitState(t, l, StateRefused, 6*time.Second)
	if !strings.Contains(st.Reason, "different home server") {
		t.Fatalf("reason %q", st.Reason)
	}
	if len(r.sharer.Sessions()) != 0 {
		t.Fatal("a wrong key opened a session")
	}
}

func TestAllowedHubs(t *testing.T) {
	r := startSharer(t, SharerConfig{AllowedHubs: []string{"slate7"}})
	l := startLink(t, r.lanAt, testKey, "laptop")
	st := waitState(t, l, StateRefused, 3*time.Second)
	if st.Reason != ReasonText(ReasonNotAllowed) {
		t.Fatalf("reason %q", st.Reason)
	}
	ok := startLink(t, r.lanAt, testKey, "slate7")
	waitState(t, ok, StateReady, 3*time.Second)
}

func TestCapStopsSharing(t *testing.T) {
	r := startSharer(t, SharerConfig{CapBytes: 40})
	l := startLink(t, r.lanAt, testKey, "slate7")
	if st := waitState(t, l, StateReady, 3*time.Second); st.CapLeft != 40 {
		t.Fatalf("cap left %d", st.CapLeft)
	}
	c := mqvpnSide(t, l)
	_, _ = c.Write([]byte("hi"))
	time.Sleep(100 * time.Millisecond)
	payload := strings.Repeat("x", 30)
	for i := 0; i < 4; i++ {
		_, _ = c.Write([]byte(payload))
		time.Sleep(50 * time.Millisecond)
	}
	st := waitState(t, l, StateRefused, 3*time.Second)
	if st.Reason != ReasonText(ReasonCap) {
		t.Fatalf("reason %q", st.Reason)
	}
	if u := r.sharer.Used(); u > 40+2*(30+3) {
		t.Fatalf("used %d bytes past a 40-byte cap", u)
	}
	// a new hub is refused while the cap is used up
	l2 := startLink(t, r.lanAt, testKey, "laptop")
	if st := waitState(t, l2, StateRefused, 3*time.Second); st.Reason != ReasonText(ReasonCap) {
		t.Fatalf("reason %q", st.Reason)
	}
}

func TestLowBatteryEndsTheSession(t *testing.T) {
	var low atomic.Bool
	r := startSharer(t, SharerConfig{Check: func() (byte, byte, bool) {
		if low.Load() {
			return ReasonBattery, 14, false
		}
		return ReasonNone, 80, true
	}})
	l := startLink(t, r.lanAt, testKey, "slate7")
	waitState(t, l, StateReady, 3*time.Second)
	low.Store(true)
	st := waitState(t, l, StateRefused, 8*time.Second)
	if st.Reason != ReasonText(ReasonBattery) {
		t.Fatalf("reason %q", st.Reason)
	}
}

func TestKeepaliveReportsBattery(t *testing.T) {
	r := startSharer(t, SharerConfig{Check: func() (byte, byte, bool) { return ReasonNone, 63, true }})
	l := startLink(t, r.lanAt, testKey, "slate7")
	waitState(t, l, StateReady, 3*time.Second)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if st := l.Status(); st.Battery == 63 && st.Charging {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("battery never reported: %+v", l.Status())
}

func TestSharerGoneMeansUnreachable(t *testing.T) {
	// nothing listens here
	dead, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	at := dead.LocalAddr().(*net.UDPAddr)
	dead.Close()
	l := startLink(t, at, testKey, "slate7")
	st := waitState(t, l, StateUnreachable, 6*time.Second)
	if st.Unanswered < 3 {
		t.Fatalf("unanswered %d", st.Unanswered)
	}
}

func TestStrangersCantUseASession(t *testing.T) {
	r := startSharer(t, SharerConfig{})
	l := startLink(t, r.lanAt, testKey, "slate7")
	waitState(t, l, StateReady, 3*time.Second)
	l.mu.Lock()
	sess := l.session
	l.mu.Unlock()
	// the same session id from another socket is ignored
	x, _ := net.DialUDP("udp4", nil, r.lanAt)
	defer x.Close()
	_, _ = x.Write(Data(nil, sess, []byte("spoof")))
	_ = x.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := x.Read(make([]byte, 100)); err == nil {
		t.Fatal("a stranger got an answer through someone else's session")
	}
	if ss := r.sharer.Sessions(); len(ss) != 1 || ss[0].Up != 0 {
		t.Fatalf("spoofed bytes counted: %+v", ss)
	}
}
