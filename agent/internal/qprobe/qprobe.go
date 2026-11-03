// Package qprobe checks that a standby server's mqvpn answers on its UDP
// port, without a tunnel (docs/architecture.md, "Multiple servers and
// failover" → Health): one 60-byte QUIC short-header packet with a random
// destination connection ID. A QUIC server that has no such connection
// answers with a Stateless Reset (RFC 9000 §10.3) and keeps no state;
// xquic, mqvpn's QUIC stack, does. Any reply means reachable.
package qprobe

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"time"
)

// Size of the probe: big enough that the server's reset (which must be
// smaller than what triggered it) is at least the RFC's 21-byte minimum.
const Size = 60

// Packet returns one probe datagram: header form 0, fixed bit 1, random
// key-phase/packet-number bits, then random bytes (connection ID + "payload").
func Packet() []byte {
	b := make([]byte, Size)
	_, _ = rand.Read(b)
	b[0] = 0x40 | (b[0] & 0x1f)
	return b
}

// Dial opens the UDP socket the probe goes out on (callers bind it to a WAN).
type Dial func(ctx context.Context, addr string) (net.Conn, error)

// Probe sends one probe to addr (host:port) and waits up to timeout for
// any datagram back.
func Probe(ctx context.Context, dial Dial, addr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if dial == nil {
		dial = func(ctx context.Context, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", addr)
		}
	}
	c, err := dial(ctx, addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	if _, err := c.Write(Packet()); err != nil {
		return err
	}
	buf := make([]byte, 1500)
	n, err := c.Read(buf)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("empty reply")
	}
	return nil
}
