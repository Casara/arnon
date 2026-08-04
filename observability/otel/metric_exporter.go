package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func newMetricExporter(
	ctx context.Context,
	config Config,
) (sdkmetric.Exporter, error) {
	options := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(
			config.MetricOTLPEndpoint,
		),
	}

	if config.OTLPInsecure {
		options = append(
			options,
			otlpmetricgrpc.WithInsecure(),
		)
	}

	exporter, err := otlpmetricgrpc.New(
		ctx,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create OTLP metric exporter: %w",
			err,
		)
	}

	return exporter, nil
}
