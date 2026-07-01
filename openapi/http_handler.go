package openapi

import (
	"encoding/json"
	"net/http"
)

// Handler exposes the OpenAPI document.
type Handler struct {
	document *Document
}

// NewHandler creates a new OpenAPI handler.
func NewHandler(
	document *Document,
) *Handler {
	return &Handler{
		document: document,
	}
}

// ServeHTTP serves the OpenAPI document.
func (handler *Handler) ServeHTTP(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json",
	)

	err := json.NewEncoder(writer).Encode(handler.document)
	if err != nil {
		http.Error(
			writer,
			"failed to serialize OpenAPI document",
			http.StatusInternalServerError,
		)
	}
}
