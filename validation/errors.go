package validation

import "errors"

// ErrUnsupportedValidationTag indicates that a validator tag
// cannot be mapped to a framework validation error.
var ErrUnsupportedValidationTag = errors.New(
	"unsupported validation tag",
)
