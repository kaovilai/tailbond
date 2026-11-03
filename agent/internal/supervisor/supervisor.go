// Package supervisor runs mqvpn, restarts it with backoff and keeps its output
// in a ring buffer for /api/logs.
package supervisor

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Entry struct {
	Seq  uint64 `json:"seq"`
	Time int64  `json:"t"` // unix ms
	Src  string `json:"src"`
	Line string `json:"line"`
}

// Ring is a bounded, concurrency-safe log buffer.
type Ring struct {
	mu   sync.Mutex
	buf  []Entry
	next uint64
	size int
	// Echo, when set, also receives each line (stderr for journald).
	Echo io.Writer
}

func NewRing(size int) *Ring { return &Ring{size: size, buf: make([]Entry, 0, size)} }

func (r *Ring) Add(src, line string) {
	r.mu.Lock()
	r.next++
	e := Entry{Seq: r.next, Time: time.Now().UnixMilli(), Src: src, Line: line}
	if len(r.buf) < r.size {
		r.buf = append(r.buf, e)
	} else {
		copy(r.buf, r.buf[1:])
		r.buf[len(r.buf)-1] = e
	}
	echo := r.Echo
	r.mu.Unlock()
	if echo != nil {
		_, _ = io.WriteString(echo, "["+src+"] "+line+"\n")
	}
}

// Since returns entries with Seq > seq, at most limit (newest kept).
func (r *Ring) Since(seq uint64, limit int) []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Entry{}
	for _, e := range r.buf {
		if e.Seq > seq {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

type State int

const (
	Stopped State = iota
	Starting
	Running
	Backoff
)

func (s State) String() string {
	return [...]string{"stopped", "starting", "running", "backoff"}[s]
}

type Spec struct {
	Bin  string
	Args []string
	// PreStart runs before every launch (config generation, cert pin check).
	// An error counts as a failed start and is retried with backoff.
	PreStart func(ctx context.Context) error
	// OnLine is called for each output line, after it is stored in the ring.
	OnLine func(line string)
	// OnStart / OnExit fire around each process lifetime.
	OnStart func()
	OnExit  func(err error)
	// PidFile (optional) records the running process. A process still
	// named in it at launch is one a previous agent left behind (no
	// parent-death signal off Linux): it is stopped first.
	PidFile string
}

type Supervisor struct {
	spec Spec
	log  *Ring

	mu          sync.Mutex
	state       State
	cancel      context.CancelFunc
	done        chan struct{}
	started     time.Time
	lastErr     string
	restarts    int
	MinBackoff  time.Duration
	MaxBackoff  time.Duration
	StableAfter time.Duration
}

func New(spec Spec, log *Ring) *Supervisor {
	return &Supervisor{spec: spec, log: log, MinBackoff: time.Second, MaxBackoff: 30 * time.Second, StableAfter: 60 * time.Second}
}

type Info struct {
	State    State
	Since    time.Time
	LastErr  string
	Restarts int
}

func (s *Supervisor) Info() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Info{s.state, s.started, s.lastErr, s.restarts}
}

func (s *Supervisor) setState(st State, err string) {
	s.mu.Lock()
	s.state = st
	if st == Running {
		s.started = time.Now()
	}
	if err != "" {
		s.lastErr = err
	}
	s.mu.Unlock()
}

// Start launches the supervision loop if it is not running.
func (s *Supervisor) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	s.lastErr = ""
	go s.loop(ctx, s.done)
}

// Stop terminates the process (SIGTERM, then SIGKILL after 5 s) and the loop.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// Restart stops and starts again, e.g. after a config change.
func (s *Supervisor) Restart() {
	s.Stop()
	s.Start()
}

func (s *Supervisor) Active() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *Supervisor) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	defer s.setState(Stopped, "")
	backoff := s.MinBackoff
	for ctx.Err() == nil {
		s.setState(Starting, "")
		began := time.Now()
		err := s.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		msg := "exited"
		if err != nil {
			msg = err.Error()
		}
		if time.Since(began) > s.StableAfter {
			backoff = s.MinBackoff
		}
		s.mu.Lock()
		s.restarts++
		s.mu.Unlock()
		s.setState(Backoff, msg)
		s.log.Add("agent", "mqvpn: "+msg+"; restarting in "+backoff.String())
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, s.MaxBackoff)
	}
}

func (s *Supervisor) runOnce(ctx context.Context) error {
	if s.spec.PreStart != nil {
		if err := s.spec.PreStart(ctx); err != nil {
			return err
		}
	}
	s.stopOrphan()
	cmd := exec.Command(s.spec.Bin, s.spec.Args...)
	cmd.SysProcAttr = procAttr()
	cmd.Env = os.Environ()
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return err
	}
	pw.Close()
	if s.spec.PidFile != "" {
		_ = os.WriteFile(s.spec.PidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644)
		defer os.Remove(s.spec.PidFile)
	}
	s.log.Add("agent", "started "+s.spec.Bin+" (pid "+strconv.Itoa(cmd.Process.Pid)+")")
	s.setState(Running, "")
	if s.spec.OnStart != nil {
		s.spec.OnStart()
	}
	lines := make(chan struct{})
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			s.log.Add("mqvpn", line)
			if s.spec.OnLine != nil {
				s.spec.OnLine(line)
			}
		}
		pr.Close()
	}()
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	var exitErr error
	select {
	case exitErr = <-waitErr:
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case exitErr = <-waitErr:
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			exitErr = <-waitErr
		}
	}
	<-lines
	if s.spec.OnExit != nil {
		s.spec.OnExit(exitErr)
	}
	if exitErr == nil {
		return errors.New("mqvpn exited")
	}
	return exitErr
}

// stopOrphan stops the process PidFile names, if it is still running:
// SIGTERM (mqvpn restores routes and DNS on it), SIGKILL after 5 s.
func (s *Supervisor) stopOrphan() {
	if s.spec.PidFile == "" {
		return
	}
	b, err := os.ReadFile(s.spec.PidFile)
	if err != nil {
		return
	}
	_ = os.Remove(s.spec.PidFile)
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 1 || pid == os.Getpid() || syscall.Kill(pid, 0) != nil {
		return
	}
	// the pid may have been reused since: only a process running our binary
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil || filepath.Base(strings.TrimSpace(string(out))) != filepath.Base(s.spec.Bin) {
		return
	}
	s.log.Add("agent", "stopping mqvpn left running by a previous agent (pid "+strconv.Itoa(pid)+")")
	_ = syscall.Kill(pid, syscall.SIGTERM)
	for range 50 {
		time.Sleep(100 * time.Millisecond)
		if syscall.Kill(pid, 0) != nil {
			return
		}
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
