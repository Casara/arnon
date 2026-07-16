package middleware

import (
	"fmt"
	"net/http"

	"github.com/Casara/arnon/httpx/routing"
)

// ServiceDesc adds a Link response header with rel="service-desc"
// (RFC 8631) pointing to path, letting a client discover the API's
// OpenAPI document (e.g. "/openapi.json") from any response without
// out-of-band documentation or a hardcoded URL - a generic HTTP
// client/tool that already understands Link headers gets this for
// free, no arnon-specific behavior required on the client side.
func ServiceDesc(
	path string,
) routing.Middleware {
	link := fmt.Sprintf(
		`<%s>; rel="service-desc"`,
		path,
	)

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writer.Header().Add(
				"Link",
				link,
			)

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
