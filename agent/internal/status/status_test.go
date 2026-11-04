package status

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/mqctl"
)

const serverStatusJSON = `{
  "ok":true,"n_clients":1,
  "clients":[{
    "user":"pixel","endpoint":"198.51.100.9:50000","scheduler":"wlb","connected_sec":42,
    "bytes_tx":1000,"bytes_rx":2000,"n_paths":3,"paths":[
      {"path_id":0,"srtt_ms":31,"min_rtt_ms":18,"cwnd":1,"in_flight":0,"bytes_tx":900,"bytes_rx":1900,
       "pkt_sent":100,"pkt_recv":99,"pkt_lost":1,"state":2,"state_label":"active","reinject_tx_bytes":0},
      {"path_id":1,"srtt_ms":80,"min_rtt_ms":60,"bytes_tx":100,"bytes_rx":100,
       "pkt_sent":10,"pkt_recv":10,"pkt_lost":0,"state":1,"state_label":"validating"},
      {"path_id":2,"srtt_ms":0,"min_rtt_ms":0,"bytes_tx":5,"bytes_rx":5,
       "pkt_sent":1,"pkt_recv":1,"pkt_lost":0,"state":4,"state_label":"closed"}
    ]}]}`

func decode[T any](t *testing.T, s string) *T {
	t.Helper()
	v := new(T)
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestServerNormalize(t *testing.T) {
	r := NewRater()
	t0 := time.Unix(1000, 0)
	st := decode[mqctl.Status](t, serverStatusJSON)
	stats := &mqctl.Stats{BytesTx: 10_000, BytesRx: 20_000, UptimeSec: 3600}

	r.Begin()
	clients, tot, up := Server(r, t0, st, stats, []string{"pixel", "slate"})
	r.End()
	if up != 3600 || tot.TxBytes != 10_000 || tot.TxBps != 0 {
		t.Fatalf("first poll totals %+v up %d", tot, up)
	}
	if len(clients) != 2 {
		t.Fatalf("want 2 clients (1 live + 1 offline), got %d", len(clients))
	}
	pixel, slate := clients[0], clients[1]
	if !pixel.Connected || !pixel.Registered || pixel.Endpoint != "198.51.100.9:50000" || len(pixel.Paths) != 3 {
		t.Fatalf("pixel: %+v", pixel)
	}
	if slate.Connected || !slate.Registered || slate.Paths == nil {
		t.Fatalf("slate: %+v", slate)
	}
	want := []string{PathActive, PathPending, PathFailed}
	for i, p := range pixel.Paths {
		if p.State != want[i] {
			t.Errorf("path %d state %q want %q", i, p.State, want[i])
		}
	}
	if pixel.Paths[0].LossPct != 1 || pixel.Paths[0].SrttMs != 31 || pixel.Paths[0].Kind != "unknown" {
		t.Errorf("path0: %+v", pixel.Paths[0])
	}

	// second poll one second later: +1000 bytes tx = 8000 bit/s, 10 new pkts with 1 lost
	st.Clients[0].Paths[0].BytesTx += 1000
	st.Clients[0].Paths[0].BytesRx += 500
	st.Clients[0].Paths[0].PktSent += 10
	st.Clients[0].Paths[0].PktLost += 1
	stats.BytesTx += 125_000
	r.Begin()
	clients, tot, _ = Server(r, t0.Add(time.Second), st, stats, []string{"pixel", "slate"})
	r.End()
	p0 := clients[0].Paths[0]
	if p0.TxBps != 8000 || p0.RxBps != 4000 {
		t.Errorf("rates tx=%d rx=%d", p0.TxBps, p0.RxBps)
	}
	// EWMA: 0.7*1 + 0.3*10 = 3.7
	if p0.LossPct != 3.7 {
		t.Errorf("loss %v", p0.LossPct)
	}
	if tot.TxBps != 1_000_000 {
		t.Errorf("total tx bps %d", tot.TxBps)
	}
}

func TestRaterCounterReset(t *testing.T) {
	r := NewRater()
	t0 := time.Unix(0, 0)
	r.Rate("x", t0, 1000, 1000, 0, 0)
	if tx, _, _ := r.Rate("x", t0.Add(time.Second), 2000, 1000, 0, 0); tx != 8000 {
		t.Fatalf("tx %d", tx)
	}
	if tx, rx, _ := r.Rate("x", t0.Add(2*time.Second), 10, 10, 0, 0); tx != 0 || rx != 0 {
		t.Fatalf("reset not detected: %d %d", tx, rx)
	}
	r.Begin()
	r.End()
	if len(r.prev) != 0 {
		t.Fatal("stale series kept")
	}
}

const clientStatusJSON = `{"ok":true,"n_clients":1,"clients":[{
  "user":"pixel","endpoint":"203.0.113.7:443","scheduler":"wlb","connected_sec":7,
  "bytes_tx":300,"bytes_rx":400,"n_paths":3,"paths":[
   {"path_id":0,"iface":"wlan0","srtt_ms":20,"min_rtt_ms":10,"bytes_tx":100,"bytes_rx":200,"pkt_sent":5,"pkt_lost":0,"state_label":"closed"},
   {"path_id":1,"iface":"wwan0","srtt_ms":50,"min_rtt_ms":40,"bytes_tx":100,"bytes_rx":100,"pkt_sent":5,"pkt_lost":0,"state_label":"active"},
   {"path_id":2,"iface":"wlan0","srtt_ms":22,"min_rtt_ms":11,"bytes_tx":10,"bytes_rx":20,"pkt_sent":1,"pkt_lost":0,"state_label":"active"}
  ]}]}`

func TestClientNormalize(t *testing.T) {
	r := NewRater()
	st := decode[mqctl.Status](t, clientStatusJSON)
	meta := func(iface string) (string, string, bool) {
		if iface == "wwan0" {
			return "cellular", "backup", true
		}
		return "wifi", "primary", true
	}
	paths, tot, up, connected := ClientFromControl(r, time.Unix(0, 0), st, meta)
	if !connected || up != 7 || tot.TxBytes != 300 {
		t.Fatalf("connected=%v up=%d tot=%+v", connected, up, tot)
	}
	if len(paths) != 2 {
		t.Fatalf("want wlan0 deduped to 2 paths, got %+v", paths)
	}
	if paths[0].Iface != "wwan0" || paths[0].Kind != "cellular" || paths[0].Priority != "backup" {
		t.Errorf("path0 %+v", paths[0])
	}
	if paths[1].Iface != "wlan0" || paths[1].ID != 2 || paths[1].State != PathActive || paths[1].Kind != "wifi" {
		t.Errorf("wlan0 should be the live path 2: %+v", paths[1])
	}
	if _, _, _, ok := ClientFromControl(r, time.Unix(0, 0), &mqctl.Status{}, meta); ok {
		t.Error("empty status reported connected")
	}
}

func TestStatusJSONShape(t *testing.T) {
	s := Status{Role: "client", Mode: "speed", State: StateConnected, Paths: []Path{{ID: 0, Iface: "wlan0", Kind: "wifi", State: PathActive, Enabled: true, Priority: "primary"}}, Clients: []Client{}, Tailscale: &Tailscale{}}
	b, _ := json.Marshal(s)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"role", "mode", "state", "server", "tunnel_ip", "uptime_sec", "totals", "paths", "clients", "tailscale"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing top-level key %q", k)
		}
	}
	p := m["paths"].([]any)[0].(map[string]any)
	for _, k := range []string{"id", "iface", "kind", "state", "srtt_ms", "min_rtt_ms", "loss_pct", "tx_bps", "rx_bps", "tx_bytes", "rx_bytes", "priority", "enabled"} {
		if _, ok := p[k]; !ok {
			t.Errorf("missing path key %q", k)
		}
	}
	ts := m["tailscale"].(map[string]any)
	for _, k := range []string{"running", "self", "routes_approved"} {
		if _, ok := ts[k]; !ok {
			t.Errorf("missing tailscale key %q", k)
		}
	}
}

// A reconnect can leave the previous session on the server until its idle
// timeout: the device shows once, with its newest session's paths.
func TestServerNewestSessionPerUser(t *testing.T) {
	st := &mqctl.Status{Clients: []mqctl.ClientInfo{
		{User: "mac", Endpoint: "198.51.100.7:50001", ConnectedSec: 95, Paths: []mqctl.Path{{PathID: 0, StateLabel: "active"}, {PathID: 1, StateLabel: "active"}, {PathID: 2, StateLabel: "active"}}},
		{User: "mac", Endpoint: "198.51.100.7:50002", ConnectedSec: 12, Paths: []mqctl.Path{{PathID: 0, StateLabel: "active"}, {PathID: 1, StateLabel: "active"}}},
		{User: "pixel", Endpoint: "203.0.113.9:4000", ConnectedSec: 400, Paths: []mqctl.Path{{PathID: 0, StateLabel: "active"}}},
	}}
	cs, _, _ := Server(NewRater(), time.Now(), st, nil, []string{"mac", "pixel"})
	if len(cs) != 2 {
		t.Fatalf("clients: %+v", cs)
	}
	if cs[0].Name != "mac" || cs[0].Endpoint != "198.51.100.7:50002" || len(cs[0].Paths) != 2 || !cs[0].Connected {
		t.Errorf("mac: %+v", cs[0])
	}
	if cs[1].Name != "pixel" || len(cs[1].Paths) != 1 {
		t.Errorf("pixel: %+v", cs[1])
	}
}
