package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Casara/arnon/httpx/routing"
)

// TestGroup_HTTPMethodsRegisterRoutes covers every convenience method
// on Group (GET, POST, ..., QUERY) that splitPattern's method
// whitelist accepts. Each one is a thin wrapper around Handle that
// prefixes the pattern with a fixed HTTP method, so a single
// table-driven test asserting the method actually reaches the
// registered handler is enough to catch a copy-paste mistake in any
// of them (e.g. PUT wired to the wrong constant).
func TestGroup_HTTPMethodsRegisterRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		method string
		mount  func(group *routing.Group, path string, handler http.Handler)
	}{
		{"GET", http.MethodGet, (*routing.Group).GET},
		{"HEAD", http.MethodHead, (*routing.Group).HEAD},
		{"POST", http.MethodPost, (*routing.Group).POST},
		{"PUT", http.MethodPut, (*routing.Group).PUT},
		{"PATCH", http.MethodPatch, (*routing.Group).PATCH},
		{"DELETE", http.MethodDelete, (*routing.Group).DELETE},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := routing.NewRouter()
			group := router.Group("/api")

			var called bool

			testCase.mount(group, "/widgets", http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				called = true

				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				httptest.NewRequest(testCase.method, "/api/widgets", nil),
			)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"expected status %d for %s, got %d",
					http.StatusOK,
					testCase.method,
					recorder.Code,
				)
			}

			if !called {
				t.Errorf("expected the %s handler to be invoked", testCase.method)
			}
		})
	}
}

// TestGroup_OPTIONSRegistersRoute exercises Group.OPTIONS separately
// from the table above: an OPTIONS request without
// Access-Control-Request-Method is not a CORS preflight, so it always
// reaches the mux and, in turn, the registered handler directly (see
// CLAUDE.md on why CORS only special-cases real preflights).
func TestGroup_OPTIONSRegistersRoute(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()
	group := router.Group("/api")

	var called bool

	group.OPTIONS("/widgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		called = true

		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodOptions, "/api/widgets", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !called {
		t.Error("expected the OPTIONS handler to be invoked")
	}
}

// TestGroup_TraceConnectAndQueryRegisterRoutes confirms Group.TRACE,
// Group.CONNECT and Group.QUERY register routes successfully, on par
// with the same three methods on Router (see
// TestRouter_ConnectTraceAndQueryRegisterRoutes) - splitPattern's
// method whitelist previously omitted CONNECT/TRACE/QUERY, which made
// Group.Handle (always routed through splitPattern via joinPattern)
// panic unconditionally for these three. Fixed in pattern.go.
func TestGroup_TraceConnectAndQueryRegisterRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		method string
		mount  func(group *routing.Group, path string, handler http.Handler)
	}{
		{"TRACE", http.MethodTrace, (*routing.Group).TRACE},
		{"CONNECT", http.MethodConnect, (*routing.Group).CONNECT},
		{"QUERY", routing.MethodQuery, (*routing.Group).QUERY},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := routing.NewRouter()
			group := router.Group("/api")

			var called bool

			testCase.mount(group, "/widgets", http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				called = true

				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				httptest.NewRequest(testCase.method, "/api/widgets", nil),
			)

			if recorder.Code != http.StatusOK {
				t.Fatalf(
					"expected status %d for %s, got %d",
					http.StatusOK,
					testCase.name,
					recorder.Code,
				)
			}

			if !called {
				t.Fatalf("expected handler to be called for %s", testCase.name)
			}
		})
	}
}

// TestGroup_Group builds a nested group and confirms the prefix is
// joined correctly and middleware registered on the parent group
// still runs for routes registered on the child - Group.Group copies
// (rather than shares the backing array of) the parent's middleware
// slice, so this also guards against the child mutating the parent.
func TestGroup_Group(t *testing.T) {
	t.Parallel()

	var parentMiddlewareCalls int

	router := routing.NewRouter()

	parent := router.Group("/api")
	parent.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			parentMiddlewareCalls++

			next.ServeHTTP(writer, request)
		})
	})

	child := parent.Group("/v1")

	var handlerCalled bool

	child.GET("/widgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		handlerCalled = true

		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/widgets", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !handlerCalled {
		t.Error("expected the nested group's handler to be invoked")
	}

	if parentMiddlewareCalls != 1 {
		t.Errorf(
			"expected the parent group's middleware to run once for the nested route, got %d",
			parentMiddlewareCalls,
		)
	}

	// Adding middleware to the child after Group() must not affect the
	// parent - Group.Group copies the middleware slice rather than
	// aliasing it.
	var childOnlyMiddlewareCalls int

	child.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			childOnlyMiddlewareCalls++

			next.ServeHTTP(writer, request)
		})
	})

	parent.GET("/health", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/health", nil),
	)

	if childOnlyMiddlewareCalls != 0 {
		t.Errorf(
			"expected child-only middleware not to run for a parent-group route, got %d calls",
			childOnlyMiddlewareCalls,
		)
	}
}
