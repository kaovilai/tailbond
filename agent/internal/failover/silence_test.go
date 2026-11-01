package failover

import (
	"testing"
	"time"
)

func TestSilence(t *testing.T) {
	s := &Silence{After: 5 * time.Second}
	now := t0
	tick := func(sent0, recv0, sent1, recv1 uint64) {
		now = now.Add(time.Second)
		s.Observe(now, 0, sent0, recv0)
		s.Observe(now, 1, sent1, recv1)
	}
	tick(10, 10, 10, 10)
	if s.Silent(now) {
		t.Fatal("silent at start")
	}
	// idle: nothing sent, nothing received — not silence
	for i := 0; i < 10; i++ {
		tick(10, 10, 10, 10)
	}
	if s.Silent(now) {
		t.Fatal("idle tunnel counted as silent")
	}
	// path 0 dies, path 1 still answers
	for i := uint64(1); i <= 10; i++ {
		tick(10+i, 10, 10+i, 10+i)
	}
	if s.Silent(now) {
		t.Fatal("one live path is enough")
	}
	// both unanswered: silent only after 5 s
	base := uint64(30)
	for i := uint64(1); i <= 4; i++ {
		tick(base+i, 10, base+i, 20)
	}
	if s.Silent(now) {
		t.Fatal("silent too early")
	}
	tick(base+6, 10, base+6, 20)
	tick(base+7, 10, base+7, 20)
	if !s.Silent(now) {
		t.Fatal("not silent after 5 s of unanswered sends")
	}
	// an answer on one path ends it
	tick(base+8, 10, base+8, 21)
	if s.Silent(now) {
		t.Fatal("still silent after an answer")
	}
	// counters going backwards (new connection) restart the path
	now = now.Add(time.Second)
	s.Observe(now, 0, 1, 1)
	s.Forget(now)
	if len(s.paths) != 1 {
		t.Fatalf("forget: %d paths", len(s.paths))
	}
	s.Reset()
	if s.Silent(now) {
		t.Fatal("silent after reset")
	}
}
