package failover

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

func prefer() Policy {
	return Policy{Mode: PolicyPrefer, SwitchAfter: 15 * time.Second, FailbackAfter: 120 * time.Second}
}

// sim drives a machine second by second: up reports whether a server can
// carry a tunnel, answers whether it answers probes (every 10 s).
type sim struct {
	t       *testing.T
	m       *Machine
	now     time.Time
	up      map[string]bool
	answers map[string]bool
	log     []string // "<second>:<server>" per switch
}

func newSim(t *testing.T, p Policy, order ...string) *sim {
	s := &sim{t: t, now: t0, up: map[string]bool{}, answers: map[string]bool{}}
	s.m = New(p, order, order[0], t0)
	for _, n := range order {
		s.up[n], s.answers[n] = true, true
	}
	return s
}

func (s *sim) run(secs int) {
	for i := 0; i < secs; i++ {
		s.now = s.now.Add(time.Second)
		if int(s.now.Sub(t0)/time.Second)%10 == 0 {
			for _, n := range s.m.Order() {
				s.m.Probe(n, s.answers[n], s.now)
			}
		}
		if to := s.m.Step(s.now, s.up[s.m.Active()]); to != "" {
			s.log = append(s.log, to)
		}
	}
}

func (s *sim) elapsed() time.Duration { return s.now.Sub(t0) }

func TestFailoverAfterSwitchAfter(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.run(30)
	if s.m.Active() != "homebox" || len(s.log) != 0 {
		t.Fatalf("healthy primary: active %s, switches %v", s.m.Active(), s.log)
	}
	s.up["homebox"], s.answers["homebox"] = false, false
	down := s.elapsed()
	for s.m.Active() == "homebox" {
		s.run(1)
		if s.elapsed()-down > time.Minute {
			t.Fatal("never switched")
		}
	}
	if got := s.elapsed() - down; got != 15*time.Second {
		t.Fatalf("switched after %v, want 15s", got)
	}
	v := s.m.View(s.now, true)
	if v.Active != "vps" || v.Reason != ReasonPrimaryDown {
		t.Fatalf("view %+v", v)
	}
}

func TestFailbackAfterHoldNoFlap(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.up["homebox"], s.answers["homebox"] = false, false
	s.run(20)
	if s.m.Active() != "vps" {
		t.Fatalf("active %s", s.m.Active())
	}
	// home answers again; the failback waits for 120 s of answers
	s.up["homebox"], s.answers["homebox"] = true, true
	back := s.elapsed()
	for s.m.Active() == "vps" {
		s.run(1)
		if s.elapsed()-back > 5*time.Minute {
			t.Fatal("never failed back")
		}
	}
	got := s.elapsed() - back
	if got < 120*time.Second || got > 131*time.Second {
		t.Fatalf("failed back after %v, want 120s (+ probe interval)", got)
	}
	if v := s.m.View(s.now, true); v.Reason != ReasonNone {
		t.Fatalf("reason after failback %q", v.Reason)
	}
	s.run(600)
	if len(s.log) != 2 {
		t.Fatalf("switches %v, want exactly vps then homebox", s.log)
	}
}

func TestFlappingPrimaryNeverFailsBack(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.up["homebox"], s.answers["homebox"] = false, false
	s.run(20)
	// home answers for 60 s, then misses one probe, forever
	for i := 0; i < 20; i++ {
		s.answers["homebox"] = true
		s.run(60)
		s.answers["homebox"] = false
		s.run(10)
	}
	if s.m.Active() != "vps" || len(s.log) != 1 {
		t.Fatalf("flapped: %v", s.log)
	}
	v := s.m.View(s.now, true)
	if v.Servers[0].State != StateUnreachable {
		t.Fatalf("primary state %s", v.Servers[0].State)
	}
}

func TestCountdownInView(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.up["homebox"], s.answers["homebox"] = false, false
	s.run(20)
	s.answers["homebox"] = true
	s.run(40) // first answered probe at 30 s → 30 s into the hold at 60 s
	v := s.m.View(s.now, true)
	if !v.Counting || v.FailbackIn != 90*time.Second {
		t.Fatalf("countdown %v counting=%v", v.FailbackIn, v.Counting)
	}
	if v.Servers[0].State != StateStandby || v.Servers[1].State != StateActive {
		t.Fatalf("states %+v", v.Servers)
	}
}

func TestFailedFailbackDoublesHold(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.up["homebox"], s.answers["homebox"] = false, false
	s.run(20)
	// home answers probes but can't carry a tunnel
	s.answers["homebox"] = true
	start := s.elapsed()
	var times []time.Duration
	for len(s.log) < 5 && s.elapsed()-start < time.Hour {
		n := len(s.log)
		s.run(1)
		if len(s.log) > n && s.m.Active() == "homebox" {
			times = append(times, s.elapsed())
		}
	}
	if len(times) < 2 {
		t.Fatalf("failbacks %v, log %v", times, s.log)
	}
	// 1st after ~120 s on vps; 2nd needs 240 s of answers after returning
	if gap := times[1] - times[0]; gap < 255*time.Second {
		t.Fatalf("second failback %v after the first, want the hold doubled", gap)
	}
}

func TestManualSuppressesFailback(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.run(15)
	if to := s.m.Manual("vps", s.now); to != "vps" {
		t.Fatalf("manual returned %q", to)
	}
	s.run(600)
	if s.m.Active() != "vps" {
		t.Fatalf("failed back from a manual choice: %v", s.log)
	}
	if v := s.m.View(s.now, true); v.Reason != ReasonManual {
		t.Fatalf("reason %q", v.Reason)
	}
	// the manual choice fails → prefer resumes
	s.up["vps"] = false
	s.run(16)
	if s.m.Active() != "homebox" {
		t.Fatalf("active %s after vps failed", s.m.Active())
	}
}

func TestAlwaysNeverSwitches(t *testing.T) {
	p := prefer()
	p.Mode, p.Pinned = PolicyAlways, "vps"
	s := newSim(t, p, "homebox", "vps")
	if s.m.Active() != "vps" {
		t.Fatalf("always vps starts on %s", s.m.Active())
	}
	s.up["vps"] = false
	s.run(300)
	if s.m.Active() != "vps" || len(s.log) != 0 {
		t.Fatalf("always switched: %v", s.log)
	}
	if v := s.m.View(s.now, false); v.Reason != ReasonPolicy || v.Servers[1].State != StateConnecting {
		t.Fatalf("view %+v", v)
	}
	// back to prefer: vps is down → homebox
	if to := s.m.SetPolicy(prefer(), s.now); to != "" {
		t.Fatalf("SetPolicy prefer switched to %q", to)
	}
	s.run(16)
	if s.m.Active() != "homebox" {
		t.Fatalf("active %s", s.m.Active())
	}
}

func TestNextSkipsUnreachable(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps-a", "vps-b")
	s.answers["vps-a"] = false
	s.run(12) // probes at 10 s
	s.up["homebox"] = false
	s.run(16)
	if s.m.Active() != "vps-b" {
		t.Fatalf("active %s, want vps-b (vps-a didn't answer)", s.m.Active())
	}
}

func TestSetOrder(t *testing.T) {
	m := New(prefer(), []string{"homebox", "vps"}, "vps", t0)
	if to := m.SetOrder([]string{"homebox"}, t0); to != "homebox" {
		t.Fatalf("removing the active server switched to %q", to)
	}
	if to := m.SetOrder([]string{"vps", "homebox"}, t0); to != "" || m.Active() != "homebox" {
		t.Fatalf("reorder switched to %q", to)
	}
	if to := m.SetOrder(nil, t0); to != "" || m.Active() != "" {
		t.Fatal("empty list")
	}
}

func TestSingleServerNeverSwitches(t *testing.T) {
	s := newSim(t, prefer(), "homebox")
	s.up["homebox"] = false
	s.run(120)
	if len(s.log) != 0 {
		t.Fatalf("switches %v", s.log)
	}
}

func TestNoPingPongBetweenDeadServers(t *testing.T) {
	s := newSim(t, prefer(), "homebox", "vps")
	s.up["homebox"], s.answers["homebox"] = false, false
	s.run(16)
	if s.m.Active() != "vps" {
		t.Fatalf("active %s", s.m.Active())
	}
	// the vps is slow to come up: homebox is known dead, so stay
	s.up["vps"] = false
	s.run(60)
	if s.m.Active() != "vps" || len(s.log) != 1 {
		t.Fatalf("switched back to a dead server: %v", s.log)
	}
	if v := s.m.View(s.now, false); v.Servers[0].State != StateUnreachable {
		t.Fatalf("homebox %s", v.Servers[0].State)
	}
	// homebox answers again but the vps is still down: failover picks it
	s.answers["homebox"], s.up["homebox"] = true, true
	s.run(12)
	if s.m.Active() != "homebox" {
		t.Fatalf("active %s, want homebox once it answers", s.m.Active())
	}
}
