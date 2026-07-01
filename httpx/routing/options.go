package routing

import (
	"net/http"

	"github.com/Casara/arnon/openapi"
)

// Option configures a Router during construction.
type Option func(*Router)

// WithOpenAPI sets the router OpenAPI registry.
func WithOpenAPI(
	registry *openapi.Registry,
) Option {
	return func(
		router *Router,
	) {
		router.openAPIRegistry = registry
	}
}

// WithInstrumentation sets the router instrumentHandler.
func WithInstrumentation(
	instrument func(
		http.Handler,
		string,
	) http.Handler,
) Option {
	return func(
		router *Router,
	) {
		router.instrumentHandler = instrument
	}
}
