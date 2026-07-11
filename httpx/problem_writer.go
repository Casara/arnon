package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Casara/arnon/problem"
)

const problemContentType = "application/problem+json; charset=utf-8"

// WriteProblem writes an RFC 7807 / RFC 9457 response.
//
// The problem is encoded into a buffer before any header is written,
// so an encoding failure can still be reported as a proper error
// response instead of corrupting a response whose status line and
// headers were already flushed to the client.
func WriteProblem(
	writer http.ResponseWriter,
	problemInstance *problem.Problem,
) {
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
