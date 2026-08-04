package routing

import (
	"net/http"
	"slices"
)

// Chain composes middlewares.
func Chain(
	handler http.Handler,
	middlewares ...Middleware,
) http.Handler {
	if len(middlewares) == 0 {
		return handler
	}

	wrappedHandler := handler

	for _, middleware := range slices.Backward(middlewares) {
		wrappedHandler = middleware(wrappedHandler)
	}

	return wrappedHandler
}
