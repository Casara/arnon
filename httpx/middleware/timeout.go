package middleware

import (
	"net/http"
	"time"

	"github.com/Casara/arnon/httpx/routing"
)

// Timeout applies request timeout.
func Timeout(
	timeout time.Duration,
) routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.TimeoutHandler(
			next,
			timeout,
			http.StatusText(
				http.StatusServiceUnavailable,
			),
		)
	}
}
