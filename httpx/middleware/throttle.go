package middleware

import (
	"net/http"
	"time"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/problem"
)

// ThrottleConfig configures the Throttle middleware.
type ThrottleConfig struct {
	// Limit is the maximum number of requests processed concurrently.
	// Must be positive.
	Limit int

	// BacklogLimit is the maximum number of requests allowed to wait
	// for a free slot once Limit is reached, instead of being
	// rejected immediately. Zero (the default) means no backlog:
	// requests are rejected as soon as Limit is reached.
	BacklogLimit int

	// BacklogTimeout is how long a queued request waits for a slot
	// before being rejected. Ignored when BacklogLimit is zero.
	BacklogTimeout time.Duration
}

// Throttle limits the number of requests processed concurrently across
// all clients, using ThrottleConfig.Limit concurrent slots and an
// optional bounded backlog for requests that arrive while every slot
// is taken.
//
// This limits concurrency, not rate: it caps how many requests may be
// in flight at once, not how many requests per second a client may
// make. For that, see RateLimit.
func Throttle(
	config ThrottleConfig,
) routing.Middleware {
	active := make(chan struct{}, config.Limit)

	var backlog chan struct{}
	if config.BacklogLimit > 0 {
		backlog = make(chan struct{}, config.BacklogLimit)
	}

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			select {
			case active <- struct{}{}:
				defer func() { <-active }()

				next.ServeHTTP(
					writer,
					request,
				)

				return

			default:
			}

			if backlog == nil {
				writeThrottled(writer)

				return
			}

			select {
			case backlog <- struct{}{}:
				defer func() { <-backlog }()

			default:
				writeThrottled(writer)

				return
			}

			timer := time.NewTimer(config.BacklogTimeout)
			defer timer.Stop()

			select {
			case active <- struct{}{}:
				defer func() { <-active }()

				next.ServeHTTP(
					writer,
					request,
				)

			case <-timer.C:
				writeThrottled(writer)

			case <-request.Context().Done():
				// Client is gone; nothing left to respond to.
			}
		})
	}
}

func writeThrottled(
	writer http.ResponseWriter,
) {
	httpx.WriteProblem(
		writer,
		problem.NewTooManyRequests(
			"the server is at capacity, try again later",
		),
	)
}
