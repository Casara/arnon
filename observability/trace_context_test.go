package observability_test

import (
	"context"
	"testing"

	"github.com/Casara/arnon/observability"
)

func TestTraceContext_RoundTripsThroughContext(t *testing.T) {
	t.Parallel()

	traceContext := observability.TraceContext{
		TraceID: "trace-abc",
		SpanID:  "span-123",
	}

	ctx := observability.WithTraceContext(context.Background(), traceContext)

	got := observability.GetTraceContext(ctx)

	if got != traceContext {
		t.Fatalf("expected %+v, got %+v", traceContext, got)
	}

	if observability.TraceID(ctx) != "trace-abc" {
		t.Fatalf("expected TraceID %q, got %q", "trace-abc", observability.TraceID(ctx))
	}

	if observability.SpanID(ctx) != "span-123" {
		t.Fatalf("expected SpanID %q, got %q", "span-123", observability.SpanID(ctx))
	}
}

func TestTraceContext_ZeroValueWhenAbsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if got := observability.GetTraceContext(ctx); got != (observability.TraceContext{}) {
		t.Fatalf("expected zero-value TraceContext, got %+v", got)
	}

	if observability.TraceID(ctx) != "" {
		t.Fatalf("expected empty TraceID, got %q", observability.TraceID(ctx))
	}

	if observability.SpanID(ctx) != "" {
		t.Fatalf("expected empty SpanID, got %q", observability.SpanID(ctx))
	}
}
