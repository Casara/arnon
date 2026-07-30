package validation

import validatorv10 "github.com/go-playground/validator/v10"

// WithConfigure applies arbitrary configuration to the underlying
// go-playground validator instance.
//
// This is the deliberate escape hatch, and the only place in this package that
// names a validator/v10 type. Reach for it when you need something arnon does
// not wrap - a struct-level validation, a custom type function, a tag name
// override.
//
// Two costs to accept knowingly. Code passed here is tied to validator/v10 and
// would not survive a change of implementation. And a validation registered
// this way runs at request time but is invisible to OpenAPI schema generation,
// because that reads the `validate` tag independently - which is the whole
// reason RegisterCustomRule exists. Prefer it whenever the rule is a tag.
func WithConfigure(
	configure func(*validatorv10.Validate) error,
) Option {
	return func(
		validate *validatorv10.Validate,
	) error {
		return configure(validate)
	}
}
