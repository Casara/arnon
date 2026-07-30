package middleware

import (
	"fmt"
	"net/http"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/problem"
)

// MaxBodyBytes rejects requests whose body exceeds maxBytes.
//
// When the client declares Content-Length up front and it already
// exceeds maxBytes, the request is rejected immediately with a 413
// Problem Details response, before the handler or binding ever runs.
//
// When the length isn't known ahead of time (chunked transfer
// encoding, or a client lying about Content-Length), the body is
// wrapped in http.MaxBytesReader as a second line of defense. In that
// case an oversized body is only discovered while it's being read -
// typically inside JSON decoding - and still ends up as a 413: see
// problem.ValidationErrorCode.StatusOverride, which is what lets a
// binding-layer error (which normally maps to a plain 400) claim the
// correct status for this specific case.
func MaxBodyBytes(
	maxBytes int64,
) routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.ContentLength > maxBytes {
				httpx.WriteProblem(
					writer,
					request,
					problem.New(
						http.StatusRequestEntityTooLarge,
						"Payload Too Large",
						fmt.Sprintf(
							"request body exceeds the maximum allowed size of %d bytes",
							maxBytes,
						),
					),
				)

				return
			}

			if request.Body != nil {
				request.Body = http.MaxBytesReader(
					writer,
					request.Body,
					maxBytes,
				)
			}

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
