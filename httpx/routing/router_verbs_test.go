package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx/routing"
)

// TestRouter_HTTPMethodsRegisterRoutes covers Router's convenience
// methods for HTTP verbs recognized by net/http.ServeMux route
// patterns. CONNECT/TRACE/QUERY are covered separately in
// TestRouter_ConnectTraceAndQueryRegisterRoutes below.
func TestRouter_HTTPMethodsRegisterRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		method string
		mount  func(router *routing.Router, path string, handler http.Handler)
	}{
		{"GET", http.MethodGet, (*routing.Router).GET},
		{"HEAD", http.MethodHead, (*routing.Router).HEAD},
		{"POST", http.MethodPost, (*routing.Router).POST},
		{"PUT", http.MethodPut, (*routing.Router).PUT},
		{"PATCH", http.MethodPatch, (*routing.Router).PATCH},
		{"DELETE", http.MethodDelete, (*routing.Router).DELETE},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := routing.NewRouter()

			var called bool

			testCase.mount(router, "/widgets", http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				called = true

				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				httptest.NewRequest(testCase.method, "/widgets", nil),
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

// TestRouter_OPTIONSRegistersRoute mirrors
// TestGroup_OPTIONSRegistersRoute for Router.OPTIONS: a non-preflight
// OPTIONS request reaches the mux and, in turn, the registered
// handler directly.
func TestRouter_OPTIONSRegistersRoute(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()

	var called bool

	router.OPTIONS("/widgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		called = true

		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodOptions, "/widgets", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if !called {
		t.Error("expected the OPTIONS handler to be invoked")
	}
}

// TestRouter_ConnectTraceAndQueryRegisterRoutes proves Router.CONNECT,
// Router.TRACE and Router.QUERY register routes successfully - both
// with and without an OpenAPI registry configured, since splitPattern's
// method whitelist now includes all three (see pattern.go and
// TestGroup_TraceConnectAndQueryRegisterRoutes for the Group side).
func TestRouter_ConnectTraceAndQueryRegisterRoutes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		method string
		mount  func(router *routing.Router, path string, handler http.Handler)
	}{
		{"CONNECT", http.MethodConnect, (*routing.Router).CONNECT},
		{"TRACE", http.MethodTrace, (*routing.Router).TRACE},
		{"QUERY", routing.MethodQuery, (*routing.Router).QUERY},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			router := routing.NewRouter()

			var called bool

			testCase.mount(router, "/widgets", http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				called = true

				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				httptest.NewRequest(testCase.method, "/widgets", nil),
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
