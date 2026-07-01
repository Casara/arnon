package validation

import (
	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/Casara/arnon/problem"
)

// PlaygroundValidator validates structs using validator/v10.
type PlaygroundValidator struct {
	validator *validatorv10.Validate
}

func New() *PlaygroundValidator {
	return &PlaygroundValidator{
		validator: validatorv10.New(),
	}
}

func (playgroundValidator *PlaygroundValidator) Validate(
	value any,
) []problem.ValidationError {
	validationErr := playgroundValidator.validator.Struct(
		value,
	)
	if validationErr == nil {
		return nil
	}

	return mapValidationErrors(
		validationErr,
		value,
	)
}
