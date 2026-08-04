// Command custom-validator swaps go-playground/validator - arnon's default -
// for ozzo-validation, to show that the default is a plug, not a hard
// dependency: validation.Validator is a one-method interface, and
// EndpointConfig.Validator accepts any implementation of it. See
// examples/internal/ozzovalidator for the adapter, and
// examples/cmd/custom-errors for the equivalent swap on error responses
// (a different kind of seam - that one is not a config field).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/casara/arnon/examples/internal/logging"
	"github.com/casara/arnon/examples/internal/ozzovalidator"
)

const readHeaderTimeout = 5 * time.Second

// CreateUserRequest carries no `validate` tag: that vocabulary belongs to
// go-playground/validator, which this example does not install. Rules live
// in Validate() below instead, ozzo-validation's own convention.
//
// One consequence worth knowing before copying this pattern: arnon's OpenAPI
// generation (openapi/reflection_field.go) reads the `validate` tag
// directly, independently of which Validator actually runs at request time.
// Swapping the runtime validator does not change what the generated schema
// says - without a `validate` tag here, the generated schema for this
// endpoint has no "required" or "format: email" on these fields, even though
// Validate() enforces both. Keeping the two in sync by hand (a `validate`
// tag purely for documentation, never read at runtime) is possible but not
// done here, since that reintroduces the "same rule in two places" drift
// RegisterCustomRule exists to avoid for the default validator.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email" sanitize:"email"`
}

// Validate satisfies ozzo-validation's Validatable interface, which
// ozzovalidator.Validator looks for via a type assertion.
func (request CreateUserRequest) Validate() error {
	err := ozzo.ValidateStruct(&request,
		ozzo.Field(&request.Name, ozzo.Required),
		ozzo.Field(&request.Email, ozzo.Required, is.Email),
	)
	if err != nil {
		// Wrapping here does not break ozzovalidator.Validator's
		// errors.As(err, &ozzo.Errors{}) - %w keeps the underlying
		// ozzo.Errors reachable through Unwrap.
		return fmt.Errorf("validate create user request: %w", err)
	}

	return nil
}

// CreateUserResponse is the response body for createUser.
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
	logging.NewLogger()

	generator := openapi.NewGenerator(
		openapi.Info{
			Title:   "Custom validator example",
			Version: "1.0.0",
		},
	)

	router := routing.NewRouter(
		routing.WithOpenAPI(generator),
	)

	router.POST("/users", httpx.Endpoint(
		createUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			Validator:     ozzovalidator.Validator{},
			OpenAPI: &openapi.Operation{
				Summary: "Create a user, validated by ozzo-validation " +
					"instead of go-playground/validator",
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
