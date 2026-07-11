package validation

import "errors"

// ErrUnsupportedValidationTag indicates that a validator tag
// cannot be mapped to a framework validation error.
var ErrUnsupportedValidationTag = errors.New(
	"unsupported validation tag",
)

// ErrCustomRuleTagEmpty indicates that a CustomRule was registered
// without a tag name.
var ErrCustomRuleTagEmpty = errors.New(
	"custom rule tag must not be empty",
)

// ErrCustomRuleFuncNil indicates that a CustomRule was registered
// without a validation function.
var ErrCustomRuleFuncNil = errors.New(
	"custom rule func must not be nil",
)
