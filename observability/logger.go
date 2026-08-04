package observability

import (
	"context"
	"log/slog"
)

type loggerContextKey struct{}

// WithLogger stores a logger in the context.
//
// This is typically used by middleware to associate a request-scoped
// logger with the current request. The stored logger may already contain
// contextual attributes such as request identifiers and tracing
// information.
func WithLogger(
	ctx context.Context,
	logger *slog.Logger,
) context.Context {
	return context.WithValue(
		ctx,
		loggerContextKey{},
		logger,
	)
}

// LoggerFromContext returns the logger associated with the current
// request context.
//
// The returned logger may contain request-scoped attributes added by
// middleware, such as request identifiers, trace identifiers and other
// correlation data.
//
// This helper allows components to obtain the current logger without
// passing *slog.Logger through every function call.
//
// If no logger is present in the context, slog.Default() is returned.
func LoggerFromContext(
	ctx context.Context,
) *slog.Logger {
	logger, ok := ctx.Value(
		loggerContextKey{},
	).(*slog.Logger)

	if !ok || logger == nil {
		return slog.Default()
	}

	return logger
}
