package middleware

import (
	"context"
)

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"

	realIPContextKey contextKey = "real_ip"
)

// RequestIDFromContext returns the request ID.
func RequestIDFromContext(
	ctx context.Context,
) string {
	value, ok := ctx.Value(
		requestIDContextKey,
	).(string)
	if !ok {
		return ""
	}

	return value
}

// RealIPFromContext returns client IP.
func RealIPFromContext(
	ctx context.Context,
) string {
	value, ok := ctx.Value(
		realIPContextKey,
	).(string)
	if !ok {
		return ""
	}

	return value
}
