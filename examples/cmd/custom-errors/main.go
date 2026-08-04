// Command custom-errors answers validation and handler failures with a flat
// {"errors": [...]} body instead of RFC 9457 Problem Details - arnon's
// default, not a requirement.
//
// Unlike examples/cmd/custom-validator's swap, this one is not a config
// field: httpx.Endpoint calls httpx.WriteProblem directly for every error
// path, so it always answers in Problem Details. What is swappable is
// everything upstream of the response: binding.Decode, sanitize.Apply and
// validation.Validator are their own independent packages, not coupled to
// httpx or to problem's wire format. They still produce
// []problem.ValidationError - that is binding/validation's shared data
// shape for "what went wrong, and where" (an RFC 6901 pointer, a code, a
// detail message) - but nothing forces that shape into RFC 9457's JSON
// envelope. customEndpoint below is what it looks like to reuse those three
// packages while writing the response yourself: about a dozen lines,
// because binding/sanitize/validation already did the actual work.
//
// None of this goes through httpx.Endpoint, so - same as
// examples/cmd/files - there is no OpenAPI generation here: OpenAPI
// registration only happens for httpx.Endpoint routes.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/binding"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/problem"
	"github.com/casara/arnon/sanitize"
	"github.com/casara/arnon/validation"

	"github.com/casara/arnon/examples/internal/customvalidators"
	"github.com/casara/arnon/examples/internal/logging"
	"github.com/casara/arnon/examples/internal/users"
)

const readHeaderTimeout = 5 * time.Second

// apiError is this example's own error shape - plain field/message/code,
// no `type`/`title`/`instance`. Any shape is fine here; RFC 9457 is not
// special-cased anywhere below this type.
type apiError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type errorResponse struct {
	Errors []apiError `json:"errors"`
}

func writeErrors(
	writer http.ResponseWriter,
	statusCode int,
	validationErrors []problem.ValidationError,
) {
	errors := make([]apiError, 0, len(validationErrors))

	for _, validationError := range validationErrors {
		field := ""
		if validationError.Source != nil {
			field = validationError.Source.Field
		}

		errors = append(errors, apiError{
			Field:   field,
			Message: validationError.Detail,
			Code:    string(validationError.Code),
		})
	}

	err := httpx.WriteJSON(writer, statusCode, errorResponse{Errors: errors})
	if err != nil {
		http.Error(writer, "encode failure", http.StatusInternalServerError)
	}
}

// customEndpoint reuses arnon's binding, sanitization and validation as-is,
// and only replaces how the result gets written - the same pipeline
// httpx.Endpoint runs (see httpx/endpoint.go), minus the parts that assume
// Problem Details.
func customEndpoint[TRequest, TResponse any](
	handler httpx.HandlerFunc[TRequest, TResponse],
	successStatus int,
) http.Handler {
	requestType := reflect.TypeFor[TRequest]()

	err := sanitize.Prepare(requestType)
	if err != nil {
		panic(fmt.Errorf("prepare sanitize tags for %s: %w", requestType, err))
	}

	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		dto, bindingErrors := binding.Decode[TRequest](request)
		if len(bindingErrors) > 0 {
			writeErrors(writer, http.StatusBadRequest, bindingErrors)

			return
		}

		sanitize.Apply(&dto)

		validationErrors := validation.Default().Validate(dto)
		if len(validationErrors) > 0 {
			writeErrors(writer, http.StatusBadRequest, validationErrors)

			return
		}

		response, err := handler(request.Context(), dto)
		if err != nil {
			slog.Error("handler failed", "error", err)
			writeErrors(writer, http.StatusInternalServerError, []problem.ValidationError{
				{Detail: "internal error", Code: "internal", Source: nil, Meta: nil},
			})

			return
		}

		err = httpx.WriteJSON(writer, successStatus, response)
		if err != nil {
			slog.Error("encode response failed", "error", err)
		}
	})
}

func main() {
	logging.NewLogger()

	customvalidators.RegisterCustomValidators()

	router := routing.NewRouter()

	// users.CreateUser and users.CreateUserRequest are the exact same
	// handler and request type examples/cmd/basic uses through
	// httpx.Endpoint - only the wrapper around them changed, which is the
	// point: the handler itself does not know or care which one answers
	// its errors.
	router.POST("/users", customEndpoint(
		users.CreateUser,
		http.StatusCreated,
	))

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
