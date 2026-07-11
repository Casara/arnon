package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Casara/arnon/httpx/middleware"
)

func TestStripSlashes_RemovesTrailingSlash(t *testing.T) {
	t.Parallel()

	var seenPath string

	handler := middleware.StripSlashes()(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		seenPath = request.URL.Path
	}))

	originalRequest := httptest.NewRequest(http.MethodGet, "/users/", nil)

	handler.ServeHTTP(httptest.NewRecorder(), originalRequest)

	if seenPath != "/users" {
		t.Errorf("expected stripped path %q, got %q", "/users", seenPath)
	}

	if originalRequest.URL.Path != "/users/" {
		t.Errorf(
			"expected original request to be left untouched, got path %q",
			originalRequest.URL.Path,
		)
	}
}

func TestStripSlashes_LeavesPathWithoutTrailingSlashUnchanged(t *testing.T) {
	t.Parallel()

	var seenPath string

	handler := middleware.StripSlashes()(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		seenPath = request.URL.Path
	}))

	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/users", nil),
	)

	if seenPath != "/users" {
		t.Errorf("expected unchanged path %q, got %q", "/users", seenPath)
	}
}

func TestStripSlashes_LeavesRootPathUnchanged(t *testing.T) {
	t.Parallel()

	var seenPath string

	handler := middleware.StripSlashes()(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		seenPath = request.URL.Path
	}))

	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if seenPath != "/" {
		t.Errorf("expected root path to stay %q, got %q", "/", seenPath)
	}
}
