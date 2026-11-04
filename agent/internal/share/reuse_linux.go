//go:build linux

package share

import "syscall"

// reusePort lets the mDNS responder share port 5353 with avahi/umdns.
func reusePort(fd int) error {
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, 0xf /* SO_REUSEPORT */, 1)
}
