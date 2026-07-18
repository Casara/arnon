package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Histogram records measurements.
type Histogram struct {
	histogram metric.Float64Histogram
}

// NewHistogram creates a histogram metric.
func NewHistogram(
	name string,
	description string,
) (*Histogram, error) {
	histogram, err := meter().Float64Histogram(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create histogram %q: %w",
			name,
			err,
		)
	}

	return &Histogram{
		histogram: histogram,
	}, nil
}

// Record adds value as an observation in the histogram, tagged with
// attributes for this call only - Histogram itself carries no base
// attributes to merge with.
func (histogram *Histogram) Record(
	ctx context.Context,
	value float64,
	attributes ...Attribute,
) {
	histogram.histogram.Record(
		ctx,
		value,
		metric.WithAttributes(
			toKeyValues(
				attributes,
			)...,
		),
	)
}
