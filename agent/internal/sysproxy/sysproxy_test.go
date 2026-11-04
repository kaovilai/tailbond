package sysproxy

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fakeMac is networksetup and route on a Mac with two services.
type fakeMac struct {
	socks map[string]*Settings
	pac   map[string]*PAC
	dev   string
	calls []string
	fail  string // a networksetup verb that fails
}

func newFake() *fakeMac {
	return &fakeMac{
		socks: map[string]*Settings{"Wi-Fi": {}, "iPhone USB": {Server: "10.0.0.9", Port: 1080}},
		pac:   map[string]*PAC{"Wi-Fi": {}, "iPhone USB": {}},
		dev:   "en0",
	}
}

func (f *fakeMac) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if name == "netstat" {
		return []byte("Routing tables\n\nInternet:\nDestination        Gateway            Flags               Netif Expire\n" +
			"0/1                10.77.0.1          UGSc                utun4\n" +
			"default            link#22            UCSIg               utun4\n" +
			"default            192.168.1.1        UGScg               " + f.dev + "\n" +
			"127                127.0.0.1          UCS                   lo0\n"), nil
	}
	if len(args) > 0 && args[0] == f.fail {
		return []byte("** Error: nope"), fmt.Errorf("exit status 1")
	}
	switch args[0] {
	case "-listnetworkserviceorder":
		return []byte(`An asterisk (*) denotes that a network service is disabled.
(1) Wi-Fi
(Hardware Port: Wi-Fi, Device: en0)

(2) iPhone USB
(Hardware Port: iPhone USB, Device: en7)

(*) Thunderbolt Bridge
(Hardware Port: Thunderbolt Bridge, Device: bridge0)
`), nil
	case "-getsocksfirewallproxy":
		s := f.socks[args[1]]
		yes := map[bool]string{true: "Yes", false: "No"}
		auth := map[bool]string{true: "1", false: "0"}
		return []byte(fmt.Sprintf("Enabled: %s\nServer: %s\nPort: %d\nAuthenticated Proxy Enabled: %s\n", yes[s.Enabled], s.Server, s.Port, auth[s.Auth])), nil
	case "-setsocksfirewallproxy":
		s := f.socks[args[1]]
		s.Server = args[2]
		s.Port, _ = strconv.Atoi(args[3])
		s.Enabled = true // networksetup turns it on
		s.Auth = len(args) > 4 && args[4] == "on"
	case "-setsocksfirewallproxystate":
		f.socks[args[1]].Enabled = args[2] == "on"
	case "-getautoproxyurl":
		p := f.pac[args[1]]
		u := p.URL
		if u == "" {
			u = "(null)"
		}
		yes := map[bool]string{true: "Yes", false: "No"}
		return []byte(fmt.Sprintf("URL: %s\nEnabled: %s\n", u, yes[p.Enabled])), nil
	case "-setautoproxyurl":
		f.pac[args[1]].URL = args[2]
		f.pac[args[1]].Enabled = true
	case "-setautoproxystate":
		f.pac[args[1]].Enabled = args[2] == "on"
	}
	return nil, nil
}

func TestActiveService(t *testing.T) {
	f := newFake()
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	ctx := context.Background()
	if s, err := m.ActiveService(ctx); err != nil || s != "Wi-Fi" {
		t.Fatalf("en0: %q %v", s, err)
	}
	f.dev = "en7"
	if s, _ := m.ActiveService(ctx); s != "iPhone USB" {
		t.Fatalf("en7: %q", s)
	}
	f.dev = "bridge0"
	if s, _ := m.ActiveService(ctx); s != "Thunderbolt Bridge" {
		t.Fatalf("disabled service: %q", s)
	}
	f.dev = "utun9"
	if _, err := m.ActiveService(ctx); err == nil {
		t.Fatal("no service for utun9, want an error")
	}
}

func TestUseAndRestore(t *testing.T) {
	f := newFake()
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	ctx := context.Background()
	if err := m.Use(ctx, "Wi-Fi", "127.0.0.1", 1889, "", ""); err != nil {
		t.Fatal(err)
	}
	if s := f.socks["Wi-Fi"]; !s.Enabled || s.Server != "127.0.0.1" || s.Port != 1889 || s.Auth {
		t.Fatalf("set: %+v", s)
	}
	if r, ok := m.Active(); !ok || r.Service != "Wi-Fi" || r.Mode != "socks" || r.Previous.Enabled {
		t.Fatalf("record: %+v %v", r, ok)
	}
	// using it again (another hub address) keeps the original "previous"
	if err := m.Use(ctx, "Wi-Fi", "127.0.0.1", 1890, "", ""); err != nil {
		t.Fatal(err)
	}
	if r, _ := m.Active(); r.Previous.Enabled || r.Previous.Port != 0 {
		t.Fatalf("previous overwritten: %+v", r.Previous)
	}
	if err := m.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if f.socks["Wi-Fi"].Enabled {
		t.Fatal("restore left the proxy on")
	}
	if _, ok := m.Active(); ok {
		t.Fatal("record left behind")
	}
	if err := m.Restore(ctx); err != nil {
		t.Fatalf("restore with no record: %v", err)
	}
}

// A service that had its own SOCKS proxy gets it back.
func TestRestorePrevious(t *testing.T) {
	f := newFake()
	f.socks["iPhone USB"].Enabled = true
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	ctx := context.Background()
	if err := m.Use(ctx, "iPhone USB", "127.0.0.1", 1889, "u", "p"); err != nil {
		t.Fatal(err)
	}
	if s := f.socks["iPhone USB"]; s.Server != "127.0.0.1" || !s.Auth {
		t.Fatalf("set: %+v", s)
	}
	if err := m.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if s := f.socks["iPhone USB"]; !s.Enabled || s.Server != "10.0.0.9" || s.Port != 1080 {
		t.Fatalf("restored: %+v", s)
	}
}

// The default route moved to another service: the first is put back
// before the second is changed.
func TestUseMovesService(t *testing.T) {
	f := newFake()
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	ctx := context.Background()
	_ = m.Use(ctx, "Wi-Fi", "127.0.0.1", 1889, "", "")
	if err := m.Use(ctx, "iPhone USB", "127.0.0.1", 1889, "", ""); err != nil {
		t.Fatal(err)
	}
	if f.socks["Wi-Fi"].Enabled {
		t.Fatal("Wi-Fi still pointed at the hub")
	}
	if r, _ := m.Active(); r.Service != "iPhone USB" {
		t.Fatalf("record: %+v", r)
	}
}

func TestPAC(t *testing.T) {
	f := newFake()
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	ctx := context.Background()
	if err := m.UsePAC(ctx, "Wi-Fi", "http://127.0.0.1:8077/proxy.pac"); err != nil {
		t.Fatal(err)
	}
	if p := f.pac["Wi-Fi"]; !p.Enabled || p.URL != "http://127.0.0.1:8077/proxy.pac" {
		t.Fatalf("pac: %+v", p)
	}
	// switching to SOCKS puts the PAC setting back first
	if err := m.Use(ctx, "Wi-Fi", "127.0.0.1", 1889, "", ""); err != nil {
		t.Fatal(err)
	}
	if f.pac["Wi-Fi"].Enabled {
		t.Fatal("PAC still on after switching to SOCKS")
	}
	if err := m.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if f.socks["Wi-Fi"].Enabled || f.pac["Wi-Fi"].Enabled {
		t.Fatalf("not restored: %+v %+v", f.socks["Wi-Fi"], f.pac["Wi-Fi"])
	}
}

// The record is written before anything changes: a failure part-way
// still leaves something to restore from.
func TestRecordBeforeChange(t *testing.T) {
	f := newFake()
	f.fail = "-setsocksfirewallproxystate"
	m := &Mac{RecordPath: filepath.Join(t.TempDir(), "r.json"), Run: f.run}
	if err := m.Use(context.Background(), "Wi-Fi", "127.0.0.1", 1889, "", ""); err == nil {
		t.Fatal("want the networksetup error")
	}
	if _, ok := m.Active(); !ok {
		t.Fatal("no record after a part-way change")
	}
}

func TestParseGet(t *testing.T) {
	s := parseGet("Enabled: Yes\nServer: fe80::1%en0\nPort: 1888\nAuthenticated Proxy Enabled: 1\n")
	if !s.Enabled || s.Server != "fe80::1%en0" || s.Port != 1888 || !s.Auth {
		t.Fatalf("%+v", s)
	}
}
