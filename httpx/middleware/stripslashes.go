package middleware

import (
	"net/http"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// StripSlashes removes a trailing slash from the request path before
// routing, so "/users/" is served by the same handler as "/users"
// instead of requiring a redirect or a separate route registration.
//
// The root path ("/") is left untouched.
func StripSlashes() routing.Middleware {
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

			strippedURL := *request.URL
			strippedURL.Path = strings.TrimSuffix(
				path,
				"/",
			)

			strippedRequest := new(http.Request)
			*strippedRequest = *request
			strippedRequest.URL = &strippedURL

			next.ServeHTTP(
				writer,
				strippedRequest,
			)
		})
	}
}
