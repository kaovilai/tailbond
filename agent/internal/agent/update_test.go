package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailbond/tailbond/agent/internal/backup"
	"github.com/tailbond/tailbond/agent/internal/config"
	"github.com/tailbond/tailbond/agent/internal/pairing"
	"github.com/tailbond/tailbond/agent/internal/update"
)

type fakeRelease struct {
	raw   []byte
	files map[string][]byte
	srv   *httptest.Server
}

func newFakeRelease(t *testing.T, version string) *fakeRelease {
	t.Helper()
	fr := &fakeRelease{files: map[string][]byte{}}
	fr.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/channels/stable.json":
			w.Write(fr.raw)
		case strings.HasPrefix(r.URL.Path, "/dl/"):
			if b, ok := fr.files[strings.TrimPrefix(r.URL.Path, "/dl/")]; ok {
				w.Write(b)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fr.srv.Close)
	fr.publish(t, version)
	return fr
}

func (fr *fakeRelease) publish(t *testing.T, version string) {
	m := update.Manifest{Format: 1, Channel: "stable", Version: version, Protocol: 1, RequiresServerProtocol: 1}
	for _, name := range []string{"tailbond-linux-x86_64.tar.gz", "tailbond_" + version + "-r1_aarch64_cortex-a53.ipk"} {
		body := []byte("artifact " + name)
		fr.files[name] = body
		h := sha256.Sum256(body)
		kind, arch := "linux", "x86_64"
		if strings.HasSuffix(name, ".ipk") {
			kind, arch = "ipk", "aarch64_cortex-a53"
		}
		m.Artifacts = append(m.Artifacts, update.Artifact{Name: name, Kind: kind, Arch: arch, URL: fr.srv.URL + "/dl/" + name, SHA256: hex.EncodeToString(h[:]), Size: int64(len(body))})
	}
	fr.raw, _ = json.Marshal(m)
}

// noAttestations is a GitHub API that has no attestation for anything.
func noAttestations(t *testing.T) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestServerMirrorAndClientVersions(t *testing.T) {
	fr := newFakeRelease(t, "0.2.0")
	a := pairServer(t)
	a.opts.Version = "0.1.0"
	a.store.Update(func(c *config.Config) error {
		c.Update = &config.UpdateSettings{Source: fr.srv.URL + "/channels", Repo: "tailbond/tailbond", Policy: "notify", Verify: "attestation"}
		return nil
	})
	a.upd.githubAPI = noAttestations(t)
	if _, err := a.addClient(context.Background(), "slate7"); err != nil {
		t.Fatal(err)
	}
	// attestation: the fake release has none (or this build can't check
	// them, -tags noattest): nothing is offered or mirrored
	a.checkManifest(context.Background())
	if v := a.updateView(); v.Available != "" || !strings.Contains(v.Error, "attestation") || v.Verify != "attestation" || v.Repo != "tailbond/tailbond" || v.Mirror.Version != "" {
		t.Fatalf("unattested manifest: %+v %+v", v, v.Mirror)
	}
	// sha256-only (setups without GitHub): offered and mirrored
	a.store.Update(func(c *config.Config) error { c.Update.Verify = "sha256-only"; return nil })
	a.checkManifest(context.Background())
	v := a.updateView()
	if v.Available != "0.2.0" || v.Source != "public" || v.Error != "" || v.Verified != "sha256" || v.Mirror == nil || v.Mirror.Version != "0.2.0" || v.Mirror.Files != 2 {
		t.Fatalf("view %+v mirror %+v", v, v.Mirror)
	}
	// served byte-identical, to tailnet sources and token holders only
	w := req(t, a, "GET", "/updates/stable.json", "100.101.9.9:4000", "", nil)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), fr.raw) {
		t.Fatalf("mirror manifest: %d", w.Code)
	}
	// with a dashboard password, a non-tailnet source needs a token
	a.store.Update(func(c *config.Config) error { c.UIPassword = "pw"; return nil })
	if w := req(t, a, "GET", "/updates/stable.json", "192.168.1.50:4000", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("LAN source without token: %d", w.Code)
	}
	key := ""
	for _, u := range a.store.Get().Server.Users {
		key = u.Key
	}
	r := httptest.NewRequest("GET", "/updates/files/tailbond-linux-x86_64.tar.gz", nil)
	r.RemoteAddr = "192.168.1.50:4000"
	r.Header.Set("Authorization", "Bearer "+APIToken(key))
	r.Header.Set("X-Tailbond-Version", "0.1.0")
	r.Header.Set("X-Tailbond-Kind", "router")
	rw := httptest.NewRecorder()
	a.Handler().ServeHTTP(rw, r)
	if rw.Code != 200 || rw.Body.String() != "artifact tailbond-linux-x86_64.tar.gz" {
		t.Fatalf("artifact with token: %d %q", rw.Code, rw.Body)
	}
	if w := req(t, a, "GET", "/updates/files/..%2Fstate.json", "100.101.9.9:4000", "", nil); w.Code != 404 {
		t.Fatalf("path escape: %d", w.Code)
	}
	// a client is reported out of date (this server runs 0.2.0 now)
	a.opts.Version = "0.2.0"
	v = a.updateView()
	if len(v.Outdated) != 1 || v.Outdated[0].Name != "slate7" || v.Outdated[0].Version != "0.1.0" || v.Outdated[0].Kind != "router" {
		t.Fatalf("outdated %+v", v.Outdated)
	}
	// a newer release: the old version's files are kept (keep-last-2)
	fr.publish(t, "0.2.1")
	a.checkManifest(context.Background())
	if _, err := os.Stat(a.mirror().FilePath("tailbond_0.2.0-r1_aarch64_cortex-a53.ipk")); err != nil {
		t.Fatal("previous version's files pruned")
	}
	// a broken manifest is refused and the mirror keeps the good one
	fr.raw = bytes.Replace(fr.raw, []byte(`"format":1`), []byte(`"format":7`), 1)
	a.checkManifest(context.Background())
	if v := a.updateView(); v.Error == "" || v.Mirror.Version != "0.2.1" {
		t.Fatalf("broken: %+v %+v", v, v.Mirror)
	}
	// switching back to attestations: the cached sha256-only manifest isn't
	// offered (or installable) any more
	a.store.Update(func(c *config.Config) error { c.Update.Verify = "attestation"; return nil })
	a.checkManifest(context.Background())
	if v := a.updateView(); v.Available != "" || v.Verified != "" || a.upd.manifest != nil {
		t.Fatalf("sha256-only cache offered under attestation: %+v", v)
	}
}

func TestUpdateUploadAndPolicy(t *testing.T) {
	fr := newFakeRelease(t, "0.3.0")
	a := pairServer(t)
	a.opts.Version = "0.1.0"
	a.store.Update(func(c *config.Config) error {
		c.Update = &config.UpdateSettings{Repo: "tailbond/tailbond", Policy: "notify", Verify: "attestation"}
		return nil
	})
	a.upd.githubAPI = noAttestations(t)
	up := func(name string, body []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/update/upload?name="+name, bytes.NewReader(body))
		r.RemoteAddr = "127.0.0.1:1"
		r.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		return w
	}
	if update.AttestSupported {
		// attestation mode, offline: the manifest waits for its .sigstore.json
		if w := up("stable.json", fr.raw); w.Code != 200 || !strings.Contains(w.Body.String(), `"waiting_for":"stable.json.sigstore.json"`) {
			t.Fatalf("manifest: %d %s", w.Code, w.Body)
		}
		if w := up("stable.json.sigstore.json", []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)); w.Code != 400 {
			t.Fatalf("bad attestation accepted: %d %s", w.Code, w.Body)
		}
	} else if w := up("stable.json", fr.raw); w.Code != 400 || !strings.Contains(w.Body.String(), "can't check GitHub attestations") {
		// -tags noattest with verify attestation: refused, says why
		t.Fatalf("noattest: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(a.mirror().Dir, "upload", "stable.json")); err == nil {
		t.Fatal("refused upload kept")
	}
	// sha256-only: the manifest is taken as uploaded
	a.store.Update(func(c *config.Config) error { c.Update.Verify = "sha256-only"; return nil })
	if w := up("stable.json", fr.raw); w.Code != 200 || !strings.Contains(w.Body.String(), `"manifest":"0.3.0"`) || !strings.Contains(w.Body.String(), `"verified":"sha256"`) {
		t.Fatalf("sha256-only upload: %d %s", w.Code, w.Body)
	}
	if w := up("tailbond-linux-x86_64.tar.gz", []byte("corrupted")); w.Code != 400 {
		t.Fatalf("bad artifact: %d", w.Code)
	}
	if w := up("tailbond-linux-x86_64.tar.gz", fr.files["tailbond-linux-x86_64.tar.gz"]); w.Code != 200 || !strings.Contains(w.Body.String(), `"checksum_ok":true`) {
		t.Fatalf("artifact: %d %s", w.Code, w.Body)
	}
	a.checkManifest(context.Background())
	if v := a.updateView(); v.Available != "0.3.0" || v.Source != "upload" {
		t.Fatalf("after upload %+v", v)
	}
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "off", "window": "25:00-01:00"}); w.Code != 400 {
		t.Fatalf("bad window: %d", w.Code)
	}
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "auto", "channel": "beta", "window": "02:00-04:00", "pin": "0.3.0"}); w.Code != 200 {
		t.Fatalf("policy: %d %s", w.Code, w.Body)
	}
	if u := a.store.Get().UpdateOrDefault(); u.Policy != "auto" || u.Channel != "beta" || u.Window != "02:00-04:00" || u.Pin != "0.3.0" {
		t.Fatalf("%+v", u)
	}
	if w := req(t, a, "POST", "/api/update/policy", "127.0.0.1:1", "", map[string]any{"policy": "sometimes"}); w.Code != 400 {
		t.Fatalf("bad policy: %d", w.Code)
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
	if k, _ := os.ReadFile(filepath.Join(stateB, "server.key")); string(k) != "TLSKEY" {
		t.Fatal("TLS key not restored")
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
