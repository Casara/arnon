package httpx

import (
	"net/http"
	"reflect"

	"github.com/Casara/arnon/httpx/binding"
	"github.com/Casara/arnon/openapi"
	"github.com/Casara/arnon/problem"
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

// Endpoint creates an HTTP endpoint.
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

	handlerFunc := func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		dto, bindingErrors := binding.Decode[TRequest](request)

		if len(bindingErrors) > 0 {
			writeValidationProblem(
				writer,
				requestValidationFailed,
				requestValidationFailed,
				bindingErrors,
			)

			return
		}

		validationErrors := config.Validator.Validate(dto)

		if len(validationErrors) > 0 {
			writeValidationProblem(
				writer,
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
				config.ProblemMapper.Map(err),
			)

			return
		}

		err = WriteJSON(writer, config.SuccessStatus, response)
		if err != nil {
			WriteProblem(
				writer,
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

		requestType: reflect.TypeFor[TRequest](),

		responseType: reflect.TypeFor[TResponse](),
	}
}

func writeValidationProblem(
	writer http.ResponseWriter,
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
		problemInstance,
	)
}
