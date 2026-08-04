package routing_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx/routing"
)

// TestGroup_JoinPathVariants exercises joinPath's three branches
// (empty prefix, empty trailing path, and the common "both sides
// present" case) indirectly through Group registration, since
// joinPath itself is unexported. Prefix and path are normalized
// (trailing/leading "/" trimmed) before being joined with a single
// "/", regardless of how either side was written.
func TestGroup_JoinPathVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		groupPath  string
		routePath  string
		wantServed string
	}{
		{"empty prefix", "", "/users", "/users"},
		{"prefix with trailing slash", "/api/", "/users", "/api/users"},
		{"route path is just a slash", "/api", "/", "/api"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := routing.NewRouter()
			group := router.Group(testCase.groupPath)

			var called bool

			group.GET(testCase.routePath, http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				called = true

				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, testCase.wantServed, nil),
			)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"expected %q to be routed to the registered handler, got status %d",
					testCase.wantServed,
					recorder.Code,
				)
			}

			if !called {
				t.Errorf("expected the handler mounted at %q to be invoked", testCase.wantServed)
			}
		})
	}
}

// TestGroup_Group_JoinsNestedPrefixes confirms Group.Group's own use
// of joinPath produces the expected combined prefix for a deeply
// nested group.
func TestGroup_Group_JoinsNestedPrefixes(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()

	nested := router.Group("/api").Group("/v1").Group("/admin")

	var called bool

	nested.GET("/users", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		called = true

		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !called {
		t.Error("expected the handler on the triply-nested group to be invoked")
	}
}

// TestGroup_Handle_MalformedPatternPanics exercises splitPattern's two
// validation error branches (besides the unsupported-method branch
// already covered by TestGroup_TraceAndQueryPanicOnRegistration),
// asserting the specific sentinel error via errors.Is rather than just
// "it panicked".
func TestGroup_Handle_MalformedPatternPanics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		wantErr error
	}{
		{"missing path", "GET", routing.ErrInvalidPattern},
		{"too many fields", "GET /a /b", routing.ErrInvalidPattern},
		{"path without leading slash", "GET users", routing.ErrInvalidPath},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recovered := recoverFromHandle(t, testCase.pattern)

			err, ok := recovered.(error)
			if !ok {
				t.Fatalf("expected panic value to be an error, got %#v", recovered)
			}

			if !errors.Is(err, testCase.wantErr) {
				t.Errorf("expected panic to wrap %v, got %v", testCase.wantErr, err)
			}
		})
	}
}

// recoverFromHandle registers pattern on a fresh Group and returns the
// recovered panic value, failing the test if Handle did not panic.
func recoverFromHandle(t *testing.T, pattern string) any {
	t.Helper()

	var recovered any

	func() {
		defer func() {
			recovered = recover()
		}()

		router := routing.NewRouter()
		group := router.Group("/api")

		group.Handle(pattern, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	}()

	if recovered == nil {
		t.Fatalf("expected Group.Handle(%q, ...) to panic", pattern)
	}

	return recovered
}
