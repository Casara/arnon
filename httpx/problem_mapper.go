package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/casara/arnon/problem"
)

// ProblemMapper converts a handler's returned error into the RFC 9457
// Problem written to the response. EndpointConfig.ProblemMapper
// defaults to DefaultProblemMapper when unset.
type ProblemMapper interface {
	Map(err error) *problem.Problem
}

// ProblemMapperFunc adapts a function into ProblemMapper.
type ProblemMapperFunc func(
	error,
) *problem.Problem

// Map calls mapper with err.
func (mapper ProblemMapperFunc) Map(
	err error,
) *problem.Problem {
	return mapper(err)
}

// DefaultProblemMapper maps errors to RFC 9457 problems.
type DefaultProblemMapper struct{}

// Map recognizes two binding failure shapes it can describe precisely
// (a malformed JSON body vs. a value of the wrong type for a field,
// the latter pointing at the exact field via a body ValidationError),
// passes an existing *problem.Problem through unchanged, and maps
// everything else to a generic 500 - the fallback exists so a handler
// error never leaks as a raw error string in the response.
func (mapper DefaultProblemMapper) Map(
	err error,
) *problem.Problem {
	var syntaxErr *json.SyntaxError

	if errors.As(err, &syntaxErr) {
		return problem.New(
			http.StatusBadRequest,
			"Malformed request body",
			"Request body contains invalid JSON",
		)
	}

	var unmarshalErr *json.UnmarshalTypeError

	if errors.As(err, &unmarshalErr) {
		return problem.New(
			http.StatusBadRequest,
			"Invalid request body",
			"Request body contains invalid values",
		).AddError(
			problem.NewBodyError(
				"invalid type",
				"/"+unmarshalErr.Field,
				problem.ValidationCodeInvalidType,
				map[string]any{
					"expected": unmarshalErr.Type.String(),
					"actual":   unmarshalErr.Value,
				},
			),
		)
	}

	var problemInstance *problem.Problem

	if errors.As(err, &problemInstance) {
		return problemInstance
	}

	return problem.New(
		http.StatusInternalServerError,
		http.StatusText(
			http.StatusInternalServerError,
		),
		"Unexpected error",
	)
}
