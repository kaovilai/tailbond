package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Manifest is <channel>.json, format 1.
type Manifest struct {
	Format                 int        `json:"format"`
	Channel                string     `json:"channel"`
	Version                string     `json:"version"`
	Released               string     `json:"released,omitempty"`
	Protocol               int        `json:"protocol"`
	RequiresServerProtocol int        `json:"requires_server_protocol"`
	Security               bool       `json:"security,omitempty"`
	NotesURL               string     `json:"notes_url,omitempty"`
	Artifacts              []Artifact `json:"artifacts"`
}

type Artifact struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"` // linux | ipk | rpm | apk
	Arch        string `json:"arch"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	VersionCode int64  `json:"version_code,omitempty"`
}

// Verification modes (config update.verify).
const (
	VerifyAttestation = "attestation" // default: a GitHub attestation for the manifest
	VerifySHA256      = "sha256-only" // trust the source (HTTPS, an upload); check SHA-256s only
)

// How a manifest was trusted (status update.verified).
const (
	VerifiedAttestation = "attestation"
	VerifiedMirror      = "mirror" // the home server's copy, over the authenticated tunnel/tailnet
	VerifiedSHA256      = "sha256"
)

// Trust decides whether a manifest from a public source or an upload is
// authentic. Attest is nil when the build has no verifier (noattest) or
// no repository is known; AttestErr says why.
type Trust struct {
	Mode      string
	Attest    *Attestor
	AttestErr error
}

// Check verifies raw against the policy; bundles is the .sigstore.json
// content (nil: ask GitHub). It returns how the manifest was trusted and the
// bundle to keep with it.
func (t Trust) Check(ctx context.Context, raw, bundles []byte) (*Manifest, string, []byte, error) {
	m, err := ParseManifest(raw)
	if err != nil {
		return nil, "", nil, err
	}
	if t.Mode == VerifySHA256 {
		return m, VerifiedSHA256, bundles, nil
	}
	if t.Attest == nil {
		err := t.AttestErr
		if err == nil {
			err = ErrNoAttestSupport
		}
		return nil, "", nil, err
	}
	d := sha256.Sum256(raw)
	b, err := t.Attest.Verify(ctx, d[:], bundles)
	if err != nil {
		return nil, "", nil, err
	}
	return m, VerifiedAttestation, b, nil
}

// ParseManifest reads and sanity-checks a manifest (it doesn't say whether
// it's authentic: see Trust).
func ParseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if m.Format != 1 {
		return nil, fmt.Errorf("manifest format %d isn't supported by this version", m.Format)
	}
	if _, ok := Parse(m.Version); !ok {
		return nil, fmt.Errorf("manifest version %q isn't semver", m.Version)
	}
	for _, a := range m.Artifacts {
		if a.Name == "" || strings.ContainsAny(a.Name, "/\\") || strings.HasPrefix(a.Name, ".") {
			return nil, fmt.Errorf("artifact name %q", a.Name)
		}
		if _, err := hex.DecodeString(a.SHA256); err != nil || len(a.SHA256) != 64 {
			return nil, fmt.Errorf("artifact %s: bad sha256", a.Name)
		}
	}
	return &m, nil
}

// Find returns the artifact of a kind for one of the archs (in order).
func (m *Manifest) Find(kind string, archs ...string) (Artifact, bool) {
	for _, arch := range archs {
		for _, a := range m.Artifacts {
			if a.Kind == kind && a.Arch == arch {
				return a, true
			}
		}
	}
	return Artifact{}, false
}

// CheckFile verifies a downloaded artifact's size and SHA-256.
func CheckFile(path string, a Artifact) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return err
	}
	if a.Size > 0 && n != a.Size {
		return fmt.Errorf("%s: %d bytes, the manifest says %d", a.Name, n, a.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, a.SHA256) {
		return fmt.Errorf("%s: checksum mismatch", a.Name)
	}
	return nil
}

// CheckBytes is CheckFile for data in memory.
func CheckBytes(b []byte, a Artifact) error {
	if a.Size > 0 && int64(len(b)) != a.Size {
		return fmt.Errorf("%s: %d bytes, the manifest says %d", a.Name, len(b), a.Size)
	}
	h := sha256.Sum256(b)
	if !strings.EqualFold(hex.EncodeToString(h[:]), a.SHA256) {
		return fmt.Errorf("%s: checksum mismatch", a.Name)
	}
	return nil
}
