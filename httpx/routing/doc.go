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
// # Middleware and the generated document
//
// A route is documented when its handler implements OpenAPIProvider. Wrapping
// a handler would normally hide that, since a middleware returns a closure and
// a closure implements nothing - so every middleware in httpx/middleware
// builds its result with Wrap, which keeps the decorated handler reachable.
// The router follows that chain, and a wrapped endpoint stays documented:
//
//	router.GET("/users/{id}", middleware.ETag()(endpoint))            // documented
//	router.GET("/users/{id}", middleware.NoCache()(middleware.ETag()(endpoint))) // documented
//
// Third-party middleware returning a bare http.HandlerFunc still hides what it
// wraps, and the route silently stops appearing in the document. Installing
// per-route middleware on a Group avoids the question entirely, and is the
// more robust option when you do not control the middleware.
//
// # Patterns
//
// A pattern is "METHOD /path". The verb methods (GET, POST, ...) are thin
// wrappers over Handle that prepend the method. A malformed pattern panics,
// the same way ServeMux treats one: route registration is startup-time wiring,
// so it is a programming error rather than a runtime condition.
package routing
