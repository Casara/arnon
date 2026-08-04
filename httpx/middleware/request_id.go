package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/casara/arnon/httpx/routing"
)

const requestIDHeader = "X-Request-Id"

// RequestID adds request ID to context.
func RequestID() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return routing.Wrap(next, http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			requestID := request.Header.Get(
				requestIDHeader,
			)

			if requestID == "" {
				requestID = uuid.NewString()
			}

			writer.Header().Set(
				requestIDHeader,
				requestID,
			)

			ctx := context.WithValue(
				request.Context(),
				requestIDContextKey,
				requestID,
			)

			next.ServeHTTP(
				writer,
				request.WithContext(ctx),
			)
		}))
	}
}
