package fraud

import (
	"context"
	"encoding"
	"fmt"

	"github.com/celestiaorg/go-header"
)

// ProofType type defines a unique proof type string.
type ProofType string

// String returns string representation of ProofType.
func (pt ProofType) String() string {
	return string(pt)
}

// Proof is a generic interface that will be used for all types of fraud proofs in the network.
type Proof[H header.Header[H]] interface {
	// Type returns the exact type of fraud proof.
	Type() ProofType
	// HeaderHash returns the block hash.
	HeaderHash() []byte
	// Height returns the block height corresponding to the Proof.
	Height() uint64
	// Validate check the validity of fraud proof.
	// Validate throws an error if some conditions don't pass and thus fraud proof is not valid.
	// NOTE: header.ExtendedHeader should pass basic validation otherwise it will panic if it's
	// malformed.
	Validate(H) error

	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// OnProof subscribes to the given Fraud Proof topic via the given Subscriber.
// In case a Fraud Proof is received, then the given handle function will be invoked.
func OnProof[H header.Header[H]](ctx context.Context, suber Subscriber[H], p ProofType, handle func(proof Proof[H])) {
	subscription, err := suber.Subscribe(p)
	if err != nil {
		return
	}
	defer subscription.Cancel()

	// At this point we receive already verified fraud proof,
	// so there is no need to call Validate.
	proof, err := subscription.Proof(ctx)
	if err != nil {
		return
	}

	handle(proof)
}

type ErrFraudExists[H header.Header[H]] struct {
	Proof []Proof[H]
}

func (e *ErrFraudExists[H]) Error() string {
	return fmt.Sprintf("fraud: %s proof exists\n", e.Proof[0].Type())
}

type ErrNoUnmarshaler struct {
	ProofType ProofType
}

func (e *ErrNoUnmarshaler) Error() string {
	return fmt.Sprintf("fraud: unmarshaler for %s type is not registered", e.ProofType)
}
