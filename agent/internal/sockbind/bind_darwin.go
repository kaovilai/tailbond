//go:build darwin

package sockbind

import (
	"net"
	"strings"
	"syscall"
)

// IP_BOUND_IF / IPV6_BOUND_IF (netinet/in.h, netinet6/in6.h)
const (
	ipBoundIf   = 25
	ipv6BoundIf = 125
)

func bind(fd int, network, dev string) error {
	ifi, err := net.InterfaceByName(dev)
	if err != nil {
		return err
	}
	if strings.HasSuffix(network, "6") {
		return syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, ipv6BoundIf, ifi.Index)
	}
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, ipBoundIf, ifi.Index)
}

// pins reports whether this platform pins sockets at all.
const pins = true
