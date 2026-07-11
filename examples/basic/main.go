// Command basic is a minimal, runnable example of a typed endpoint,
// request validation and generated OpenAPI documentation.
package main

import (
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

const readHeaderTimeout = 5 * time.Second

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
	// the context before CORS/Logging need it, and Logging runs
	// innermost among these so it captures the full chain's duration
	// and the real handler's status code.
	router.Use(
		middleware.StripSlashes(),
		middleware.RealIP(),
		middleware.RequestID(),
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins: []string{"*"},
		}),
		middleware.Logging(logger),
	)

	api := router.Group("/api")

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
