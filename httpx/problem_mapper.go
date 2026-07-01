package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Casara/arnon/problem"
)

// ProblemMapper maps errors into problems.
type ProblemMapper interface {
	Map(err error) *problem.Problem
}

// ProblemMapperFunc adapts a function into ProblemMapper.
type ProblemMapperFunc func(
	error,
) *problem.Problem

func (mapper ProblemMapperFunc) Map(
	err error,
) *problem.Problem {
	return mapper(err)
}

// DefaultProblemMapper maps errors to RFC 7807 problems.
type DefaultProblemMapper struct{}

// Map maps an error to a problem.
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
