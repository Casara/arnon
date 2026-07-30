package middleware

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/problem"
)

// Timeout aborts a request that takes longer than timeout to handle,
// responding with a 503 Service Unavailable Problem Details response
// instead of letting the handler run indefinitely.
//
// Modeled on http.TimeoutHandler: the handler runs in its own
// goroutine against a private ResponseWriter that only buffers
// writes, so nothing reaches the real http.ResponseWriter until we
// know which side won the race - the handler finishing, or the
// deadline. Unlike http.TimeoutHandler, the timeout response itself
// is Problem Details (via httpx.WriteProblem), not plain text: the
// stdlib helper responds in plain text, which would be the one place
// in the framework that bypasses RFC 9457, the only error format
// arnon otherwise guarantees.
//
// A panic in the handler goroutine is re-raised (panic(recovered)) in
// the goroutine running Timeout itself, instead of being swallowed
// here, so an outer Recover still sees it and produces a proper
// Problem Details 500 with the panic logged. Without a Recover
// anywhere in the chain, this repanic is still caught by net/http's
// own per-connection recover (net/http.conn.serve): the stack trace
// is logged to the server's error log and that connection is closed,
// but the process and every other in-flight connection are
// unaffected - the client just sees the connection drop instead of a
// Problem Details response.
func Timeout(
	timeout time.Duration,
) routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			ctx, cancel := context.WithTimeout(
				request.Context(),
				timeout,
			)
			defer cancel()

			request = request.WithContext(ctx)

			buffered := &timeoutResponseWriter{
				header: make(http.Header),
			}

			done := make(chan struct{})

			panicChan := make(chan any, 1)

			go func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						panicChan <- recovered
					}
				}()

				next.ServeHTTP(buffered, request)

				close(done)
			}()

			select {
			case recovered := <-panicChan:
				panic(recovered)

			case <-done:
				buffered.mu.Lock()
				defer buffered.mu.Unlock()

				maps.Copy(writer.Header(), buffered.header)

				if !buffered.wroteHeader {
					buffered.statusCode = http.StatusOK
				}

				writer.WriteHeader(buffered.statusCode)

				_, _ = writer.Write(buffered.buffer.Bytes())

			case <-ctx.Done():
				buffered.mu.Lock()
				defer buffered.mu.Unlock()

				buffered.err = http.ErrHandlerTimeout

				httpx.WriteProblem(
					writer,
					request,
					problem.NewServiceUnavailable(
						"the request took too long to process",
					),
				)
			}
		})
	}
}

// timeoutResponseWriter buffers a handler's response until Timeout
// knows whether it finished before the deadline. Once the deadline
// wins, err is set and any further Write/WriteHeader call from the
// (still running) handler goroutine becomes a safe no-op instead of
// racing the Problem Details response already sent on the real
// http.ResponseWriter.
type timeoutResponseWriter struct {
	mu sync.Mutex

	header     http.Header
	buffer     bytes.Buffer
	statusCode int

	wroteHeader bool
	err         error
}

func (writer *timeoutResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *timeoutResponseWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.err != nil {
		return 0, writer.err
	}

	if !writer.wroteHeader {
		writer.writeHeaderLocked(http.StatusOK)
	}

	n, err := writer.buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("buffer response body: %w", err)
	}

	return n, nil
}

func (writer *timeoutResponseWriter) WriteHeader(statusCode int) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	writer.writeHeaderLocked(statusCode)
}

func (writer *timeoutResponseWriter) writeHeaderLocked(statusCode int) {
	if writer.err != nil || writer.wroteHeader {
		return
	}

	writer.wroteHeader = true
	writer.statusCode = statusCode
}
