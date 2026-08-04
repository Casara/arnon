package patch

import (
	"bytes"
	"fmt"
	"net/http"
)

// bufferedResponseWriter captures a handler's response instead of
// sending it to a real client, so its status/headers/body can be
// inspected before deciding what to do next - the same "buffer
// everything, decide at the end" shape already used by
// httpx/middleware's etagResponseWriter/compressResponseWriter, with
// its own copy here since neither is exported.
type bufferedResponseWriter struct {
	header      http.Header
	buffer      bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{
		header: make(http.Header),
	}
}

func (writer *bufferedResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}

	n, err := writer.buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("buffer response body: %w", err)
	}

	return n, nil
}

func (writer *bufferedResponseWriter) WriteHeader(statusCode int) {
	if writer.wroteHeader {
		return
	}

	writer.wroteHeader = true
	writer.statusCode = statusCode
}

// isSuccess reports whether the buffered response is 2xx. A handler
// that never called WriteHeader/Write at all defaults to 200, same as
// net/http itself.
func (writer *bufferedResponseWriter) isSuccess() bool {
	if !writer.wroteHeader {
		return true
	}

	return writer.statusCode >= http.StatusOK && writer.statusCode < http.StatusMultipleChoices
}

// copyTo replays the buffered response onto destination verbatim.
func (writer *bufferedResponseWriter) copyTo(destination http.ResponseWriter) {
	for name, values := range writer.header {
		for _, value := range values {
			destination.Header().Add(name, value)
		}
	}

	statusCode := writer.statusCode
	if !writer.wroteHeader {
		statusCode = http.StatusOK
	}

	destination.WriteHeader(statusCode)

	_, _ = destination.Write(writer.buffer.Bytes())
}
