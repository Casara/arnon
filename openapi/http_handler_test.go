package openapi_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestHandler_ServeHTTPWritesJSONDocument confirms the Handler serves
// the wrapped Document as JSON, with the expected Content-Type.
func TestHandler_ServeHTTPWritesJSONDocument(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodGet,
		"/users",
		openapi.Operation{},
		struct{}{},
		struct{}{},
	)

	document := generator.Generate()

	handler := openapi.NewHandler(&document)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", got)
	}

	var decoded openapi.Document

	err := json.Unmarshal(recorder.Body.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("expected valid JSON body, got error: %v", err)
	}

	if decoded.Info.Title != "Test API" {
		t.Errorf("expected decoded document title %q, got %q", "Test API", decoded.Info.Title)
	}
}

// TestHandler_ServeHTTPWritesInternalServerErrorOnEncodeFailure
// confirms that a Document which cannot be marshaled to JSON (here, a
// schema carrying a NaN minimum - encoding/json rejects NaN/Inf
// floats) makes ServeHTTP fall back to a 500 response instead of
// writing a half-encoded body.
func TestHandler_ServeHTTPWritesInternalServerErrorOnEncodeFailure(t *testing.T) {
	t.Parallel()

	nan := math.NaN()

	document := openapi.Document{
		OpenAPI: openapi.OpenAPIVersion3_2,
		Info:    openapi.Info{Title: "Broken", Version: "1.0.0"},
		Paths:   map[string]openapi.PathItem{},
		Components: openapi.Components{
			Schemas: map[string]openapi.Schema{
				"Broken": {Type: "number", Minimum: &nan},
			},
		},
	}

	handler := openapi.NewHandler(&document)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}
