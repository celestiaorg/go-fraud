package fraudserv

import (
	"context"
	"fmt"
	"reflect"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/celestiaorg/go-fraud"
)

// subscription wraps pubsub subscription and handles Fraud Proof from the pubsub topic.
type subscription struct {
	subscription *pubsub.Subscription
}

func (s *subscription) Proof(ctx context.Context) (fraud.Proof, error) {
	if s.subscription == nil {
		panic("fraud: subscription is not created")
	}
	data, err := s.subscription.Next(ctx)
	if err != nil {
		return nil, err
	}
	proof, ok := data.ValidatorData.(fraud.Proof)
	if !ok {
		panic(fmt.Sprintf("fraud: unexpected type received %s", reflect.TypeOf(data.ValidatorData)))
	}
	return proof, nil
}

func (s *subscription) Cancel() {
	s.subscription.Cancel()
}
