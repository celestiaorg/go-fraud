package fraudserv

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/celestiaorg/go-header"

	"github.com/celestiaorg/go-fraud"
)

var (
	log    = logging.Logger("fraudserv")
	tracer = otel.Tracer("fraudserv")
)

const (
	// fraudRequests is the amount of external requests that will be tried to get fraud proofs from
	// other peers.
	fraudRequests = 5

	// headThreshold specifies the maximum allowable height of the Proof
	// relative to the network head to be verified.
	headThreshold uint64 = 20
)

// ProofService is responsible for validating and propagating Fraud Proofs.
// It implements the Service interface.
type ProofService[H header.Header[H]] struct {
	networkID string

	ctx    context.Context
	cancel context.CancelFunc

	topicsLk sync.RWMutex
	topics   map[fraud.ProofType]*pubsub.Topic

	storesLk sync.RWMutex
	stores   map[fraud.ProofType]datastore.Datastore

	verifiersLk sync.RWMutex
	verifiers   map[fraud.ProofType]fraud.Verifier[H]

	pubsub        *pubsub.PubSub
	host          host.Host
	headerGetter  fraud.HeaderFetcher[H]
	headGetter    fraud.HeadGetter[H]
	unmarshal     fraud.ProofUnmarshaler[H]
	ds            datastore.Datastore
	syncerEnabled bool
}

func NewProofService[H header.Header[H]](
	p *pubsub.PubSub,
	host host.Host,
	headerGetter fraud.HeaderFetcher[H],
	headGetter fraud.HeadGetter[H],
	unmarshal fraud.ProofUnmarshaler[H],
	ds datastore.Datastore,
	syncerEnabled bool,
	networkID string,
) *ProofService[H] {
	return &ProofService[H]{
		pubsub:        p,
		host:          host,
		headerGetter:  headerGetter,
		headGetter:    headGetter,
		unmarshal:     unmarshal,
		verifiers:     make(map[fraud.ProofType]fraud.Verifier[H]),
		topics:        make(map[fraud.ProofType]*pubsub.Topic),
		stores:        make(map[fraud.ProofType]datastore.Datastore),
		ds:            ds,
		networkID:     networkID,
		syncerEnabled: syncerEnabled,
	}
}

// registerProofTopics registers  as pubsub topics to be joined.
func (f *ProofService[H]) registerProofTopics() error {
	for _, proofType := range f.unmarshal.List() {
		t, err := join(f.pubsub, proofType, f.networkID, f.processIncoming)
		if err != nil {
			return err
		}
		f.topicsLk.Lock()
		f.topics[proofType] = t
		f.topicsLk.Unlock()
	}
	return nil
}

// Start joins fraud proofs topics, sets the stream handler for fraudProtocolID and starts syncing
// if syncer is enabled.
func (f *ProofService[H]) Start(context.Context) error {
	f.ctx, f.cancel = context.WithCancel(context.Background())
	if err := f.registerProofTopics(); err != nil {
		return err
	}
	id := protocolID(f.networkID)
	log.Infow("starting fraud proof service", "protocol ID", id)

	f.host.SetStreamHandler(id, f.handleFraudMessageRequest)
	if f.syncerEnabled {
		go f.syncFraudProofs(f.ctx, id)
	}
	return nil
}

// Stop removes the stream handler and cancels the underlying ProofService
func (f *ProofService[H]) Stop(context.Context) (err error) {
	f.host.RemoveStreamHandler(protocolID(f.networkID))
	f.topicsLk.Lock()
	for tp, topic := range f.topics {
		delete(f.topics, tp)
		err = errors.Join(topic.Close())
	}
	f.topicsLk.Unlock()
	f.cancel()
	return
}

func (f *ProofService[H]) Subscribe(proofType fraud.ProofType) (_ fraud.Subscription[H], err error) {
	f.topicsLk.Lock()
	defer f.topicsLk.Unlock()
	t, ok := f.topics[proofType]
	if !ok {
		return nil, fmt.Errorf("topic for %s does not exist", proofType)
	}
	subs, err := t.Subscribe()
	if err != nil {
		return nil, err
	}
	return &subscription[H]{subs}, nil
}

func (f *ProofService[H]) Broadcast(ctx context.Context, p fraud.Proof[H]) error {
	bin, err := p.MarshalBinary()
	if err != nil {
		return err
	}
	f.topicsLk.RLock()
	t, ok := f.topics[p.Type()]
	f.topicsLk.RUnlock()
	if !ok {
		return fmt.Errorf("fraud: unmarshaler for %s proof is not registered", p.Type())
	}
	return t.Publish(ctx, bin)
}

func (f *ProofService[H]) AddVerifier(proofType fraud.ProofType, verifier fraud.Verifier[H]) error {
	f.verifiersLk.Lock()
	defer f.verifiersLk.Unlock()
	if _, ok := f.verifiers[proofType]; ok {
		return fmt.Errorf("verifier for proof type %s already exist", proofType)
	}
	f.verifiers[proofType] = verifier
	return nil
}

// processIncoming encompasses the logic for validating fraud proofs.
func (f *ProofService[H]) processIncoming(
	ctx context.Context,
	proofType fraud.ProofType,
	from peer.ID,
	msg *pubsub.Message,
) (res pubsub.ValidationResult) {
	ctx, span := tracer.Start(ctx, "process_proof", trace.WithAttributes(
		attribute.String("proof_type", string(proofType)),
	))
	defer span.End()

	defer func() {
		r := recover()
		if r != nil {
			err := fmt.Errorf("PANIC while processing a proof: %s", r)
			log.Error(err)
			span.RecordError(err)
			res = pubsub.ValidationReject
		}
	}()

	// unmarshal message to the Proof.
	// Peer will be added to black list if unmarshalling fails.
	proof, err := f.unmarshal.Unmarshal(proofType, msg.Data)
	if err != nil {
		log.Errorw("unmarshalling failed", "err", err)
		if !errors.Is(err, &fraud.ErrNoUnmarshaler{}) {
			f.pubsub.BlacklistPeer(from)
		}
		span.RecordError(err)
		return pubsub.ValidationReject
	}
	// check the fraud proof locally and ignore if it has been already stored locally.
	if f.verifyLocal(ctx, proofType, hex.EncodeToString(proof.HeaderHash()), msg.Data) {
		span.AddEvent("received_known_fraud_proof", trace.WithAttributes(
			attribute.String("proof_type", string(proof.Type())),
			attribute.Int64("block_height", int64(proof.Height())), //nolint:gosec
			attribute.String("block_hash", hex.EncodeToString(proof.HeaderHash())),
			attribute.String("from_peer", from.String()),
		))
		return pubsub.ValidationIgnore
	}

	head, err := f.headGetter(ctx)
	if err != nil {
		log.Errorw("failed to fetch current network head to verify a fraud proof",
			"err", err, "proofType", proof.Type(), "height", proof.Height())
		return pubsub.ValidationIgnore
	}

	if head.Height()+headThreshold < proof.Height() {
		err = fmt.Errorf("received proof above the max threshold."+
			"maxHeight: %d, proofHeight: %d, proofType: %s",
			head.Height()+headThreshold,
			proof.Height(),
			proof.Type(),
		)
		log.Error(err)
		span.RecordError(err)
		return pubsub.ValidationReject
	}

	msg.ValidatorData = proof

	// fetch extended header in order to verify the fraud proof.
	extHeader, err := f.headerGetter(ctx, proof.Height())
	if err != nil {
		log.Errorw("failed to fetch header to verify a fraud proof",
			"err", err, "proofType", proof.Type(), "height", proof.Height())
		return pubsub.ValidationIgnore
	}

	// execute the verifier for proof type if exists
	f.verifiersLk.RLock()
	verifier, ok := f.verifiers[proofType]
	f.verifiersLk.RUnlock()
	if ok {
		status, err := verifier(proof)
		if err != nil {
			log.Errorw("failed to run the verifier", "err", err, "proofType", proof.Type())
			return pubsub.ValidationReject
		}
		if !status {
			log.Errorw("invalid fraud proof", "proofType", proof.Type())
			return pubsub.ValidationReject
		}
	}

	// validate the fraud proof.
	// Peer will be added to black list if the validation fails.
	err = proof.Validate(extHeader)
	if err != nil {
		log.Errorw("proof validation err: ",
			"err", err, "proofType", proof.Type(), "height", proof.Height())
		f.pubsub.BlacklistPeer(from)
		span.RecordError(err)
		return pubsub.ValidationReject
	}

	span.AddEvent("received_valid_proof", trace.WithAttributes(
		attribute.String("proof_type", string(proof.Type())),
		attribute.Int64("block_height", int64(proof.Height())), //nolint:gosec
		attribute.String("block_hash", hex.EncodeToString(proof.HeaderHash())),
		attribute.String("from_peer", from.String()),
	))

	// add the fraud proof to storage.
	err = f.put(ctx, proof.Type(), hex.EncodeToString(proof.HeaderHash()), msg.Data)
	if err != nil {
		log.Errorw("failed to store fraud proof", "err", err)
		span.RecordError(err)
	}

	span.SetStatus(codes.Ok, "")
	return pubsub.ValidationAccept
}

func (f *ProofService[H]) Get(ctx context.Context, proofType fraud.ProofType) ([]fraud.Proof[H], error) {
	f.storesLk.Lock()
	store, ok := f.stores[proofType]
	if !ok {
		store = initStore(proofType, f.ds)
		f.stores[proofType] = store
	}
	f.storesLk.Unlock()

	return getAll(ctx, store, proofType, f.unmarshal)
}

// put adds a fraud proof to the local storage.
func (f *ProofService[H]) put(ctx context.Context, proofType fraud.ProofType, hash string, data []byte) error {
	f.storesLk.Lock()
	store, ok := f.stores[proofType]
	if !ok {
		store = initStore(proofType, f.ds)
		f.stores[proofType] = store
	}
	f.storesLk.Unlock()
	return put(ctx, store, hash, data)
}

// verifyLocal checks if a fraud proof has been stored locally.
func (f *ProofService[H]) verifyLocal(ctx context.Context, proofType fraud.ProofType, hash string, data []byte) bool {
	f.storesLk.RLock()
	storage, ok := f.stores[proofType]
	f.storesLk.RUnlock()
	if !ok {
		return false
	}

	proof, err := getByHash(ctx, storage, hash)
	if err != nil {
		if !errors.Is(err, datastore.ErrNotFound) {
			log.Error(err)
		}
		return false
	}

	return bytes.Equal(proof, data)
}
