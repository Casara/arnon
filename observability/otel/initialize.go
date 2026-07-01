package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Initialize installs OpenTelemetry components and returns a shutdown
// function.
func Initialize(config Config) (
	func(context.Context) error,
	error,
) {
	config = config.withDefaults()

	var tracerProvider *sdktrace.TracerProvider

	var meterProvider *sdkmetric.MeterProvider

	var err error

	if config.TracesEnabled {
		tracerProvider, err = newTracerProvider(
			context.Background(),
			config,
		)
		if err != nil {
			return nil, err
		}
	}

	if config.MetricsEnabled {
		meterProvider, err = newMeterProvider(
			context.Background(),
			config,
		)
		if err != nil {
			return nil, err
		}

		err = startRuntimeMetrics()
		if err != nil {
			return nil, err
		}
	}

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return shutdown(meterProvider, tracerProvider), nil
}

func shutdown(
	meterProvider *sdkmetric.MeterProvider,
	traceProvider *sdktrace.TracerProvider,
) func(
	ctx context.Context,
) error {
	return func(
		ctx context.Context,
	) error {
		var firstErr error

		if meterProvider != nil {
			err := meterProvider.Shutdown(ctx)
			if err != nil {
				firstErr = err
			}
		}

		err := traceProvider.Shutdown(ctx)
		if err != nil && firstErr == nil {
			firstErr = err
		}

		return firstErr
	}
}
