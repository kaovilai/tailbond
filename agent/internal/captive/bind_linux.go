package captive

import (
	"context"
	"net"
	"syscall"
)

// dialNet binds the socket to dev (SO_BINDTODEVICE), so the probe and its
// DNS go out over that WAN whatever the routing table prefers.
func dialNet(ctx context.Context, dev, network, addr string) (net.Conn, error) {
	d := net.Dialer{Control: func(_, _ string, c syscall.RawConn) error {
		if dev == "" {
			return nil
		}
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, dev)
		}); err != nil {
			return err
		}
		return serr
	}}
	return d.DialContext(ctx, network, addr)
}
