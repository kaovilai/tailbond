package sockbind

import (
	"errors"
	"net"
	"syscall"
	"testing"
)

// A socket pinned to the loopback interface still reaches loopback, and
// an unknown interface is an error (Linux and macOS pin; elsewhere no-op).
func TestControlLoopback(t *testing.T) {
	lo := ""
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagLoopback != 0 {
			lo = i.Name
		}
	}
	if lo == "" {
		t.Skip("no loopback interface")
	}
	srv, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	d := net.Dialer{Control: Control(lo)}
	c, err := d.Dial("udp4", srv.LocalAddr().String())
	if errors.Is(err, syscall.EPERM) {
		t.Skipf("pinning needs privileges here: %v", err)
	}
	if err != nil {
		t.Fatalf("pinned to %s: %v", lo, err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("x")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 4)
	if n, _, err := srv.ReadFrom(buf); err != nil || n != 1 {
		t.Fatalf("read: %d %v", n, err)
	}
	if Control("")("udp4", "", nil) != nil {
		t.Fatal("unpinned control must not fail")
	}
}

func TestControlUnknownInterface(t *testing.T) {
	d := net.Dialer{Control: Control("tbnosuch0")}
	c, err := d.Dial("udp4", "127.0.0.1:9")
	if err == nil {
		c.Close()
	}
	if pins && err == nil {
		t.Fatal("pinning to a missing interface should fail")
	}
}
