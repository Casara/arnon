package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx/middleware"
)

func newNoopHandler() http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	})
}

func TestRedirectSlashes_RedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	handler := middleware.RedirectSlashes()(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/users/?limit=10", nil),
	)

	if recorder.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusPermanentRedirect, recorder.Code)
	}

	if got := recorder.Header().Get("Location"); got != "/users?limit=10" {
		t.Errorf("expected Location %q, got %q", "/users?limit=10", got)
	}
}

func TestRedirectSlashes_LeavesPathWithoutTrailingSlashUnchanged(t *testing.T) {
	t.Parallel()

	handler := middleware.RedirectSlashes()(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestRedirectSlashes_LeavesRootPathUnchanged(t *testing.T) {
	t.Parallel()

	handler := middleware.RedirectSlashes()(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
