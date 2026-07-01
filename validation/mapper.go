package validation

import (
	"errors"
	"fmt"
	"strings"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/Casara/arnon/problem"
)

func mapValidationErrors(
	validationErr error,
	value any,
) []problem.ValidationError {
	var validationErrors validatorv10.ValidationErrors
	if !errors.As(validationErr, &validationErrors) {
		return []problem.ValidationError{
			problem.NewBodyError(
				validationErr.Error(),
				"/",
				problem.ValidationCodeInvalidType,
				nil,
			),
		}
	}

	structFieldMap := buildFieldMap(
		value,
	)

	errors := make(
		[]problem.ValidationError,
		0,
		len(validationErrors),
	)

	for _, fieldErr := range validationErrors {
		source, ok := structFieldMap[fieldErr.StructField()]

		if !ok {
			source = problem.ValidationSource{
				In: problem.ValidationLocationBody,
				Field: "/" + strings.ToLower(
					fieldErr.Field(),
				),
			}
		}

		validationError := mapFieldError(
			fieldErr,
			source,
		)

		errors = append(
			errors,
			validationError,
		)
	}

	return errors
}

// mapFieldError translates validator tags into framework validation errors.
//
// Validation rules are intentionally mapped through a single switch
// statement to keep all validator-to-problem translations visible in
// one place.
func mapFieldError(
	fieldErr validatorv10.FieldError,
	source problem.ValidationSource,
) problem.ValidationError {
	switch fieldErr.Tag() {
	case "required":
		return buildValidationError(
			source,
			"field is required",
			problem.ValidationCodeRequired,
			nil,
		)

	case "min":
		return buildValidationError(
			source,
			fmt.Sprintf(
				"must contain at least %s characters",
				fieldErr.Param(),
			),
			problem.ValidationCodeMinLength,
			map[string]any{
				"min": fieldErr.Param(),
			},
		)

	case "max":
		return buildValidationError(
			source,
			fmt.Sprintf(
				"must contain at most %s characters",
				fieldErr.Param(),
			),
			problem.ValidationCodeMaxLength,
			map[string]any{
				"max": fieldErr.Param(),
			},
		)

	case "gt":
		return buildValidationError(
			source,
			"must be greater than "+fieldErr.Param(),
			problem.ValidationCodeGreaterThan,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "gte":
		return buildValidationError(
			source,
			"must be greater than or equal to "+fieldErr.Param(),
			problem.ValidationCodeGreaterThanOrEqual,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "lt":
		return buildValidationError(
			source,
			"must be less than "+fieldErr.Param(),
			problem.ValidationCodeLessThan,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "lte":
		return buildValidationError(
			source,
			"must be less than or equal to "+fieldErr.Param(),
			problem.ValidationCodeLessThanOrEqual,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "len":
		return buildValidationError(
			source,
			"must match length "+fieldErr.Param(),
			problem.ValidationCodeExactLength,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "oneof":
		allowed := strings.Fields(
			fieldErr.Param(),
		)

		return buildValidationError(
			source,
			"must be one of: "+strings.Join(
				allowed,
				", ",
			),
			problem.ValidationCodeInvalidValue,
			map[string]any{
				"allowed": allowed,
			},
		)

	case "eq":
		return buildValidationError(
			source,
			"must equal "+fieldErr.Param(),
			problem.ValidationCodeEqual,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "ne":
		return buildValidationError(
			source,
			"must not equal "+fieldErr.Param(),
			problem.ValidationCodeNotEqual,
			map[string]any{
				"value": fieldErr.Param(),
			},
		)

	case "uuid":
		return buildValidationError(
			source,
			"invalid UUID",
			problem.ValidationCodeInvalidUUID,
			nil,
		)

	case "email":
		return buildValidationError(
			source,
			"invalid email",
			problem.ValidationCodeInvalidEmail,
			nil,
		)

	case "url":
		return buildValidationError(
			source,
			"invalid URL",
			problem.ValidationCodeInvalidURL,
			nil,
		)

	default:
		return buildValidationError(
			source,
			"validation failed",
			problem.ValidationCodeValidationFailed,
			map[string]any{
				"rule":  fieldErr.Tag(),
				"param": fieldErr.Param(),
			},
		)
	}
}

// buildValidationError creates a validation error for the given location.
//
// When the location is unknown, a body validation error is returned.
func buildValidationError(
	source problem.ValidationSource,
	detail string,
	code problem.ValidationErrorCode,
	meta map[string]any,
) problem.ValidationError {
	type validationErrorBuilder func(
		detail string,
		field string,
		code problem.ValidationErrorCode,
		meta map[string]any,
	) problem.ValidationError

	validationErrorBuilders := map[problem.ValidationLocation]validationErrorBuilder{
		problem.ValidationLocationPath:   problem.NewPathError,
		problem.ValidationLocationQuery:  problem.NewQueryError,
		problem.ValidationLocationHeader: problem.NewHeaderError,
		problem.ValidationLocationBody:   problem.NewBodyError,
	}

	builder, ok := validationErrorBuilders[source.In]

	if !ok {
		builder = problem.NewBodyError
	}

	return builder(
		detail,
		source.Field,
		code,
		meta,
	)
}
