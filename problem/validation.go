package problem

// ValidationError describes a validation problem.
type ValidationError struct {
	Detail string `json:"detail,omitempty"`

	Code ValidationErrorCode `json:"code,omitempty"`

	Source *ValidationSource `json:"source,omitempty"`

	Meta map[string]any `json:"meta,omitempty"`
}

// NewBodyError creates a ValidationError sourced from the request body,
// with field as a JSON pointer (e.g. "/name").
func NewBodyError(
	detail string,
	field string,
	code ValidationErrorCode,
	meta map[string]any,
) ValidationError {
	return newValidationError(
		detail,
		ValidationLocationBody,
		field,
		code,
		meta,
	)
}

// NewPathError creates a ValidationError sourced from a path parameter.
func NewPathError(
	detail string,
	field string,
	code ValidationErrorCode,
	meta map[string]any,
) ValidationError {
	return newValidationError(
		detail,
		ValidationLocationPath,
		field,
		code,
		meta,
	)
}

// NewQueryError creates a ValidationError sourced from a query
// parameter.
func NewQueryError(
	detail string,
	field string,
	code ValidationErrorCode,
	meta map[string]any,
) ValidationError {
	return newValidationError(
		detail,
		ValidationLocationQuery,
		field,
		code,
		meta,
	)
}

// NewHeaderError creates a ValidationError sourced from a request
// header.
func NewHeaderError(
	detail string,
	field string,
	code ValidationErrorCode,
	meta map[string]any,
) ValidationError {
	return newValidationError(
		detail,
		ValidationLocationHeader,
		field,
		code,
		meta,
	)
}

func newValidationError(
	detail string,
	location ValidationLocation,
	field string,
	code ValidationErrorCode,
	meta map[string]any,
) ValidationError {
	return ValidationError{
		Detail: detail,
		Code:   code,
		Source: &ValidationSource{
			In:    location,
			Field: field,
		},
		Meta: meta,
	}
}
