package problem

import "net/http"

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

	ValidationCodeInvalidType     ValidationErrorCode = "invalid_type"
	ValidationCodeMalformedJSON   ValidationErrorCode = "malformed_json"
	ValidationCodePayloadTooLarge ValidationErrorCode = "payload_too_large"
)

// StatusOverride returns the HTTP status a validation error with this
// code should produce instead of the default 400 Bad Request used for
// binding/validation failures, or 0 when the default applies.
//
// This exists because binding errors (see httpx/binding) don't carry
// their own HTTP status - httpx.Endpoint maps every one of them to
// 400 by default - so a handful of codes that genuinely mean
// something else (e.g. a request body that's simply too large) need
// an explicit escape hatch.
func (code ValidationErrorCode) StatusOverride() int {
	switch code {
	case ValidationCodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	default:
		return 0
	}
}
