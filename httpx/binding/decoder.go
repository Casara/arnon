package binding

import (
	"net/http"

	"github.com/Casara/arnon/problem"
)

// Decode binds an HTTP request into a T, reading path, query, header
// and JSON body values in that order. It never returns early: all four
// sources are always attempted so a caller sees every binding failure
// at once instead of one at a time across repeated requests.
func Decode[T any](
	request *http.Request,
) (
	T,
	[]problem.ValidationError,
) {
	var dto T

	validationErrors := make([]problem.ValidationError, 0, 4)

	validationErrors = append(
		validationErrors,
		bindPath(request, &dto)...,
	)

	validationErrors = append(
		validationErrors,
		bindQuery(request, &dto)...,
	)

	validationErrors = append(
		validationErrors,
		bindHeader(request, &dto)...,
	)

	validationErrors = append(
		validationErrors,
		bindJSON(request, &dto)...,
	)

	return dto, validationErrors
}
