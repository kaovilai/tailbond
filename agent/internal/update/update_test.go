package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
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

func TestParseManifestAndTrust(t *testing.T) {
	raw := manifestBytes(t, sampleManifest())
	m, err := ParseManifest(raw)
	if err != nil || m.Version != "0.2.0" {
		t.Fatalf("%v %+v", err, m)
	}
	// artifact names can't escape the mirror directory; checksums are hex
	for _, mut := range []func(*Manifest){
		func(m *Manifest) { m.Artifacts[0].Name = "../etc/passwd" },
		func(m *Manifest) { m.Artifacts[0].Name = ".hidden" },
		func(m *Manifest) { m.Artifacts[0].SHA256 = strings.Repeat("z", 64) },
		func(m *Manifest) { m.Format = 2 },
		func(m *Manifest) { m.Version = "dev" },
	} {
		bad := sampleManifest()
		mut(&bad)
		if _, err := ParseManifest(manifestBytes(t, bad)); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	ctx := context.Background()
	// sha256-only: the manifest as is, its bundle kept if there was one
	if m, how, b, err := (Trust{Mode: VerifySHA256}).Check(ctx, raw, []byte("x")); err != nil || how != VerifiedSHA256 || m.Version != "0.2.0" || string(b) != "x" {
		t.Fatalf("sha256-only: %v %s", err, how)
	}
	// attestation without a verifier (router build / no repo): refused
	if _, _, _, err := (Trust{Mode: VerifyAttestation, AttestErr: ErrNoRepo}).Check(ctx, raw, nil); err != ErrNoRepo {
		t.Fatalf("no repo: %v", err)
	}
	if _, _, _, err := (Trust{Mode: VerifyAttestation}).Check(ctx, raw, nil); err != ErrNoAttestSupport {
		t.Fatalf("no verifier: %v", err)
	}
}

func TestVersionOrder(t *testing.T) {
	order := []string{"0.1.0-beta.1", "0.1.0-beta.2", "0.1.0-beta.11", "0.1.0-rc.1", "v0.1.0", "0.1.1", "0.2.0", "1.0.0"}
	for i := 1; i < len(order); i++ {
		if !Newer(order[i], order[i-1]) || Newer(order[i-1], order[i]) {
			t.Errorf("%s should be newer than %s", order[i], order[i-1])
		}
	}
	if !Same("1.2.3-beta.1-r1", "1.2.3-beta.1") || !Same("0.2.0-r1", "0.2.0") {
		t.Error("ipk revision after a pre-release")
	}
	if !Same("0.1.0-r96", "v0.1.0") || !Same("0.1.0", "0.1.0+build.5") {
		t.Error("revision / build metadata should be ignored")
	}
	for _, bad := range []string{"dev", "abc1234", "0.1", "0.1.0-dirty", "v0.1.0-5-gabc123-dirty", "0.2.0-3-gabc1234", "1.x.0", ""} {
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
	body := []byte("new release")
	h := sha256.Sum256(body)
	m := sampleManifest()
	m.Artifacts = []Artifact{{Name: "tailbond-linux-x86_64.tar.gz", Kind: "linux", Arch: "x86_64", SHA256: hex.EncodeToString(h[:]), Size: int64(len(body))}}
	raw := manifestBytes(t, m)
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Tailbond-Version")
		switch r.URL.Path {
		case "/updates/stable.json":
			w.Write(raw)
		case "/updates/files/tailbond-linux-x86_64.tar.gz":
			w.Write(body)
		case "/public/stable.json":
			w.Write(raw) // no attestation next to it
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	f := &Fetcher{}
	strict := Trust{Mode: VerifyAttestation, AttestErr: ErrNoRepo}
	srcs := []Source{{Base: srv.URL + "/nothing", Mirror: true}, {Base: srv.URL + "/updates", Mirror: true, Header: http.Header{"X-Tailbond-Version": {"0.1.0"}}}, {Base: srv.URL + "/public"}}
	// the home server's mirror is trusted as is (authenticated channel)
	got, err := f.Manifest(context.Background(), srcs, "stable", strict)
	if err != nil || got.Manifest.Version != "0.2.0" || got.From.Base != srv.URL+"/updates" || gotHeader != "0.1.0" || got.Verified != VerifiedMirror {
		t.Fatalf("manifest: %v %+v header %q", err, got, gotHeader)
	}
	// a public source needs an attestation, unless sha256-only
	pub := srcs[2:]
	if _, err := f.Manifest(context.Background(), pub, "stable", strict); err == nil {
		t.Fatal("public manifest without an attestation accepted")
	}
	if got, err := f.Manifest(context.Background(), pub, "stable", Trust{Mode: VerifySHA256}); err != nil || got.Verified != VerifiedSHA256 {
		t.Fatalf("sha256-only: %v %+v", err, got)
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
	// no source answering: error, no manifest
	if _, err := f.Manifest(context.Background(), srcs[:1], "stable", strict); err == nil {
		t.Fatal("nothing should verify")
	}

	mir := Mirror{Dir: t.TempDir(), MaxBytes: 1 << 20}
	bundle := []byte(`{"mediaType":"x"}`)
	if err := mir.Store("stable", raw, bundle); err != nil {
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
	m2, b2, ok := mir.Manifest("stable")
	if !ok || !bytes.Equal(m2, raw) || !bytes.Equal(b2, bundle) {
		t.Fatal("mirror manifest not byte-identical")
	}
	// a newer manifest without an attestation (sha256-only): the bundle
	// moves with the old one to .prev
	m.Version = "0.2.1"
	raw3 := manifestBytes(t, m)
	if err := mir.Store("stable", raw3, nil); err != nil {
		t.Fatal(err)
	}
	if _, b3, _ := mir.Manifest("stable"); b3 != nil {
		t.Fatal("stale bundle kept with the new manifest")
	}
	if b, _ := os.ReadFile(filepath.Join(mir.Dir, "stable.prev.json"+BundleSuffix)); !bytes.Equal(b, bundle) {
		t.Fatal("previous bundle not kept")
	}

}
