package httpx

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Casara/arnon/httpx/binding"
	"github.com/Casara/arnon/openapi"
	"github.com/Casara/arnon/problem"
	"github.com/Casara/arnon/sanitize"
	"github.com/Casara/arnon/validation"
)

const requestValidationFailed = "Request validation failed"

// EndpointConfig configures an endpoint.
type EndpointConfig struct {
	Validator validation.Validator

	ProblemMapper ProblemMapper

	SuccessStatus int

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

		operation: config.OpenAPI,

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
