package share

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Sharer timing.
const (
	handshakeTTL = 10 * time.Second
	sessionIdle  = 30 * time.Second
	maxPending   = 64
	maxSessions  = 8
)

// SharerConfig is one sharer's setup.
type SharerConfig struct {
	// ID identifies this sharer (stable per device); Label is what hubs show
	// ("Pixel's 5G").
	ID, Label string
	// Key is the home server's share key.
	Key []byte
	// Server is the home server's mqvpn address (host:port). The relay
	// forwards to it and nowhere else.
	Server string
	// Dial opens the upstream for one session: a UDP socket to server over
	// the cellular network (bound to it).
	Dial func(ctx context.Context, server string) (net.Conn, error)
	// CapBytes stops sharing once this many payload bytes went through in
	// total (0 = no cap); UsedBytes is what was used before this run.
	CapBytes, UsedBytes uint64
	// AllowedHubs: hub ids that may use this share; empty = any hub paired
	// with the same server.
	AllowedHubs []string
	// Check (optional) says whether sharing may go on right now: a Reason*
	// (ReasonNone = yes) plus battery percent (255 = unknown) and charging.
	Check func() (reason byte, battery byte, charging bool)
	// Logf (optional) logs.
	Logf func(format string, args ...any)
}

type pending struct {
	sharerNonce [NonceLen]byte
	hubID       string
	addr        string
	at          time.Time
}

type session struct {
	id       [SessionLen]byte
	hubID    string
	hubAddr  net.Addr
	up       net.Conn
	upBytes  atomic.Uint64
	dnBytes  atomic.Uint64
	lastSeen atomic.Int64 // unix nanos
	since    time.Time
	once     sync.Once
}

// SessionStats is one hub's use of the share.
type SessionStats struct {
	HubID string    `json:"hub"`
	Addr  string    `json:"addr"`
	Up    uint64    `json:"up_bytes"`
	Down  uint64    `json:"down_bytes"`
	Since time.Time `json:"since"`
}

// Sharer is the relay on the device that shares its cellular data.
type Sharer struct {
	cfg  SharerConfig
	conn net.PacketConn

	mu       sync.Mutex
	pending  map[[NonceLen]byte]pending
	sessions map[[SessionLen]byte]*session
	byHub    map[string]*session
	used     atomic.Uint64
}

// NewSharer checks cfg.
func NewSharer(cfg SharerConfig) (*Sharer, error) {
	if len(cfg.Key) == 0 || cfg.Server == "" || cfg.Dial == nil {
		return nil, errors.New("share: sharer needs a key, a server and a dialer")
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	s := &Sharer{
		cfg:      cfg,
		pending:  map[[NonceLen]byte]pending{},
		sessions: map[[SessionLen]byte]*session{},
		byHub:    map[string]*session{},
	}
	s.used.Store(cfg.UsedBytes)
	return s, nil
}

// Used is the payload bytes shared so far (UsedBytes included).
func (s *Sharer) Used() uint64 { return s.used.Load() }

func (s *Sharer) capLeft() uint64 {
	if s.cfg.CapBytes == 0 {
		return NoCap
	}
	u := s.used.Load()
	if u >= s.cfg.CapBytes {
		return 0
	}
	return s.cfg.CapBytes - u
}

func (s *Sharer) check() (byte, byte, bool) {
	if s.cfg.Check == nil {
		return ReasonNone, 255, false
	}
	return s.cfg.Check()
}

// Sessions lists the hubs using the share now.
func (s *Sharer) Sessions() []SessionStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SessionStats, 0, len(s.sessions))
	for _, x := range s.sessions {
		out = append(out, SessionStats{HubID: x.hubID, Addr: x.hubAddr.String(), Up: x.upBytes.Load(), Down: x.dnBytes.Load(), Since: x.since})
	}
	slices.SortFunc(out, func(a, b SessionStats) int { return a.Since.Compare(b.Since) })
	return out
}

// Serve answers hubs on conn (the LAN listener) until ctx ends.
func (s *Sharer) Serve(ctx context.Context, conn net.PacketConn) error {
	s.conn = conn
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	go s.janitor(ctx)
	defer s.closeAll(ReasonPaused)
	buf := make([]byte, 65536)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		if n < 1 {
			continue
		}
		s.handle(ctx, buf[:n], addr)
	}
}

func (s *Sharer) handle(ctx context.Context, b []byte, from net.Addr) {
	switch b[0] {
	case TypeData:
		x := s.sessionFrom(b, from)
		if x == nil {
			return
		}
		if s.capLeft() == 0 {
			s.end(x, ReasonCap, true)
			return
		}
		p := b[DataHeader:]
		// counted before it leaves, so the counters and the cap are current
		// by the time anyone sees the datagram's effect; undone on failure
		x.upBytes.Add(uint64(len(p)))
		s.used.Add(uint64(len(p)))
		if _, err := x.up.Write(p); err != nil {
			x.upBytes.Add(^uint64(len(p) - 1))
			s.used.Add(^uint64(len(p) - 1))
		}
		x.lastSeen.Store(time.Now().UnixNano())
	case TypeKeepalive:
		x := s.sessionFrom(b, from)
		if x == nil || len(b) < 1+SessionLen+4 {
			return
		}
		x.lastSeen.Store(time.Now().UnixNano())
		reason, battery, charging := s.check()
		if reason == ReasonNone && s.capLeft() == 0 {
			reason = ReasonCap
		}
		if reason != ReasonNone {
			s.end(x, reason, true)
			return
		}
		ack := KAAck{Session: x.id, Up: x.upBytes.Load(), Down: x.dnBytes.Load(), CapLeft: s.capLeft(), Battery: battery}
		ack.Seq = uint32(b[1+SessionLen])<<24 | uint32(b[2+SessionLen])<<16 | uint32(b[3+SessionLen])<<8 | uint32(b[4+SessionLen])
		if charging {
			ack.Flags |= FlagCharging
		}
		_, _ = s.conn.WriteTo(ack.Marshal(), from)
	case TypeBye:
		if x := s.sessionFrom(b, from); x != nil {
			s.end(x, ReasonNone, false)
		}
	case TypeHello:
		var h Hello
		if h.Unmarshal(b) != nil {
			return
		}
		var sn [NonceLen]byte
		_, _ = rand.Read(sn[:])
		s.mu.Lock()
		if len(s.pending) >= maxPending {
			s.mu.Unlock()
			return
		}
		s.pending[h.HubNonce] = pending{sharerNonce: sn, hubID: h.HubID, addr: from.String(), at: time.Now()}
		s.mu.Unlock()
		c := Challenge{HubNonce: h.HubNonce, SharerNonce: sn, SharerID: s.cfg.ID, Label: s.cfg.Label}
		_, _ = s.conn.WriteTo(c.Marshal(), from)
	case TypeAuth:
		var a Auth
		if a.Unmarshal(b) != nil {
			return
		}
		s.mu.Lock()
		p, ok := s.pending[a.HubNonce]
		if ok {
			delete(s.pending, a.HubNonce)
		}
		s.mu.Unlock()
		// a wrong key, a replay or a stranger gets no answer at all
		if !ok || p.sharerNonce != a.SharerNonce || p.addr != from.String() ||
			!Equal(a.MAC, AuthMAC(s.cfg.Key, a.HubNonce, a.SharerNonce, p.hubID)) {
			return
		}
		reason, _, _ := s.check()
		if reason == ReasonNone && len(s.cfg.AllowedHubs) > 0 && !slices.Contains(s.cfg.AllowedHubs, p.hubID) {
			reason = ReasonNotAllowed
		}
		if reason == ReasonNone && s.capLeft() == 0 {
			reason = ReasonCap
		}
		if reason != ReasonNone {
			r := Refuse{HubNonce: a.HubNonce, Reason: reason, MAC: RefuseMAC(s.cfg.Key, a.HubNonce, a.SharerNonce, reason)}
			_, _ = s.conn.WriteTo(r.Marshal(), from)
			s.cfg.Logf("share: refused hub %s (%s)", p.hubID, ReasonText(reason))
			return
		}
		x, err := s.open(ctx, p.hubID, from)
		if err != nil {
			r := Refuse{HubNonce: a.HubNonce, Reason: ReasonNoCellular, MAC: RefuseMAC(s.cfg.Key, a.HubNonce, a.SharerNonce, ReasonNoCellular)}
			_, _ = s.conn.WriteTo(r.Marshal(), from)
			s.cfg.Logf("share: no upstream for hub %s: %v", p.hubID, err)
			return
		}
		capLeft := s.capLeft()
		w := Welcome{HubNonce: a.HubNonce, Session: x.id, CapLeft: capLeft, MAC: WelcomeMAC(s.cfg.Key, a.HubNonce, a.SharerNonce, x.id, capLeft)}
		_, _ = s.conn.WriteTo(w.Marshal(), from)
		s.cfg.Logf("share: hub %s (%s) connected", p.hubID, from)
	}
}

// sessionFrom finds the frame's session, from its own hub's address only.
func (s *Sharer) sessionFrom(b []byte, from net.Addr) *session {
	id, ok := SessionOf(b)
	if !ok {
		return nil
	}
	s.mu.Lock()
	x := s.sessions[id]
	s.mu.Unlock()
	if x == nil || x.hubAddr.String() != from.String() {
		return nil
	}
	return x
}

func (s *Sharer) open(ctx context.Context, hubID string, from net.Addr) (*session, error) {
	dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	up, err := s.cfg.Dial(dctx, s.cfg.Server)
	cancel()
	if err != nil {
		return nil, err
	}
	x := &session{hubID: hubID, hubAddr: from, up: up, since: time.Now()}
	_, _ = rand.Read(x.id[:])
	x.lastSeen.Store(time.Now().UnixNano())
	s.mu.Lock()
	old := s.byHub[hubID]
	if old == nil && len(s.sessions) >= maxSessions {
		s.mu.Unlock()
		_ = up.Close()
		return nil, errors.New("too many hubs")
	}
	s.sessions[x.id] = x
	s.byHub[hubID] = x
	s.mu.Unlock()
	if old != nil {
		s.end(old, ReasonReplaced, true)
	}
	go s.pumpDown(x)
	return x, nil
}

// pumpDown: server → hub.
func (s *Sharer) pumpDown(x *session) {
	buf := make([]byte, 65536)
	out := make([]byte, 0, 65536+DataHeader)
	for {
		n, err := x.up.Read(buf[DataHeader:])
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			s.end(x, ReasonNoCellular, true)
			return
		}
		out = Data(out, x.id, buf[DataHeader:DataHeader+n])
		// counted first, as upstream: the hub may ask for the counters
		// (KEEPALIVE) the moment this frame reaches it
		x.dnBytes.Add(uint64(n))
		s.used.Add(uint64(n))
		if _, err := s.conn.WriteTo(out, x.hubAddr); err != nil {
			x.dnBytes.Add(^uint64(n - 1))
			s.used.Add(^uint64(n - 1))
		}
	}
}

// end closes a session; bye tells the hub why.
func (s *Sharer) end(x *session, reason byte, bye bool) {
	x.once.Do(func() {
		s.mu.Lock()
		delete(s.sessions, x.id)
		if s.byHub[x.hubID] == x {
			delete(s.byHub, x.hubID)
		}
		s.mu.Unlock()
		_ = x.up.Close()
		if bye && s.conn != nil {
			_, _ = s.conn.WriteTo(Bye(x.id, reason), x.hubAddr)
		}
		s.cfg.Logf("share: hub %s left (%s)", x.hubID, ReasonText(reason))
	})
}

func (s *Sharer) closeAll(reason byte) {
	s.mu.Lock()
	all := make([]*session, 0, len(s.sessions))
	for _, x := range s.sessions {
		all = append(all, x)
	}
	s.mu.Unlock()
	for _, x := range all {
		s.end(x, reason, true)
	}
}

// Stop ends every session, telling the hubs why (battery, cap, paused).
func (s *Sharer) Stop(reason byte) { s.closeAll(reason) }

func (s *Sharer) janitor(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		var idle []*session
		s.mu.Lock()
		for k, p := range s.pending {
			if now.Sub(p.at) > handshakeTTL {
				delete(s.pending, k)
			}
		}
		for _, x := range s.sessions {
			if now.Sub(time.Unix(0, x.lastSeen.Load())) > sessionIdle {
				idle = append(idle, x)
			}
		}
		s.mu.Unlock()
		for _, x := range idle {
			s.end(x, ReasonIdle, true)
		}
		if reason, _, _ := s.check(); reason != ReasonNone {
			s.closeAll(reason)
		} else if s.capLeft() == 0 {
			s.closeAll(ReasonCap)
		}
	}
}
