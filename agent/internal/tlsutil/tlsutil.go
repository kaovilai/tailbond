// Package tlsutil creates the fallback self-signed server certificate and
// checks a server's certificate against the pairing pin.
package tlsutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

// EnsureSelfSigned creates certFile/keyFile (ECDSA P-256, 10 years) if missing.
func EnsureSelfSigned(certFile, keyFile string, names []string) error {
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
		return nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	cn := "tailbond"
	if len(names) > 0 && names[0] != "" {
		cn = names[0]
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn, Organization: []string{"tailbond"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, n := range names {
		if n == "" {
			continue
		}
		if ip := net.ParseIP(n); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, n)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		return err
	}
	return os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
}

// FingerprintFile returns the hex SHA-256 of the first certificate's DER.
func FingerprintFile(certFile string) (string, error) {
	b, err := os.ReadFile(certFile)
	if err != nil {
		return "", err
	}
	blk, _ := pem.Decode(b)
	if blk == nil || blk.Type != "CERTIFICATE" {
		return "", errors.New("no certificate in " + certFile)
	}
	return Fingerprint(blk.Bytes), nil
}

func Fingerprint(der []byte) string {
	s := sha256.Sum256(der)
	return hex.EncodeToString(s[:])
}

// ErrPinMismatch is fatal: the server presented a different certificate.
var ErrPinMismatch = errors.New("server certificate does not match the pinned SHA-256")

// CheckPin performs a QUIC handshake with the mqvpn server (ALPN h3) and
// compares its leaf certificate with pin. It is an early, readable check
// only: mqvpn itself is given the pin (CertSHA256) and enforces it on every
// handshake of the tunnel.
func CheckPin(ctx context.Context, addr, sni, pin string) error {
	pin = strings.ToLower(strings.ReplaceAll(pin, ":", ""))
	var got string
	cfg := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true, // verified below by pin
		NextProtos:         []string{"h3"},
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("no certificate")
			}
			got = Fingerprint(raw[0])
			if got != pin {
				return ErrPinMismatch
			}
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	conn, err := quic.DialAddr(ctx, addr, cfg, &quic.Config{HandshakeIdleTimeout: 6 * time.Second})
	if got != "" && got != pin {
		return fmt.Errorf("%w (got %s)", ErrPinMismatch, got)
	}
	if err != nil {
		if got == pin {
			// The cert matched; a later failure (ALPN, transport params) is
			// not a security problem.
			return nil
		}
		return fmt.Errorf("pin check handshake with %s: %w", addr, err)
	}
	_ = conn.CloseWithError(0, "pin check")
	return nil
}
