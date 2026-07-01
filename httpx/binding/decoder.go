package binding

import (
	"net/http"

	"github.com/Casara/arnon/problem"
)

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
