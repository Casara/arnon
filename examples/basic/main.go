// Command basic is a minimal, runnable example of a typed endpoint,
// request validation and generated OpenAPI documentation.
package main

import (
	"compress/gzip"
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/openapi"
)

const (
	readHeaderTimeout = 5 * time.Second

	maxRequestBodyBytes = 1024

	rateLimitRequests = 100
	rateLimitWindow   = time.Minute
)

type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,notblank"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

func createUser(
	_ context.Context,
	request CreateUserRequest,
) (CreateUserResponse, error) {
	slog.Debug("creating user", "name", request.Name, "email", request.Email)

	return CreateUserResponse{ID: "usr_123"}, nil
}

func main() {
	logger := NewLogger()

	registerCustomValidators()

	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(openapi.NewRegistry(generator)),
	)

	// Order matters: Chain wraps outermost-first, so StripSlashes runs
	// before anything else reads the path, RealIP/RequestID populate
	// the context before CORS/Logging/RateLimit need it, SecureHeaders
	// sits early so its headers land on every response (including a
	// 429 from RateLimit or a 404 from the mux), RateLimit rejects
	// abusive clients before anything downstream does real work, and
	// Compress stays close to CORS/Logging so it wraps the actual
	// response body. Logging runs innermost among these so it captures
	// the full chain's duration and the real handler's status code.
	router.Use(
		middleware.StripSlashes(),
		middleware.RealIP(),
		middleware.RequestID(),
		middleware.SecureHeaders(middleware.SecureHeadersConfig{}),
		middleware.RateLimit(middleware.RateLimitConfig{
			RequestLimit: rateLimitRequests,
			WindowLength: rateLimitWindow,
		}),
		middleware.Compress(gzip.DefaultCompression),
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins: []string{"*"},
		}),
		middleware.Logging(logger),
	)

	api := router.Group("/api")

	// Scoped to /api, not global: AllowContentType/MaxBodyBytes only
	// make sense for endpoints that read a JSON body, and NoCache only
	// matters for responses reflecting per-request state - /openapi.json
	// and /docs are static enough to benefit from normal caching.
	api.Use(
		middleware.AllowContentType("application/json"),
		middleware.MaxBodyBytes(maxRequestBodyBytes),
		middleware.NoCache(),
	)

	api.POST("/users", httpx.Endpoint(
		createUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			// OpenAPI must be set (even to an empty *Operation) for the
			// route to be registered in the generated document - it is
			// the opt-in per endpoint, everything inside it (schemas,
			// parameters, default responses) is still generated
			// automatically.
			//
			// SuccessStatus is repeated here because
			// EndpointConfig.SuccessStatus (what the handler actually
			// returns) and Operation.SuccessStatus (what the generated
			// doc documents) are independent fields; keep them in sync
			// by hand until the framework unifies them.
			OpenAPI: &openapi.Operation{
				Summary:       "Create a user",
				SuccessStatus: http.StatusCreated,
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(nil))

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
