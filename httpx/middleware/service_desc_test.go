package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Casara/arnon/httpx/middleware"
)

func TestServiceDesc_SetsLinkHeader(t *testing.T) {
	t.Parallel()

	handler := middleware.ServiceDesc("/openapi.json")(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	want := `</openapi.json>; rel="service-desc"`

	if got := recorder.Header().Get("Link"); got != want {
		t.Errorf("expected Link header %q, got %q", want, got)
	}
}

func TestServiceDesc_AddsToExistingLinkHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.ServiceDesc("/openapi.json")(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Add("Link", `</page/2>; rel="next"`)
		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	links := recorder.Header().Values("Link")

	if len(links) != 2 {
		t.Fatalf("expected 2 Link headers, got %d: %v", len(links), links)
	}

	if links[0] != `</openapi.json>; rel="service-desc"` {
		t.Errorf("unexpected first Link header: %q", links[0])
	}

	if links[1] != `</page/2>; rel="next"` {
		t.Errorf("unexpected second Link header: %q", links[1])
	}
}
