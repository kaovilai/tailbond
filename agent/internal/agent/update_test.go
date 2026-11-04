package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/backup"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/pairing"
	"github.com/kaovilai/tailbond/agent/internal/status"
)

// fakeGitHub is GitHub's latest-release API plus the release's files.
type fakeGitHub struct {
	mu      sync.Mutex
	version string
	files   map[string][]byte
	sums    map[string]string // name -> sha256 (override = tampered)
	calls   int
	srv     *httptest.Server
}

func targz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	return b.Bytes()
}

func newFakeGitHub(t *testing.T, version string) *fakeGitHub {
	t.Helper()
	g := &fakeGitHub{}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		switch {
		case r.URL.Path == "/repos/kaovilai/tailbond/releases/latest":
			g.calls++
			var as []map[string]any
			for n := range g.files {
				as = append(as, map[string]any{"name": n, "browser_download_url": g.srv.URL + "/dl/" + n, "size": len(g.files[n])})
			}
			as = append(as, map[string]any{"name": "SHA256SUMS", "browser_download_url": g.srv.URL + "/dl/SHA256SUMS"})
			json.NewEncoder(w).Encode(map[string]any{"tag_name": "v" + g.version, "html_url": "https://github.com/kaovilai/tailbond/releases/tag/v" + g.version, "assets": as})
		case r.URL.Path == "/dl/SHA256SUMS":
			for n, b := range g.files {
				s := sha256.Sum256(b)
				h := hex.EncodeToString(s[:])
				if o, ok := g.sums[n]; ok {
					h = o
				}
				fmt.Fprintf(w, "%s  %s\n", h, n)
			}
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			if b, ok := g.files[strings.TrimPrefix(r.URL.Path, "/dl/")]; ok {
				w.Write(b)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(g.srv.Close)
	g.publish(t, version)
	return g
}

func (g *fakeGitHub) publish(t *testing.T, version string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.version, g.sums = version, map[string]string{}
	g.files = map[string][]byte{
		bundleName() + ".tar.gz": targz(t, map[string]string{
			bundleName() + "/install.sh":   "#!/bin/sh\n",
			bundleName() + "/VERSION":      version + "\n",
			bundleName() + "/../../escape": "x",
		}),
		"tailbond_" + version + "-r1_aarch64_cortex-a53.ipk":            []byte("ipk a53"),
		"tailbond_" + version + "-r1_aarch64_cortex-a53_neon-vfpv4.ipk": []byte("ipk neon"),
	}
}

type runCall struct {
	name      string
	args, env []string
}

// updAgent: a server agent on 0.1.0 checking g, with a recording runner.
func updAgent(t *testing.T, g *fakeGitHub, method string) (*Agent, *[]runCall) {
	a := pairServer(t)
	a.opts.Version = "0.1.0"
	a.upd.method = method
	var calls []runCall
	a.upd.runner = func(name string, args, env []string) error {
		calls = append(calls, runCall{name, args, env})
		return nil
	}
	night := time.Date(2026, 11, 3, 3, 30, 0, 0, time.Local)
	a.upd.now = func() time.Time { return night }
	a.upd.dnfAuto = func() string { return "download" }
	a.store.Update(func(c *config.Config) error {
		c.Update = &config.UpdateSettings{APIURL: g.srv.URL}
		return nil
	})
	return a, &calls
}

func TestUpdateCheckNotifyAndInstall(t *testing.T) {
	ctx := context.Background()
	g := newFakeGitHub(t, "0.2.0")
	a, calls := updAgent(t, g, methodStatic)
	// notify (default): shown, not installed
	a.updateTick(ctx, false)
	v := a.updateView()
	if v.Available != "0.2.0" || v.Policy != "notify" || v.Method != "static" || v.Repo != "kaovilai/tailbond" || !strings.HasSuffix(v.NotesURL, "/v0.2.0") || v.Error != "" || len(*calls) != 0 {
		t.Fatalf("notify: %+v calls %v", v, *calls)
	}
	// Update now: downloads, checks, unpacks, runs the new bundle's install.sh --update
	if w := req(t, a, "POST", "/api/update/install", "127.0.0.1:1", "", map[string]any{}); w.Code != http.StatusAccepted {
		t.Fatalf("install: %d %s", w.Code, w.Body)
	}
	a.updateTick(ctx, true)
	if len(*calls) != 1 {
		t.Fatalf("runner calls %v (%+v)", *calls, a.updateView())
	}
	c := (*calls)[0]
	bundle := filepath.Join(a.store.Get().StateDir, "update", "0.2.0", bundleName())
	if c.name != filepath.Join(bundle, "install.sh") || strings.Join(c.args, " ") != "--update --bundle-dir "+bundle {
		t.Fatalf("ran %+v", c)
	}
	if b, _ := os.ReadFile(filepath.Join(bundle, "VERSION")); strings.TrimSpace(string(b)) != "0.2.0" {
		t.Fatal("bundle not unpacked")
	}
	if _, err := os.Stat(filepath.Join(a.store.Get().StateDir, "escape")); err == nil {
		t.Fatal("tar path escaped the update directory")
	}
	if v := a.updateView(); v.State != "installing" {
		t.Fatalf("state %s", v.State)
	}
	// a tampered bundle (SHA256SUMS disagrees): failed, nothing run
	g.publish(t, "0.3.0")
	g.mu.Lock()
	g.sums[bundleName()+".tar.gz"] = strings.Repeat("0", 64)
	g.mu.Unlock()
	a.updateTick(ctx, true)
	if v := a.updateView(); v.State != "failed" || !strings.Contains(v.Error, "SHA-256") || len(*calls) != 1 {
		t.Fatalf("tampered: %+v calls %d", v, len(*calls))
	}
}

func TestUpdateAutoOffDevAndMethods(t *testing.T) {
	ctx := context.Background()
	g := newFakeGitHub(t, "0.2.0")
	a, calls := updAgent(t, g, methodStatic)
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "sometimes"}); w.Code != 400 {
		t.Fatalf("bad policy: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"repo": "https://gitlab.com/a/b"}); w.Code != 400 {
		t.Fatalf("bad repo: %d", w.Code)
	}
	// auto: installs by itself
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "auto"}); w.Code != 200 {
		t.Fatalf("policy: %d %s", w.Code, w.Body)
	}
	// ... at night only (devices drop for a few seconds)
	a.upd.now = func() time.Time { return time.Date(2026, 11, 3, 14, 0, 0, 0, time.Local) }
	a.updateTick(ctx, false)
	if len(*calls) != 0 || a.updateView().Available != "0.2.0" {
		t.Fatalf("auto installed in the afternoon: %v", *calls)
	}
	a.upd.now = func() time.Time { return time.Date(2026, 11, 3, 3, 30, 0, 0, time.Local) }
	a.updateTick(ctx, false)
	if len(*calls) != 1 || a.updateView().AutoWindow != "03:00-05:00" {
		t.Fatalf("auto didn't install at night: %+v", a.updateView())
	}
	// a dev build shows releases but never installs by itself
	a.opts.Version = "v0.1.0-12-gabcdef"
	*calls = nil
	a.updateTick(ctx, false)
	if len(*calls) != 0 || a.updateView().Available != "0.2.0" {
		t.Fatalf("dev build: %+v %v", a.updateView(), *calls)
	}
	a.opts.Version = "0.1.0"
	// off: no checks at all
	req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "off"})
	g.mu.Lock()
	before := g.calls
	g.mu.Unlock()
	a.updateTick(ctx, false)
	g.mu.Lock()
	after := g.calls
	g.mu.Unlock()
	if after != before || a.updateView().State != "off" {
		t.Fatalf("off checked anyway: %d -> %d, %+v", before, after, a.updateView())
	}
	// dnf / none: no one-click install, the reason says what to do
	a.upd.method = methodDnf
	req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "notify"})
	a.updateTick(ctx, false)
	if v := a.updateView(); v.DnfAutomatic != "download" {
		t.Fatalf("dnf_automatic: %+v", v)
	}
	for m, want := range map[string]string{methodDnf: "dnf upgrade", methodNone: "by hand"} {
		a.upd.method = m
		if w := req(t, a, "POST", "/api/update/install", "127.0.0.1:1", "", map[string]any{}); w.Code != 400 || !strings.Contains(w.Body.String(), want) {
			t.Fatalf("%s: %d %s", m, w.Code, w.Body)
		}
	}
	// up to date: nothing available
	a.upd.method = methodStatic
	a.opts.Version = "0.2.0"
	req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "notify"})
	a.updateTick(ctx, false)
	if v := a.updateView(); v.Available != "" {
		t.Fatalf("up to date: %+v", v)
	}
	if w := req(t, a, "POST", "/api/update/install", "127.0.0.1:1", "", map[string]any{}); w.Code != http.StatusConflict {
		t.Fatalf("install when up to date: %d", w.Code)
	}
	// GitHub unreachable: the error is shown
	g.srv.Close()
	a.updateTick(ctx, false)
	if v := a.updateView(); !strings.Contains(v.Error, "couldn't check") {
		t.Fatalf("unreachable: %+v", v)
	}
}

func TestUpdateIPKAndResult(t *testing.T) {
	ctx := context.Background()
	g := newFakeGitHub(t, "0.2.0")
	a, calls := updAgent(t, g, methodOpkg)
	a.upd.pkgDir = t.TempDir()
	a.upd.ipkCmd = "/usr/lib/tailbond/update-ipk"
	os.WriteFile(filepath.Join(a.upd.pkgDir, "tailbond_0.1.0-r1_aarch64_cortex-a53.ipk"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(a.upd.pkgDir, "current"), []byte("tailbond_0.1.0-r1_aarch64_cortex-a53.ipk\n"), 0o644)
	a.updateTick(ctx, true)
	if len(*calls) != 1 {
		t.Fatalf("no install: %+v", a.updateView())
	}
	c := (*calls)[0]
	want := []string{filepath.Join(a.upd.pkgDir, "tailbond_0.2.0-r1_aarch64_cortex-a53.ipk"), filepath.Join(a.upd.pkgDir, "tailbond_0.1.0-r1_aarch64_cortex-a53.ipk")}
	if c.name != a.upd.ipkCmd || strings.Join(c.args, " ") != strings.Join(want, " ") || !strings.Contains(strings.Join(c.env, " "), "TB_ROLE=server") {
		t.Fatalf("ran %+v", c)
	}
	if b, _ := os.ReadFile(want[0]); string(b) != "ipk a53" {
		t.Fatal("ipk content")
	}
	// the installers' result file
	res := `{"result":"rolled_back","version":"0.2.0","from":"0.1.0","at":"2026-11-03T03:00:00Z","detail":"the agent didn't answer within 90s"}`
	os.WriteFile(filepath.Join(a.store.Get().StateDir, resultFileName), []byte(res), 0o644)
	a.loadUpdateResult()
	if v := a.updateView(); v.State != "rolled_back" || v.LastResult != "rolled_back" || v.LastResultVersion != "0.2.0" || !strings.Contains(v.Detail, "90s") {
		t.Fatalf("result: %+v", v)
	}
	// automatic updates don't try a version that didn't start here again;
	// Update now does
	*calls = nil
	req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "auto"})
	a.updateTick(ctx, false)
	if len(*calls) != 0 {
		t.Fatalf("auto retried the rolled-back version: %v", *calls)
	}
	a.updateTick(ctx, true)
	if len(*calls) != 1 {
		t.Fatal("Update now didn't retry it")
	}
}

func TestOutdatedClients(t *testing.T) {
	a := pairServer(t)
	a.opts.Version = "0.2.0"
	res, err := a.addClient(context.Background(), "slate7")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := pairing.Decode(res.URI)
	r := httptest.NewRequest("GET", "/api/pairing", nil)
	r.RemoteAddr = "100.101.9.9:4000"
	r.Header.Set("Authorization", "Bearer "+APIToken(p.Key))
	r.Header.Set("X-Tailbond-Version", "0.1.0")
	r.Header.Set("X-Tailbond-Kind", "router")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("pairing: %d", w.Code)
	}
	v := a.updateView()
	if len(v.Outdated) != 1 || v.Outdated[0] != (status.Outdated{Name: "slate7", Kind: "router", Version: "0.1.0"}) {
		t.Fatalf("outdated %+v", v.Outdated)
	}
}

func TestPairingRefreshEndpoint(t *testing.T) {
	a := pairServer(t)
	res, err := a.addClient(context.Background(), "slate7")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := pairing.Decode(res.URI)
	a.store.Update(func(c *config.Config) error { c.Server.PublicAddr = "new-home.example.net"; return nil })
	w := req(t, a, "GET", "/api/pairing", "100.101.9.9:4000", APIToken(p.Key), nil)
	var got struct{ Name, URI string }
	decode(t, w, &got)
	np, err := pairing.Decode(got.URI)
	if w.Code != 200 || err != nil || got.Name != "slate7" || np.Key != p.Key || !strings.HasPrefix(np.Server, "new-home.example.net:") {
		t.Fatalf("%d %+v %+v", w.Code, got, np)
	}
	if w := req(t, a, "GET", "/api/pairing", "100.101.9.9:4000", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("without token: %d", w.Code)
	}
}

func TestBackupRestoreAPI(t *testing.T) {
	a := pairServer(t)
	if _, err := a.addClient(context.Background(), "slate7"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a.store.Get().StateDir, "server.key"), []byte("TLSKEY"), 0o600)
	// a client's pushed backup moves with the server
	os.MkdirAll(filepath.Join(a.store.Get().StateDir, "client-backups", "slate7"), 0o700)
	os.WriteFile(filepath.Join(a.store.Get().StateDir, "client-backups", "slate7", "slate7-1.tbbackup"), []byte("CLIENTBACKUP"), 0o600)
	const pw = "a long passphrase"
	if w := req(t, a, "POST", "/api/backup", "127.0.0.1:1", "", map[string]string{"passphrase": "short"}); w.Code != 400 {
		t.Fatalf("short passphrase: %d", w.Code)
	}
	w := req(t, a, "POST", "/api/backup", "127.0.0.1:1", "", map[string]string{"passphrase": pw})
	if w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte(backup.Magic)) || !strings.Contains(w.Header().Get("Content-Disposition"), ".tbbackup") {
		t.Fatalf("backup: %d %q", w.Code, w.Header())
	}
	file := base64.StdEncoding.EncodeToString(w.Body.Bytes())
	if bytes.Contains(w.Body.Bytes(), []byte("slate7")) {
		t.Fatal("backup not encrypted")
	}
	var sum map[string]any
	decode(t, req(t, a, "GET", "/api/backup/summary", "127.0.0.1:1", "", nil), &sum)
	if s, _ := json.Marshal(sum); strings.Contains(string(s), a.store.Get().Server.Users[0].Key) || !strings.Contains(string(s), "slate7") {
		t.Fatalf("summary %s", s)
	}

	// restore onto a fresh server (a new machine)
	b := pairServer(t)
	b.store.Update(func(c *config.Config) error { c.Server.PublicAddr = "new-vps.example.net"; return nil })
	if w := req(t, b, "POST", "/api/restore/preview", "127.0.0.1:1", "", map[string]any{"file": file, "passphrase": "wrong passphrase!"}); w.Code != 400 || !strings.Contains(w.Body.String(), "wrong passphrase") {
		t.Fatalf("wrong passphrase: %d %s", w.Code, w.Body)
	}
	var prev struct {
		Changes  []backup.Change
		Warnings []string
		Role     string
	}
	w = req(t, b, "POST", "/api/restore/preview", "127.0.0.1:1", "", map[string]any{"file": file, "passphrase": pw})
	decode(t, w, &prev)
	if w.Code != 200 || prev.Role != "server" || len(prev.Changes) == 0 {
		t.Fatalf("preview %d %+v", w.Code, prev)
	}
	if w := req(t, b, "POST", "/api/restore", "127.0.0.1:1", "", map[string]any{"file": file, "passphrase": pw}); w.Code != 400 {
		t.Fatalf("restore without confirm: %d", w.Code)
	}
	stateB := b.store.Get().StateDir
	w = req(t, b, "POST", "/api/restore", "127.0.0.1:1", "", map[string]any{"file": file, "passphrase": pw, "confirm": true})
	if w.Code != 200 {
		t.Fatalf("restore: %d %s", w.Code, w.Body)
	}
	got := b.store.Get()
	if len(got.Server.Users) != 1 || got.Server.Users[0].Key != a.store.Get().Server.Users[0].Key || got.StateDir != stateB {
		t.Fatalf("restored %+v", got.Server.Users)
	}
	if got.Server.PublicAddr != "new-vps.example.net" {
		t.Fatalf("restore replaced this machine's public address: %q", got.Server.PublicAddr)
	}
	if k, _ := os.ReadFile(filepath.Join(stateB, "server.key")); string(k) != "TLSKEY" {
		t.Fatal("TLS key not restored")
	}
	if k, _ := os.ReadFile(filepath.Join(stateB, "client-backups", "slate7", "slate7-1.tbbackup")); string(k) != "CLIENTBACKUP" {
		t.Fatal("the client's pushed backup didn't move with the server")
	}
	ents, _ := os.ReadDir(filepath.Join(stateB, "backups"))
	if len(ents) != 1 || !strings.HasPrefix(ents[0].Name(), "before-restore-") {
		t.Fatalf("rollback copy: %v", ents)
	}
	// a client backup can't be restored on a server
	c := clientAgent(t, nil)
	cw := req(t, c, "POST", "/api/backup", lan, "", map[string]string{"passphrase": pw})
	if w := req(t, b, "POST", "/api/restore/preview", "127.0.0.1:1", "", map[string]any{"file": base64.StdEncoding.EncodeToString(cw.Body.Bytes()), "passphrase": pw}); w.Code != 400 || !strings.Contains(w.Body.String(), "client backup") {
		t.Fatalf("role mismatch: %d %s", w.Code, w.Body)
	}
}

func TestBackupPush(t *testing.T) {
	a := pairServer(t)
	res, _ := a.addClient(context.Background(), "slate7")
	p, _ := pairing.Decode(res.URI)
	blob, _ := backup.Seal([]byte(`{"format":1}`), "a long passphrase", backup.Header{Role: "client"})
	push := func(body []byte, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/backups/push", bytes.NewReader(body))
		r.RemoteAddr = "100.101.9.9:4000"
		r.Header.Set("Content-Type", "application/octet-stream")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	if w := push(blob, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("without token: %d", w.Code)
	}
	if w := push([]byte("not a backup"), APIToken(p.Key)); w.Code != 400 {
		t.Fatalf("junk: %d", w.Code)
	}
	if w := push(blob, APIToken(p.Key)); w.Code != 200 {
		t.Fatalf("push: %d %s", w.Code, w.Body)
	}
	var list []backupEntry
	decode(t, req(t, a, "GET", "/api/backups", "127.0.0.1:1", "", nil), &list)
	if len(list) != 1 || list[0].Client != "slate7" || strings.Contains(list[0].Name, "/") {
		t.Fatalf("list %+v", list)
	}
}
