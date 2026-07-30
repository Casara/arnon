package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/casara/arnon/httpx/middleware"
)

func TestNoCache_SetsNoCacheHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.NoCache()(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	cases := map[string]string{
		"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
		"Expires":         time.Unix(0, 0).UTC().Format(http.TimeFormat),
	}

	for header, want := range cases {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("header %q: expected %q, got %q", header, want, got)
		}
	}
}

func TestNoCache_StripsConditionalRequestHeaders(t *testing.T) {
	t.Parallel()

	conditionalHeaders := []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}

	var seenHeaders http.Header

	handler := middleware.NoCache()(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		seenHeaders = request.Header
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, header := range conditionalHeaders {
		request.Header.Set(header, "some-value")
	}

	handler.ServeHTTP(httptest.NewRecorder(), request)

	for _, header := range conditionalHeaders {
		if got := seenHeaders.Get(header); got != "" {
			t.Errorf(
				"expected header %q to be stripped before reaching the handler, got %q",
				header,
				got,
			)
		}
	}
}
