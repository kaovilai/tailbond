package captive

import (
	"context"
	"net"

	"github.com/kaovilai/tailbond/agent/internal/sockbind"
)

// dialNet pins the socket to dev (sockbind: SO_BINDTODEVICE on Linux,
// IP_BOUND_IF on macOS), so the probe and its DNS go out over that WAN
// whatever the routing table prefers.
func dialNet(ctx context.Context, dev, network, addr string) (net.Conn, error) {
	d := net.Dialer{Control: sockbind.Control(dev)}
	return d.DialContext(ctx, network, addr)
}
