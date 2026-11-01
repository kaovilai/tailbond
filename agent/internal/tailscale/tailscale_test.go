package tailscale

import "testing"

const statusFixture = `{
  "BackendState": "Running",
  "MagicDNSSuffix": "tail1234.ts.net",
  "CurrentTailnet": {"Name": "alex@example.com", "MagicDNSSuffix": "tail1234.ts.net"},
  "Self": {"HostName": "homebox", "DNSName": "homebox.tail1234.ts.net.", "OS": "linux",
           "TailscaleIPs": ["100.101.1.1", "fd7a:115c:a1e0::1"], "Online": true,
           "PrimaryRoutes": ["10.77.0.0/24"], "AllowedIPs": ["100.101.1.1/32", "10.77.0.0/24"]},
  "Peer": {
    "nodekey:a": {"HostName": "nas", "DNSName": "nas.tail1234.ts.net.", "OS": "linux",
                  "TailscaleIPs": ["100.101.2.2"], "Online": true, "LastSeen": "0001-01-01T00:00:00Z"},
    "nodekey:b": {"HostName": "Work-Laptop", "DNSName": "work-laptop.tail1234.ts.net.", "OS": "macOS",
                  "TailscaleIPs": ["100.101.3.3"], "Online": false, "LastSeen": "2026-10-30T08:00:00Z"},
    "nodekey:c": {"HostName": "apple-tv", "DNSName": "apple-tv.tail1234.ts.net.", "OS": "tvOS",
                  "TailscaleIPs": ["100.101.4.4"], "Online": true}
  }
}`

func TestParseStatusPeers(t *testing.T) {
	info := &Info{}
	if err := parseStatus([]byte(statusFixture), "10.77.0.0/24", info); err != nil {
		t.Fatal(err)
	}
	if !info.Running || !info.RoutesApproved || info.Tailnet != "alex@example.com" {
		t.Fatalf("info = %+v", info)
	}
	if info.SelfNode == nil || info.SelfNode.DNSName != "homebox.tail1234.ts.net" || !info.SelfNode.Online {
		t.Fatalf("self = %+v", info.SelfNode)
	}
	// online first, then by name
	want := []string{"apple-tv", "nas", "Work-Laptop"}
	if len(info.PeerList) != len(want) {
		t.Fatalf("peers = %+v", info.PeerList)
	}
	for i, n := range info.PeerList {
		if n.HostName != want[i] {
			t.Fatalf("peer %d = %s, want %s", i, n.HostName, want[i])
		}
	}
	nas := info.PeerList[1]
	if nas.DNSName != "nas.tail1234.ts.net" || nas.TailscaleIPs[0] != "100.101.2.2" || !nas.LastSeen.IsZero() {
		t.Fatalf("nas = %+v", nas)
	}
	if lap := info.PeerList[2]; lap.Online || lap.LastSeen.IsZero() || lap.OS != "macOS" {
		t.Fatalf("laptop = %+v", lap)
	}
	if info.Peers != 3 || info.PeersOnline != 2 {
		t.Fatalf("counts = %d/%d", info.PeersOnline, info.Peers)
	}
}
