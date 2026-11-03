package qprobe

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestPacketShape(t *testing.T) {
	for i := 0; i < 100; i++ {
		p := Packet()
		if len(p) != Size || p[0]&0xc0 != 0x40 {
			t.Fatalf("first byte %#x len %d: want short header with the fixed bit", p[0], len(p))
		}
	}
}

func TestProbe(t *testing.T) {
	srv, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := srv.ReadFrom(buf)
			if err != nil {
				return
			}
			if n == Size && buf[0]&0xc0 == 0x40 {
				_, _ = srv.WriteTo(buf[:43], from) // a stateless reset is smaller
			}
		}
	}()
	if err := Probe(context.Background(), nil, srv.LocalAddr().String(), time.Second); err != nil {
		t.Fatalf("answered probe: %v", err)
	}
	// nothing listening: no reply (or a refused error) within the timeout
	dead, _ := net.ListenPacket("udp", "127.0.0.1:0")
	addr := dead.LocalAddr().String()
	dead.Close()
	start := time.Now()
	if err := Probe(context.Background(), nil, addr, 300*time.Millisecond); err == nil {
		t.Fatal("probe of a closed port succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("timeout not honoured")
	}
}
