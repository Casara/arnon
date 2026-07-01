package problem

import (
	"errors"
	"net/http"
)

var (
	ErrEmptyExtensionKey = errors.New(
		"problem extension key cannot be empty",
	)

	ErrReservedExtensionKey = errors.New(
		"problem extension key is reserved",
	)
)

const (
	defaultUnexpectedDetail = "An unexpected error occurred"
)

// NewBadRequest creates a 400 problem.
func NewBadRequest(
	detail string,
) *Problem {
	return New(
		http.StatusBadRequest,
		http.StatusText(
			http.StatusBadRequest,
		),
		detail,
	)
}

// NewUnauthorized creates a 401 problem.
func NewUnauthorized(
	detail string,
) *Problem {
	return New(
		http.StatusUnauthorized,
		http.StatusText(
			http.StatusUnauthorized,
		),
		detail,
	)
}

// NewForbidden creates a 403 problem.
func NewForbidden(
	detail string,
) *Problem {
	return New(
		http.StatusForbidden,
		http.StatusText(
			http.StatusForbidden,
		),
		detail,
	)
}

// NewNotFound creates a 404 problem.
func NewNotFound(
	detail string,
) *Problem {
	return New(
		http.StatusNotFound,
		http.StatusText(
			http.StatusNotFound,
		),
		detail,
	)
}

// NewConflict creates a 409 problem.
func NewConflict(
	detail string,
) *Problem {
	return New(
		http.StatusConflict,
		http.StatusText(
			http.StatusConflict,
		),
		detail,
	)
}

// NewUnprocessableEntity creates a 422 problem.
func NewUnprocessableEntity(
	detail string,
) *Problem {
	return New(
		http.StatusUnprocessableEntity,
		http.StatusText(
			http.StatusUnprocessableEntity,
		),
		detail,
	)
}

// NewTooManyRequests creates a 429 problem.
func NewTooManyRequests(
	detail string,
) *Problem {
	return New(
		http.StatusTooManyRequests,
		http.StatusText(
			http.StatusTooManyRequests,
		),
		detail,
	)
}

// NewInternal creates a 500 problem.
func NewInternal(
	detail string,
) *Problem {
	if detail == "" {
		detail = defaultUnexpectedDetail
	}

	return New(
		http.StatusInternalServerError,
		http.StatusText(
			http.StatusInternalServerError,
		),
		detail,
	)
}
