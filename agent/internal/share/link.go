package share

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"time"
)

// Hub timing.
const (
	helloEvery = time.Second
	// A keepalive a second, gone after 3.5 s without an answer: mqvpn can't
	// tell a relay path from a dead one by itself, so the hub has to.
	keepaliveEvery = time.Second
	linkTimeout    = 3500 * time.Millisecond
	refusedRetry   = 30 * time.Second
)

// Link states.
const (
	StateConnecting  = "connecting"  // handshake under way
	StateReady       = "ready"       // session open: the path can carry traffic
	StateRefused     = "refused"     // the sharer said no (Reason)
	StateUnreachable = "unreachable" // no answer to HELLO
	StateLost        = "lost"        // the session stopped answering
)

// LinkConfig is the hub's end of one share.
type LinkConfig struct {
	// Sharer is the sharer's LAN address.
	Sharer *net.UDPAddr
	// Key is this hub's pass for the sharer (PassKey, from the home
	// server), valid until Exp (unix seconds); HubID is the name it was
	// issued to. SetPass replaces them.
	Key   []byte
	Exp   uint64
	HubID string
	// LocalPort (optional) keeps mqvpn's relay address when a link is
	// replaced (the sharer moved): 0 = any port.
	LocalPort int
	Logf      func(format string, args ...any)
	// OnChange (optional) is called, without locks held, when the state
	// changes.
	OnChange func(LinkStatus)
}

// LinkStatus is a share as the hub sees it.
type LinkStatus struct {
	State    string    `json:"state"`
	SharerID string    `json:"sharer_id,omitempty"`
	Label    string    `json:"label,omitempty"`
	Addr     string    `json:"addr"`
	Reason   string    `json:"reason,omitempty"`
	Up       uint64    `json:"up_bytes"`   // payload bytes hub → sharer
	Down     uint64    `json:"down_bytes"` // payload bytes sharer → hub
	CapLeft  uint64    `json:"cap_left"`   // NoCap = none
	Battery  int       `json:"battery"`    // percent, -1 unknown
	Charging bool      `json:"charging"`
	Since    time.Time `json:"since,omitempty"`
	// Unanswered HELLOs since the last answer (client isolation shows here).
	Unanswered int `json:"unanswered"`
}

// Link carries one mqvpn relay path: mqvpn sends to Local(), the link
// frames each datagram for the sharer, and the sharer's frames come back
// to mqvpn's socket.
type Link struct {
	cfg   LinkConfig
	local *net.UDPConn // 127.0.0.1:port, mqvpn's relay address
	lan   *net.UDPConn

	mu        sync.Mutex
	key       []byte
	exp       uint64
	st        LinkStatus
	hubNonce  [NonceLen]byte
	sharerNon [NonceLen]byte
	session   [SessionLen]byte
	mqvpn     *net.UDPAddr // mqvpn's path socket, learned from its first datagram
	lastAck   time.Time
	retryAt   time.Time
	seq       uint32
	authTries int // CHALLENGEs answered without a WELCOME
}

// NewLink opens the link's sockets.
func NewLink(cfg LinkConfig) (*Link, error) {
	if cfg.Sharer == nil || len(cfg.Key) == 0 {
		return nil, errors.New("share: link needs a sharer address and a key")
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	local, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: cfg.LocalPort})
	if err != nil && cfg.LocalPort != 0 {
		local, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	}
	if err != nil {
		return nil, err
	}
	lan, err := net.ListenUDP("udp", nil)
	if err != nil {
		local.Close()
		return nil, err
	}
	return &Link{cfg: cfg, key: cfg.Key, exp: cfg.Exp, local: local, lan: lan, st: LinkStatus{State: StateConnecting, Addr: cfg.Sharer.String(), CapLeft: NoCap, Battery: -1}}, nil
}

// Local is the address mqvpn's relay path sends to.
func (l *Link) Local() *net.UDPAddr { return l.local.LocalAddr().(*net.UDPAddr) }

// Status is a snapshot.
func (l *Link) Status() LinkStatus {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.st
}

// SetPass replaces the pass for the next handshakes (a running session
// keeps its own until the sharer ends it). A link refused for its old pass
// tries again at once.
func (l *Link) SetPass(key []byte, exp uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if string(key) == string(l.key) && exp == l.exp {
		return
	}
	l.key, l.exp = key, exp
	l.authTries = 0
	if l.st.State == StateRefused {
		l.retryAt = time.Time{}
	}
}

// Ready reports an open session.
func (l *Link) Ready() bool { return l.Status().State == StateReady }

// Run drives the link until ctx ends, then says BYE and closes.
func (l *Link) Run(ctx context.Context) {
	go l.fromLAN()
	go l.fromMqvpn()
	t := time.NewTicker(helloEvery)
	defer t.Stop()
	for {
		l.tick(time.Now())
		select {
		case <-ctx.Done():
			l.mu.Lock()
			s, ready := l.session, l.st.State == StateReady
			l.mu.Unlock()
			if ready {
				_, _ = l.lan.WriteToUDP(Bye(s, ReasonPaused), l.cfg.Sharer)
			}
			_ = l.lan.Close()
			_ = l.local.Close()
			return
		case <-t.C:
		}
	}
}

func (l *Link) tick(now time.Time) {
	l.mu.Lock()
	before := l.st.State
	defer l.notify(before)
	defer l.mu.Unlock()
	switch l.st.State {
	case StateReady:
		if now.Sub(l.lastAck) > linkTimeout {
			l.st.State, l.st.Reason = StateLost, ReasonText(ReasonIdle)
			l.cfg.Logf("share: %s (%s) stopped answering", l.st.Label, l.cfg.Sharer)
			l.hello(now)
			return
		}
		if now.Sub(l.lastAck) >= keepaliveEvery/2 || l.seq == 0 {
			l.seq++
			_, _ = l.lan.WriteToUDP(Keepalive(l.session, l.seq), l.cfg.Sharer)
		}
	case StateRefused:
		if now.After(l.retryAt) {
			l.hello(now)
		}
	default:
		l.hello(now)
	}
}

// hello (re)starts the handshake. Caller holds l.mu.
func (l *Link) hello(now time.Time) {
	if l.st.State == StateRefused {
		l.st.State = StateConnecting
	}
	if l.authTries >= 3 {
		// answers, but never accepts our key: another home server's device
		l.authTries = 0
		l.st.State, l.st.Reason = StateRefused, "it's paired with a different home server"
		l.retryAt = now.Add(refusedRetry)
		return
	}
	_, _ = rand.Read(l.hubNonce[:])
	l.st.Unanswered++
	if l.st.Unanswered >= 3 && l.st.State != StateLost && l.st.State != StateRefused {
		l.st.State = StateUnreachable
	}
	h := Hello{HubNonce: l.hubNonce, Exp: l.exp, HubID: l.cfg.HubID}
	_, _ = l.lan.WriteToUDP(h.Marshal(), l.cfg.Sharer)
}

func (l *Link) fromLAN() {
	buf := make([]byte, 65536)
	for {
		n, from, err := l.lan.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n < 1 || !from.IP.Equal(l.cfg.Sharer.IP) || from.Port != l.cfg.Sharer.Port {
			continue
		}
		b := buf[:n]
		if b[0] == TypeData {
			l.mu.Lock()
			ok := l.st.State == StateReady && len(b) >= DataHeader && string(b[1:DataHeader]) == string(l.session[:])
			dst := l.mqvpn
			if ok {
				l.st.Down += uint64(n - DataHeader)
			}
			l.mu.Unlock()
			if ok && dst != nil {
				_, _ = l.local.WriteToUDP(b[DataHeader:], dst)
			}
			continue
		}
		l.control(b, time.Now())
	}
}

// notify reports a state change; deferred before the unlock, so it runs
// after it.
func (l *Link) notify(before string) {
	if l.cfg.OnChange == nil {
		return
	}
	st := l.Status()
	if st.State != before {
		l.cfg.OnChange(st)
	}
}

func (l *Link) control(b []byte, now time.Time) {
	l.mu.Lock()
	before := l.st.State
	defer l.notify(before)
	defer l.mu.Unlock()
	switch b[0] {
	case TypeChallenge:
		var c Challenge
		if c.Unmarshal(b) != nil || c.HubNonce != l.hubNonce || l.st.State == StateReady {
			return
		}
		l.sharerNon = c.SharerNonce
		l.st.Unanswered = 0
		l.authTries++
		l.st.SharerID, l.st.Label = c.SharerID, c.Label
		a := Auth{HubNonce: c.HubNonce, SharerNonce: c.SharerNonce, MAC: AuthMAC(l.key, c.HubNonce, c.SharerNonce, l.cfg.HubID)}
		_, _ = l.lan.WriteToUDP(a.Marshal(), l.cfg.Sharer)
	case TypeWelcome:
		var w Welcome
		if w.Unmarshal(b) != nil || w.HubNonce != l.hubNonce || l.st.State == StateReady ||
			!Equal(w.MAC, WelcomeMAC(l.key, w.HubNonce, l.sharerNon, w.Session, w.CapLeft)) {
			return
		}
		l.session = w.Session
		l.authTries = 0
		l.st.State, l.st.Reason, l.st.Unanswered = StateReady, "", 0
		l.st.CapLeft, l.st.Since = w.CapLeft, now
		l.lastAck, l.seq = now, 0
		l.cfg.Logf("share: %s (%s) ready", l.st.Label, l.cfg.Sharer)
	case TypeRefuse:
		var r Refuse
		if r.Unmarshal(b) != nil || r.HubNonce != l.hubNonce ||
			!Equal(r.MAC, RefuseMAC(l.key, r.HubNonce, l.sharerNon, r.Reason)) {
			return
		}
		l.st.State, l.st.Reason, l.st.Unanswered = StateRefused, ReasonText(r.Reason), 0
		l.retryAt = now.Add(refusedRetry)
		l.cfg.Logf("share: %s (%s) refused: %s", l.st.Label, l.cfg.Sharer, l.st.Reason)
	case TypeKAAck:
		var a KAAck
		if a.Unmarshal(b) != nil || a.Session != l.session || l.st.State != StateReady {
			return
		}
		l.lastAck = now
		l.st.CapLeft, l.st.Charging = a.CapLeft, a.Flags&FlagCharging != 0
		l.st.Battery = -1
		if a.Battery <= 100 {
			l.st.Battery = int(a.Battery)
		}
	case TypeBye:
		s, ok := SessionOf(b)
		if !ok || s != l.session || l.st.State != StateReady || len(b) < 1+SessionLen+1 {
			return
		}
		reason := b[1+SessionLen]
		l.st.State, l.st.Reason = StateRefused, ReasonText(reason)
		l.retryAt = now.Add(refusedRetry)
		if reason == ReasonReplaced || reason == ReasonIdle || reason == ReasonExpired {
			l.retryAt = now // with a fresh pass, if the server gave one
		}
		l.cfg.Logf("share: %s (%s) ended: %s", l.st.Label, l.cfg.Sharer, l.st.Reason)
	}
}

func (l *Link) fromMqvpn() {
	buf := make([]byte, 65536)
	out := make([]byte, 0, 65536+DataHeader)
	for {
		n, from, err := l.local.ReadFromUDP(buf)
		if err != nil {
			return
		}
		l.mu.Lock()
		l.mqvpn = from
		ready, s := l.st.State == StateReady, l.session
		if ready {
			l.st.Up += uint64(n)
		}
		l.mu.Unlock()
		if !ready {
			continue // mqvpn retries; the path comes up once the session does
		}
		out = Data(out, s, buf[:n])
		_, _ = l.lan.WriteToUDP(out, l.cfg.Sharer)
	}
}
