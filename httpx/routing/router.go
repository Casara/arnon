package routing

import "net/http"

// MethodQuery represents a custom QUERY HTTP method.
//
// QUERY is not defined by net/http but is supported by the router
// to allow safe, body-carrying query operations.
const MethodQuery = "QUERY"

// Router wraps ServeMux with middleware support.
type Router struct {
	mux *http.ServeMux

	middlewares []Middleware

	openAPIRegistry OpenAPIRegistrar

	instrumentHandler Instrumenter
}

// NewRouter creates a router.
func NewRouter(
	options ...Option,
) *Router {
	router := &Router{
		mux: http.NewServeMux(),
	}

	for _, option := range options {
		option(router)
	}

	return router
}

// ServeHTTP implements http.Handler.
//
// Global middleware (registered via Use) wraps the mux itself here, so
// it runs before route matching. That is required for middleware that
// needs to influence which route matches - e.g. StripSlashes rewriting
// the path before net/http.ServeMux tries to match it - and it is also
// why unmatched routes (404s) still go through RequestID/Logging/CORS/
// etc. instead of skipping global middleware entirely.
func (router *Router) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	Chain(
		router.mux,
		router.middlewares...,
	).ServeHTTP(
		writer,
		request,
	)
}

// Use registers global middleware.
func (router *Router) Use(
	middlewares ...Middleware,
) {
	router.middlewares = append(
		router.middlewares,
		middlewares...,
	)
}

// Handle registers a route from a "METHOD /path" pattern.
//
// It panics if the pattern is malformed - see ErrInvalidPattern,
// ErrInvalidMethod and ErrInvalidPath. Route registration is startup-time
// wiring, so a bad pattern is a programming error, the same way
// net/http.ServeMux treats one.
func (router *Router) Handle(
	pattern string,
	handler http.Handler,
) {
	router.register(
		pattern,
		handler,
	)
}

// GET registers a handler for HTTP GET requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("GET "+pattern, handler)
func (router *Router) GET(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodGet+" "+pattern,
		handler,
	)
}

// HEAD registers a handler for HTTP HEAD requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("HEAD "+pattern, handler)
func (router *Router) HEAD(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodHead+" "+pattern,
		handler,
	)
}

// POST registers a handler for HTTP POST requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("POST "+pattern, handler)
func (router *Router) POST(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodPost+" "+pattern,
		handler,
	)
}

// PUT registers a handler for HTTP PUT requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("PUT "+pattern, handler)
func (router *Router) PUT(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodPut+" "+pattern,
		handler,
	)
}

// PATCH registers a handler for HTTP PATCH requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("PATCH "+pattern, handler)
func (router *Router) PATCH(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodPatch+" "+pattern,
		handler,
	)
}

// DELETE registers a handler for HTTP DELETE requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("DELETE "+pattern, handler)
func (router *Router) DELETE(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodDelete+" "+pattern,
		handler,
	)
}

// CONNECT registers a handler for HTTP CONNECT requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("CONNECT "+pattern, handler)
func (router *Router) CONNECT(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodConnect+" "+pattern,
		handler,
	)
}

// OPTIONS registers a handler for HTTP OPTIONS requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("OPTIONS "+pattern, handler)
func (router *Router) OPTIONS(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodOptions+" "+pattern,
		handler,
	)
}

// TRACE registers a handler for HTTP TRACE requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("TRACE "+pattern, handler)
func (router *Router) TRACE(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		http.MethodTrace+" "+pattern,
		handler,
	)
}

// QUERY registers a handler for HTTP QUERY requests.
//
// QUERY is an extension method and is not defined by the Go standard
// library. This method is provided for protocols and APIs that support
// QUERY semantics.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	router.Handle("QUERY "+pattern, handler)
func (router *Router) QUERY(
	pattern string,
	handler http.Handler,
) {
	router.Handle(
		MethodQuery+" "+pattern,
		handler,
	)
}

// Group creates a route group.
func (router *Router) Group(
	prefix string,
) *Group {
	return &Group{
		router: router,
		prefix: prefix,
	}
}

func (router *Router) register(
	pattern string,
	handler http.Handler,
	extraMiddlewares ...Middleware,
) {
	router.registerOpenAPI(
		pattern,
		handler,
	)

	// router.middlewares wrap the mux itself in ServeHTTP, not here -
	// only group-scoped middlewares are applied per-route, since
	// net/http.ServeMux has no notion of prefix-scoped middleware.
	wrapped := Chain(
		handler,
		extraMiddlewares...,
	)

	if router.instrumentHandler != nil {
		wrapped = router.instrumentHandler(
			wrapped,
			pattern,
		)
	}

	router.mux.Handle(
		pattern,
		wrapped,
	)
}
