package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// RealIP extracts client IP.
func RealIP() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			ip := request.Header.Get(
				"X-Forwarded-For",
			)

			if ip == "" {
				ip = request.Header.Get(
					"X-Real-IP",
				)
			}

			if ip == "" {
				host, _, err := net.SplitHostPort(
					request.RemoteAddr,
				)
				if err == nil {
					ip = host
				}
			}

			ip = strings.TrimSpace(ip)

			ctx := context.WithValue(
				request.Context(),
				realIPContextKey,
				ip,
			)

			next.ServeHTTP(
				writer,
				request.WithContext(ctx),
			)
		})
	}
}
