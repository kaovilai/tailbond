package agent

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/status"
)

func clientAgent(t *testing.T, edit func(*config.Config)) *Agent {
	t.Helper()
	cfg := config.Default()
	cfg.Role = config.RoleClient
	cfg.DisableTailscale = true
	cfg.StateDir = t.TempDir()
	cfg.MqvpnBin = "/bin/false"
	cfg.HooksDir = t.TempDir()
	if edit != nil {
		edit(cfg)
	}
	cfg.ApplyDefaults()
	a := New(config.NewStore(filepath.Join(t.TempDir(), "tailbond.json"), cfg), Options{})
	a.loadServers()
	t.Cleanup(a.sup.Stop)
	return a
}

func pairURI(t *testing.T, server, name, subnet string) string {
	t.Helper()
	p := &pairing.Payload{V: pairing.Version, Server: server, User: "slate7", Key: "k-" + name, ServerName: name, Subnet: subnet}
	if subnet != "" {
		p.DNS = []string{strings.TrimSuffix(subnet, "0/24") + "1"}
	}
	u, err := pairing.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

const lan = "192.168.8.100:1"

func TestServersAddOrderRemove(t *testing.T) {
	a := clientAgent(t, nil)
	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	if w := req(t, a, "POST", "/api/pair", lan, "", map[string]string{"uri": home}); w.Code != 200 {
		t.Fatalf("pair: %d %s", w.Code, w.Body)
	}
	vps := pairURI(t, "198.51.100.9:8443", "vps", "10.83.142.0/24")
	var added struct{ Name string }
	w := req(t, a, "POST", "/api/servers", lan, "", map[string]string{"uri": vps})
	if w.Code != 200 {
		t.Fatalf("add: %d %s", w.Code, w.Body)
	}
	decode(t, w, &added)
	if added.Name != "vps" {
		t.Fatalf("added as %q", added.Name)
	}
	// same server again, and another one in the same subnet as homebox
	w = req(t, a, "POST", "/api/servers", lan, "", map[string]string{"uri": vps})
	var e struct{ Error, Conflict, Subnet string }
	decode(t, w, &e)
	if w.Code != http.StatusConflict || e.Error != "exists" || e.Conflict != "vps" {
		t.Fatalf("duplicate: %d %+v", w.Code, e)
	}
	clash := pairURI(t, "192.0.2.50:443", "cabin", "")
	w = req(t, a, "POST", "/api/servers", lan, "", map[string]string{"uri": clash})
	decode(t, w, &e)
	if w.Code != http.StatusConflict || e.Error != "subnet_in_use" || e.Conflict != "homebox" || e.Subnet != "10.77.0.0/24" {
		t.Fatalf("clash: %d %+v", w.Code, e)
	}

	a.poll(context.Background())
	st := a.Snapshot()
	if len(st.Servers) != 2 || st.ActiveServer != "homebox" || st.Servers[0].Role != "primary" || st.Servers[1].Role != "backup" ||
		st.Servers[1].Subnet != "10.83.142.0/24" || st.Servers[1].Server != "198.51.100.9:8443" || !st.Servers[0].Active {
		t.Fatalf("status servers %+v active %q", st.Servers, st.ActiveServer)
	}
	if st.Failover == nil || st.Failover.Policy != "prefer" || st.Failover.SwitchAfterSec != 15 || st.Failover.FailbackAfterSec != 120 {
		t.Fatalf("failover %+v", st.Failover)
	}

	// reorder: vps becomes the primary; pairing_uri follows
	if w := req(t, a, "POST", "/api/servers/order", lan, "", map[string]any{"names": []string{"vps", "homebox"}}); w.Code != 200 {
		t.Fatalf("order: %d %s", w.Code, w.Body)
	}
	if c := a.store.Get().Client; c.PairingURI != vps || c.Servers[0].Name != "vps" {
		t.Fatalf("after reorder %+v", c.Servers)
	}
	if w := req(t, a, "POST", "/api/servers/order", lan, "", map[string]any{"names": []string{"vps"}}); w.Code != 400 {
		t.Fatalf("partial order accepted: %d", w.Code)
	}

	// manual choice, then "always use"
	if w := req(t, a, "POST", "/api/servers/active", lan, "", map[string]string{"name": "homebox"}); w.Code != 200 {
		t.Fatalf("active: %d %s", w.Code, w.Body)
	}
	a.poll(context.Background())
	if st := a.Snapshot(); st.ActiveServer != "homebox" || st.Failover.Reason != "manual" {
		t.Fatalf("manual: %q %+v", st.ActiveServer, st.Failover)
	}
	if w := req(t, a, "POST", "/api/servers/policy", lan, "", map[string]string{"policy": "always", "server": "nope"}); w.Code != 400 {
		t.Fatalf("always unknown server: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/servers/policy", lan, "", map[string]string{"policy": "always", "server": "homebox"}); w.Code != 200 {
		t.Fatalf("always: %d %s", w.Code, w.Body)
	}
	a.poll(context.Background())
	if st := a.Snapshot(); st.ActiveServer != "homebox" || st.Failover.Reason != "policy" || st.Failover.Server != "homebox" {
		t.Fatalf("always: %q %+v", st.ActiveServer, st.Failover)
	}

	// removing the "always" server falls back to prefer and the other one
	if w := req(t, a, "DELETE", "/api/servers/homebox", lan, "", nil); w.Code != 200 {
		t.Fatalf("delete: %d", w.Code)
	}
	a.poll(context.Background())
	st = a.Snapshot()
	if len(st.Servers) != 1 || st.ActiveServer != "vps" || st.Failover.Policy != "prefer" || !a.paired() {
		t.Fatalf("after delete %+v %q %+v", st.Servers, st.ActiveServer, st.Failover)
	}
	if w := req(t, a, "DELETE", "/api/servers/homebox", lan, "", nil); w.Code != 404 {
		t.Fatalf("delete twice: %d", w.Code)
	}
	// unpair clears the whole list
	req(t, a, "DELETE", "/api/pair", lan, "", nil)
	if c := a.store.Get().Client; len(c.Servers) != 0 || c.PairingURI != "" || a.paired() {
		t.Fatalf("unpair left %+v", c.Servers)
	}
}

func TestReplacingPrimaryKeepsBackups(t *testing.T) {
	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	vps := pairURI(t, "198.51.100.9:8443", "vps", "10.83.142.0/24")
	a := clientAgent(t, func(c *config.Config) {
		c.Client.Servers = []config.ServerEntry{{Name: "homebox", PairingURI: home}, {Name: "vps", PairingURI: vps}}
	})
	home2 := pairURI(t, "203.0.113.8:443", "homebox", "10.77.0.0/24")
	if w := req(t, a, "POST", "/api/pair", lan, "", map[string]string{"uri": home2}); w.Code != 200 {
		t.Fatalf("re-pair: %d %s", w.Code, w.Body)
	}
	c := a.store.Get().Client
	if len(c.Servers) != 2 || c.Servers[0].PairingURI != home2 || c.Servers[1].Name != "vps" {
		t.Fatalf("%+v", c.Servers)
	}
	// a new primary clashing with the backup's subnet is refused
	bad := pairURI(t, "203.0.113.9:443", "other", "10.83.142.0/24")
	if w := req(t, a, "POST", "/api/pair", lan, "", map[string]string{"uri": bad}); w.Code != http.StatusConflict {
		t.Fatalf("clashing primary: %d", w.Code)
	}
}

func TestSwitchHookEnv(t *testing.T) {
	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	vps := pairURI(t, "198.51.100.9:8443", "vps", "10.83.142.0/24")
	a := clientAgent(t, func(c *config.Config) {
		c.Client.Servers = []config.ServerEntry{{Name: "homebox", PairingURI: home}, {Name: "vps", PairingURI: vps}}
	})
	a.mu.Lock()
	a.client.serverIP = "203.0.113.7"
	a.rt.tunnelIP = "10.77.0.2"
	a.markReadyLocked()
	up := a.rt.upEnv
	a.mu.Unlock()
	if up["TB_SERVER_NAME"] != "homebox" || up["TB_SUBNET"] != "10.77.0.0/24" || up["TB_DNS_IP"] != "10.77.0.1" || up["TB_SERVER_IP"] != "203.0.113.7" {
		t.Fatalf("up env %v", up)
	}
	// switch: the down hook still describes homebox, flagged as a switch
	a.mu.Lock()
	s := a.client.server("vps")
	a.client.activeServer, a.client.payload, a.client.serverIP = "vps", s.payload, "198.51.100.9"
	a.client.switchPending = true
	down := a.downEnvLocked()
	again := a.downEnvLocked()
	cur := a.hookEnvLocked()
	a.mu.Unlock()
	if down["TB_SERVER_IP"] != "203.0.113.7" || down["TB_SUBNET"] != "10.77.0.0/24" || down["TB_SWITCH"] != "1" {
		t.Fatalf("down env %v", down)
	}
	if again["TB_SWITCH"] != "" {
		t.Fatal("TB_SWITCH not consumed")
	}
	if cur["TB_SERVER_NAME"] != "vps" || cur["TB_SUBNET"] != "10.83.142.0/24" || cur["TB_DNS_IP"] != "10.83.142.1" {
		t.Fatalf("new env %v", cur)
	}
}

func TestJoinAddRenumbersFreshServer(t *testing.T) {
	srv := pairServer(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "100.101.9.9:40000"
		srv.Handler().ServeHTTP(w, r)
	}))
	defer ts.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))

	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	cli := clientAgent(t, func(c *config.Config) {
		c.Client.Servers = []config.ServerEntry{{Name: "homebox", PairingURI: home}}
	})
	cli.apiPort = port
	if w := req(t, cli, "POST", "/api/join", lan, "", map[string]any{"server": "127.0.0.1", "name": "slate7", "add": true}); w.Code != http.StatusAccepted {
		t.Fatalf("join: %d %s", w.Code, w.Body)
	}
	waitFor(t, func() bool { return cli.joinView().State == joinWaiting })
	srv.mu.Lock()
	pend := srv.pairReqs.pending()
	srv.mu.Unlock()
	req(t, srv, "POST", "/api/pair-requests/"+pend[0].ID+"/approve", "127.0.0.1:1", "", map[string]string{})
	waitFor(t, func() bool { return cli.joinView().State != joinWaiting })
	if j := cli.joinView(); j.State != joinApproved || j.Added == "" {
		t.Fatalf("join %+v", j)
	}
	moved := srv.store.Get().Server.Subnet
	if moved == "10.77.0.0/24" || !strings.HasPrefix(moved, "10.") {
		t.Fatalf("server subnet %s", moved)
	}
	c := cli.store.Get().Client
	if len(c.Servers) != 2 || c.Servers[0].Name != "homebox" {
		t.Fatalf("client servers %+v", c.Servers)
	}
	p, _ := pairing.Decode(c.Servers[1].PairingURI)
	if p.Subnet != moved || p.DNS[0] != strings.TrimSuffix(moved, "0/24")+"1" {
		t.Fatalf("payload subnet %s dns %v, server %s", p.Subnet, p.DNS, moved)
	}
}

func TestJoinAddRefusesClashWhenServerHasClients(t *testing.T) {
	srv := pairServer(t)
	if _, err := srv.addClient(context.Background(), "pixel"); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.RemoteAddr = "100.101.9.9:40000"
		srv.Handler().ServeHTTP(w, r)
	}))
	defer ts.Close()
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	cli := clientAgent(t, func(c *config.Config) {
		c.Client.Servers = []config.ServerEntry{{Name: "homebox", PairingURI: home}}
	})
	cli.apiPort = port
	req(t, cli, "POST", "/api/join", lan, "", map[string]any{"server": "127.0.0.1", "add": true})
	waitFor(t, func() bool { return cli.joinView().State == joinWaiting })
	srv.mu.Lock()
	pend := srv.pairReqs.pending()
	srv.mu.Unlock()
	req(t, srv, "POST", "/api/pair-requests/"+pend[0].ID+"/approve", "127.0.0.1:1", "", map[string]string{})
	waitFor(t, func() bool { return cli.joinView().State != joinWaiting })
	if j := cli.joinView(); j.State != joinError || j.Error != "subnet_in_use" {
		t.Fatalf("join %+v", j)
	}
	if srv.store.Get().Server.Subnet != "10.77.0.0/24" || len(cli.store.Get().Client.Servers) != 1 {
		t.Fatal("server with clients moved, or the clashing server was added")
	}
}

func TestPeersHostFollowsActiveServer(t *testing.T) {
	p := &pairing.Payload{Subnet: "10.83.142.0/24"}
	if got := p.TunnelSubnet().Addr().Next().String(); got != "10.83.142.1" {
		t.Fatal(got)
	}
}

func TestStatusServersOff(t *testing.T) {
	home := pairURI(t, "203.0.113.7:443", "homebox", "10.77.0.0/24")
	vps := pairURI(t, "198.51.100.9:8443", "vps", "10.83.142.0/24")
	off := false
	a := clientAgent(t, func(c *config.Config) {
		c.Client.Servers = []config.ServerEntry{{Name: "homebox", PairingURI: home}, {Name: "vps", PairingURI: vps, Enabled: &off}}
	})
	st := &status.Status{}
	a.serversView(st, false)
	if len(st.Servers) != 2 || st.Servers[1].State != "off" || st.Servers[0].State != "connecting" {
		t.Fatalf("%+v", st.Servers)
	}
}
