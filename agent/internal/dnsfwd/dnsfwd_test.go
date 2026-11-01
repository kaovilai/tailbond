package dnsfwd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoute(t *testing.T) {
	rc := filepath.Join(t.TempDir(), "resolv.conf")
	_ = os.WriteFile(rc, []byte("nameserver 192.0.2.53\n"), 0o644)
	up := true
	f := &Forwarder{
		ResolvConf:  rc,
		Suffixes:    func() []string { return []string{"tail1234.ts.net", "corp.example"} },
		TailscaleUp: func() bool { return up },
	}
	cases := map[string]string{
		"homebox.tail1234.ts.net.": MagicDNS,
		"x.ts.net":                 MagicDNS,
		"printer.corp.example.":    MagicDNS,
		"example.com.":             "192.0.2.53:53",
		"notts.net.":               "192.0.2.53:53",
	}
	for name, want := range cases {
		if got := f.Route(name); len(got) != 1 || got[0] != want {
			t.Errorf("Route(%s) = %v want %s", name, got, want)
		}
	}
	up = false
	if got := f.Route("homebox.tail1234.ts.net."); got[0] != "192.0.2.53:53" {
		t.Errorf("tailscale down should use system resolver, got %v", got)
	}
}
