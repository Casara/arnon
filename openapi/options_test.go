package openapi_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestNewGenerator_WithServersSetsDocumentServers confirms WithServers
// populates Document.Servers - Servers has no dedicated NewGenerator
// parameter of its own, only this option.
func TestNewGenerator_WithServersSetsDocumentServers(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(
		openapi.Info{Title: "Test API", Version: "1.0.0"},
		openapi.WithServers(
			openapi.Server{URL: "https://api.example.com/v1", Description: "Production"},
			openapi.Server{URL: "http://localhost:8080", Description: "Local"},
		),
	)

	document := generator.Generate()

	if len(document.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d: %+v", len(document.Servers), document.Servers)
	}

	if document.Servers[0].URL != "https://api.example.com/v1" {
		t.Errorf("expected first server URL to be the production one, got %+v", document.Servers[0])
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(string(encoded), "https://api.example.com/v1") {
		t.Errorf("expected servers to be present in the encoded document, got %s", encoded)
	}
}

// TestNewGenerator_NoOptionsLeavesServersAndExternalDocsUnset confirms
// the zero-options case (every existing NewGenerator call site) still
// omits servers/externalDocs entirely, rather than emitting an empty
// array/null.
func TestNewGenerator_NoOptionsLeavesServersAndExternalDocsUnset(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	document := generator.Generate()

	if document.Servers != nil {
		t.Errorf("expected Servers to be nil, got %+v", document.Servers)
	}

	if document.ExternalDocs != nil {
		t.Errorf("expected ExternalDocs to be nil, got %+v", document.ExternalDocs)
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if strings.Contains(string(encoded), "servers") ||
		strings.Contains(string(encoded), "externalDocs") {
		t.Errorf("expected no servers/externalDocs key in the encoded document, got %s", encoded)
	}
}

// TestNewGenerator_WithExternalDocsSetsDocumentExternalDocs confirms
// WithExternalDocs populates Document.ExternalDocs - like Servers, it
// has no dedicated NewGenerator parameter of its own.
func TestNewGenerator_WithExternalDocsSetsDocumentExternalDocs(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(
		openapi.Info{Title: "Test API", Version: "1.0.0"},
		openapi.WithExternalDocs(&openapi.ExternalDocs{
			Description: "Find more info here",
			URL:         "https://example.com",
		}),
	)

	document := generator.Generate()

	if document.ExternalDocs == nil {
		t.Fatalf("expected ExternalDocs to be set")
	}

	if document.ExternalDocs.URL != "https://example.com" {
		t.Errorf("expected ExternalDocs.URL to be set, got %+v", document.ExternalDocs)
	}
}
