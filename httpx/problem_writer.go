package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/casara/arnon/problem"
)

const problemContentType = "application/problem+json; charset=utf-8"

// WriteProblem writes an RFC 7807 / RFC 9457 response.
//
// The problem is encoded into a buffer before any header is written,
// so an encoding failure can still be reported as a proper error
// response instead of corrupting a response whose status line and
// headers were already flushed to the client.
//
// If problemInstance.Instance is empty, it is set (in place - avoid
// sharing a single *problem.Problem across concurrent responses) to
// request.URL.Path before encoding, so every problem response is
// traceable back to the request that produced it without every call
// site having to remember to call problemInstance.WithInstance itself.
// This is what RFC 9457 §3.1.6 means by "a URI reference that
// identifies the specific occurrence of the problem" - the RFC's own
// non-normative example uses a path exactly like this
// ("/account/12345/msgs/abc"). An explicitly set Instance is never
// overwritten.
func WriteProblem(
	writer http.ResponseWriter,
	request *http.Request,
	problemInstance *problem.Problem,
) {
	if problemInstance.Instance == "" {
		_ = problemInstance.WithInstance(request.URL.Path)
	}

	buffer := &bytes.Buffer{}

	err := json.NewEncoder(buffer).Encode(problemInstance)
	if err != nil {
		http.Error(
			writer,
			fmt.Sprintf(
				"problem encode failure: %v",
				err,
			),
			http.StatusInternalServerError,
		)

		return
	}

	writer.Header().Set(
		"Content-Type",
		problemContentType,
	)

	writer.WriteHeader(
		problemInstance.StatusCode(),
	)

	_, _ = writer.Write(buffer.Bytes())
}
