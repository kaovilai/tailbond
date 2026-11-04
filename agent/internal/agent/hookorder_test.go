package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A down hook queued at shutdown must not run while hookWorker is still in
// the middle of the up hook (PR 19 review): drainHooks waits for it.
func TestDrainHooksWaitsForRunningHook(t *testing.T) {
	a := &Agent{hookq: make(chan func(), 32)}
	var running, overlapped atomic.Bool
	var order []string
	var mu sync.Mutex
	note := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }
	hook := func(name string, d time.Duration) func() {
		return func() {
			if running.Swap(true) {
				overlapped.Store(true)
			}
			note(name + " start")
			time.Sleep(d)
			note(name + " end")
			running.Store(false)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.hookWorker(ctx); close(done) }()
	a.hookq <- hook("server-up", 300*time.Millisecond)
	for !running.Load() { // the worker has started the up hook
		time.Sleep(time.Millisecond)
	}
	cancel() // shutdown: the agent queues the down hook and drains
	a.hookq <- hook("server-down", 10*time.Millisecond)
	a.drainHooks()
	<-done
	if overlapped.Load() {
		t.Fatalf("hooks overlapped: %v", order)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 4 || order[0] != "server-up start" || order[1] != "server-up end" || order[2] != "server-down start" {
		t.Fatalf("order %v", order)
	}
}
