// Package routing provides the HTTP router.
//
// Router wraps net/http.ServeMux, so route patterns are the standard library's
// own ("/users/{id}", read back with request.PathValue) and Router itself is
// an http.Handler that goes straight into an http.Server. What it adds over
// the bare mux is middleware composition, route groups, and registration of
// OpenAPI operations contributed by handlers built with httpx.Endpoint.
//
// # Middleware scope
//
// Middleware registered with Router.Use wraps the entire mux, not each route.
// That is what lets pre-routing middleware such as StripSlashes work at all,
// and what makes a 404 still pass through RequestID and Logging. Middleware
// registered with Group.Use is applied per route inside the group, because
// ServeMux has no notion of a prefix to hang it on.
//
// For the recommended global chain, and the ordering constraints between its
// entries, use httpx/middleware.BuildChain rather than assembling one by hand.
//
// # Patterns
//
// A pattern is "METHOD /path". The verb methods (GET, POST, ...) are thin
// wrappers over Handle that prepend the method. A malformed pattern panics,
// the same way ServeMux treats one: route registration is startup-time wiring,
// so it is a programming error rather than a runtime condition.
package routing
