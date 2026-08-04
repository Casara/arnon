package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casara/arnon/httpx/routing"
)

// CORSConfig configures CORS. AllowedOrigins is the only field
// without a safe default - an empty/unset list allows no origin at
// all, there's no implicit wildcard.
type CORSConfig struct {
	// AllowedOrigins is checked against the request's Origin header
	// (case-insensitive exact match), or "*" to allow any origin.
	AllowedOrigins []string

	// AllowedMethods defaults to GET, POST, PUT, PATCH, DELETE,
	// OPTIONS when empty.
	AllowedMethods []string

	// AllowedHeaders defaults to Accept, Authorization, Content-Type,
	// Origin when empty.
	AllowedHeaders []string

	// ExposedHeaders sets Access-Control-Expose-Headers when non-empty.
	// Left unset, only the CORS-safelisted response headers are
	// readable by client-side JavaScript.
	ExposedHeaders []string

	// AllowCredentials sets Access-Control-Allow-Credentials: true,
	// permitting cookies/credentials on cross-origin requests.
	AllowCredentials bool

	// MaxAge sets Access-Control-Max-Age (in seconds) on a preflight
	// response, controlling how long the browser may cache it.
	MaxAge time.Duration
}

// CORS creates a CORS middleware.
func CORS(config CORSConfig) routing.Middleware {
	allowedMethods := config.AllowedMethods

	if len(allowedMethods) == 0 {
		allowedMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		}
	}

	allowedHeaders := config.AllowedHeaders

	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Origin",
		}
	}

	maxAge := int(config.MaxAge.Seconds())

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				// The response always depends on the Origin request
				// header (whether Access-Control-Allow-Origin ends up
				// echoing it back or is omitted entirely), so a shared
				// cache needs Vary: Origin to avoid serving a response
				// computed for one origin to a request from another.
				writer.Header().Add("Vary", "Origin")

				origin := request.Header.Get("Origin")

				if origin != "" && isAllowedOrigin(origin, config.AllowedOrigins) {
					writer.Header().Set(
						"Access-Control-Allow-Origin",
						origin,
					)

					if config.AllowCredentials {
						writer.Header().Set(
							"Access-Control-Allow-Credentials",
							"true",
						)
					}
				}

				writer.Header().Set(
					"Access-Control-Allow-Methods",
					strings.Join(
						allowedMethods,
						", ",
					),
				)

				writer.Header().Set(
					"Access-Control-Allow-Headers",
					strings.Join(
						allowedHeaders,
						", ",
					),
				)

				if len(config.ExposedHeaders) > 0 {
					writer.Header().Set(
						"Access-Control-Expose-Headers",
						strings.Join(
							config.ExposedHeaders,
							", ",
						),
					)
				}

				if maxAge > 0 {
					writer.Header().Set(
						"Access-Control-Max-Age",
						strconv.Itoa(maxAge),
					)
				}

				// Only a genuine CORS preflight - OPTIONS carrying
				// Access-Control-Request-Method (Fetch §4.1's "CORS-preflight
				// request") - is answered here. A bare OPTIONS request (no
				// browser doing a preflight, e.g. a client probing
				// capabilities per RFC 9110 §9.3.7) falls through to next
				// instead: if the path has no explicit OPTIONS handler,
				// that reaches net/http.ServeMux's own 405 response, which
				// already carries an Allow header reflecting the methods
				// actually registered for that path - a generic blanket 204
				// here would otherwise mask that (and would also make an
				// explicit OPTIONS handler registered via Router.OPTIONS
				// unreachable, since this middleware runs first).
				if request.Method == http.MethodOptions &&
					request.Header.Get("Access-Control-Request-Method") != "" {
					writer.WriteHeader(http.StatusNoContent)

					return
				}

				next.ServeHTTP(writer, request)
			},
		)
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return true
		}

		if strings.EqualFold(allowedOrigin, origin) {
			return true
		}
	}

	return false
}
