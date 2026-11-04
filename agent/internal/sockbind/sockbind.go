// Package sockbind pins a socket to one network interface, so its traffic
// leaves through that interface whatever the routing table prefers
// (captive-portal probes per WAN, a sharer's upstream over its cellular).
//
// Linux: SO_BINDTODEVICE. macOS: IP_BOUND_IF / IPV6_BOUND_IF (by index,
// resolved when the socket is made). Elsewhere: not pinned.
package sockbind

import "syscall"

// Control returns a net.Dialer / net.ListenConfig Control function that
// pins the socket to dev; dev "" leaves it unpinned.
func Control(dev string) func(network, address string, c syscall.RawConn) error {
	return func(network, _ string, c syscall.RawConn) error {
		if dev == "" {
			return nil
		}
		var serr error
		if err := c.Control(func(fd uintptr) { serr = bind(int(fd), network, dev) }); err != nil {
			return err
		}
		return serr
	}
}
