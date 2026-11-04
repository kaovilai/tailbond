//go:build !linux

package share

import "syscall"

func reusePort(fd int) error {
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
