package otel_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/Casara/arnon/observability"
	arnonotel "github.com/Casara/arnon/observability/otel"
)

// TestNewHandler_PropagatesTraceAndSpanIDIntoContext installs a real
// SDK tracer provider (via otel.SetTracerProvider) so the span
// otelhttp.NewHandler creates has a valid, non-zero SpanContext -
// against the default no-op provider, traceContextHandler would never
// find a valid span to propagate. otel.SetTracerProvider is
// process-wide global state, so this must not run in parallel with
// any other test that also sets it.
//
//nolint:paralleltest // mutates the process-wide otel global tracer provider, see comment above
func TestNewHandler_PropagatesTraceAndSpanIDIntoContext(t *testing.T) {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	previous := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	var gotTraceID, gotSpanID string

	inner := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotTraceID = observability.TraceID(request.Context())
		gotSpanID = observability.SpanID(request.Context())

		writer.WriteHeader(http.StatusOK)
	})

	handler := arnonotel.NewHandler(inner, "test-span")

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if gotTraceID == "" {
		t.Fatalf("expected a non-empty TraceID to reach the inner handler")
	}

	if gotSpanID == "" {
		t.Fatalf("expected a non-empty SpanID to reach the inner handler")
	}
}

func TestNewHandler_DelegatesToInnerHandlerResponse(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	})

	handler := arnonotel.NewHandler(inner, "test-span")

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}
