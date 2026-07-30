package httpx

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/casara/arnon/httpx/binding"
	"github.com/casara/arnon/openapi"
	"github.com/casara/arnon/problem"
	"github.com/casara/arnon/sanitize"
	"github.com/casara/arnon/validation"
)

const requestValidationFailed = "Request validation failed"

// EndpointConfig configures an endpoint. Every field is optional: the zero
// value produces a working endpoint with request validation, RFC 9457 error
// responses and a 200 success status. See WithDefaults for what gets filled
// in.
type EndpointConfig struct {
	// Validator checks the bound request against its `validate` struct tags.
	// Defaults to validation.Default(), the shared instance - set this only
	// to plug in a different implementation or one configured separately.
	Validator validation.Validator

	// ProblemMapper converts an error returned by the handler into the
	// problem document sent to the client. Defaults to DefaultProblemMapper,
	// which passes a *problem.Problem through unchanged and turns anything
	// else into a 500 without leaking its message. Override it to map your
	// own domain errors onto statuses.
	ProblemMapper ProblemMapper

	// SuccessStatus is the status written when the handler returns no error.
	// Defaults to 200 OK; set it to 201 for a creation endpoint, and so on.
	//
	// The generated OpenAPI document documents the success response under this
	// same status - it is copied onto the operation - so there is nothing to
	// keep in sync by hand.
	SuccessStatus int

	// OpenAPI opts this route into the generated OpenAPI document. A route is
	// only registered when this is non-nil - an empty &openapi.Operation{} is
	// enough - so documentation is explicit per endpoint, never automatic.
	// Leaving it nil produces a fully functional, undocumented route.
	OpenAPI *openapi.Operation
}

// WithDefaults returns a copy of the configuration with missing values
// replaced by framework defaults.
//
// The default validator enables request validation, the default problem
// mapper converts errors into RFC 9457 problem responses, and the
// default success status is HTTP 200 OK.
func (config EndpointConfig) WithDefaults() EndpointConfig {
	if config.Validator == nil {
		config.Validator = validation.Default()
	}

	if config.ProblemMapper == nil {
		config.ProblemMapper = DefaultProblemMapper{}
	}

	if config.SuccessStatus == 0 {
		config.SuccessStatus = http.StatusOK
	}

	return config
}

// Endpoint wraps a typed handler into an http.Handler: it binds the
// request (path/query/header/JSON body), sanitizes it, validates it,
// calls handler, and writes the result - a success response on the
// happy path, or an RFC 9457 Problem (via config.ProblemMapper) if
// binding, validation, or handler itself returns an error.
//
// It panics if TRequest carries a `sanitize` tag naming a transform that was
// never registered (see sanitize.Prepare and sanitize.RegisterFunc), and - via
// the default validator it installs when config.Validator is nil - if a custom
// validation rule is misconfigured (see validation.Default). Both are
// startup-time programming errors: Endpoint is meant to be called while wiring
// routes, not per request, so the panic surfaces at boot rather than on the one
// request that happens to populate the field.
func Endpoint[
	TRequest any,
	TResponse any,
](
	handler HandlerFunc[
		TRequest,
		TResponse,
	],
	config EndpointConfig,
) http.Handler {
	config = config.WithDefaults()

	requestType := reflect.TypeFor[TRequest]()

	err := sanitize.Prepare(requestType)
	if err != nil {
		panic(fmt.Errorf("prepare sanitize tags for %s: %w", requestType, err))
	}

	handlerFunc := func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if !acceptsJSON(request.Header.Get("Accept")) {
			WriteProblem(
				writer,
				request,
				problem.NewNotAcceptable(
					"this endpoint only produces application/json",
				),
			)

			return
		}

		dto, bindingErrors := binding.Decode[TRequest](request)

		if len(bindingErrors) > 0 {
			writeValidationProblem(
				writer,
				request,
				requestValidationFailed,
				requestValidationFailed,
				bindingErrors,
			)

			return
		}

		sanitize.Apply(&dto)

		validationErrors := config.Validator.Validate(dto)

		if len(validationErrors) > 0 {
			writeValidationProblem(
				writer,
				request,
				requestValidationFailed,
				requestValidationFailed,
				validationErrors,
			)

			return
		}

		response, err := handler(
			request.Context(),
			dto,
		)
		if err != nil {
			WriteProblem(
				writer,
				request,
				config.ProblemMapper.Map(err),
			)

			return
		}

		err = WriteJSON(writer, config.SuccessStatus, response)
		if err != nil {
			WriteProblem(
				writer,
				request,
				problem.New(
					http.StatusInternalServerError,
					"Response serialization failure",
					"Failed to encode response",
				),
			)
		}
	}

	return &endpointHandler{
		handler: handlerFunc,

		operation: operationFor(config),

		requestType: requestType,

		responseType: reflect.TypeFor[TResponse](),
	}
}

func writeValidationProblem(
	writer http.ResponseWriter,
	request *http.Request,
	title string,
	detail string,
	validationErrors []problem.ValidationError,
) {
	statusCode := http.StatusBadRequest

	for _, validationError := range validationErrors {
		if override := validationError.Code.StatusOverride(); override != 0 {
			statusCode = override

			break
		}
	}

	problemInstance := problem.New(
		statusCode,
		title,
		detail,
	)

	for _, validationError := range validationErrors {
		_ = problemInstance.AddError(validationError)
	}

	WriteProblem(
		writer,
		request,
		problemInstance,
	)
}

// operationFor returns the operation to publish, with SuccessStatus derived
// from the endpoint's own so the generated document cannot disagree with what
// the handler returns. An explicit non-zero value on the operation wins, for
// the rare case of documenting a different status on purpose.
//
// The operation is copied rather than mutated: the caller's
// EndpointConfig.OpenAPI is a pointer they may well reuse across endpoints.
func operationFor(config EndpointConfig) *openapi.Operation {
	if config.OpenAPI == nil {
		return nil
	}

	operation := *config.OpenAPI

	if operation.SuccessStatus == 0 {
		operation.SuccessStatus = config.SuccessStatus
	}

	return &operation
}
