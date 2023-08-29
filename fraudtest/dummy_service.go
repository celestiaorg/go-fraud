package fraudtest

import (
	"context"

	"github.com/celestiaorg/go-fraud"
	"github.com/celestiaorg/go-header"
	"github.com/celestiaorg/go-header/headertest"
)

type DummyService[H header.Header[H]] struct{}

func (d *DummyService[H]) Broadcast(context.Context, fraud.Proof[H]) error {
	return nil
}

func (d *DummyService[H]) Subscribe(fraud.ProofType) (fraud.Subscription[H], error) {
	return &subscription[H]{}, nil
}

func (d *DummyService[H]) Get(context.Context, fraud.ProofType) ([]fraud.Proof[*headertest.DummyHeader], error) {
	return nil, nil
}

type subscription[H header.Header[H]] struct{}

func (s *subscription[H]) Proof(context.Context) (fraud.Proof[H], error) {
	return nil, nil
}

func (s *subscription[H]) Cancel() {}
