//go:build !noattest

package update

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// AttestSupported: this build verifies GitHub attestations.
const AttestSupported = true

// Attestor verifies GitHub artifact attestations for one repository.
type Attestor struct {
	o  AttestOptions
	id verify.CertificateIdentity

	mu     sync.Mutex
	tm     root.TrustedMaterial
	tmAt   time.Time
	public bool // the trusted material is Sigstore's public-good instance
	// test hook: trusted material and verifier options without TUF
	material func() (root.TrustedMaterial, bool, error)
}

// NewAttestor checks the options; the trusted root is loaded on first use.
func NewAttestor(o AttestOptions) (*Attestor, error) {
	if o.Repo == "" {
		return nil, ErrNoRepo
	}
	if !ValidRepo(o.Repo) {
		return nil, fmt.Errorf("update.repo %q isn't owner/name", o.Repo)
	}
	if o.Workflow == "" {
		o.Workflow = DefaultWorkflow
	}
	if !ValidWorkflow(o.Workflow) {
		return nil, fmt.Errorf("update.workflow %q isn't .github/workflows/<file>.yml", o.Workflow)
	}
	if o.RefRegex == "" {
		o.RefRegex = "refs/tags/v"
	}
	if o.API == "" {
		o.API = "https://api.github.com"
	}
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 2 * time.Minute}
	}
	repoURL := "https://github.com/" + o.Repo
	san, err := verify.NewSANMatcher("", "^"+regexp.QuoteMeta(repoURL+"/"+o.Workflow+"@")+o.RefRegex)
	if err != nil {
		return nil, err
	}
	iss, err := verify.NewIssuerMatcher(GitHubIssuer, "")
	if err != nil {
		return nil, err
	}
	id, err := verify.NewCertificateIdentity(san, iss, certificate.Extensions{SourceRepositoryURI: repoURL})
	if err != nil {
		return nil, err
	}
	return &Attestor{o: o, id: id}, nil
}

// Repo is the repository the attestations must come from.
func (a *Attestor) Repo() string { return a.o.Repo }

func (a *Attestor) trusted() (root.TrustedMaterial, bool, error) {
	if a.material != nil {
		return a.material()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tm != nil && time.Since(a.tmAt) < 24*time.Hour {
		return a.tm, a.public, nil
	}
	if a.o.TrustedRoot != "" {
		tr, err := root.NewTrustedRootFromPath(a.o.TrustedRoot)
		if err != nil {
			return nil, false, fmt.Errorf("update.trusted_root: %w", err)
		}
		a.tm, a.tmAt, a.public = tr, time.Now(), false
		return tr, false, nil
	}
	opts := tuf.DefaultOptions()
	if a.o.CacheDir != "" {
		opts.CachePath = a.o.CacheDir
	}
	tr, err := root.FetchTrustedRootWithOptions(opts)
	if err != nil {
		// offline: the cached TUF metadata, while it's still valid
		opts.ForceCache = true
		var err2 error
		if tr, err2 = root.FetchTrustedRootWithOptions(opts); err2 != nil {
			if a.tm != nil { // keep what we had
				return a.tm, a.public, nil
			}
			return nil, false, fmt.Errorf("Sigstore trusted root: %w", err)
		}
	}
	a.tm, a.tmAt, a.public = tr, time.Now(), true
	return tr, true, nil
}

// Verify checks that one of the bundles (the .sigstore.json file; nil =
// ask the GitHub API) attests sha256 digest for this repository's release
// workflow. It returns the bundle that verified, to keep next to the
// manifest (offline checks, the mirror).
func (a *Attestor) Verify(ctx context.Context, digest []byte, bundles []byte) ([]byte, error) {
	cands := splitBundles(bundles)
	var apiErr error
	if len(cands) == 0 {
		cands, apiErr = a.fromAPI(ctx, digest)
		if len(cands) == 0 {
			if apiErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrNoAttestation, apiErr)
			}
			return nil, ErrNoAttestation
		}
	}
	tm, public, err := a.trusted()
	if err != nil {
		return nil, err
	}
	var errs []string
	for _, c := range cands {
		if err := a.verifyOne(tm, public, digest, c); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		return c, nil
	}
	return nil, fmt.Errorf("GitHub attestation doesn't verify for %s: %s", a.o.Repo, strings.Join(dedupe(errs), "; "))
}

func (a *Attestor) verifyOne(tm root.TrustedMaterial, public bool, digest, raw []byte) error {
	var b bundle.Bundle
	if err := b.UnmarshalJSON(raw); err != nil {
		return fmt.Errorf("bundle: %w", err)
	}
	opts := []verify.VerifierOption{verify.WithObserverTimestamps(1)}
	if public {
		// Sigstore public-good: certificates carry SCTs and the signature
		// is in the Rekor transparency log
		opts = append(opts, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1))
	} else if tl, _ := b.TlogEntries(); len(tl) > 0 {
		opts = append(opts, verify.WithTransparencyLog(1))
	}
	v, err := verify.NewVerifier(tm, opts...)
	if err != nil {
		return err
	}
	res, err := v.Verify(&b, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(a.id)))
	if err != nil {
		return err
	}
	if res.Statement == nil || res.Statement.GetPredicateType() != ProvenancePredicate {
		return errors.New("not a build provenance attestation")
	}
	return nil
}

// fromAPI asks GET /repos/{repo}/attestations/sha256:{digest}. Bundles come
// inline or (lately) as a snappy-compressed blob behind bundle_url.
func (a *Attestor) fromAPI(ctx context.Context, digest []byte) ([][]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/attestations/sha256:%s?per_page=30&predicate_type=%s",
		strings.TrimRight(a.o.API, "/"), a.o.Repo, hex.EncodeToString(digest), ProvenancePredicate)
	body, err := a.get(ctx, url, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Attestations []struct {
			Bundle    json.RawMessage `json:"bundle"`
			BundleURL string          `json:"bundle_url"`
		} `json:"attestations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("GitHub API: %w", err)
	}
	var out [][]byte
	var errs []string
	for _, at := range resp.Attestations {
		switch {
		case len(at.Bundle) > 0 && string(at.Bundle) != "null":
			out = append(out, at.Bundle)
		case strings.HasPrefix(at.BundleURL, "https://"):
			b, err := a.get(ctx, at.BundleURL, false)
			if err == nil && !json.Valid(b) {
				b, err = snappy.Decode(nil, b)
			}
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			out = append(out, b)
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "; "))
	}
	return out, nil
}

func (a *Attestor) get(ctx context.Context, url string, api bool) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if api {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if t := os.Getenv("TB_GITHUB_TOKEN"); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
	}
	resp, err := a.o.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && api {
		return []byte(`{"attestations":[]}`), nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if err == nil && len(b) > 8<<20 {
		err = errors.New("attestation too large")
	}
	return b, err
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
