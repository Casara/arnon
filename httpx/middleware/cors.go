package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CORSConfig configures CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string

	AllowedMethods []string

	AllowedHeaders []string

	ExposedHeaders []string

	AllowCredentials bool

	MaxAge time.Duration
}

// CORS creates a CORS middleware.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
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

				if request.Method == http.MethodOptions {
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
