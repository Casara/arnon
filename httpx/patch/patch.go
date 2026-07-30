package patch

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/precondition"
	"github.com/casara/arnon/problem"
)

// Config configures From.
type Config struct {
	// OnApplyError builds the problem response written when the
	// incoming patch document is malformed or fails to apply.
	// Defaults to a 400 Bad Request including err's own message -
	// unlike middleware.RateLimitConfig.OnCounterError (which
	// deliberately hides an internal storage error), this error
	// describes a problem with the client's own request body, the
	// same category binding/validation errors already fall into.
	OnApplyError func(err error) *problem.Problem
}

// From derives a PATCH http.Handler from an existing GET and PUT
// handler for the same resource: RFC 7386 (JSON Merge Patch) and
// RFC 6902 (JSON Patch) are both supported, selected by the incoming
// request's Content-Type (see apply in format.go for the exact
// rules).
//
// It works by request replay, not by teaching get/put anything about
// patching: an internal GET fetches the resource's current JSON
// representation, the patch is applied to those raw bytes, and the
// result is replayed as an internal PUT - reusing that handler's
// existing binding/sanitize/validation entirely unchanged. get and
// put are called directly, never through a Router, so whatever
// middleware should apply to them (auth, logging, ETag, ...) must
// already wrap get/put themselves, or wrap the registered PATCH route
// with the same Group.Use stack as the real GET/PUT routes.
//
// The original incoming request's own If-Match/If-Unmodified-Since (a
// client's optimistic-concurrency intent, from an earlier GET) is
// checked against the internal GET's ETag/Last-Modified via
// httpx/precondition - a mismatch is a 412 Precondition Failed, and
// put is never called. If the internal GET's response carries an
// ETag/Last-Modified, it is separately copied onto the internal PUT's
// If-Match/If-Unmodified-Since too - mirroring huma's own autopatch
// package. That second propagation only closes the internal GET-to-PUT
// race if put itself calls precondition.Check/CheckRequest with its
// own atomically-read current state (From has no way to do that part
// for it - see httpx/precondition's package doc for why).
func From(
	get http.Handler,
	put http.Handler,
	config Config,
) http.Handler {
	onApplyError := config.OnApplyError
	if onApplyError == nil {
		onApplyError = defaultOnApplyError
	}

	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		getRequest := request.Clone(request.Context())
		getRequest.Method = http.MethodGet
		getRequest.Body = http.NoBody
		getRequest.ContentLength = 0

		getResponse := newBufferedResponseWriter()

		get.ServeHTTP(getResponse, getRequest)

		if !getResponse.isSuccess() {
			getResponse.copyTo(writer)

			return
		}

		problemInstance := precondition.CheckRequest(
			request,
			getResponse.header.Get("ETag"),
			parseLastModified(getResponse.header.Get("Last-Modified")),
			precondition.Config{Require: false},
		)
		if problemInstance != nil {
			httpx.WriteProblem(writer, request, problemInstance)

			return
		}

		patchBody, err := io.ReadAll(request.Body)
		if err != nil {
			httpx.WriteProblem(
				writer,
				request,
				problem.NewBadRequest(
					fmt.Sprintf("failed to read request body: %v", err),
				),
			)

			return
		}

		merged, err := apply(
			request.Header.Get("Content-Type"),
			getResponse.buffer.Bytes(),
			patchBody,
		)
		if err != nil {
			writeApplyError(writer, request, onApplyError, err)

			return
		}

		putRequest := request.Clone(request.Context())
		putRequest.Method = http.MethodPut
		putRequest.Body = io.NopCloser(bytes.NewReader(merged))
		putRequest.ContentLength = int64(len(merged))
		putRequest.Header.Set("Content-Type", "application/json; charset=utf-8")

		if etag := getResponse.header.Get("ETag"); etag != "" {
			putRequest.Header.Set("If-Match", etag)
		}

		if lastModified := getResponse.header.Get("Last-Modified"); lastModified != "" {
			putRequest.Header.Set("If-Unmodified-Since", lastModified)
		}

		put.ServeHTTP(writer, putRequest)
	})
}

// writeApplyError reports a failure to apply the incoming patch
// document. An unsupported Content-Type is always a plain 415 - it's
// unambiguous and not something OnApplyError needs to customize;
// everything else (a malformed or inapplicable patch document) goes
// through OnApplyError, since that failure is about the specific
// shape of the client's own patch body.
func writeApplyError(
	writer http.ResponseWriter,
	request *http.Request,
	onApplyError func(error) *problem.Problem,
	err error,
) {
	if errors.Is(err, ErrUnsupportedContentType) {
		httpx.WriteProblem(
			writer,
			request,
			problem.New(
				http.StatusUnsupportedMediaType,
				http.StatusText(http.StatusUnsupportedMediaType),
				err.Error(),
			),
		)

		return
	}

	httpx.WriteProblem(writer, request, onApplyError(err))
}

func defaultOnApplyError(err error) *problem.Problem {
	return problem.NewBadRequest(err.Error())
}

// parseLastModified parses header as an HTTP-date (RFC 9110 §5.6.7),
// returning the zero time.Time for an absent or unparseable value -
// precondition.Check/CheckRequest already treat a zero lastModified as
// unverifiable and ignore it, exactly the right behavior here too.
func parseLastModified(header string) time.Time {
	parsed, err := http.ParseTime(header)
	if err != nil {
		return time.Time{}
	}

	return parsed
}
