package patch

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/problem"
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
// If the internal GET's response carries an ETag/Last-Modified, it is
// copied onto the internal PUT's If-Match/If-Unmodified-Since -
// mirroring huma's own autopatch package. This is inert today: arnon
// has no If-Match/optimistic-concurrency mechanism yet (see
// CLAUDE.md's 428/If-Match note), so nothing currently rejects a PUT
// whose precondition fails, and a lost update between the internal
// GET and PUT is possible under concurrent PATCHes to the same
// resource. Propagating the headers now means this won't need
// rework once that mechanism exists.
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
