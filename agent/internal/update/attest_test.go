//go:build !noattest

package update

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/root"
)

// testdata: real GitHub artifact attestations of cli/cli's release
// gh_2.103.3_linux_arm64.tar.gz (fetched from the GitHub API, 2026-11-02),
// and a snapshot of Sigstore's public-good trusted root. They verify offline.
//   - gh-provenance.sigstore.json: actions/attest-build-provenance style
//     (SLSA provenance v1, Sigstore public-good: Fulcio + Rekor), workflow
//     .github/workflows/deployment.yml@refs/heads/trunk;
//   - .sn: the same, snappy-compressed, as the API's bundle_url serves it;
//   - gh-release-attestation.sigstore.json: GitHub's own release
//     attestation (another predicate and identity: must not count).
const ghDigest = "ee32df2323c8847f5fac6cb6b4b69ca75828e9329a8f22f677021f7b3306ce63"

func ghOpts() AttestOptions {
	return AttestOptions{Repo: "cli/cli", Workflow: ".github/workflows/deployment.yml", RefRegex: "refs/heads/trunk$", TrustedRoot: "testdata/sigstore-trusted-root.json"}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// publicGood makes the attestor treat the snapshot as Sigstore's
// public-good root (SCTs and the transparency log required), as in
// production, without TUF.
func publicGood(t *testing.T, a *Attestor) {
	tr, err := root.NewTrustedRootFromPath("testdata/sigstore-trusted-root.json")
	if err != nil {
		t.Fatal(err)
	}
	a.material = func() (root.TrustedMaterial, bool, error) { return tr, true, nil }
}

func TestAttestationRealBundle(t *testing.T) {
	ctx := context.Background()
	digest, _ := hex.DecodeString(ghDigest)
	prov := fixture(t, "gh-provenance.sigstore.json")
	rel := fixture(t, "gh-release-attestation.sigstore.json")

	a, err := NewAttestor(ghOpts())
	if err != nil {
		t.Fatal(err)
	}
	publicGood(t, a)
	got, err := a.Verify(ctx, digest, prov)
	if err != nil || !bytes.Equal(got, prov) {
		t.Fatalf("real attestation: %v", err)
	}
	// several bundles (JSON lines): the one that matches is used
	if got, err := a.Verify(ctx, digest, append(append(append([]byte{}, rel...), '\n'), prov...)); err != nil || !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(prov)) {
		t.Fatalf("bundle list: %v", err)
	}
	// a trusted_root.json from the config (custom instance) works too
	b, _ := NewAttestor(ghOpts())
	if _, err := b.Verify(ctx, digest, prov); err != nil {
		t.Fatalf("trusted_root file: %v", err)
	}

	other := append([]byte{}, digest...)
	other[0] ^= 1
	fails := []struct {
		name   string
		o      AttestOptions
		digest []byte
		bundle []byte
	}{
		{"another artifact", ghOpts(), other, prov},
		{"another repository", func() AttestOptions { o := ghOpts(); o.Repo = "cli/cli2"; return o }(), digest, prov},
		{"another workflow", func() AttestOptions { o := ghOpts(); o.Workflow = ".github/workflows/ci.yml"; return o }(), digest, prov},
		{"built from a branch, releases come from tags", func() AttestOptions { o := ghOpts(); o.RefRegex = ""; return o }(), digest, prov},
		{"GitHub's release attestation isn't build provenance", ghOpts(), digest, rel},
		{"tampered statement (another digest for the same name)", ghOpts(), other, tamper(t, prov, ghDigest, hex.EncodeToString(other))},
		{"not a bundle", ghOpts(), digest, []byte(`{"x":1}`)},
	}
	for _, f := range fails {
		a, err := NewAttestor(f.o)
		if err != nil {
			t.Fatal(err)
		}
		publicGood(t, a)
		_, err = a.Verify(ctx, f.digest, f.bundle)
		t.Logf("%s: %v", f.name, err)
		if err == nil {
			t.Errorf("%s: verified", f.name)
		}
	}
	if _, err := NewAttestor(AttestOptions{}); err != ErrNoRepo {
		t.Fatalf("no repo: %v", err)
	}
	if _, err := NewAttestor(AttestOptions{Repo: "a/b", Workflow: "../x.yml"}); err == nil {
		t.Fatal("bad workflow accepted")
	}
}

// The manifest comes without its .sigstore.json: the GitHub API has it,
// inline or behind a snappy-compressed bundle_url.
func TestAttestationFromAPI(t *testing.T) {
	prov := fixture(t, "gh-provenance.sigstore.json")
	sn := fixture(t, "gh-provenance.sigstore.json.sn")
	var inline bool
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/cli/cli/attestations/sha256:"+ghDigest:
			if r.URL.Query().Get("predicate_type") != ProvenancePredicate {
				t.Errorf("predicate filter: %s", r.URL.RawQuery)
			}
			if inline {
				w.Write([]byte(`{"attestations":[{"bundle":` + string(prov) + `}]}`))
				return
			}
			w.Write([]byte(`{"attestations":[{"bundle":null,"bundle_url":"` + strings.Replace(srv.URL, "http://", "https://", 1) + `/blob.json.sn"}]}`))
		case r.URL.Path == "/blob.json.sn":
			w.Write(sn)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	// bundle_url must be https: route it to the test server
	tr := &http.Transport{}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		return tr.RoundTrip(r)
	})}
	o := ghOpts()
	o.API, o.Client = srv.URL, client
	digest, _ := hex.DecodeString(ghDigest)
	for _, in := range []bool{false, true} {
		inline = in
		a, _ := NewAttestor(o)
		publicGood(t, a)
		got, err := a.Verify(context.Background(), digest, nil)
		if err != nil || !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(prov)) {
			t.Fatalf("inline=%v: %v", in, err)
		}
	}
	// nothing attested: ErrNoAttestation
	a, _ := NewAttestor(o)
	publicGood(t, a)
	other := append([]byte{}, digest...)
	other[0] ^= 1
	if _, err := a.Verify(context.Background(), other, nil); !errors.Is(err, ErrNoAttestation) {
		t.Fatalf("unknown digest: %v", err)
	}
}

// tamper rewrites the DSSE payload (the in-toto statement), keeping the
// signature: it must no longer verify.
func tamper(t *testing.T, bundle []byte, from, to string) []byte {
	t.Helper()
	var b map[string]any
	if err := json.Unmarshal(bundle, &b); err != nil {
		t.Fatal(err)
	}
	env := b["dsseEnvelope"].(map[string]any)
	payload, err := base64.StdEncoding.DecodeString(env["payload"].(string))
	if err != nil || !bytes.Contains(payload, []byte(from)) {
		t.Fatalf("payload: %v", err)
	}
	env["payload"] = base64.StdEncoding.EncodeToString(bytes.Replace(payload, []byte(from), []byte(to), 1))
	out, _ := json.Marshal(b)
	return out
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
