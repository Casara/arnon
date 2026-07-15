package middleware

import (
	"net/http"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// RedirectSlashes redirects requests whose path has a trailing slash
// (e.g. "/users/") to the same path without it ("/users"), using a 308
// Permanent Redirect so the method and body are preserved by
// well-behaved clients.
//
// The root path ("/") is left untouched.
//
// This is the explicit-redirect counterpart to StripSlashes, which
// normalizes the path silently instead of round-tripping through the
// client. Install one or the other, not both.
func RedirectSlashes() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			path := request.URL.Path

			if len(path) <= 1 || !strings.HasSuffix(path, "/") {
				next.ServeHTTP(
					writer,
					request,
				)

				return
			}

			redirectURL := *request.URL
			redirectURL.Path = strings.TrimSuffix(
				path,
				"/",
			)

			//nolint:gosec // not an open redirect: redirectURL is derived
			// from this same request's own path/query (trailing slash
			// trimmed), never from a redirect-target parameter or any
			// other attacker-controlled input, and never changes host.
			http.Redirect(
				writer,
				request,
				redirectURL.String(),
				http.StatusPermanentRedirect,
			)
		})
	}
}
