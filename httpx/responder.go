package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const jsonContentType = "application/json; charset=utf-8"

// WriteJSON writes a JSON response.
//
// The value is encoded into a buffer before any header is written, so
// an encoding failure can be reported to the caller as a recoverable
// error instead of corrupting a response whose status line and
// headers were already flushed to the client. Once the header is
// written, a failure to flush the buffer to the client (e.g. a
// disconnected client) is no longer reported: the caller has no safe
// way to write a different response at that point.
func WriteJSON(
	writer http.ResponseWriter,
	statusCode int,
	value any,
) error {
	buffer := &bytes.Buffer{}

	err := json.NewEncoder(buffer).Encode(value)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}

	writer.Header().Set(
		"Content-Type",
		jsonContentType,
	)

	writer.WriteHeader(statusCode)

	_, _ = writer.Write(buffer.Bytes())

	return nil
}
