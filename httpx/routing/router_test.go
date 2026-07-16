package routing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Casara/arnon/httpx/routing"
)

// stripTrailingSlash is a minimal stand-in for
// httpx/middleware.StripSlashes, kept local to avoid a routing ->
// httpx/middleware import (which .go-arch-lint.yml forbids: middleware
// depends on routing, not the other way around).
func stripTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if len(request.URL.Path) > 1 {
			request.URL.Path = strings.TrimSuffix(request.URL.Path, "/")
		}

		next.ServeHTTP(writer, request)
	})
}

func TestRouter_GlobalMiddlewareRunsBeforeRouteMatching(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()
	router.Use(stripTrailingSlash)

	router.GET("/users", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()

	// Without path rewriting happening before net/http.ServeMux
	// matches the route, "/users/" would 404: the mux only registered
	// the exact pattern "/users".
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/users/", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestRouter_GlobalMiddlewareRunsForUnmatchedRoutes(t *testing.T) {
	t.Parallel()

	var sawUnmatchedRequest bool

	router := routing.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			sawUnmatchedRequest = true

			next.ServeHTTP(writer, request)
		})
	})

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/does-not-exist", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	if !sawUnmatchedRequest {
		t.Error("expected global middleware to run even for a 404")
	}
}

func TestRouter_GroupMiddlewareOnlyAppliesWithinGroup(t *testing.T) {
	t.Parallel()

	var groupMiddlewareCalls int

	router := routing.NewRouter()

	group := router.Group("/api")
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			groupMiddlewareCalls++

			next.ServeHTTP(writer, request)
		})
	})

	noop := http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	})

	group.GET("/users", noop)
	router.GET("/health", noop)

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/health", nil),
	)

	if groupMiddlewareCalls != 0 {
		t.Fatalf(
			"expected group middleware not to run for routes outside the group, got %d calls",
			groupMiddlewareCalls,
		)
	}

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/users", nil),
	)

	if groupMiddlewareCalls != 1 {
		t.Errorf(
			"expected group middleware to run once for a route inside the group, got %d calls",
			groupMiddlewareCalls,
		)
	}
}

func TestRouter_GlobalMiddlewareRunsBeforeGroupMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	router := routing.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			order = append(order, "global")

			next.ServeHTTP(writer, request)
		})
	})

	group := router.Group("/api")
	group.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			order = append(order, "group")

			next.ServeHTTP(writer, request)
		})
	})

	group.GET("/users", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		order = append(order, "handler")

		writer.WriteHeader(http.StatusOK)
	}))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/users", nil),
	)

	want := []string{"global", "group", "handler"}

	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}

	for i, step := range want {
		if order[i] != step {
			t.Errorf("expected order %v, got %v", want, order)

			break
		}
	}
}

// TestRouter_MountsArbitraryContentTypeHandlers proves the Router
// itself has no JSON coupling: httpx.Endpoint is JSON-only by
// design, but Router.Handle/GET/POST/etc accept any http.Handler, so
// a route that needs to return XML, a PDF, or any other content type
// (or file) can be mounted as a plain handler right alongside typed
// JSON endpoints, using the exact same registration methods.
func TestRouter_MountsArbitraryContentTypeHandlers(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()

	router.GET("/report.pdf", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/pdf")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("%PDF-1.4 fake report"))
	}))

	router.GET("/feed.xml", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><feed></feed>`))
	}))

	pdfRecorder := httptest.NewRecorder()
	router.ServeHTTP(pdfRecorder, httptest.NewRequest(http.MethodGet, "/report.pdf", nil))

	if pdfRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d for the PDF route, got %d", http.StatusOK, pdfRecorder.Code)
	}

	if got := pdfRecorder.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("expected Content-Type %q, got %q", "application/pdf", got)
	}

	xmlRecorder := httptest.NewRecorder()
	router.ServeHTTP(xmlRecorder, httptest.NewRequest(http.MethodGet, "/feed.xml", nil))

	if xmlRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d for the XML route, got %d", http.StatusOK, xmlRecorder.Code)
	}

	if got := xmlRecorder.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "application/xml; charset=utf-8", got)
	}
}
