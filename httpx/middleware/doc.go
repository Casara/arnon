// Package middleware provides the framework's HTTP middleware.
//
// Everything here returns a routing.Middleware - a plain
// func(http.Handler) http.Handler - so each one composes with any other
// net/http middleware, in or out of this project.
//
// # Order matters, so don't assemble the chain by hand
//
// Several of these have ordering constraints between them: RequestID has to
// run before Logging for the log line to carry the id, ETag before Compress so
// the validator describes the bytes actually sent, Recover outermost so it
// catches panics from everything else. Nothing at runtime can check that -
// a middleware is just a function, with no identity to inspect.
//
// BuildChain exists for exactly this reason: it takes a ChainConfig of what
// you want enabled and returns the chain already in the right relative order.
// Custom middleware that needs a specific position goes in through
// ChainConfig.Extra, anchored to one of the built-in stages.
//
// # Global versus group
//
// Recover, Timeout, StripSlashes/RedirectSlashes, RealIP, RequestID,
// SecureHeaders, RateLimit, Throttle, ETag, Compress, CORS, ServiceDesc and
// Logging are global: install them with Router.Use. AllowContentType,
// MaxBodyBytes and NoCache are per-route concerns and belong on a Group.
//
// # Deployment note
//
// RealIP reads client-controlled headers and has no trusted-proxy list, so it
// is only safe behind a proxy that overwrites them. Read its documentation
// before putting it in front of RateLimit.
package middleware
