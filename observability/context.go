package observability

import "context"

type traceContextKey struct{}

// WithTraceContext stores trace information in the context.
func WithTraceContext(
	ctx context.Context,
	traceContext TraceContext,
) context.Context {
	return context.WithValue(
		ctx,
		traceContextKey{},
		traceContext,
	)
}

// GetTraceContext returns trace information from the context.
func GetTraceContext(
	ctx context.Context,
) TraceContext {
	traceContext, _ := ctx.Value(
		traceContextKey{},
	).(TraceContext)

	return traceContext
}

// TraceID returns the current trace identifier.
func TraceID(
	ctx context.Context,
) string {
	return GetTraceContext(
		ctx,
	).TraceID
}

// SpanID returns the current span identifier.
func SpanID(
	ctx context.Context,
) string {
	return GetTraceContext(
		ctx,
	).SpanID
}
