package otel

import (
	"context"

	"go.opentelemetry.io/otel"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newTracerProvider creates and registers the OpenTelemetry tracer provider.
func newTracerProvider(
	ctx context.Context,
	config Config,
) (*sdktrace.TracerProvider, error) {
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(
			newResource(config),
		),

		sdktrace.WithSampler(
			sdktrace.AlwaysSample(),
		),
	}

	if config.TraceOTLPEndpoint != "" {
		exporter, err := newTraceExporter(
			ctx,
			config,
		)
		if err != nil {
			return nil, err
		}

		options = append(
			options,
			sdktrace.WithBatcher(exporter),
		)
	}

	provider := sdktrace.NewTracerProvider(
		options...,
	)

	otel.SetTracerProvider(
		provider,
	)

	return provider, nil
}
