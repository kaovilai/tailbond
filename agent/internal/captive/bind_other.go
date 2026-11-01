//go:build !linux

package captive

import (
	"context"
	"net"
)

func dialNet(ctx context.Context, _ string, network, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}
