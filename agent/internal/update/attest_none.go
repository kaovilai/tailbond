//go:build noattest

package update

import "context"

// AttestSupported: this build (-tags noattest, the router) has no
// attestation verifier; it updates from its server's mirror.
const AttestSupported = false

type Attestor struct{ o AttestOptions }

func NewAttestor(o AttestOptions) (*Attestor, error) { return nil, ErrNoAttestSupport }

func (a *Attestor) Repo() string { return a.o.Repo }

func (a *Attestor) Verify(ctx context.Context, digest []byte, bundles []byte) ([]byte, error) {
	return nil, ErrNoAttestSupport
}
