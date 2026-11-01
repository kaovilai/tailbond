package failover

import "time"

// Silence tells, from the control API's per-path packet counters, whether
// the server still answers: the mqvpn client sends keep-alive PINGs, so a
// live path's received count keeps moving whenever its sent count does. A
// path whose sends have gone unanswered for After is silent; the tunnel is
// silent when every path is. (mqvpn itself only gives up after its 120 s
// idle timeout.)
type Silence struct {
	After time.Duration
	paths map[uint64]*seen
}

type seen struct {
	sent, recv uint64
	since      time.Time // first unanswered send, zero while answered
	at         time.Time
}

// Observe records one path's counters at now.
func (s *Silence) Observe(now time.Time, id, sent, recv uint64) {
	if s.paths == nil {
		s.paths = map[uint64]*seen{}
	}
	p, ok := s.paths[id]
	if !ok || sent < p.sent || recv < p.recv {
		s.paths[id] = &seen{sent: sent, recv: recv, at: now}
		return
	}
	switch {
	case recv > p.recv:
		p.since = time.Time{}
	case sent > p.sent && p.since.IsZero():
		p.since = now
	}
	p.sent, p.recv, p.at = sent, recv, now
}

// Forget drops paths not observed since before (closed paths).
func (s *Silence) Forget(before time.Time) {
	for id, p := range s.paths {
		if p.at.Before(before) {
			delete(s.paths, id)
		}
	}
}

// Reset starts over (mqvpn restarted).
func (s *Silence) Reset() { s.paths = nil }

// Silent: at least one path, and every path has had unanswered sends for After.
func (s *Silence) Silent(now time.Time) bool {
	if len(s.paths) == 0 {
		return false
	}
	for _, p := range s.paths {
		if p.since.IsZero() || now.Sub(p.since) < s.After {
			return false
		}
	}
	return true
}
