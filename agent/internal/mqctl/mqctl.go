// Package mqctl talks to mqvpn's JSON control API
// (third_party/mqvpn/docs/control-api.md): one newline-terminated request per
// TCP connection, one JSON response, then the server closes.
package mqctl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

type Client struct {
	Addr    string
	Timeout time.Duration
}

func New(addr string) *Client { return &Client{Addr: addr, Timeout: 2 * time.Second} }

// ErrUnavailable means nothing listens on the control port (e.g. an unpatched
// client-mode mqvpn); callers degrade instead of failing.
var ErrUnavailable = errors.New("mqvpn control API unavailable")

type APIError struct{ Msg string }

func (e *APIError) Error() string { return "mqvpn: " + e.Msg }

func (c *Client) Do(ctx context.Context, req map[string]any, out any) error {
	d := net.Dialer{Timeout: c.Timeout}
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	b, _ := json.Marshal(req)
	if _, err := conn.Write(append(b, '\n')); err != nil {
		return err
	}
	r := bufio.NewReaderSize(conn, 64<<10)
	line, err := r.ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return fmt.Errorf("read control response: %w", err)
	}
	var env struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(line, &env); err != nil {
		return fmt.Errorf("decode control response: %w", err)
	}
	if !env.OK {
		return &APIError{env.Error}
	}
	if out != nil {
		return json.Unmarshal(line, out)
	}
	return nil
}

type Path struct {
	PathID          uint64 `json:"path_id"`
	Iface           string `json:"iface"` // client-mode patch only
	SrttMs          uint64 `json:"srtt_ms"`
	MinRttMs        uint64 `json:"min_rtt_ms"`
	Cwnd            uint64 `json:"cwnd"`
	InFlight        uint64 `json:"in_flight"`
	BytesTx         uint64 `json:"bytes_tx"`
	BytesRx         uint64 `json:"bytes_rx"`
	PktSent         uint64 `json:"pkt_sent"`
	PktRecv         uint64 `json:"pkt_recv"`
	PktLost         uint64 `json:"pkt_lost"`
	State           int    `json:"state"`
	StateLabel      string `json:"state_label"`
	ReinjectTxBytes uint64 `json:"reinject_tx_bytes"`
}

type ClientInfo struct {
	User         string `json:"user"`
	Endpoint     string `json:"endpoint"`
	Scheduler    string `json:"scheduler"`
	ConnectedSec uint64 `json:"connected_sec"`
	BytesTx      uint64 `json:"bytes_tx"`
	BytesRx      uint64 `json:"bytes_rx"`
	NPaths       int    `json:"n_paths"`
	Paths        []Path `json:"paths"`
}

type Status struct {
	NClients int          `json:"n_clients"`
	Clients  []ClientInfo `json:"clients"`
}

type Stats struct {
	NClients  int    `json:"n_clients"`
	BytesTx   uint64 `json:"bytes_tx"`
	BytesRx   uint64 `json:"bytes_rx"`
	DgramLost uint64 `json:"dgram_lost"`
	UptimeSec uint64 `json:"uptime_sec"`
}

func (c *Client) Status(ctx context.Context) (*Status, error) {
	s := &Status{}
	return s, c.Do(ctx, map[string]any{"cmd": "get_status"}, s)
}

func (c *Client) Stats(ctx context.Context) (*Stats, error) {
	s := &Stats{}
	return s, c.Do(ctx, map[string]any{"cmd": "get_stats"}, s)
}

func (c *Client) AddUser(ctx context.Context, name, key string) error {
	return c.Do(ctx, map[string]any{"cmd": "add_user", "name": name, "key": key}, nil)
}

func (c *Client) RemoveUser(ctx context.Context, name string) error {
	return c.Do(ctx, map[string]any{"cmd": "remove_user", "name": name}, nil)
}

func (c *Client) ListUsers(ctx context.Context) ([]string, error) {
	var out struct {
		Users []string `json:"users"`
	}
	err := c.Do(ctx, map[string]any{"cmd": "list_users"}, &out)
	return out.Users, err
}
