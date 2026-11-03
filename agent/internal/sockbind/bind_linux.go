//go:build linux

package sockbind

import "syscall"

func bind(fd int, _ string, dev string) error {
	return syscall.SetsockoptString(fd, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, dev)
}

// pins reports whether this platform pins sockets at all.
const pins = true
