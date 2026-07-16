// Command basic is the minimal, runnable example of arnon: a typed
// endpoint, request validation and generated OpenAPI documentation,
// with no middleware at all. See examples/cmd/middleware for the same
// endpoint wrapped in arnon's full middleware stack.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Casara/arnon/examples/internal/customvalidators"
	"github.com/Casara/arnon/examples/internal/logging"
	"github.com/Casara/arnon/examples/internal/users"
	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/openapi"
)

const readHeaderTimeout = 5 * time.Second

func main() {
	logging.NewLogger()

	customvalidators.RegisterCustomValidators()

	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(openapi.NewRegistry(generator)),
	)

	router.POST("/users", httpx.Endpoint(
		users.CreateUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			// OpenAPI must be set (even to an empty *Operation) for the
			// route to be registered in the generated document - it is
			// opt-in per endpoint, everything inside it (schemas,
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

	// Demonstrates path binding plus []string binding from both query
	// (repeated keys: "?tags=a&tags=b") and header (Accept-Language,
	// repeated lines or one comma-joined line - both work the same
	// way, see users.GetUserRequest).
	router.GET("/users/{id}", httpx.Endpoint(
		users.GetUser,
		httpx.EndpointConfig{
			OpenAPI: &openapi.Operation{
				Summary: "Get a user",
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
