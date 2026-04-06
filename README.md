# go-fraud

[![Go Reference](https://pkg.go.dev/badge/github.com/celestiaorg/go-fraud.svg)](https://pkg.go.dev/github.com/celestiaorg/go-fraud)
[![GitHub release](https://img.shields.io/github/v/release/celestiaorg/go-fraud.svg)](https://github.com/celestiaorg/go-fraud/releases)
[![Go CI](https://github.com/celestiaorg/go-fraud/actions/workflows/go-ci.yml/badge.svg)](https://github.com/celestiaorg/go-fraud/actions/workflows/go-ci.yml)

A Go library for broadcasting, subscribing to, and verifying fraud
proofs over a libp2p network. It is designed for modular blockchain
architectures where nodes need a protocol for detecting and
propagating invalid block data.

## Overview

go-fraud provides:

- **Fraud proof interfaces** — generic `Proof`, `Service`,
  `Broadcaster`, `Subscriber`, and `Getter` interfaces that
  can be implemented for any proof type.
- **libp2p-based proof service** (`fraudserv`) — a ready-to-use
  implementation that uses libp2p pubsub for proof propagation,
  a request/response protocol for syncing proofs from peers,
  and a local datastore for persistence.
- **Test helpers** (`fraudtest`) — dummy proof types and a mock
  service for use in unit tests.
- **OpenTelemetry instrumentation** — built-in tracing and
  metrics for proof processing.

## Installation

```shell
go get github.com/celestiaorg/go-fraud
```

## Usage

### Defining a proof type

Implement the `fraud.Proof` interface for your proof type:

```go
type Proof[H header.Header[H]] interface {
    Type() ProofType
    HeaderHash() []byte
    Height() uint64
    Validate(H) error
    encoding.BinaryMarshaler
    encoding.BinaryUnmarshaler
}
```

### Creating a proof service

```go
proofService := fraudserv.NewProofService(
    pubsub,        // libp2p pubsub instance
    host,          // libp2p host
    headerGetter,  // func to fetch headers by height
    headGetter,    // func to get the current network head
    unmarshaler,   // proof unmarshaler
    datastore,     // datastore for persistence
    syncerEnabled, // enable syncing proofs from peers
    networkID,     // network identifier
)
```

### Subscribing to fraud proofs

```go
fraud.OnProof(ctx, proofService, myProofType, func(proof fraud.Proof[H]) {
    // handle the verified fraud proof (e.g. halt the node)
})
```

## Development

### Prerequisites

- Go 1.24+

### Common commands

```shell
make deps       # install dependencies
make test       # run unit tests
make lint       # run linters
make fmt        # format code
make benchmark  # run benchmarks
```

## License

[Apache 2.0](LICENSE)
