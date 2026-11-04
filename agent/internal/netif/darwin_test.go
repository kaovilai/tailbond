//go:build darwin

package netif

import (
	"net"
	"syscall"
	"testing"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

// On a real Mac: discovery runs, lists no tunnel or loopback, and the
// loopback counters move when traffic crosses it.
func TestDarwinDiscoverAndCounters(t *testing.T) {
	ifs, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ifs {
		t.Logf("%s kind=%s label=%q up=%v carrier=%v addrs=%v", i.Name, i.Kind, i.Label, i.Up, i.Carrier, i.Addrs)
		if darwinSkipped(i.Name) {
			t.Errorf("%s listed", i.Name)
		}
	}
	tx0, rx0, ok := Counters("lo0")
	if !ok {
		lo, _ := net.InterfaceByName("lo0")
		b, err := route.FetchRIB(syscall.AF_UNSPEC, route.RIBType(unix.NET_RT_IFLIST2), 0)
		t.Fatalf("no counters for lo0 (index %v): %d bytes, %v; first message type %v", lo, len(b), err, func() any {
			if len(b) > 3 {
				return b[3]
			}
			return nil
		}())
	}
	srv, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	c, err := net.Dial("udp4", srv.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for range 20 {
		_, _ = c.Write(make([]byte, 1000))
	}
	tx1, rx1, _ := Counters("lo0")
	if tx1 < tx0+20000 || rx1 < rx0+20000 {
		t.Fatalf("lo0 counters barely moved: tx %d -> %d, rx %d -> %d", tx0, tx1, rx0, rx1)
	}
}
