package otel

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/casara/arnon/observability"
)

// NewHandler wraps an HTTP handler with OpenTelemetry instrumentation.
//
// Besides creating and propagating HTTP spans through otelhttp, this
// adapter exposes trace information through the observability package so that
// other framework components can access trace identifiers without
// directly depending on OpenTelemetry APIs.
//
// This helps keep OpenTelemetry-specific concerns isolated within the
// otel package.
func NewHandler(
	handler http.Handler,
	spanName string,
) http.Handler {
	traceHandler := &traceContextHandler{
		next: handler,
	}

	return otelhttp.NewHandler(
		traceHandler,
		spanName,
		otelhttp.WithTracerProvider(
			otel.GetTracerProvider(),
		),
	)
}

// traceContextHandler bridges OpenTelemetry tracing information into the
// observability package.
//
// arnon intentionally exposes tracing information through
// the observability package rather than OpenTelemetry APIs. This handler extracts
// identifiers from the active span and makes them available through the
// observability context helpers.
type traceContextHandler struct {
	next http.Handler
}

func (handler *traceContextHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	spanContext := trace.SpanFromContext(
		request.Context(),
	).SpanContext()

	if spanContext.IsValid() {
		request = request.WithContext(
			observability.WithTraceContext(
				request.Context(),
				observability.TraceContext{
					TraceID: spanContext.TraceID().String(),
					SpanID:  spanContext.SpanID().String(),
				},
			),
		)
	}

	handler.next.ServeHTTP(
		writer,
		request,
	)
}
