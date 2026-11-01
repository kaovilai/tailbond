package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
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
	"time"
)

func key(t *testing.T, seed byte) ed25519.PrivateKey {
	t.Helper()
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
}

func manifestBytes(t *testing.T, m Manifest) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func sampleManifest() Manifest {
	return Manifest{Format: 1, Channel: "stable", Version: "0.2.0", Protocol: 1, RequiresServerProtocol: 1,
		Artifacts: []Artifact{{Name: "tailbond-linux-x86_64.tar.gz", Kind: "linux", Arch: "x86_64", URL: "https://example/x", SHA256: strings.Repeat("a", 64), Size: 1}}}
}

func TestVerify(t *testing.T) {
	priv := key(t, 1)
	pub := priv.Public().(ed25519.PublicKey)
	raw := manifestBytes(t, sampleManifest())
	sig := Sign(raw, priv)
	m, err := Verify(raw, sig, []ed25519.PublicKey{key(t, 9).Public().(ed25519.PublicKey), pub})
	if err != nil || m.Version != "0.2.0" {
		t.Fatalf("good signature: %v %+v", err, m)
	}
	// tampered manifest
	bad := bytes.Replace(raw, []byte(`"0.2.0"`), []byte(`"9.9.9"`), 1)
	if _, err := Verify(bad, sig, []ed25519.PublicKey{pub}); err != ErrBadSignature {
		t.Fatalf("tampered: %v", err)
	}
	// wrong key
	if _, err := Verify(raw, sig, []ed25519.PublicKey{key(t, 2).Public().(ed25519.PublicKey)}); err != ErrUnknownKey {
		t.Fatalf("wrong key: %v", err)
	}
	// a key with the right id field but a different key: forge the id
	var s Signature
	json.Unmarshal(sig, &s)
	other := key(t, 3)
	s.KeyID = KeyID(other.Public().(ed25519.PublicKey))
	forged, _ := json.Marshal(s)
	if _, err := Verify(raw, forged, []ed25519.PublicKey{other.Public().(ed25519.PublicKey)}); err != ErrBadSignature {
		t.Fatalf("relabelled signature: %v", err)
	}
	if _, err := Verify(raw, sig, nil); err != ErrNoKeys {
		t.Fatalf("no keys: %v", err)
	}
	if _, err := Verify(raw, []byte("not json"), []ed25519.PublicKey{pub}); err == nil {
		t.Fatal("garbage signature accepted")
	}
	// artifact names can't escape the mirror directory
	m2 := sampleManifest()
	m2.Artifacts[0].Name = "../etc/passwd"
	raw2 := manifestBytes(t, m2)
	if _, err := Verify(raw2, Sign(raw2, priv), []ed25519.PublicKey{pub}); err == nil {
		t.Fatal("path in artifact name accepted")
	}
}

func TestParseKeys(t *testing.T) {
	pub := key(t, 1).Public().(ed25519.PublicKey)
	b64 := base64.StdEncoding.EncodeToString(pub)
	keys := ParseKeys(b64+", junk,"+b64, "")
	if len(keys) != 1 || !keys[0].Equal(pub) {
		t.Fatalf("%v", keys)
	}
	if KeyID(pub) != KeyID(keys[0]) || len(KeyID(pub)) != 16 {
		t.Fatal("key id")
	}
}

func TestVersionOrder(t *testing.T) {
	order := []string{"0.1.0-beta.1", "0.1.0-beta.2", "0.1.0-beta.11", "0.1.0-rc.1", "v0.1.0", "0.1.1", "0.2.0", "1.0.0"}
	for i := 1; i < len(order); i++ {
		if !Newer(order[i], order[i-1]) || Newer(order[i-1], order[i]) {
			t.Errorf("%s should be newer than %s", order[i], order[i-1])
		}
	}
	if !Same("0.1.0-r96", "v0.1.0") || !Same("0.1.0", "0.1.0+build.5") {
		t.Error("revision / build metadata should be ignored")
	}
	for _, bad := range []string{"dev", "abc1234", "0.1", "0.1.0-dirty", "v0.1.0-5-gabc123-dirty", "1.x.0", ""} {
		if _, ok := Parse(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
	if v, _ := Parse("v1.2.3-beta.4"); v.String() != "1.2.3-beta.4" {
		t.Error(v.String())
	}
}

func TestDecide(t *testing.T) {
	m := sampleManifest()
	at := func(h, min int) time.Time { return time.Date(2026, 11, 2, h, min, 0, 0, time.Local) }
	auto := Policy{Mode: PolicyAuto, Window: "03:00-05:00", DeferKbps: 300}
	cases := []struct {
		name   string
		p      Policy
		s      Situation
		action string
	}{
		{"server outside window", auto, Situation{Now: at(14, 0), Server: true, Current: "0.1.0", Manifest: &m}, ActionWait},
		{"server inside window", auto, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionInstall},
		{"window wraps midnight", Policy{Mode: PolicyAuto, Window: "23:00-01:00"}, Situation{Now: at(0, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionInstall},
		{"client any time", auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m, ServerProtocol: 1}, ActionInstall},
		{"client busy", auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m, TrafficKbps: 900}, ActionWait},
		{"up to date", auto, Situation{Now: at(3, 30), Server: true, Current: "0.2.0", Manifest: &m}, ActionNone},
		{"newer installed", auto, Situation{Now: at(3, 30), Server: true, Current: "0.3.0", Manifest: &m}, ActionNone},
		{"dev build", auto, Situation{Now: at(3, 30), Server: true, Current: "dev", Manifest: &m}, ActionNone},
		{"off", Policy{Mode: PolicyOff}, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionNone},
		{"notify", Policy{Mode: PolicyNotify}, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionWait},
		{"requested ignores window", auto, Situation{Now: at(14, 0), Server: true, Current: "0.1.0", Manifest: &m, Requested: true}, ActionInstall},
		{"failed before", auto, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m, Failed: []string{"0.2.0"}}, ActionNone},
		{"pinned to current", Policy{Mode: PolicyAuto, Pin: "0.1.0"}, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionNone},
		{"pinned to the offered one", Policy{Mode: PolicyAuto, Pin: "0.2.0"}, Situation{Now: at(3, 30), Server: true, Current: "0.1.0", Manifest: &m}, ActionInstall},
	}
	for _, c := range cases {
		if d := Decide(c.p, c.s); d.Action != c.action {
			t.Errorf("%s: %+v, want %s", c.name, d, c.action)
		}
	}
	// compatibility: a client waits for its server
	m2 := m
	m2.RequiresServerProtocol = 2
	if d := Decide(auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m2, ServerProtocol: 1}); d.Action != ActionBlocked {
		t.Errorf("old server: %+v", d)
	}
	if d := Decide(auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m2, ServerProtocol: 1, Requested: true}); d.Action != ActionBlocked {
		t.Errorf("requested on old server: %+v", d)
	}
	if d := Decide(auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m2, ServerProtocol: 2}); d.Action != ActionInstall {
		t.Errorf("new server: %+v", d)
	}
	// security releases install while busy (clients)
	m3 := m
	m3.Security = true
	if d := Decide(auto, Situation{Now: at(14, 0), Current: "0.1.0", Manifest: &m3, TrafficKbps: 900}); d.Action != ActionInstall {
		t.Errorf("security while busy: %+v", d)
	}
	if !ValidWindow("03:00-05:00") || ValidWindow("3-5") || ValidWindow("25:00-01:00") || !ValidWindow("") {
		t.Error("ValidWindow")
	}
}

func tarball(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	p := filepath.Join(t.TempDir(), "x.tar.gz")
	os.WriteFile(p, buf.Bytes(), 0o644)
	return p
}

func TestUnpackAndTrial(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	tb := tarball(t, map[string]string{"tailbond-linux-x86_64/bin/tailbondd": "#!/bin/sh\n", "tailbond-linux-x86_64/bin/mqvpn": "m", "tailbond-linux-x86_64/hooks/server-up": "h", "tailbond-linux-x86_64/install.sh": "i"})
	if err := s.Unpack(tb, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"tailbondd", "mqvpn", "hooks/server-up"} {
		if _, err := os.Stat(filepath.Join(s.VersionDir("0.2.0"), f)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(s.VersionDir("0.2.0"), "install.sh")); err == nil {
		t.Fatal("unpacked an unrelated file")
	}
	if err := s.Unpack(tarball(t, map[string]string{"x/mqvpn": "m"}), "0.3.0"); err == nil {
		t.Fatal("archive without tailbondd accepted")
	}

	t0 := time.Now()
	// base 0.1.0, nothing installed: run the base
	if exe := s.Boot("0.1.0", t0); exe != "" {
		t.Fatalf("exec %q", exe)
	}
	if err := s.Install("0.2.0", Trial{ClientsBefore: 2}); err != nil {
		t.Fatal(err)
	}
	// the base execs the new version during its trial
	if exe := s.Boot("0.1.0", t0); exe != s.Binary("0.2.0", "tailbondd") {
		t.Fatalf("exec %q", exe)
	}
	if st := s.Load(); st.Trial == nil || st.Trial.Boots != 1 || st.Prev != "" || st.Trial.ClientsBefore != 2 {
		t.Fatalf("state %+v", st)
	}
	// it keeps crashing: after 3 boots the base rolls back to itself
	s.Boot("0.1.0", t0)
	s.Boot("0.1.0", t0)
	if exe := s.Boot("0.1.0", t0); exe != "" {
		t.Fatalf("still exec'ing %q after 4 boots", exe)
	}
	st := s.Load()
	if st.Active != "" || st.Trial != nil || len(st.Failed) != 1 || st.Last == nil || st.Last.Result != "rolled_back" || st.Last.Version != "0.2.0" {
		t.Fatalf("after rollback %+v", st)
	}
	// a version that passes its trial stays
	s.Unpack(tb, "0.2.1")
	s.Install("0.2.1", Trial{})
	s.Boot("0.1.0", t0)
	if err := s.Commit("healthy"); err != nil {
		t.Fatal(err)
	}
	if st := s.Load(); st.Active != "0.2.1" || st.Trial != nil || st.Last.Result != "ok" {
		t.Fatalf("after commit %+v", st)
	}
	for i := 0; i < 5; i++ {
		if exe := s.Boot("0.1.0", t0); exe != s.Binary("0.2.1", "tailbondd") {
			t.Fatal("committed version not exec'd")
		}
	}
	// a trial that runs out of time (hung, not crashing)
	s.Unpack(tb, "0.2.2")
	s.Install("0.2.2", Trial{})
	if exe := s.Boot("0.1.0", time.Now().Add(TrialTimeout+time.Second)); exe != s.Binary("0.2.1", "tailbondd") {
		t.Fatalf("timed-out trial not rolled back to 0.2.1: %q", exe)
	}
	if st := s.Load(); st.Active != "0.2.1" || st.Prev != "" {
		t.Fatalf("%+v", st)
	}
	// a newer package base wins over an older update
	if exe := s.Boot("0.3.0", t0); exe != "" {
		t.Fatalf("older update exec'd over a newer base: %q", exe)
	}
	// pruning keeps only active + prev
	ents, _ := os.ReadDir(filepath.Join(s.Dir, "versions"))
	if len(ents) > 2 {
		t.Fatalf("versions kept: %d", len(ents))
	}
}

func TestFetchAndMirror(t *testing.T) {
	priv := key(t, 7)
	body := []byte("new release")
	h := sha256.Sum256(body)
	m := sampleManifest()
	m.Artifacts = []Artifact{{Name: "tailbond-linux-x86_64.tar.gz", Kind: "linux", Arch: "x86_64", SHA256: hex.EncodeToString(h[:]), Size: int64(len(body))}}
	raw := manifestBytes(t, m)
	sig := Sign(raw, priv)
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Tailbond-Version")
		switch r.URL.Path {
		case "/updates/stable.json":
			w.Write(raw)
		case "/updates/stable.json.sig":
			w.Write(sig)
		case "/updates/files/tailbond-linux-x86_64.tar.gz":
			w.Write(body)
		case "/public/stable.json", "/public/stable.json.sig":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	f := &Fetcher{}
	keys := []ed25519.PublicKey{priv.Public().(ed25519.PublicKey)}
	srcs := []Source{{Base: srv.URL + "/nothing", Mirror: true}, {Base: srv.URL + "/updates", Mirror: true, Header: http.Header{"X-Tailbond-Version": {"0.1.0"}}}, {Base: srv.URL + "/public"}}
	got, err := f.Manifest(context.Background(), srcs, "stable", keys)
	if err != nil || got.Manifest.Version != "0.2.0" || got.From.Base != srv.URL+"/updates" || gotHeader != "0.1.0" {
		t.Fatalf("manifest: %v %+v header %q", err, got, gotHeader)
	}
	dst := filepath.Join(t.TempDir(), "dl", "x.tar.gz")
	if _, err := f.Download(context.Background(), srcs, m.Artifacts[0], dst); err != nil {
		t.Fatal(err)
	}
	// a corrupted artifact is refused
	bad := m.Artifacts[0]
	bad.SHA256 = strings.Repeat("0", 64)
	if _, err := f.Download(context.Background(), srcs, bad, dst+"2"); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	// no source answering a valid signature: error, no manifest
	if _, err := f.Manifest(context.Background(), srcs[:1], "stable", keys); err == nil {
		t.Fatal("nothing should verify")
	}

	mir := Mirror{Dir: t.TempDir(), MaxBytes: 1 << 20}
	if err := mir.Store("stable", raw, sig); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(mir.FilePath("tailbond-linux-x86_64.tar.gz"), body, 0o644)
	os.WriteFile(mir.FilePath("stale.ipk"), body, 0o644)
	mir.Prune([]*Manifest{&m}, nil)
	if _, err := os.Stat(mir.FilePath("stale.ipk")); err == nil {
		t.Fatal("stale artifact kept")
	}
	if _, err := os.Stat(mir.FilePath("tailbond-linux-x86_64.tar.gz")); err != nil {
		t.Fatal("current artifact pruned")
	}
	if mir.FilePath("../x") != "" {
		t.Fatal("path escape")
	}
	m2, s2, ok := mir.Manifest("stable")
	if !ok || !bytes.Equal(m2, raw) || !bytes.Equal(s2, sig) {
		t.Fatal("mirror manifest not byte-identical")
	}
}
