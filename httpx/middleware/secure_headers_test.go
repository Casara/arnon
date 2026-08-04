package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/casara/arnon/httpx/middleware"
)

func TestSecureHeaders_SetsDefaultHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.SecureHeaders(middleware.SecureHeadersConfig{})(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	cases := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}

	for header, want := range cases {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("header %q: expected %q, got %q", header, want, got)
		}
	}

	if got := recorder.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("expected no Strict-Transport-Security by default, got %q", got)
	}
}

func TestSecureHeaders_EnablesHSTSWhenConfigured(t *testing.T) {
	t.Parallel()

	handler := middleware.SecureHeaders(middleware.SecureHeadersConfig{
		HSTSMaxAge: 24 * time.Hour,
	})(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	want := "max-age=86400; includeSubDomains"

	if got := recorder.Header().Get("Strict-Transport-Security"); got != want {
		t.Errorf("expected Strict-Transport-Security %q, got %q", want, got)
	}
}

func TestSecureHeaders_UsesCustomFrameOptions(t *testing.T) {
	t.Parallel()

	handler := middleware.SecureHeaders(middleware.SecureHeadersConfig{
		FrameOptions: "SAMEORIGIN",
	})(newNoopHandler())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := recorder.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("expected X-Frame-Options %q, got %q", "SAMEORIGIN", got)
	}
}
