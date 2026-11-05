// Package tcplisten opens the agent's TCP listeners.
//
// Go listens with MPTCP by default (1.24+). Some kernels carry an MPTCP
// stack that accepts the listening socket but never completes a connection
// to it, loopback included (GL.iNet's 5.4 kernel on a Brume 2, GL-MT2500):
// the agent runs, its API on 127.0.0.1 times out, and the admin panel says
// "tailbond not running". So in "auto" mode a new listener is first opened
// with MPTCP and probed with a plain TCP connection; if that doesn't
// connect, it is reopened as plain TCP, and so is every later listener.
//
// Modes (config "tcp_mode", UCI main.tcp_mode, env TAILBOND_TCP_MODE):
//
//	auto   MPTCP, falling back to plain TCP when a probe fails (default)
//	mptcp  MPTCP, no probe
//	tcp    plain TCP, never MPTCP
package tcplisten

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ModeAuto  = "auto"
	ModeMPTCP = "mptcp"
	ModeTCP   = "tcp"
)

// ProbeTimeout bounds the self-connect. A healthy kernel completes the
// handshake in microseconds (no accept needed: the kernel finishes it).
const ProbeTimeout = 1500 * time.Millisecond

var (
	mu     sync.Mutex
	mode   = ModeAuto
	broken atomic.Bool // auto mode saw an MPTCP listener that didn't connect
	logf   = func(string, ...any) {}
)

// Valid reports whether m is "" (auto) or a known mode.
func Valid(m string) bool {
	switch strings.ToLower(m) {
	case "", ModeAuto, ModeMPTCP, ModeTCP:
		return true
	}
	return false
}

// SetMode picks the mode ("" = auto). The environment variable
// TAILBOND_TCP_MODE wins over m, so a stuck router can be fixed without
// editing the config. An unknown value means auto. logger (optional)
// receives the one-line notice when auto falls back.
func SetMode(m string, logger func(string, ...any)) {
	if v := os.Getenv("TAILBOND_TCP_MODE"); v != "" {
		m = v
	}
	m = strings.ToLower(m)
	if !Valid(m) || m == "" {
		m = ModeAuto
	}
	mu.Lock()
	mode = m
	if logger != nil {
		logf = logger
	}
	mu.Unlock()
	broken.Store(false)
}

// Mode returns the current mode.
func Mode() string {
	mu.Lock()
	defer mu.Unlock()
	return mode
}

// FellBack reports whether auto mode has switched to plain TCP.
func FellBack() bool { return broken.Load() }

func listen(network, addr string, mptcp bool) (net.Listener, error) {
	lc := net.ListenConfig{}
	lc.SetMultipathTCP(mptcp)
	return lc.Listen(context.Background(), network, addr)
}

// Listen is net.Listen for "tcp", "tcp4" and "tcp6" under the current mode.
func Listen(network, addr string) (net.Listener, error) {
	mu.Lock()
	m, log := mode, logf
	mu.Unlock()
	switch {
	case m == ModeTCP:
		return listen(network, addr, false)
	case m == ModeMPTCP:
		return listen(network, addr, true)
	case broken.Load():
		return listen(network, addr, false)
	}
	ln, err := listen(network, addr, true)
	if err != nil {
		return nil, err
	}
	if probe(ln.Addr()) == nil {
		return ln, nil
	}
	// The probe failed: this kernel's MPTCP listener doesn't take plain
	// TCP connections. Reopen the same address as plain TCP.
	ln.Close()
	broken.Store(true)
	log("tcp: an MPTCP listener on %s didn't accept a connection; using plain TCP (set tcp_mode to override)", addr)
	return listen(network, addr, false)
}

// probe connects to a listener the way a client does (plain TCP).
func probe(a net.Addr) error {
	ta, ok := a.(*net.TCPAddr)
	if !ok {
		return nil
	}
	ip := ta.IP
	if ip == nil || ip.IsUnspecified() {
		ip = net.IPv4(127, 0, 0, 1)
		if ta.IP != nil && ta.IP.To4() == nil {
			ip = net.IPv6loopback
		}
	}
	// plain TCP whatever GODEBUG says: the point is a client that has no MPTCP
	d := net.Dialer{Timeout: ProbeTimeout}
	d.SetMultipathTCP(false)
	c, err := d.Dial("tcp", net.JoinHostPort(ip.String(), strconv.Itoa(ta.Port)))
	if err != nil {
		return err
	}
	return c.Close()
}
