// Package ozzovalidator adapts github.com/go-ozzo/ozzo-validation/v4 to
// arnon's validation.Validator interface, for examples/cmd/custom-validator:
// proof that go-playground/validator is arnon's default, not a requirement -
// EndpointConfig.Validator accepts anything with a one-method
// Validate(value any) []problem.ValidationError shape.
package ozzovalidator

import (
	"errors"

	"github.com/casara/arnon/problem"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// Validator implements validation.Validator by delegating to a value's own
// ozzo-validation Validate() method - ozzo.Validatable, ozzo's own interface
// for "this type knows how to check itself". A request type wanting
// validation under this Validator implements that method itself; see
// examples/cmd/custom-validator for one.
//
// Values that don't implement ozzo.Validatable pass through unchecked, same
// as a struct with no `validate` tag does under arnon's default validator.
type Validator struct{}

// Validate satisfies validation.Validator.
func (Validator) Validate(value any) []problem.ValidationError {
	validatable, ok := value.(ozzo.Validatable)
	if !ok {
		return nil
	}

	err := validatable.Validate()
	if err == nil {
		return nil
	}

	var ozzoErrors ozzo.Errors

	if !errors.As(err, &ozzoErrors) {
		// A Validate() that fails outside ValidateStruct (a struct-level
		// check with no single offending field) can't be attributed to one
		// field, so it becomes one body-level error instead.
		return []problem.ValidationError{
			problem.NewBodyError(err.Error(), "", "invalid", nil),
		}
	}

	validationErrors := make([]problem.ValidationError, 0, len(ozzoErrors))

	for field, fieldErr := range ozzoErrors {
		code := problem.ValidationErrorCode("invalid")
		detail := fieldErr.Error()

		// ozzo's own rules (Required, is.Email, ...) return an ozzo.Error,
		// which carries a stable machine code - e.g. "validation_required".
		// A custom ozzo.RuleFunc that returns a plain error still works, it
		// just falls back to the generic code above.
		var coded ozzo.Error

		if errors.As(fieldErr, &coded) {
			code = problem.ValidationErrorCode(coded.Code())
			detail = coded.Message()
		}

		// ozzo.ErrorTag defaults to "json", so field is already the JSON
		// field name (e.g. "email"), the same vocabulary
		// binding/validation use for their own RFC 6901 pointers.
		validationErrors = append(validationErrors, problem.NewBodyError(
			detail,
			"/"+field,
			code,
			nil,
		))
	}

	return validationErrors
}
