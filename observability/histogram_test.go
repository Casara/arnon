package observability_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/casara/arnon/observability"
)

// TestHistogram_RecordRecordsValueAndAttributes has the same
// must-not-be-parallel constraint as
// TestCounter_AddRecordsValueAndAttributes, for the same reason:
// otel.SetMeterProvider is process-wide global state.
//
//nolint:paralleltest // mutates the process-wide otel global meter provider, see counter_test.go
func TestHistogram_RecordRecordsValueAndAttributes(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	previous := otel.GetMeterProvider()

	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	histogram, err := observability.NewHistogram(
		"observability_test_histogram",
		"a histogram created by a test",
	)
	if err != nil {
		t.Fatalf("NewHistogram failed: %v", err)
	}

	ctx := context.Background()

	histogram.Record(ctx, 1.5, observability.String("key", "value"))
	histogram.Record(ctx, 2.5, observability.String("key", "value"))

	var collected metricdata.ResourceMetrics

	err = reader.Collect(ctx, &collected)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	metric := findMetric(t, collected, "observability_test_histogram")

	hist, ok := metric.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected metricdata.Histogram[float64], got %T", metric.Data)
	}

	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected exactly one data point, got %d", len(hist.DataPoints))
	}

	dataPoint := hist.DataPoints[0]

	if dataPoint.Count != 2 {
		t.Fatalf("expected 2 recorded observations, got %d", dataPoint.Count)
	}

	if dataPoint.Sum != 4 {
		t.Fatalf("expected sum 4 (1.5 + 2.5), got %v", dataPoint.Sum)
	}

	value, ok := dataPoint.Attributes.Value(attribute.Key("key"))
	if !ok || value.AsString() != "value" {
		t.Fatalf("expected attribute key=value, got ok=%v value=%v", ok, value)
	}
}
