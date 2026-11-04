//go:build darwin

package netif

import (
	"net"
	"testing"
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
		t.Fatal("no counters for lo0")
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
