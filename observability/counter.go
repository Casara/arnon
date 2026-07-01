package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Counter records monotonically increasing values.
type Counter struct {
	counter metric.Int64Counter
}

// NewCounter creates a counter metric.
func NewCounter(
	name string,
	description string,
) (*Counter, error) {
	counter, err := meter().Int64Counter(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create counter %q: %w",
			name,
			err,
		)
	}

	return &Counter{
		counter: counter,
	}, nil
}

// Add increments the counter.
func (counter *Counter) Add(
	ctx context.Context,
	value int64,
	attributes ...Attribute,
) {
	counter.counter.Add(
		ctx,
		value,
		metric.WithAttributes(
			toKeyValues(
				attributes,
			)...,
		),
	)
}
