package otel

import (
	"context"

	"go.opentelemetry.io/otel"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// newMeterProvider creates and registers the OpenTelemetry meter provider.
func newMeterProvider(
	ctx context.Context,
	config Config,
) (*sdkmetric.MeterProvider, error) {
	exporter, err := newMetricExporter(
		ctx,
		config,
	)
	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			reader,
		),

		sdkmetric.WithResource(
			newResource(config),
		),
	)

	otel.SetMeterProvider(
		provider,
	)

	return provider, nil
}
