package routing

import (
	"net/http"
)

// Group groups routes and middleware.
type Group struct {
	router *Router

	prefix string

	middlewares []Middleware
}

// Use registers middleware for a group.
func (group *Group) Use(
	middlewares ...Middleware,
) {
	group.middlewares = append(
		group.middlewares,
		middlewares...,
	)
}

// Handle registers a route.
func (group *Group) Handle(
	pattern string,
	handler http.Handler,
) {
	fullPattern := joinPattern(
		group.prefix,
		pattern,
	)

	group.router.register(
		fullPattern,
		handler,
		group.middlewares...,
	)
}

// GET registers a handler for HTTP GET requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("GET "+pattern, handler)
func (group *Group) GET(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodGet+" "+pattern,
		handler,
	)
}

// HEAD registers a handler for HTTP HEAD requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("HEAD "+pattern, handler)
func (group *Group) HEAD(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodHead+" "+pattern,
		handler,
	)
}

// POST registers a handler for HTTP POST requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("POST "+pattern, handler)
func (group *Group) POST(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodPost+" "+pattern,
		handler,
	)
}

// PUT registers a handler for HTTP PUT requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("PUT "+pattern, handler)
func (group *Group) PUT(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodPut+" "+pattern,
		handler,
	)
}

// PATCH registers a handler for HTTP PATCH requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("PATCH "+pattern, handler)
func (group *Group) PATCH(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodPatch+" "+pattern,
		handler,
	)
}

// DELETE registers a handler for HTTP DELETE requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("DELETE "+pattern, handler)
func (group *Group) DELETE(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodDelete+" "+pattern,
		handler,
	)
}

// CONNECT registers a handler for HTTP CONNECT requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("CONNECT "+pattern, handler)
func (group *Group) CONNECT(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodConnect+" "+pattern,
		handler,
	)
}

// OPTIONS registers a handler for HTTP OPTIONS requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("OPTIONS "+pattern, handler)
func (group *Group) OPTIONS(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		http.MethodOptions+" "+pattern,
		handler,
	)
}

// TRACE registers a handler for HTTP TRACE requests.
//
// This is a convenience wrapper around Handle and behaves identically
// to calling:
//
//	group.Handle("TRACE "+pattern, handler)
func (group *Group) TRACE(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
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
//	group.Handle("QUERY "+pattern, handler)
func (group *Group) QUERY(
	pattern string,
	handler http.Handler,
) {
	group.Handle(
		MethodQuery+" "+pattern,
		handler,
	)
}

// Group creates a nested group.
func (group *Group) Group(
	prefix string,
) *Group {
	return &Group{
		router: group.router,
		prefix: joinPath(
			group.prefix,
			prefix,
		),
		middlewares: append(
			[]Middleware{},
			group.middlewares...,
		),
	}
}
