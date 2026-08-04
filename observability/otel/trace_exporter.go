package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newTraceExporter(
	ctx context.Context,
	config Config,
) (sdktrace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(
			config.TraceOTLPEndpoint,
		),
	}

	if config.OTLPInsecure {
		options = append(
			options,
			otlptracegrpc.WithInsecure(),
		)
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create OTLP trace exporter: %w",
			err,
		)
	}

	return exporter, nil
}
