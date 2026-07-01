package middleware

import (
	"fmt"
	"net/http"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/problem"
)

// Recover handles panics.
func Recover() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				httpx.WriteProblem(
					writer,
					problem.New(
						http.StatusInternalServerError,
						"Internal Server Error",
						fmt.Sprintf(
							"%v",
							recovered,
						),
					),
				)
			}()

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
