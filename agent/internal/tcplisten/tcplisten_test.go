package tcplisten

import (
	"net"
	"testing"
)

func TestModes(t *testing.T) {
	for _, m := range []string{"", "auto", "MPTCP", "tcp"} {
		if !Valid(m) {
			t.Errorf("Valid(%q) = false", m)
		}
	}
	if Valid("udp") {
		t.Error(`Valid("udp") = true`)
	}
	t.Setenv("TAILBOND_TCP_MODE", "")
	SetMode("bogus", nil)
	if Mode() != ModeAuto {
		t.Errorf("unknown mode = %q, want auto", Mode())
	}
	t.Setenv("TAILBOND_TCP_MODE", "tcp")
	SetMode("mptcp", nil)
	if Mode() != ModeTCP {
		t.Errorf("env should win: mode = %q", Mode())
	}
}

// Every mode yields a listener that a plain TCP client can connect to.
func TestListenAccepts(t *testing.T) {
	t.Setenv("TAILBOND_TCP_MODE", "")
	for _, m := range []string{ModeAuto, ModeMPTCP, ModeTCP} {
		SetMode(m, t.Logf)
		ln, err := Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("%s: %v", m, err)
		}
		if err := probe(ln.Addr()); err != nil {
			t.Errorf("%s: plain TCP connect: %v", m, err)
		}
		ln.Close()
	}
}

func TestProbeFailsOnClosedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	a := ln.Addr()
	ln.Close()
	if probe(a) == nil {
		t.Error("probe of a closed port succeeded")
	}
}
