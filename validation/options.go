package validation

import validatorv10 "github.com/go-playground/validator/v10"

// WithConfigure applies arbitrary configuration to the underlying
// go-playground validator instance.
func WithConfigure(
	configure func(*validatorv10.Validate) error,
) Option {
	return func(
		validate *validatorv10.Validate,
	) error {
		return configure(validate)
	}
}
