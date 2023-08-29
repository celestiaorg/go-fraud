package fraud

import (
	"context"

	"github.com/ipfs/go-datastore"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/celestiaorg/go-header"
)

var meter = otel.Meter("fraud")

// WithMetrics enables metrics to monitor fraud proofs.
func WithMetrics[H header.Header[H]](store Getter[H], unmarshaler ProofUnmarshaler[H]) {
	for _, proofType := range unmarshaler.List() {
		counter, err := meter.Int64ObservableGauge(string(proofType),
			metric.WithDescription("Stored fraud proof"),
		)
		if err != nil {
			panic(err)
		}

		callback := func(ctx context.Context, observer metric.Observer) error {
			proofs, err := store.Get(ctx, proofType)
			switch err {
			case nil:
				observer.ObserveInt64(counter, int64(len(proofs)),
					metric.WithAttributes(
						attribute.String("proof_type", string(proofType))))
			case datastore.ErrNotFound:
				observer.ObserveInt64(counter, 0,
					metric.WithAttributes(attribute.String("err", "not_found")))
			default:
				observer.ObserveInt64(counter, 0,
					metric.WithAttributes(attribute.String("err", "unknown")))
			}
			return nil
		}
		_, err = meter.RegisterCallback(callback, counter)
		if err != nil {
			panic(err)
		}
	}
}
