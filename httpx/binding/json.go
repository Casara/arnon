package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Casara/arnon/problem"
)

var errInvalidJSONType = errors.New(
	"invalid JSON value type",
)

func bindJSON[T any](
	request *http.Request,
	target *T,
) []problem.ValidationError {
	if request.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(
		request.Body,
	)

	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)
	if err != nil {
		return mapJSONError(err)
	}

	return nil
}

func mapJSONError(
	err error,
) []problem.ValidationError {
	var syntaxErr *json.SyntaxError

	if errors.As(err, &syntaxErr) {
		return []problem.ValidationError{
			problem.NewBodyError(
				"request body contains invalid JSON",
				"/",
				problem.ValidationCodeMalformedJSON,
				map[string]any{
					"offset": syntaxErr.Offset,
				},
			),
		}
	}

	var unmarshalTypeErr *json.UnmarshalTypeError

	if errors.As(err, &unmarshalTypeErr) {
		return []problem.ValidationError{
			problem.NewBodyError(
				fmt.Sprintf(
					"invalid type for field %q",
					unmarshalTypeErr.Field,
				),
				"/"+unmarshalTypeErr.Field,
				problem.ValidationCodeInvalidType,
				map[string]any{
					"expected": unmarshalTypeErr.Type.String(),
					"actual":   unmarshalTypeErr.Value,
				},
			),
		}
	}

	if errors.Is(err, io.EOF) {
		return nil
	}

	return []problem.ValidationError{
		problem.NewBodyError(
			err.Error(),
			"/",
			problem.ValidationCodeInvalidType,
			map[string]any{
				"reason": errInvalidJSONType.Error(),
			},
		),
	}
}
