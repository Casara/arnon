package validation

import (
	"fmt"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/casara/arnon/problem"
)

// PlaygroundValidator validates structs using validator/v10.
type PlaygroundValidator struct {
	validator *validatorv10.Validate
}

// Option configures a PlaygroundValidator during construction.
type Option func(*validatorv10.Validate) error

// New creates a new PlaygroundValidator.
//
// A validator instance is intended to be reused for the lifetime of the
// application. It maintains internal caches that improve validation
// performance.
//
// Use options to customize the underlying validator instance when
// registering aliases, custom validation rules or tag name functions.
func New(
	options ...Option,
) (*PlaygroundValidator, error) {
	validate := validatorv10.New()

	for _, rule := range registeredCustomRules() {
		// validatorv10.FieldLevel satisfies FieldContext structurally - same
		// method set - so the rule is handed straight through with no
		// adapter, and arnon's own API never names the library's type.
		err := validate.RegisterValidation(
			rule.Tag,
			func(field validatorv10.FieldLevel) bool {
				return rule.Func(field)
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"register custom rule %q: %w",
				rule.Tag,
				err,
			)
		}
	}

	for _, option := range options {
		err := option(validate)
		if err != nil {
			return nil, err
		}
	}

	return &PlaygroundValidator{
		validator: validate,
	}, nil
}

// Validate validates a value and returns mapped validation errors.
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
