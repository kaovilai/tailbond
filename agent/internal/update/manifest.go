package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// Signature is <channel>.json.sig.
type Signature struct {
	KeyID string `json:"key_id"`
	Sig   string `json:"sig"`
}

var (
	ErrNoKeys       = errors.New("this build has no update key")
	ErrUnknownKey   = errors.New("manifest signed with a key this build doesn't trust")
	ErrBadSignature = errors.New("manifest signature doesn't match")
)

// KeyID is the hex of the first 8 bytes of SHA-256(public key).
func KeyID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:8])
}

// ParseKeys reads base64 raw public keys (comma or space separated);
// invalid entries are skipped.
func ParseKeys(lists ...string) []ed25519.PublicKey {
	var out []ed25519.PublicKey
	seen := map[string]bool{}
	for _, l := range lists {
		for _, s := range strings.FieldsFunc(l, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
			b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
			if err != nil || len(b) != ed25519.PublicKeySize || seen[string(b)] {
				continue
			}
			seen[string(b)] = true
			out = append(out, ed25519.PublicKey(b))
		}
	}
	return out
}

// Verify checks sig over the exact manifest bytes and parses the manifest.
func Verify(manifest, sig []byte, keys []ed25519.PublicKey) (*Manifest, error) {
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	var s Signature
	if err := json.Unmarshal(sig, &s); err != nil {
		return nil, fmt.Errorf("signature file: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s.Sig))
	if err != nil || len(raw) != ed25519.SignatureSize {
		return nil, ErrBadSignature
	}
	var key ed25519.PublicKey
	for _, k := range keys {
		if KeyID(k) == strings.ToLower(s.KeyID) {
			key = k
		}
	}
	if key == nil {
		return nil, ErrUnknownKey
	}
	if !ed25519.Verify(key, manifest, raw) {
		return nil, ErrBadSignature
	}
	var m Manifest
	if err := json.Unmarshal(manifest, &m); err != nil {
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
		if len(a.SHA256) != 64 {
			return nil, fmt.Errorf("artifact %s: bad sha256", a.Name)
		}
	}
	return &m, nil
}

// Sign makes a signature file (tests and the e2e's fake release server;
// releases are signed by scripts/release/sign-manifest.sh).
func Sign(manifest []byte, priv ed25519.PrivateKey) []byte {
	pub := priv.Public().(ed25519.PublicKey)
	b, _ := json.Marshal(Signature{KeyID: KeyID(pub), Sig: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, manifest))})
	return b
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
