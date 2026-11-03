package mqconf

import (
	"strings"
	"testing"
)

type kv struct{ sec, key, val string }

func parse(t *testing.T, ini string) []kv {
	t.Helper()
	var out []kv
	sec := ""
	for _, line := range strings.Split(ini, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			sec = strings.Trim(line, "[]")
		default:
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				t.Fatalf("bad line %q", line)
			}
			out = append(out, kv{sec, strings.TrimSpace(k), strings.TrimSpace(v)})
		}
	}
	return out
}

func get(kvs []kv, sec, key string) []string {
	var vals []string
	for _, x := range kvs {
		if x.sec == sec && x.key == key {
			vals = append(vals, x.val)
		}
	}
	return vals
}

func one(t *testing.T, kvs []kv, sec, key, want string) {
	t.Helper()
	got := get(kvs, sec, key)
	if want == "" {
		if len(got) != 0 {
			t.Errorf("[%s] %s = %v, want unset", sec, key, got)
		}
		return
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("[%s] %s = %v, want %q", sec, key, got, want)
	}
}

func clientParams(mode string) ClientParams {
	return ClientParams{
		Server: "203.0.113.7:443", ServerName: "homebox.tail1234.ts.net", Key: "k1",
		DNS: []string{"10.77.0.1"}, Paths: []string{"wlan0", "wwan0"}, Mode: mode,
		ControlAddr: "127.0.0.1:9077", ManageRoutes: true,
	}
}

func TestClientModes(t *testing.T) {
	cases := []struct {
		mode, sched, reinj, cc, redundant, reqSched, hybrid string
		reorder                                             bool
	}{
		{"speed", "wlb", "deadline", "", "", "true", "true", true},
		{"streaming", "wlb", "idle", "bbr", "true", "", "true", false},
		{"redundant", "minrtt", "idle", "bbr", "true", "true", "true", false},
		{"failover", "minrtt", "off", "", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			ini, err := Client(clientParams(c.mode))
			if err != nil {
				t.Fatal(err)
			}
			kvs := parse(t, ini)
			one(t, kvs, "Multipath", "Scheduler", c.sched)
			one(t, kvs, "Multipath", "Reinjection", c.reinj)
			one(t, kvs, "Multipath", "CC", c.cc)
			one(t, kvs, "Multipath", "Redundant", c.redundant)
			one(t, kvs, "Multipath", "RequestServerScheduler", c.reqSched)
			one(t, kvs, "Hybrid", "Enabled", c.hybrid)
			if c.reorder {
				one(t, kvs, "Reorder", "Enabled", "true")
				if p := get(kvs, "ReorderRule", "Profile"); len(p) != 2 || p[0] != "fiber_lte" {
					t.Errorf("reorder rules = %v", p)
				}
			} else {
				one(t, kvs, "Reorder", "Enabled", "")
			}
			one(t, kvs, "Server", "Address", "203.0.113.7:443")
			one(t, kvs, "Server", "ServerName", "homebox.tail1234.ts.net")
			one(t, kvs, "Server", "Insecure", "false")
			one(t, kvs, "Auth", "Key", "k1")
			one(t, kvs, "Interface", "TunName", "tb0")
			one(t, kvs, "Interface", "DNS", "10.77.0.1")
			one(t, kvs, "Interface", "ManageRoutes", "true")
			one(t, kvs, "Control", "Listen", "127.0.0.1:9077")
			if p := get(kvs, "Multipath", "Path"); strings.Join(p, ",") != "wlan0,wwan0" {
				t.Errorf("paths = %v", p)
			}
		})
	}
}

func TestClientNeedsPairing(t *testing.T) {
	p := clientParams("speed")
	p.Key = ""
	if _, err := Client(p); err == nil {
		t.Fatal("client without key accepted")
	}
	p = clientParams("bogus")
	if _, err := Client(p); err == nil {
		t.Fatal("unknown mode accepted")
	}
}

func serverParams(mode string) ServerParams {
	return ServerParams{
		Port: 443, Subnet: "10.77.0.0/24", Subnet6: "fd77:77::/112", TunName: "tb0",
		CertFile: "/var/lib/tailbond/server.crt", KeyFile: "/var/lib/tailbond/server.key",
		Users: []User{{"pixel", "a2V5MQ=="}, {"slate", "a2V5Mg=="}},
		Mode:  mode, ControlAddr: "127.0.0.1:9077",
		EgressAllow: []string{"100.64.0.0/10", "fd7a:115c:a1e0::/48"},
	}
}

func TestServer(t *testing.T) {
	for _, mode := range []string{"speed", "streaming", "redundant", "failover"} {
		ini, err := Server(serverParams(mode))
		if err != nil {
			t.Fatal(err)
		}
		kvs := parse(t, ini)
		m, _ := Mode(mode)
		one(t, kvs, "Interface", "Listen", "0.0.0.0:443")
		one(t, kvs, "Interface", "Subnet", "10.77.0.0/24")
		one(t, kvs, "Interface", "Subnet6", "fd77:77::/112")
		one(t, kvs, "Interface", "TunName", "tb0")
		one(t, kvs, "TLS", "Cert", "/var/lib/tailbond/server.crt")
		one(t, kvs, "Auth", "Key", "")
		if u := get(kvs, "Auth", "User"); len(u) != 2 || u[0] != "pixel:a2V5MQ==" {
			t.Errorf("users = %v", u)
		}
		one(t, kvs, "Multipath", "Scheduler", m.Scheduler)
		one(t, kvs, "Multipath", "AllowClientScheduler", "true")
		// Redundant is a client-only key.
		one(t, kvs, "Multipath", "Redundant", "")
		one(t, kvs, "Control", "Listen", "127.0.0.1:9077")
		one(t, kvs, "Hybrid", "Enabled", "true")
		if e := get(kvs, "Hybrid", "EgressAllow"); len(e) != 2 {
			t.Errorf("egress allow = %v", e)
		}
		one(t, kvs, "Reorder", "Enabled", "true")
	}
}

func TestServerGlobalKeyOnlyWithoutUsers(t *testing.T) {
	p := serverParams("speed")
	p.Users = nil
	if _, err := Server(p); err == nil {
		t.Fatal("server without any credential accepted")
	}
	p.GlobalKey = "Z2xvYmFs"
	ini, err := Server(p)
	if err != nil {
		t.Fatal(err)
	}
	one(t, parse(t, ini), "Auth", "Key", "Z2xvYmFs")
}

func TestServerRejectsInjection(t *testing.T) {
	p := serverParams("speed")
	p.Users = []User{{"evil\nListen", "x"}}
	if _, err := Server(p); err == nil {
		t.Fatal("bad user name accepted")
	}
	p.Users = []User{{"ok", "a:b"}}
	if _, err := Server(p); err == nil {
		t.Fatal("key containing ':' accepted")
	}
}

func TestValidUserName(t *testing.T) {
	for _, s := range []string{"pixel", "slate-7", "a.b_c"} {
		if !ValidUserName(s) {
			t.Errorf("%q rejected", s)
		}
	}
	for _, s := range []string{"", "-x", "a b", "a:b", strings.Repeat("a", 64), "../x"} {
		if ValidUserName(s) {
			t.Errorf("%q accepted", s)
		}
	}
}

func TestClientMTU(t *testing.T) {
	p := clientParams("speed")
	ini, err := Client(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ini, "MTU") {
		t.Fatalf("MTU 0 must leave mqvpn's default, got:\n%s", ini)
	}
	p.MTU = 1280
	if ini, err = Client(p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ini, "MTU = 1280") {
		t.Fatalf("want MTU = 1280 under [Interface], got:\n%s", ini)
	}
}
