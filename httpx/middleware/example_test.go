package middleware_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/casara/arnon/httpx/middleware"
	"github.com/casara/arnon/httpx/routing"
)

// BuildChain assembles the recommended global chain in a fixed relative order,
// so enabling or disabling entries can't accidentally break constraints like
// "RequestID before Logging" or "ETag before Compress". Only the fields you set
// contribute a middleware.
func ExampleBuildChain() {
	chain := middleware.BuildChain(middleware.ChainConfig{
		Recover:         true,
		Timeout:         5 * time.Second,
		RequestID:       true,
		RealIP:          true,
		ETag:            true,
		ServiceDescPath: "/openapi.json",
		Logger:          slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})

	router := routing.NewRouter()
	router.Use(chain...)
	router.GET("/ping", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		fmt.Fprint(writer, "pong")
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/ping", nil),
	)

	fmt.Println("middlewares:", len(chain))
	fmt.Println("status:", recorder.Code)
	fmt.Println("request id set:", recorder.Header().Get("X-Request-Id") != "")
	fmt.Println("etag set:", recorder.Header().Get("ETag") != "")
	fmt.Println("link:", recorder.Header().Get("Link"))
	// Output:
	// middlewares: 7
	// status: 200
	// request id set: true
	// etag set: true
	// link: </openapi.json>; rel="service-desc"
}

// Custom middleware that must sit at a specific point in the chain goes through
// ChainConfig.Extra, anchored to one of the built-in stages, rather than
// through a second ordering API.
func ExampleBuildChain_extra() {
	brand := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writer.Header().Set("X-Powered-By", "arnon")
			next.ServeHTTP(writer, request)
		})
	}

	chain := middleware.BuildChain(middleware.ChainConfig{
		Recover: true,
		ETag:    true,
		Extra: []middleware.ExtraMiddleware{
			{Middleware: brand, Before: middleware.AnchorETag},
		},
	})

	router := routing.NewRouter()
	router.Use(chain...)
	router.GET("/ping", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		fmt.Fprint(writer, "pong")
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/ping", nil),
	)

	fmt.Println(recorder.Header().Get("X-Powered-By"))
	// Output:
	// arnon
}

// ETag answers a conditional GET with 304 and no body once the client has the
// current validator.
func ExampleETag() {
	handler := middleware.ETag()(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		fmt.Fprint(writer, "hello")
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(
		first,
		httptest.NewRequest(http.MethodGet, "/greeting", nil),
	)

	etag := first.Header().Get("ETag")

	conditional := httptest.NewRequest(http.MethodGet, "/greeting", nil)
	conditional.Header.Set("If-None-Match", etag)

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, conditional)

	fmt.Println(first.Code, first.Body.Len())
	fmt.Println(second.Code, second.Body.Len())
	// Output:
	// 200 5
	// 304 0
}

// CORS only intercepts an OPTIONS request that is a genuine preflight, i.e. one
// carrying Access-Control-Request-Method. A bare OPTIONS falls through to the
// router, which answers with a real Allow header.
func ExampleCORS() {
	router := routing.NewRouter()
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{http.MethodGet},
	}))
	router.GET("/users", http.HandlerFunc(func(
		http.ResponseWriter, *http.Request,
	) {
	}))

	preflight := httptest.NewRequest(http.MethodOptions, "/users", nil)
	preflight.Header.Set("Origin", "https://app.example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, preflight)

	fmt.Println("preflight:", recorder.Code,
		recorder.Header().Get("Access-Control-Allow-Origin"))

	bare := httptest.NewRecorder()
	router.ServeHTTP(
		bare,
		httptest.NewRequest(http.MethodOptions, "/users", nil),
	)

	fmt.Println("bare OPTIONS:", bare.Code, bare.Header().Get("Allow"))
	// Output:
	// preflight: 204 https://app.example.com
	// bare OPTIONS: 405 GET, HEAD
}

// RealIP reads client-controlled headers, so it needs to know which peers are
// allowed to set them. A request from outside the trusted networks is keyed on
// RemoteAddr, which cannot be forged over TCP - without this, a caller reaching
// the server directly picks its own rate-limit key on every request.
func ExampleRealIP() {
	handler := middleware.RealIP(
		middleware.WithTrustedProxies("10.0.0.0/8"),
	)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		fmt.Println(middleware.RealIPFromContext(request.Context()))
	}))

	// Behind the load balancer: the forwarded address is used.
	fromProxy := httptest.NewRequest(http.MethodGet, "/", nil)
	fromProxy.RemoteAddr = "10.1.2.3:44444"
	fromProxy.Header.Set("X-Forwarded-For", "198.51.100.7")

	handler.ServeHTTP(httptest.NewRecorder(), fromProxy)

	// Straight from the internet, claiming to be someone else: ignored.
	direct := httptest.NewRequest(http.MethodGet, "/", nil)
	direct.RemoteAddr = "203.0.113.9:44444"
	direct.Header.Set("X-Forwarded-For", "198.51.100.7")

	handler.ServeHTTP(httptest.NewRecorder(), direct)
	// Output:
	// 198.51.100.7
	// 203.0.113.9
}
