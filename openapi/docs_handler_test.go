package openapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casara/arnon/openapi"
)

// TestNewDocsHandler_NilConfigUsesDefaults confirms a nil *DocsConfig
// falls back to defaultDocsConfig entirely (title, OpenAPI URL) and
// that the served page reflects those defaults.
func TestNewDocsHandler_NilConfigUsesDefaults(t *testing.T) {
	t.Parallel()

	handler := openapi.NewDocsHandler(openapi.DocsConfig{})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "text/html; charset=utf-8", got)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, "<title>API Documentation</title>") {
		t.Errorf("expected default title in body, got %s", body)
	}

	if !strings.Contains(body, `apiDescriptionUrl="/openapi.json"`) {
		t.Errorf("expected default OpenAPI URL in body, got %s", body)
	}

	if strings.Contains(body, `<link rel="icon"`) {
		t.Error("expected no favicon link when FaviconURL is unset")
	}

	if strings.Contains(body, `logo=`) {
		t.Error("expected no logo attribute when LogoURL is unset")
	}
}

// TestNewDocsHandler_PartialConfigOverridesOnlySetFields confirms that
// only the non-zero fields of a partially filled DocsConfig override
// the defaults - e.g. setting just Title leaves OpenAPIURL as
// "/openapi.json".
func TestNewDocsHandler_PartialConfigOverridesOnlySetFields(t *testing.T) {
	t.Parallel()

	handler := openapi.NewDocsHandler(openapi.DocsConfig{
		Title: "Custom Docs",
	})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs", nil))

	body := recorder.Body.String()

	if !strings.Contains(body, "<title>Custom Docs</title>") {
		t.Errorf("expected custom title in body, got %s", body)
	}

	if !strings.Contains(body, `apiDescriptionUrl="/openapi.json"`) {
		t.Errorf("expected the default OpenAPI URL to survive a partial config, got %s", body)
	}
}

// TestNewDocsHandler_FullConfigSetsFaviconAndLogo confirms
// FaviconURL, LogoURL and OpenAPIURL are honored in the rendered HTML.
func TestNewDocsHandler_FullConfigSetsFaviconAndLogo(t *testing.T) {
	t.Parallel()

	handler := openapi.NewDocsHandler(openapi.DocsConfig{
		Title:      "Full Docs",
		OpenAPIURL: "/spec.json",
		FaviconURL: "/favicon.ico",
		LogoURL:    "/logo.png",
	})

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs", nil))

	body := recorder.Body.String()

	if !strings.Contains(body, `<link rel="icon" href="/favicon.ico" />`) {
		t.Errorf("expected favicon link in body, got %s", body)
	}

	if !strings.Contains(body, `logo="/logo.png"`) {
		t.Errorf("expected logo attribute in body, got %s", body)
	}

	if !strings.Contains(body, `apiDescriptionUrl="/spec.json"`) {
		t.Errorf("expected custom OpenAPI URL in body, got %s", body)
	}
}
