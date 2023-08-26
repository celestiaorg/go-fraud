package fraudtest

import (
	"encoding/json"
	"errors"

	"github.com/celestiaorg/go-fraud"
	"github.com/celestiaorg/go-header"
)

const DummyProofType fraud.ProofType = "DummyProof"

type DummyProof[H header.Header[H]] struct {
	Valid bool
}

func NewValidProof[H header.Header[H]]() *DummyProof[H] {
	return &DummyProof[H]{true}
}

func NewInvalidProof[H header.Header[H]]() *DummyProof[H] {
	return &DummyProof[H]{false}
}

func (m *DummyProof[H]) Type() fraud.ProofType {
	return "DummyProof"
}

func (m *DummyProof[H]) HeaderHash() []byte {
	return []byte("hash")
}

func (m *DummyProof[H]) Height() uint64 {
	return 1
}

func (m *DummyProof[H]) Validate(H) error {
	if !m.Valid {
		return errors.New("DummyProof: proof is not valid")
	}
	return nil
}

func (m *DummyProof[H]) MarshalBinary() (data []byte, err error) {
	return json.Marshal(m)
}

func (m *DummyProof[H]) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}
