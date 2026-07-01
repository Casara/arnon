package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Casara/arnon/problem"
)

const problemContentType = "application/problem+json; charset=utf-8"

// WriteProblem writes an RFC 7807 / RFC 9457 response.
func WriteProblem(
	writer http.ResponseWriter,
	problemInstance *problem.Problem,
) {
	writer.Header().Set(
		"Content-Type",
		problemContentType,
	)

	statusCode := problemInstance.StatusCode()

	writer.WriteHeader(statusCode)

	err := json.NewEncoder(writer).Encode(problemInstance)
	if err != nil {
		http.Error(
			writer,
			fmt.Sprintf(
				"problem encode failure: %v",
				err,
			),
			http.StatusInternalServerError,
		)
	}
}
