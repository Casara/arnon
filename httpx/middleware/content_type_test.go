package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casara/arnon/httpx/middleware"
)

func TestAllowContentType_AllowsListedType(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowContentType("application/json")(newNoopHandler())

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestAllowContentType_RejectsUnlistedType(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowContentType("application/json")(newNoopHandler())

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("<xml/>"))
	request.Header.Set("Content-Type", "application/xml")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, recorder.Code)
	}
}

func TestAllowContentType_RejectsMalformedContentType(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowContentType("application/json")(newNoopHandler())

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	request.Header.Set("Content-Type", ";;;")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestAllowContentType_AllowsMissingContentType(t *testing.T) {
	t.Parallel()

	handler := middleware.AllowContentType("application/json")(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
