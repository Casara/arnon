package middleware

import (
	"net/http"
	"time"

	"github.com/casara/arnon/httpx/routing"
)

// noCacheHeaders are the response headers NoCache sets on every
// request, instructing clients and intermediate caches (proxies,
// CDNs) not to store or reuse the response. Expires is a fixed,
// already-expired value (the Unix epoch), matching what nginx and
// most reverse proxies expect to treat a response as immediately
// stale. Ported from chi's middleware.NoCache, including the
// nginx-specific X-Accel-Expires.
//
//nolint:gochecknoglobals // read-only reference data, not per-request state
var noCacheHeaders = map[string]string{
	"Expires":         time.Unix(0, 0).UTC().Format(http.TimeFormat),
	"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

// conditionalRequestHeaders are stripped from the incoming request by
// NoCache before it reaches the handler. They let a client (or an
// intermediate cache replaying a client's request) ask for a
// conditional/cached response (e.g. "only send the body if it changed
// since ETag X"); honoring them anywhere downstream would contradict
// NoCache's "always fresh" intent, so they never reach the handler in
// the first place. Ported from chi's middleware.NoCache.
var conditionalRequestHeaders = []string{ //nolint:gochecknoglobals // read-only reference data
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// NoCache sets response headers that instruct clients and
// intermediate caches (proxies, CDNs) not to store or reuse the
// response, and strips conditional-request headers from the incoming
// request so no downstream code can short-circuit with a
// conditional/cached response instead.
//
// Useful for endpoints whose responses must always be re-fetched,
// e.g. anything reflecting per-request or authenticated state.
func NoCache() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return routing.Wrap(next, http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			for _, name := range conditionalRequestHeaders {
				request.Header.Del(name)
			}

			header := writer.Header()

			for name, value := range noCacheHeaders {
				header.Set(name, value)
			}

			next.ServeHTTP(
				writer,
				request,
			)
		}))
	}
}
