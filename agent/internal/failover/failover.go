// Package failover decides which of a client's servers carries the tunnel
// (docs/architecture.md, "Multiple servers and failover"). It is a pure
// state machine: the agent feeds it the active tunnel's health and the
// standby servers' probe results with explicit times, and performs the
// switches it returns.
package failover

import "time"

const (
	PolicyPrefer = "prefer"
	PolicyAlways = "always"

	ReasonNone        = ""
	ReasonPrimaryDown = "primary_down"
	ReasonManual      = "manual"
	ReasonPolicy      = "policy"

	StateActive      = "active"
	StateConnecting  = "connecting"
	StateStandby     = "standby"
	StateUnreachable = "unreachable"
	StateUnknown     = "unknown"

	// A failback that doesn't come up doubles that server's hold, up to
	// 1<<maxPenalty times; staying up this long clears it.
	maxPenalty   = 4
	penaltyReset = 10 * time.Minute
)

type Policy struct {
	Mode          string // PolicyPrefer | PolicyAlways
	Pinned        string // PolicyAlways: the server to use
	SwitchAfter   time.Duration
	FailbackAfter time.Duration
}

type probe struct {
	known   bool
	ok      bool
	okSince time.Time // start of the current run of answered probes
	lastOK  time.Time
}

type Machine struct {
	p     Policy
	order []string

	active       string
	activeSince  time.Time // when active became active
	lastWorking  time.Time // last time the active tunnel worked
	workingSince time.Time // start of the current working run
	manual       bool
	failingBack  string // the server a failback just went to, until it works
	failed       bool   // the switch in progress leaves a server that stopped working

	probes  map[string]*probe
	penalty map[string]int
}

// New starts on active (or the first server if active isn't listed).
func New(p Policy, order []string, active string, now time.Time) *Machine {
	m := &Machine{p: p, probes: map[string]*probe{}, penalty: map[string]int{}}
	m.order = append([]string(nil), order...)
	if m.index(active) < 0 && len(m.order) > 0 {
		active = m.order[0]
	}
	if m.p.Mode == PolicyAlways && m.index(m.p.Pinned) >= 0 {
		active = m.p.Pinned
	}
	m.active, m.activeSince = active, now
	return m
}

func (m *Machine) Active() string  { return m.active }
func (m *Machine) Order() []string { return append([]string(nil), m.order...) }
func (m *Machine) Policy() Policy  { return m.p }

func (m *Machine) index(name string) int {
	for i, n := range m.order {
		if n == name {
			return i
		}
	}
	return -1
}

func (m *Machine) probeOf(name string) *probe {
	p := m.probes[name]
	if p == nil {
		p = &probe{}
		m.probes[name] = p
	}
	return p
}

// switchTo makes name active. The server left behind must prove itself
// again with fresh probes before anything fails back to it; one left
// because it stopped working counts as unreachable until a probe answers.
func (m *Machine) switchTo(name string, now time.Time) string {
	if name == "" || name == m.active {
		return ""
	}
	if m.failed {
		m.probes[m.active] = &probe{known: true}
	} else {
		delete(m.probes, m.active)
	}
	m.failed = false
	delete(m.probes, name)
	m.active, m.activeSince = name, now
	m.lastWorking, m.workingSince = time.Time{}, time.Time{}
	return name
}

// SetOrder replaces the server list (reorder, add, remove). It returns the
// server to switch to when the active one is gone ("" otherwise).
func (m *Machine) SetOrder(order []string, now time.Time) string {
	m.order = append([]string(nil), order...)
	for n := range m.probes {
		if m.index(n) < 0 {
			delete(m.probes, n)
		}
	}
	if m.index(m.active) >= 0 || len(m.order) == 0 {
		if len(m.order) == 0 {
			m.active = ""
		}
		return m.enforcePolicy(now)
	}
	m.manual = false
	return m.switchTo(m.order[0], now)
}

// SetPolicy changes the policy; "always" switches at once.
func (m *Machine) SetPolicy(p Policy, now time.Time) string {
	m.p = p
	if p.Mode != PolicyAlways {
		return ""
	}
	m.manual = false
	return m.enforcePolicy(now)
}

func (m *Machine) enforcePolicy(now time.Time) string {
	if m.p.Mode == PolicyAlways && m.index(m.p.Pinned) >= 0 && m.active != m.p.Pinned {
		return m.switchTo(m.p.Pinned, now)
	}
	return ""
}

// Manual is "Use this server now": switch at once, and don't fail back
// while it works.
func (m *Machine) Manual(name string, now time.Time) string {
	if m.index(name) < 0 {
		return ""
	}
	m.manual = m.index(name) > 0
	m.failingBack = ""
	return m.switchTo(name, now)
}

// Probe records a standby server's probe result.
func (m *Machine) Probe(name string, ok bool, now time.Time) {
	if name == m.active || m.index(name) < 0 {
		return
	}
	p := m.probeOf(name)
	if ok {
		if !p.known || !p.ok {
			p.okSince = now
		}
		p.lastOK = now
	}
	p.known, p.ok = true, ok
}

// Step advances the machine: working is whether the active tunnel works
// right now. It returns the server to switch to, or "".
func (m *Machine) Step(now time.Time, working bool) string {
	if len(m.order) == 0 || m.active == "" {
		return ""
	}
	if working {
		m.lastWorking = now
		if m.workingSince.IsZero() {
			m.workingSince = now
		}
		if m.failingBack == m.active {
			m.failingBack = ""
		}
		if now.Sub(m.workingSince) >= penaltyReset {
			delete(m.penalty, m.active)
		}
	} else {
		m.workingSince = time.Time{}
	}
	if m.p.Mode == PolicyAlways && m.index(m.p.Pinned) >= 0 {
		return m.enforcePolicy(now)
	}

	// failover: the active server hasn't worked for SwitchAfter
	ref := m.activeSince
	if m.lastWorking.After(ref) {
		ref = m.lastWorking
	}
	if !working && now.Sub(ref) >= m.p.SwitchAfter {
		if m.failingBack == m.active && m.penalty[m.active] < maxPenalty {
			m.penalty[m.active]++
		}
		m.failingBack = ""
		if next := m.next(); next != "" {
			m.manual = false
			m.failed = true
			return m.switchTo(next, now)
		}
		return ""
	}

	// failback: an earlier server has answered every probe for its hold
	if m.manual || !working {
		return ""
	}
	ai := m.index(m.active)
	for i := 0; i < ai; i++ {
		name := m.order[i]
		p := m.probes[name]
		if p != nil && p.known && p.ok && now.Sub(p.okSince) >= m.hold(name) {
			m.failingBack = name
			return m.switchTo(name, now)
		}
	}
	return ""
}

func (m *Machine) hold(name string) time.Duration {
	return m.p.FailbackAfter << m.penalty[name]
}

// next is the server after the active one in list order (wrapping),
// preferring one whose last probe answered, then one not probed yet; ""
// when all of them are known to be unreachable.
func (m *Machine) next() string {
	ai := m.index(m.active)
	var cands []string
	for k := 1; k < len(m.order); k++ {
		cands = append(cands, m.order[(ai+k)%len(m.order)])
	}
	for _, c := range cands {
		if p := m.probes[c]; p != nil && p.known && p.ok {
			return c
		}
	}
	for _, c := range cands {
		if p := m.probes[c]; p == nil || !p.known || p.ok {
			return c
		}
	}
	// every other server is known to be down: keep trying this one
	return ""
}

// ServerView is one server as /api/status shows it.
type ServerView struct {
	Name   string
	State  string
	Active bool
	LastOK time.Time
}

// View summarises the machine for the status API.
type View struct {
	Active     string
	Servers    []ServerView
	Reason     string
	Since      time.Time
	FailbackIn time.Duration // > 0 while counting down to a failback
	Counting   bool
}

func (m *Machine) View(now time.Time, working bool) View {
	v := View{Active: m.active, Since: m.activeSince}
	for _, n := range m.order {
		sv := ServerView{Name: n, State: StateUnknown}
		if n == m.active {
			sv.Active, sv.LastOK = true, m.lastWorking
			sv.State = StateConnecting
			if working {
				sv.State = StateActive
			}
		} else if p := m.probes[n]; p != nil && p.known {
			sv.LastOK = p.lastOK
			sv.State = StateUnreachable
			if p.ok {
				sv.State = StateStandby
			}
		}
		v.Servers = append(v.Servers, sv)
	}
	if ai := m.index(m.active); ai > 0 {
		switch {
		case m.p.Mode == PolicyAlways:
			v.Reason = ReasonPolicy
		case m.manual:
			v.Reason = ReasonManual
		default:
			v.Reason = ReasonPrimaryDown
			for i := 0; i < ai; i++ {
				p := m.probes[m.order[i]]
				if p != nil && p.known && p.ok {
					left := m.hold(m.order[i]) - now.Sub(p.okSince)
					if left < 0 {
						left = 0
					}
					if !v.Counting || left < v.FailbackIn {
						v.FailbackIn, v.Counting = left, true
					}
				}
			}
		}
	}
	return v
}
