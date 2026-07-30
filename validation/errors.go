package validation

import "errors"

// ErrUnsupportedValidationTag indicates that a validator tag
// cannot be mapped to a framework validation error.
var ErrUnsupportedValidationTag = errors.New(
	"validation: unsupported validation tag",
)

// ErrCustomRuleTagEmpty indicates that a CustomRule was registered
// without a tag name.
var ErrCustomRuleTagEmpty = errors.New(
	"validation: custom rule tag must not be empty",
)

// ErrCustomRuleFuncNil indicates that a CustomRule was registered
// without a validation function.
var ErrCustomRuleFuncNil = errors.New(
	"validation: custom rule func must not be nil",
)
