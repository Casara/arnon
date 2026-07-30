package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/casara/arnon/httpx/routing"
)

// SecureHeadersConfig configures SecureHeaders. The zero value is
// safe to use as-is - every field has a working default.
type SecureHeadersConfig struct {
	// FrameOptions sets X-Frame-Options. Defaults to "DENY".
	FrameOptions string

	// HSTSMaxAge, when non-zero, enables Strict-Transport-Security
	// with this max-age. Left at zero (disabled) by default: HSTS
	// tells browsers to refuse plain HTTP to this host for the given
	// duration, which breaks local HTTP development if turned on
	// blindly. Only set this when actually serving over HTTPS.
	HSTSMaxAge time.Duration
}

// SecureHeaders sets a baseline set of response headers that mitigate
// common browser-based attacks: MIME-sniffing (X-Content-Type-Options),
// clickjacking (X-Frame-Options) and, optionally, protocol downgrade
// (Strict-Transport-Security).
//
// Most of this matters more for HTML responses (e.g. the /docs UI)
// than for JSON API responses - a JSON body can't be framed the way a
// page can - but X-Content-Type-Options and HSTS apply regardless of
// content type, and setting all of them is cheap, so this is offered
// as a single opt-in middleware rather than split per concern.
func SecureHeaders(
	config SecureHeadersConfig,
) routing.Middleware {
	frameOptions := config.FrameOptions
	if frameOptions == "" {
		frameOptions = "DENY"
	}

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			header := writer.Header()

			header.Set(
				"X-Content-Type-Options",
				"nosniff",
			)

			header.Set(
				"X-Frame-Options",
				frameOptions,
			)

			header.Set(
				"Referrer-Policy",
				"no-referrer",
			)

			if config.HSTSMaxAge > 0 {
				header.Set(
					"Strict-Transport-Security",
					fmt.Sprintf(
						"max-age=%d; includeSubDomains",
						int(config.HSTSMaxAge.Seconds()),
					),
				)
			}

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
