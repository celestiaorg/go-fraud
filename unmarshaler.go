package fraud

import (
	"github.com/celestiaorg/go-header"
)

// MultiUnmarshaler contains a mapping of all registered proof
// types to their unmarshal functions.
type MultiUnmarshaler[H header.Header[H]] struct {
	Unmarshalers map[ProofType]func([]byte) (Proof[H], error)
}

func (d MultiUnmarshaler[H]) List() []ProofType {
	types := make([]ProofType, len(d.Unmarshalers))
	for tp := range d.Unmarshalers {
		types = append(types, tp)
	}
	return types
}

func (d MultiUnmarshaler[H]) Unmarshal(proofType ProofType, data []byte) (Proof[H], error) {
	uf, ok := d.Unmarshalers[proofType]
	if !ok {
		return nil, &ErrNoUnmarshaler{ProofType: proofType}
	}

	return uf(data)
}
