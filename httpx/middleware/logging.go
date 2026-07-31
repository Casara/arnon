package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/observability"
)

// Logging logs HTTP requests using a structured logger.
//
// The middleware creates a request-scoped logger enriched with request
// metadata such as the HTTP method, path, request identifier and trace
// information when available.
//
// If RequestID or OpenTelemetry middlewares are not installed, the
// corresponding attributes are simply omitted from the log entry.
func Logging(
	logger *slog.Logger,
) routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return routing.Wrap(next, http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			start := time.Now()

			requestLogger := newRequestLogger(logger, request)

			ctx := observability.WithLogger(
				request.Context(),
				requestLogger,
			)

			responseRecorder := newResponseWriter(writer)

			next.ServeHTTP(
				responseRecorder,
				request.WithContext(ctx),
			)

			attrs := []any{
				slog.Int(
					"status_code",
					responseRecorder.statusCode,
				),

				slog.Duration(
					"duration",
					time.Since(start),
				),
			}

			if realIP := RealIPFromContext(ctx); realIP != "" {
				attrs = append(
					attrs,

					slog.String(
						"real_ip",
						realIP,
					),
				)
			}

			requestLogger.Info(
				"http request",
				attrs...,
			)
		}))
	}
}

// newRequestLogger creates a logger enriched with request metadata.
//
// Optional attributes such as request_id, trace_id and span_id are only
// included when the corresponding middleware has populated them in the
// request context.
func newRequestLogger(
	logger *slog.Logger,
	request *http.Request,
) *slog.Logger {
	attrs := []any{
		slog.String(
			"method",
			request.Method,
		),

		slog.String(
			"path",
			request.URL.Path,
		),
	}

	if requestID := RequestIDFromContext(
		request.Context(),
	); requestID != "" {
		attrs = append(
			attrs,

			slog.String(
				"request_id",
				requestID,
			),
		)
	}

	if traceID := observability.TraceID(
		request.Context(),
	); traceID != "" {
		attrs = append(
			attrs,

			slog.String(
				"trace_id",
				traceID,
			),

			slog.String(
				"span_id",
				observability.SpanID(
					request.Context(),
				),
			),
		)
	}

	return logger.With(
		attrs...,
	)
}
