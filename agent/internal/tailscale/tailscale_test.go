package tailscale

import (
	"net/netip"
	"testing"
	"time"
)

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

func TestParseWhois(t *testing.T) {
	const out = `{"Node":{"ID":42,"StableID":"nSLATE7CNTRL","Name":"slate7.tail1234.ts.net.","ComputedName":"slate7",
	  "Hostinfo":{"Hostname":"GL-BE3600","OS":"linux"},"Addresses":["100.101.9.9/32"]},
	  "UserProfile":{"ID":7,"LoginName":"alex@example.com","DisplayName":"Alex"}}`
	w, err := ParseWhois([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if w.NodeID != "nSLATE7CNTRL" || w.Node != "slate7" || w.DNSName != "slate7.tail1234.ts.net" || w.Hostname != "GL-BE3600" || w.User != "alex@example.com" || w.UserName != "Alex" {
		t.Fatalf("whois = %+v", w)
	}
	if _, err := ParseWhois([]byte(`{}`)); err == nil {
		t.Fatal("empty whois accepted")
	}
	w, _ = ParseWhois([]byte(`{"Node":{"ID":9,"Name":"tv.tail1234.ts.net."}}`))
	if w == nil || w.NodeID != "9" || w.Node != "tv" {
		t.Fatalf("fallbacks: %+v", w)
	}
}

func TestIsTailnetIP(t *testing.T) {
	for s, want := range map[string]bool{"100.64.0.1": true, "100.127.255.254": true, "100.128.0.1": false, "10.77.0.2": false,
		"fd7a:115c:a1e0::5": true, "fd77:77::2": false, "::ffff:100.101.1.1": true} {
		if got := IsTailnetIP(netip.MustParseAddr(s)); got != want {
			t.Errorf("%s: %v", s, got)
		}
	}
}

func TestPeerConn(t *testing.T) {
	// relay is claimed only for a peer with traffic in the last couple of
	// minutes and no direct path: an idle peer is not "relayed", it just has
	// no live connection (that was the bug behind "everything is via relay")
	wasNow := now
	now = func() time.Time { return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC) }
	defer func() { now = wasNow }()
	fresh, stale := `"2026-11-05T11:59:30Z"`, `"2026-11-05T09:00:00Z"`
	info := &Info{}
	err := parseStatus([]byte(`{"BackendState":"Running","Self":{"HostName":"homebox","DNSName":"homebox.t.ts.net.","TailscaleIPs":["100.101.1.1"]},
	 "Peer":{"a":{"HostName":"nas","Online":true,"CurAddr":"192.168.1.20:41641","Relay":"fra","LastHandshake":`+fresh+`},
	         "b":{"HostName":"slate7","Online":true,"CurAddr":"","Relay":"fra","LastHandshake":`+fresh+`},
	         "c":{"HostName":"idle","Online":true,"CurAddr":"","Relay":""},
	         "d":{"HostName":"old","Online":false,"CurAddr":"","Relay":"nyc","LastHandshake":`+fresh+`},
	         "e":{"HostName":"quiet","Online":true,"CurAddr":"","Relay":"fra","LastHandshake":`+stale+`},
	         "f":{"HostName":"unused","Online":true,"CurAddr":"","Relay":"fra"}}}`), "", info)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, p := range info.PeerList {
		got[p.HostName] = p.Conn + "/" + p.Relay
	}
	if got["nas"] != "direct/" || got["slate7"] != "relay/fra" || got["idle"] != "/" || got["old"] != "/" || got["quiet"] != "/" || got["unused"] != "/" {
		t.Fatalf("%v", got)
	}
}

func TestParseNetcheck(t *testing.T) {
	// tailscale netcheck --format=json (opt.Bool fields are strings)
	n, err := ParseNetcheck([]byte(`{"UDP":true,"IPv4":true,"GlobalV4":"203.0.113.7:41641","IPv6":false,
	 "MappingVariesByDestIP":"true","UPnP":"false","PMP":"","PCP":"true","PreferredDERP":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if !n.UDP || !n.MappingVariesByDestIP || n.UPnP || n.PMP || !n.PCP || n.GlobalV4 != "203.0.113.7:41641" || n.PreferredDERP != 4 {
		t.Fatalf("%+v", n)
	}
}
