package routing

import "net/http"

// Instrumenter wraps a handler for tracing, given the route pattern it was
// registered under. Named rather than left as a bare func type so the contract
// is visible in the documentation.
type Instrumenter func(handler http.Handler, pattern string) http.Handler

// Option configures a Router during construction.
type Option func(*Router)

// WithOpenAPI makes the router register every route whose handler implements
// OpenAPIProvider - that is, every httpx.Endpoint with EndpointConfig.OpenAPI
// set - into registrar. Pass an *openapi.Generator:
//
//	generator := openapi.NewGenerator(openapi.Info{Title: "API", Version: "1"})
//	router := routing.NewRouter(routing.WithOpenAPI(generator))
//
// Without this option the router serves the same routes and generates no
// document.
func WithOpenAPI(
	registrar OpenAPIRegistrar,
) Option {
	return func(
		router *Router,
	) {
		router.openAPIRegistry = registrar
	}
}

// WithInstrumentation wraps every registered handler with instrument, which
// receives the handler and its route pattern so a span can be named after the
// route rather than the raw URL.
//
// otel.NewHandler from arnon/observability/otel has exactly this shape.
//
// Note that this runs inside the mux's dispatch, below any middleware
// installed with Router.Use. A global middleware therefore executes before the
// span exists - which is why middleware.Logging has to be registered on a
// Group to pick up trace_id.
func WithInstrumentation(
	instrument Instrumenter,
) Option {
	return func(
		router *Router,
	) {
		router.instrumentHandler = instrument
	}
}
