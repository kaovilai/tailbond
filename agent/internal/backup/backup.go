// Package backup reads and writes encrypted tailbond backups (.tbbackup,
// docs/architecture.md "Backup and restore"): a magic line, a JSON header
// (authenticated) and the AES-256-GCM sealed payload, keyed with scrypt.
package backup

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	Magic         = "TBBACKUP1"
	MinPassphrase = 10
	// scrypt cost: 32 MiB of memory, fine on a phone and a travel router
	ScryptN = 1 << 15
	ScryptR = 8
	ScryptP = 1
	MaxSize = 2 << 20
)

type Header struct {
	V            int    `json:"v"`
	KDF          string `json:"kdf"`
	N            int    `json:"n"`
	R            int    `json:"r"`
	P            int    `json:"p"`
	Salt         string `json:"salt"`
	Nonce        string `json:"nonce"`
	Role         string `json:"role"`
	Platform     string `json:"platform"`
	Host         string `json:"host,omitempty"`
	AgentVersion string `json:"agent_version,omitempty"`
	Created      string `json:"created"`
}

var (
	ErrFormat     = errors.New("not a tailbond backup")
	ErrPassphrase = errors.New("wrong passphrase or damaged file")
	ErrShort      = fmt.Errorf("the passphrase needs at least %d characters", MinPassphrase)
)

func deriveKey(pass string, h Header) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(h.Salt)
	if err != nil || len(salt) < 16 {
		return nil, ErrFormat
	}
	if h.KDF != "scrypt" || h.N < 1<<14 || h.N > 1<<20 || h.N&(h.N-1) != 0 || h.R < 1 || h.R > 32 || h.P < 1 || h.P > 16 {
		return nil, fmt.Errorf("%w: unsupported key derivation", ErrFormat)
	}
	return scrypt.Key([]byte(pass), salt, h.N, h.R, h.P, 32)
}

// Seal encrypts plaintext. h describes the backup (role, platform, host,
// agent version); the crypto fields are filled in.
func Seal(plaintext []byte, pass string, h Header) ([]byte, error) {
	if len([]rune(pass)) < MinPassphrase {
		return nil, ErrShort
	}
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	h.V, h.KDF, h.N, h.R, h.P = 1, "scrypt", ScryptN, ScryptR, ScryptP
	h.Salt, h.Nonce = base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(nonce)
	if h.Created == "" {
		h.Created = time.Now().UTC().Format(time.RFC3339)
	}
	hdr, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(pass, h)
	if err != nil {
		return nil, err
	}
	gcm, err := aead(key)
	if err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, hdr)
	var out bytes.Buffer
	out.WriteString(Magic + "\n")
	out.Write(hdr)
	out.WriteString("\n")
	out.WriteString(base64.StdEncoding.EncodeToString(ct))
	out.WriteString("\n")
	return out.Bytes(), nil
}

func aead(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// ReadHeader parses a backup's header without the passphrase.
func ReadHeader(data []byte) (Header, []byte, []byte, error) {
	var h Header
	if len(data) > MaxSize*2 {
		return h, nil, nil, fmt.Errorf("%w: too large", ErrFormat)
	}
	lines := strings.SplitN(string(data), "\n", 4)
	if len(lines) < 3 || strings.TrimRight(lines[0], "\r") != Magic {
		return h, nil, nil, ErrFormat
	}
	hdr := []byte(strings.TrimRight(lines[1], "\r"))
	if err := json.Unmarshal(hdr, &h); err != nil || h.V != 1 {
		return h, nil, nil, ErrFormat
	}
	ct, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[2]))
	if err != nil {
		return h, nil, nil, ErrFormat
	}
	return h, hdr, ct, nil
}

// Open decrypts a backup. A wrong passphrase, a damaged payload and an
// edited header all give ErrPassphrase (GCM can't tell them apart).
func Open(data []byte, pass string) (Header, []byte, error) {
	h, hdr, ct, err := ReadHeader(data)
	if err != nil {
		return h, nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(h.Nonce)
	if err != nil || len(nonce) != 12 {
		return h, nil, ErrFormat
	}
	key, err := deriveKey(pass, h)
	if err != nil {
		return h, nil, err
	}
	gcm, err := aead(key)
	if err != nil {
		return h, nil, err
	}
	pt, err := gcm.Open(nil, nonce, ct, hdr)
	if err != nil {
		return h, nil, ErrPassphrase
	}
	return h, pt, nil
}
