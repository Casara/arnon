package middleware

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/problem"
)

// AllowContentType restricts accepted request Content-Type values to
// an explicit allow-list (e.g. "application/json"), rejecting
// anything else with a 415 Problem Details response.
//
// Requests without a Content-Type header are let through unchecked:
// binding already tolerates a missing body, so there is nothing to
// validate here. A Content-Type is compared ignoring parameters (e.g.
// "application/json; charset=utf-8" matches "application/json").
func AllowContentType(
	contentTypes ...string,
) routing.Middleware {
	allowed := make(
		map[string]struct{},
		len(contentTypes),
	)

	for _, contentType := range contentTypes {
		allowed[contentType] = struct{}{}
	}

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			rawContentType := request.Header.Get("Content-Type")

			if rawContentType == "" {
				next.ServeHTTP(
					writer,
					request,
				)

				return
			}

			mediaType, _, err := mime.ParseMediaType(
				rawContentType,
			)
			if err != nil {
				httpx.WriteProblem(
					writer,
					request,
					problem.New(
						http.StatusBadRequest,
						"Malformed Content-Type",
						"Request Content-Type header could not be parsed",
					),
				)

				return
			}

			if _, ok := allowed[mediaType]; !ok {
				httpx.WriteProblem(
					writer,
					request,
					problem.New(
						http.StatusUnsupportedMediaType,
						"Unsupported Media Type",
						fmt.Sprintf(
							"Content-Type %q is not supported",
							mediaType,
						),
					),
				)

				return
			}

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
