package fraudserv

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	ds_sync "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/go-header/headertest"

	"github.com/celestiaorg/go-fraud/fraudtest"
)

func TestStore_Put(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer t.Cleanup(cancel)

	p := fraudtest.NewValidProof[*headertest.DummyHeader]()
	bin, err := p.MarshalBinary()
	require.NoError(t, err)
	ds := ds_sync.MutexWrap(datastore.NewMapDatastore())
	store := namespace.Wrap(ds, makeKey(p.Type()))
	err = put(ctx, store, string(p.HeaderHash()), bin)
	require.NoError(t, err)
}

func TestStore_GetAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer t.Cleanup(cancel)

	proof := fraudtest.NewValidProof[*headertest.DummyHeader]()
	bin, err := proof.MarshalBinary()
	require.NoError(t, err)
	ds := ds_sync.MutexWrap(datastore.NewMapDatastore())
	proofStore := namespace.Wrap(ds, makeKey(proof.Type()))

	err = put(ctx, proofStore, string(proof.HeaderHash()), bin)
	require.NoError(t, err)

	proofs, err := getAll[*headertest.DummyHeader](ctx, proofStore, proof.Type(), unmarshaler)
	require.NoError(t, err)
	require.NotEmpty(t, proofs)
	require.NoError(t, proof.Validate(nil))
}

func Test_GetAllFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer t.Cleanup(cancel)

	proof := fraudtest.NewValidProof[*headertest.DummyHeader]()
	ds := ds_sync.MutexWrap(datastore.NewMapDatastore())
	store := namespace.Wrap(ds, makeKey(proof.Type()))

	proofs, err := getAll[*headertest.DummyHeader](ctx, store, proof.Type(), unmarshaler)
	require.Error(t, err)
	require.ErrorIs(t, err, datastore.ErrNotFound)
	require.Nil(t, proofs)
}

func Test_getByHash(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer t.Cleanup(cancel)

	proof := fraudtest.NewValidProof[*headertest.DummyHeader]()
	ds := ds_sync.MutexWrap(datastore.NewMapDatastore())
	store := namespace.Wrap(ds, makeKey(proof.Type()))
	bin, err := proof.MarshalBinary()
	require.NoError(t, err)
	err = put(ctx, store, string(proof.HeaderHash()), bin)
	require.NoError(t, err)
	_, err = getByHash(ctx, store, string(proof.HeaderHash()))
	require.NoError(t, err)
}
