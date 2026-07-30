package middleware

import (
	"net/http"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/observability"
	"github.com/casara/arnon/problem"
)

// Recover handles panics.
//
// The recovered value is logged server-side only, through the
// request-scoped logger (observability.LoggerFromContext, falling
// back to slog.Default() if none is set) - never included in the
// response. RFC 9457 §3.1.5 warns that the "detail" member can carry
// sensitive information, and a raw panic value in Go often does (an
// exact variable value, a nil pointer dereference message, sometimes
// a file path), so problem.NewInternal("") is used instead, which
// falls back to its own generic, safe detail.
//
// Install this after RequestID/Logging (i.e. so it sits closer to the
// actual handler) if request_id/trace_id should be attached to the
// logged panic - middleware installed outside Recover in the chain
// enriches its own downstream request's context, not the *http.Request
// value Recover itself is holding, so an outermost Recover only ever
// sees the context the request started with.
func Recover() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			ctx := request.Context()

			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				observability.LoggerFromContext(ctx).Error(
					"recovered from panic",
					"panic", recovered,
				)

				httpx.WriteProblem(
					writer,
					request,
					problem.NewInternal(""),
				)
			}()

			next.ServeHTTP(
				writer,
				request,
			)
		})
	}
}
