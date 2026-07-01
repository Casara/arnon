package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

// WriteJSON writes a JSON response.
func WriteJSON(
	writer http.ResponseWriter,
	statusCode int,
	value any,
) error {
	writer.Header().Set(
		"Content-Type",
		jsonContentType,
	)

	writer.WriteHeader(statusCode)

	err := json.NewEncoder(writer).Encode(value)
	if err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}
