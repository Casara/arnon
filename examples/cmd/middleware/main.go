// Command middleware demonstrates arnon's full middleware stack
// wrapping the same typed endpoints used by examples/cmd/basic:
// panic recovery, security headers, rate limiting, conditional GET
// (ETag), compression, CORS, OpenAPI discovery (Link), request
// logging, a JSON-only content-type allow-list, a request body size
// limit and no-cache response headers.
//
// The global chain is built with middleware.BuildChain instead of a
// hand-ordered middleware.Router.Use(...) call: BuildChain hardcodes
// the same relative order every time, so enabling/disabling entries here
// can't accidentally break it - see docs/architecture/project-context.md,
// "Middleware order", for why each constraint exists (RequestID before
// Logging, ETag before Compress, Recover outermost, ...).
//
// It also demonstrates ChainConfig.Extra (serverBrandMiddleware
// below): a hand-written middleware arnon has no field for, inserted
// at a specific point relative to the built-ins via a ChainAnchor.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/middleware"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"

	"github.com/casara/arnon/examples/internal/customvalidators"
	"github.com/casara/arnon/examples/internal/logging"
	"github.com/casara/arnon/examples/internal/users"
)

const (
	readHeaderTimeout = 5 * time.Second

	requestTimeout = 5 * time.Second

	maxRequestBodyBytes = 1024

	rateLimitRequests = 100
	rateLimitWindow   = time.Minute
)

func main() {
	logger := logging.NewLogger()

	customvalidators.RegisterCustomValidators()

	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(generator),
	)

	router.Use(middleware.BuildChain(middleware.ChainConfig{
		Recover:       true,
		Timeout:       requestTimeout,
		StripSlashes:  true,
		RealIP:        true,
		RequestID:     true,
		SecureHeaders: &middleware.SecureHeadersConfig{},
		RateLimit: &middleware.RateLimitConfig{
			RequestLimit: rateLimitRequests,
			WindowLength: rateLimitWindow,
		},
		// serverBrandMiddleware isn't a built-in, so it has no
		// ChainConfig field of its own - Extra anchors it right before
		// ETag instead, so ETag's buffered response (and a subsequent
		// 304 from a conditional GET) both carry the header it sets.
		Extra: []middleware.ExtraMiddleware{
			{
				Middleware: serverBrandMiddleware(),
				Before:     middleware.AnchorETag,
			},
		},
		ETag:     true,
		Compress: &middleware.CompressConfig{},
		CORS: &middleware.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
		ServiceDescPath: "/openapi.json",
		Logger:          logger,
	})...)

	api := router.Group("/api")

	// Scoped to /api, not global: AllowContentType/MaxBodyBytes only
	// make sense for endpoints that read a JSON body, and NoCache only
	// matters for responses reflecting per-request state - /openapi.json
	// and /docs are static enough to benefit from normal caching (and
	// /users/{id} below relies on the opposite: ETag, not NoCache).
	api.Use(
		middleware.AllowContentType("application/json"),
		middleware.MaxBodyBytes(maxRequestBodyBytes),
	)

	api.POST("/users", httpx.Endpoint(
		users.CreateUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			// OpenAPI must be set (even to an empty *Operation) for the
			// route to be registered in the generated document - it is
			// the opt-in per endpoint, everything inside it (schemas,
			// parameters, default responses) is still generated
			// automatically.
			OpenAPI: &openapi.Operation{
				Summary: "Create a user",
			},
		},
	))

	// Same handler as examples/cmd/basic, but here it also goes
	// through ETag: a repeated GET with If-None-Match set to the
	// previous response's ETag gets a 304 with no body back.
	api.GET("/users/{id}", httpx.Endpoint(
		users.GetUser,
		httpx.EndpointConfig{
			OpenAPI: &openapi.Operation{
				Summary: "Get a user",
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(openapi.DocsConfig{}))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	err := server.ListenAndServe()
	if err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// serverBrandMiddleware is a stand-in for a hand-written or
// third-party middleware arnon knows nothing about - it exists only
// to demonstrate middleware.ChainConfig.Extra above. It sets a
// header identifying this server; nothing about the header itself
// requires a specific position, but wiring it in through Extra is
// exactly what a middleware with a real ordering requirement (e.g.
// one that must read a value RateLimit or RealIP set in the request
// context) would do too.
func serverBrandMiddleware() routing.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			writer.Header().Set("X-Powered-By", "arnon-example")

			next.ServeHTTP(writer, request)
		})
	}
}
