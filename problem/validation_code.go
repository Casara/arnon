package problem

// ValidationErrorCode represents a stable validation error code.
type ValidationErrorCode string

// Validation error codes used by RFC 9457 validation errors.
const (
	ValidationCodeRequired ValidationErrorCode = "required"

	ValidationCodeMinLength ValidationErrorCode = "min_length"
	ValidationCodeMaxLength ValidationErrorCode = "max_length"

	ValidationCodeGreaterThan        ValidationErrorCode = "greater_than"
	ValidationCodeGreaterThanOrEqual ValidationErrorCode = "greater_than_or_equal"
	ValidationCodeLessThan           ValidationErrorCode = "less_than"
	ValidationCodeLessThanOrEqual    ValidationErrorCode = "less_than_or_equal"

	ValidationCodeExactLength      ValidationErrorCode = "exact_length"
	ValidationCodeInvalidValue     ValidationErrorCode = "invalid_value"
	ValidationCodeEqual            ValidationErrorCode = "equal"
	ValidationCodeNotEqual         ValidationErrorCode = "not_equal"
	ValidationCodeInvalidUUID      ValidationErrorCode = "invalid_uuid"
	ValidationCodeInvalidEmail     ValidationErrorCode = "invalid_email"
	ValidationCodeInvalidURL       ValidationErrorCode = "invalid_url"
	ValidationCodeValidationFailed ValidationErrorCode = "validation_failed"

	ValidationCodeInvalidType   ValidationErrorCode = "invalid_type"
	ValidationCodeMalformedJSON ValidationErrorCode = "malformed_json"
)
