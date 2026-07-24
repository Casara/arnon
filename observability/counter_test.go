package observability_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/Casara/arnon/observability"
)

// TestCounter_AddRecordsValueAndAttributes drives a real
// sdkmetric.MeterProvider (via otel.SetMeterProvider) instead of the
// no-op default, so it must not run in parallel with any other test
// in this package that does the same - otel.SetMeterProvider mutates
// process-wide global state, and two tests racing on it could each
// read the other's provider.
//
//nolint:paralleltest // mutates the process-wide otel global meter provider, see comment above
func TestCounter_AddRecordsValueAndAttributes(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	previous := otel.GetMeterProvider()

	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	counter, err := observability.NewCounter(
		"observability_test_counter",
		"a counter created by a test",
	)
	if err != nil {
		t.Fatalf("NewCounter failed: %v", err)
	}

	ctx := context.Background()

	counter.Add(ctx, 3, observability.String("key", "value"))
	counter.Add(ctx, 2, observability.String("key", "value"))

	var collected metricdata.ResourceMetrics

	err = reader.Collect(ctx, &collected)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	metric := findMetric(t, collected, "observability_test_counter")

	sum, ok := metric.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected metricdata.Sum[int64], got %T", metric.Data)
	}

	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected exactly one data point, got %d", len(sum.DataPoints))
	}

	dataPoint := sum.DataPoints[0]

	if dataPoint.Value != 5 {
		t.Fatalf("expected the two Add calls to accumulate to 5, got %d", dataPoint.Value)
	}

	value, ok := dataPoint.Attributes.Value(attribute.Key("key"))
	if !ok || value.AsString() != "value" {
		t.Fatalf("expected attribute key=value, got ok=%v value=%v", ok, value)
	}
}

// findMetric locates the named metric among every scope Collect
// returned, failing the test if it isn't found.
func findMetric(
	t *testing.T,
	resourceMetrics metricdata.ResourceMetrics,
	name string,
) metricdata.Metrics {
	t.Helper()

	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == name {
				return metric
			}
		}
	}

	t.Fatalf("metric %q not found in collected data", name)

	return metricdata.Metrics{}
}
