package middleware

import "net/http"

type responseWriter struct {
	http.ResponseWriter

	statusCode int
}

func newResponseWriter(
	writer http.ResponseWriter,
) *responseWriter {
	return &responseWriter{
		ResponseWriter: writer,
		statusCode:     http.StatusOK,
	}
}

func (writer *responseWriter) WriteHeader(
	statusCode int,
) {
	writer.statusCode = statusCode

	writer.ResponseWriter.WriteHeader(
		statusCode,
	)
}

func (writer *responseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}
