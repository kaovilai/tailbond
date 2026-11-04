//go:build darwin

package share

import "syscall"

// reusePort lets the mDNS responder share port 5353 with mDNSResponder,
// which holds it with SO_REUSEPORT.
func reusePort(fd int) error {
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
}
