// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kaovilai/tailbond/agent/internal/report"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

func TestReportPreviewAndURL(t *testing.T) {
	a := pairServer(t)
	a.Logf("dial 203.0.113.7:443 with token=hunter2 from homebox.tail1234.ts.net")
	w := req(t, a, "GET", "/api/report", "127.0.0.1:1", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/report = %d %s", w.Code, w.Body)
	}
	var got map[string]string
	json.Unmarshal(w.Body.Bytes(), &got)
	if got["component"] == "" || got["version"] != "test" || got["device"] == "" || got["notice"] != report.Notice || got["security_url"] == "" {
		t.Fatalf("preview: %+v", got)
	}
	for _, bad := range []string{"203.0.113.7", "hunter2", "tail1234"} {
		if strings.Contains(w.Body.String(), bad) {
			t.Errorf("preview has %q", bad)
		}
	}
	if !strings.Contains(got["logs"], "[public IPv4]:443") {
		t.Errorf("logs: %q", got["logs"])
	}
	// the user's edits are redacted again
	w = req(t, a, "POST", "/api/report", "127.0.0.1:1", "", map[string]any{
		"component": got["component"], "version": got["version"], "device": got["device"],
		"what": "It stops; my key=hunter2 and server 198.51.100.4", "logs": got["logs"],
	})
	var out map[string]string
	json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != http.StatusOK || !strings.HasPrefix(out["url"], "https://github.com/kaovilai/tailbond/issues/new?template=bug_report.yml&") ||
		len(out["url"]) > report.MaxURL || !strings.HasPrefix(out["file_name"], "tailbond-report-") {
		t.Fatalf("POST: %d %+v", w.Code, out)
	}
	if strings.Contains(out["url"], "hunter2") || strings.Contains(out["full"], "198.51.100.4") || !strings.Contains(out["full"], "### Logs") {
		t.Errorf("not redacted: %+v", out)
	}
}

func TestCrashKeptUntilDismissed(t *testing.T) {
	a := pairServer(t)
	// the previous run's fatal error output
	os.WriteFile(a.crashPath(crashFile), []byte("panic: boom token=hunter2\ngoroutine 1 [running]:\n"), 0o600)
	a.setupCrash()
	if a.crashAt() == "" {
		t.Fatal("no crash kept")
	}
	if fi, _ := os.Stat(a.crashPath(crashFile)); fi == nil || fi.Size() != 0 {
		t.Error("this run's crash file isn't empty")
	}
	a.poll(t.Context())
	if a.Snapshot().Crash == "" {
		t.Error("status has no crash")
	}
	var got map[string]string
	json.Unmarshal(req(t, a, "GET", "/api/report", "127.0.0.1:1", "", nil).Body.Bytes(), &got)
	if !strings.Contains(got["crash"], "panic: boom token=[redacted]") || !strings.Contains(got["crash"], "stopped with a fatal error") {
		t.Errorf("crash: %q", got["crash"])
	}
	req(t, a, "POST", "/api/report/crash/dismiss", "127.0.0.1:1", "", map[string]any{})
	if a.crashAt() != "" {
		t.Error("still kept after dismiss")
	}
	// a clean run keeps nothing
	a.setupCrash()
	if a.crashAt() != "" {
		t.Error("a crash from an empty crash.log")
	}
}

func TestMqvpnCrashSignal(t *testing.T) {
	a := pairServer(t)
	a.logs.Add("mqvpn", "path wifi up")
	err := exec.Command("sh", "-c", "kill -SEGV $$").Run()
	a.noteChildCrash(err)
	b, _ := os.ReadFile(a.crashPath(crashLast))
	if !strings.Contains(string(b), "mqvpn stopped with signal segmentation fault") || !strings.Contains(string(b), "path wifi up") {
		t.Fatalf("crash record: %q", b)
	}
	// a normal stop isn't a crash
	os.Remove(a.crashPath(crashLast))
	a.noteChildCrash(exec.Command("sh", "-c", "kill -TERM $$").Run())
	a.noteChildCrash(errors.New("exit status 1"))
	if _, err := os.Stat(a.crashPath(crashLast)); err == nil {
		t.Error("SIGTERM recorded as a crash")
	}
}

// The work VPN's internal names (its split-DNS domains) are this machine's
// secret: the report says [work name].
func TestReportRedactsWorkNames(t *testing.T) {
	a := pairServer(t)
	a.mu.Lock()
	a.snap = &status.Status{Role: "client", Coexist: &status.Coexist{DNSDomains: []string{"corp.example.", "netbird.cloud"},
		Foreign: []status.CoexistForeign{
			{Name: "NetBird", Kind: "netbird", Iface: "utun100", Addrs: []string{"100.92.1.2/16"}, Prefixes: []string{"100.92.0.0/16", "10.20.0.0/16"}},
			{Name: "Tailscale", Kind: "tailscale", Iface: "utun4", Addrs: []string{"100.101.1.1/32"}, Prefixes: []string{"100.64.0.0/10"}},
		}}}
	a.mu.Unlock()
	a.Logf("resolving git.eu.corp.example via 10.92.0.5; peer laptop.netbird.cloud; not notcorp.example.org")
	a.Logf("netbird 100.92.1.2 routes 10.20.0.0/16 (host 10.20.3.4); tailnet 100.101.1.1; home 192.168.8.1")
	var got map[string]string
	json.Unmarshal(req(t, a, "GET", "/api/report", "127.0.0.1:1", "", nil).Body.Bytes(), &got)
	if !strings.Contains(got["logs"], "resolving [work name] via 10.92.0.5; peer [work name];") {
		t.Errorf("logs: %q", got["logs"])
	}
	if !strings.Contains(got["logs"], "netbird [work address] routes [work network] (host [work address]); tailnet 100.101.1.1; home 192.168.8.1") {
		t.Errorf("work addresses: %q", got["logs"])
	}
	if got["account_note"] != report.AccountNote {
		t.Errorf("account note: %q", got["account_note"])
	}
	var out map[string]string
	json.Unmarshal(req(t, a, "POST", "/api/report", "127.0.0.1:1", "", map[string]any{"what": "can't open wiki.corp.example"}).Body.Bytes(), &out)
	if strings.Contains(out["url"], "corp.example") || !strings.Contains(out["full"], "can't open [work name]") {
		t.Errorf("POST: %q", out["full"])
	}
}
