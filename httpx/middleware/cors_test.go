package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/httpx/routing"
)

func TestCORS_PreflightRequestGetsNoContentAndHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	})(newNoopHandler())

	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin to be set, got %q", got)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
}

func TestCORS_AlwaysSetsVaryOrigin(t *testing.T) {
	t.Parallel()

	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	})(newNoopHandler())

	cases := []struct {
		name   string
		origin string
	}{
		{"allowed origin", "https://example.com"},
		{"disallowed origin", "https://evil.example"},
		{"no origin header", ""},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("Vary"); got != "Origin" {
				t.Errorf("expected Vary: Origin, got %q", got)
			}
		})
	}
}

// TestCORS_OnlyReflectsAllowedOrigin is the regression test for a gap
// mutation testing found: TestCORS_AlwaysSetsVaryOrigin already had a
// case per origin scenario (allowed/disallowed/no origin) but only
// ever asserted on Vary, never on Access-Control-Allow-Origin itself -
// so isAllowedOrigin's actual allow/deny decision, the one thing this
// middleware exists to get right, had no test that would fail if it
// were ever inverted.
func TestCORS_OnlyReflectsAllowedOrigin(t *testing.T) {
	t.Parallel()

	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	})(newNoopHandler())

	cases := []struct {
		name          string
		origin        string
		wantReflected bool
	}{
		{"allowed origin is reflected back", "https://example.com", true},
		{"disallowed origin gets no header", "https://evil.example", false},
		{"no origin header gets no header", "", false},
		{
			"allowed origin comparison is case-insensitive",
			"HTTPS://EXAMPLE.COM",
			true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if testCase.origin != "" {
				request.Header.Set("Origin", testCase.origin)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			got := recorder.Header().Get("Access-Control-Allow-Origin")

			if testCase.wantReflected && got != testCase.origin {
				t.Errorf(
					"expected Access-Control-Allow-Origin %q, got %q",
					testCase.origin,
					got,
				)
			}

			if !testCase.wantReflected && got != "" {
				t.Errorf(
					"expected no Access-Control-Allow-Origin header, got %q",
					got,
				)
			}
		})
	}
}

// TestCORS_WildcardReflectsRequestOriginRatherThanLiteralAsterisk
// proves AllowedOrigins: []string{"*"} still echoes the request's own
// Origin value back (not a literal "*"), which is what lets it be
// safely combined with AllowCredentials - a literal "*" is rejected
// by browsers whenever Access-Control-Allow-Credentials: true is also
// present (Fetch spec).
func TestCORS_WildcardReflectsRequestOriginRatherThanLiteralAsterisk(t *testing.T) {
	t.Parallel()

	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})(newNoopHandler())

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://any-origin.example")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().
		Get("Access-Control-Allow-Origin"); got != "https://any-origin.example" {
		t.Errorf(
			"expected the request's own origin to be reflected, got %q",
			got,
		)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials: true, got %q", got)
	}
}

func TestCORS_BareOptionsRequestFallsThroughToNext(t *testing.T) {
	t.Parallel()

	nextCalled := false

	handler := middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"*"},
	})(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		nextCalled = true

		writer.WriteHeader(http.StatusOK)
	}))

	// No Access-Control-Request-Method: not a CORS preflight, just a
	// bare OPTIONS request (e.g. a client probing capabilities).
	request := httptest.NewRequest(http.MethodOptions, "/", nil)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if !nextCalled {
		t.Error("expected a bare OPTIONS request to fall through to next")
	}

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestCORS_BareOptionsReflectsRealAllowFromRouter(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()

	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"*"},
	}))

	router.GET("/users", newNoopHandler())
	router.POST("/users", newNoopHandler())

	// Bare OPTIONS on a path registered for GET/POST but not OPTIONS:
	// with CORS no longer swallowing every OPTIONS request, this
	// reaches net/http.ServeMux's own 405 response, which reflects the
	// methods actually registered for /users - not a fixed, path-agnostic
	// list from CORSConfig.
	request := httptest.NewRequest(http.MethodOptions, "/users", nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}

	allow := recorder.Header().Get("Allow")

	allowedMethods := strings.Split(allow, ",")
	for i, method := range allowedMethods {
		allowedMethods[i] = strings.TrimSpace(method)
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
		if !slices.Contains(allowedMethods, method) {
			t.Errorf("expected Allow %q to contain %q", allow, method)
		}
	}
}
