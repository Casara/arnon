package problem

// ValidationError describes a validation problem.
type ValidationError struct {
	Detail string `json:"detail,omitempty"`

	Code ValidationErrorCode `json:"code,omitempty"`

	Source *ValidationSource `json:"source,omitempty"`

	Meta map[string]any `json:"meta,omitempty"`
}

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
