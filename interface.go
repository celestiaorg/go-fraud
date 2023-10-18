package fraud

import (
	"context"

	"github.com/celestiaorg/go-header"
)

// HeaderFetcher aliases a function that is used to fetch an ExtendedHeader from store by height.
type HeaderFetcher[H header.Header[H]] func(context.Context, uint64) (H, error)

// HeadFetcher aliases a function that is used to get current network head.
type HeadFetcher[H header.Header[H]] func(ctx context.Context) (H, error)

// Verifier is a function that is executed as part of processing the incoming fraud proof
type Verifier[H header.Header[H]] func(fraud Proof[H]) (bool, error)

// ProofUnmarshaler contains methods that allow an instance of ProofService
// to access unmarshalers for various ProofTypes.
type ProofUnmarshaler[H header.Header[H]] interface {
	// List supported ProofTypes.
	List() []ProofType
	// Unmarshal decodes bytes into a Proof of a given ProofType.
	Unmarshal(ProofType, []byte) (Proof[H], error)
}

// Service encompasses the behavior necessary to subscribe and broadcast
// fraud proofs within the network.
type Service[H header.Header[H]] interface {
	Subscriber[H]
	Broadcaster[H]
	Getter[H]
}

// Broadcaster is a generic interface that sends a `Proof` to all nodes subscribed on the
// Broadcaster's topic.
type Broadcaster[H header.Header[H]] interface {
	// Broadcast takes a fraud `Proof` data structure interface and broadcasts it to local
	// subscriptions and peers. It may additionally cache/persist Proofs for future
	// access via Getter and to serve Proof requests to peers in the network.
	Broadcast(context.Context, Proof[H]) error
}

// Subscriber encompasses the behavior necessary to
// subscribe/unsubscribe from new FraudProof events from the
// network.
type Subscriber[H header.Header[H]] interface {
	// Subscribe allows to subscribe on a Proof pub sub topic by its type.
	Subscribe(ProofType) (Subscription[H], error)
	// AddVerifier allows for supplying additional verification logic which
	// will be run as part of processing the incoming fraud proof.
	// This only supplements the main validation done by Proof.Validate
	AddVerifier(ProofType, Verifier[H]) error
}

// Getter encompasses the behavior to fetch stored fraud proofs.
type Getter[H header.Header[H]] interface {
	// Get fetches fraud proofs from the disk by its type.
	Get(context.Context, ProofType) ([]Proof[H], error)
}

// Subscription returns a valid proof if one is received on the topic.
type Subscription[H header.Header[H]] interface {
	// Proof returns already verified valid proof.
	Proof(context.Context) (Proof[H], error)
	Cancel()
}
