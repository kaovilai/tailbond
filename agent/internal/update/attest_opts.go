package update

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

// GitHub artifact attestations (docs/architecture.md, "Updates"): the
// release job attests the channel manifests and every artifact with
// actions/attest-build-provenance (Sigstore keyless: a short-lived
// certificate for the workflow's identity, no private key anywhere). A
// manifest is authentic when an attestation for its SHA-256 verifies against
// Sigstore's trusted root and names this repository's release workflow.
// Artifacts are then checked against the manifest's SHA-256s.
//
// Builds with -tags noattest (the router) leave the verifier out (it adds
// ~8 MB to the binary); they take updates from their server's mirror, which
// verified them, over the authenticated tunnel / tailnet.

const (
	// GitHubIssuer is the OIDC issuer of GitHub Actions certificates.
	GitHubIssuer = "https://token.actions.githubusercontent.com"
	// ProvenancePredicate is what attest-build-provenance attests.
	ProvenancePredicate = "https://slsa.dev/provenance/v1"
	// DefaultWorkflow is the workflow that builds releases.
	DefaultWorkflow = ".github/workflows/ci.yml"
	// BundleSuffix is the attestation bundle next to a manifest:
	// <channel>.json.sigstore.json.
	BundleSuffix = ".sigstore.json"
)

var (
	ErrNoAttestSupport = errors.New("this build can't check GitHub attestations (it takes updates from its server's copy; or set update.verify to sha256-only)")
	ErrNoRepo          = errors.New("this build doesn't know which GitHub repository its releases come from: set update.repo, or update.verify to sha256-only")
	ErrNoAttestation   = errors.New("no GitHub attestation for this manifest")
	repoRE             = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	workflowRE         = regexp.MustCompile(`^\.github/workflows/[A-Za-z0-9_.-]+\.ya?ml$`)
)

// ValidRepo checks "owner/name".
func ValidRepo(s string) bool { return repoRE.MatchString(s) }

// ValidWorkflow checks ".github/workflows/<file>.yml".
func ValidWorkflow(s string) bool { return workflowRE.MatchString(s) }

// AttestOptions configures the attestation check.
type AttestOptions struct {
	Repo     string // "owner/name" (required)
	Workflow string // default DefaultWorkflow
	// RefRegex matches the git ref in the certificate's SAN after "@";
	// default "refs/tags/v" (releases are built from version tags).
	RefRegex string
	// TrustedRoot is a Sigstore trusted_root.json (GitHub's own instance
	// for private repos: `gh attestation trusted-root`; tests). Empty:
	// Sigstore's public-good instance through TUF, cached in CacheDir.
	TrustedRoot string
	CacheDir    string
	// API is the GitHub API base (default https://api.github.com), used
	// when a manifest comes without its .sigstore.json.
	API    string
	Client *http.Client
}

// splitBundles reads one bundle, a JSON-lines file of bundles, or a GitHub
// API answer ({"attestations":[{"bundle":{...}}]}) into single bundles.
func splitBundles(b []byte) [][]byte {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	var api struct {
		Attestations []struct {
			Bundle json.RawMessage `json:"bundle"`
		} `json:"attestations"`
	}
	if json.Unmarshal(b, &api) == nil && api.Attestations != nil {
		var out [][]byte
		for _, a := range api.Attestations {
			if len(a.Bundle) > 0 && string(a.Bundle) != "null" {
				out = append(out, a.Bundle)
			}
		}
		return out
	}
	var one map[string]json.RawMessage
	if json.Unmarshal(b, &one) == nil {
		return [][]byte{b}
	}
	var out [][]byte
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<10), 8<<20)
	for sc.Scan() {
		if l := bytes.TrimSpace(sc.Bytes()); len(l) > 0 && json.Valid(l) {
			out = append(out, append([]byte(nil), l...))
		}
	}
	return out
}

// joinBundles is the inverse for storing several bundles in one file.
func joinBundles(bs [][]byte) []byte {
	var out []string
	for _, b := range bs {
		var c bytes.Buffer
		if json.Compact(&c, b) == nil {
			out = append(out, c.String())
		}
	}
	return []byte(strings.Join(out, "\n") + "\n")
}
