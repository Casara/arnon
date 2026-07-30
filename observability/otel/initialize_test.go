package otel_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	arnonotel "github.com/casara/arnon/observability/otel"
)

// resetGlobalProviders saves the current global tracer/meter providers
// and restores them once the test finishes, so one test's
// otel.SetTracerProvider/otel.SetMeterProvider call (made indirectly
// through Initialize) can't leak into another. Every test in this
// file calls Initialize, which mutates this process-wide global
// state, so none of them can run in parallel with each other -
// callers must not add t.Parallel() to a test using this helper.
func resetGlobalProviders(t *testing.T) {
	t.Helper()

	previousTracer := otel.GetTracerProvider()
	previousMeter := otel.GetMeterProvider()

	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracer)
		otel.SetMeterProvider(previousMeter)
	})
}

//nolint:paralleltest // Initialize mutates the process-wide otel global providers, see resetGlobalProviders
func TestInitialize_NothingEnabledShutsDownWithoutPanicking(t *testing.T) {
	resetGlobalProviders(t)

	shutdown, err := arnonotel.Initialize(arnonotel.Config{
		ServiceName: "arnon-test",
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Regression test: shutdown used to call traceProvider.Shutdown
	// unconditionally, even when TracesEnabled was false and
	// traceProvider was left nil - a guaranteed nil pointer dereference.
	err = shutdown(context.Background())
	if err != nil {
		t.Fatalf("expected no error when nothing was enabled, got %v", err)
	}
}

//nolint:paralleltest // Initialize mutates the process-wide otel global providers, see resetGlobalProviders
func TestInitialize_TracesEnabledInstallsRealTracerProvider(t *testing.T) {
	resetGlobalProviders(t)

	shutdown, err := arnonotel.Initialize(arnonotel.Config{
		ServiceName:       "arnon-test",
		TracesEnabled:     true,
		TraceOTLPEndpoint: "127.0.0.1:1",
		OTLPInsecure:      true,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatalf(
			"expected a real *sdktrace.TracerProvider to be installed, got %T",
			otel.GetTracerProvider(),
		)
	}

	assertShutdownRespectsDeadline(t, shutdown)
}

//nolint:paralleltest // Initialize mutates the process-wide otel global providers, see resetGlobalProviders
func TestInitialize_MetricsEnabledInstallsRealMeterProvider(t *testing.T) {
	resetGlobalProviders(t)

	shutdown, err := arnonotel.Initialize(arnonotel.Config{
		ServiceName:        "arnon-test",
		MetricsEnabled:     true,
		MetricOTLPEndpoint: "127.0.0.1:1",
		OTLPInsecure:       true,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Fatalf(
			"expected a real *sdkmetric.MeterProvider to be installed, got %T",
			otel.GetMeterProvider(),
		)
	}

	assertShutdownRespectsDeadline(t, shutdown)
}

//nolint:paralleltest // Initialize mutates the process-wide otel global providers, see resetGlobalProviders
func TestInitialize_BothEnabledTogether(t *testing.T) {
	resetGlobalProviders(t)

	shutdown, err := arnonotel.Initialize(arnonotel.Config{
		ServiceName:        "arnon-test",
		TracesEnabled:      true,
		MetricsEnabled:     true,
		TraceOTLPEndpoint:  "127.0.0.1:1",
		MetricOTLPEndpoint: "127.0.0.1:1",
		OTLPInsecure:       true,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatalf(
			"expected a real *sdktrace.TracerProvider to be installed, got %T",
			otel.GetTracerProvider(),
		)
	}

	if _, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Fatalf(
			"expected a real *sdkmetric.MeterProvider to be installed, got %T",
			otel.GetMeterProvider(),
		)
	}

	assertShutdownRespectsDeadline(t, shutdown)
}

// assertShutdownRespectsDeadline calls shutdown with a short-lived
// context and fails the test if it takes drastically longer than that
// deadline to return - proving shutdown doesn't hang forever against
// an unreachable OTLP endpoint, regardless of whether the export
// itself succeeds (it won't: 127.0.0.1:1 refuses every connection).
func assertShutdownRespectsDeadline(t *testing.T, shutdown func(context.Context) error) {
	t.Helper()

	const deadline = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	start := time.Now()
	_ = shutdown(ctx)
	elapsed := time.Since(start)

	if elapsed > deadline*5 {
		t.Fatalf(
			"shutdown took %v, well past its %v deadline - it isn't respecting the context",
			elapsed,
			deadline,
		)
	}
}
