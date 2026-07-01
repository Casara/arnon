package binding

import (
	"net/http"
	"reflect"

	"github.com/Casara/arnon/problem"
)

func bindHeader[T any](
	request *http.Request,
	target *T,
) []problem.ValidationError {
	targetValue := reflect.ValueOf(target)

	if targetValue.Kind() != reflect.Pointer {
		return nil
	}

	targetElement := targetValue.Elem()

	targetType := targetElement.Type()

	validationErrors := make(
		[]problem.ValidationError,
		0,
	)

	for index := range targetElement.NumField() {
		field := targetElement.Field(index)

		structField := targetType.Field(index)

		tag := structField.Tag.Get("header")
		if tag == "" {
			continue
		}

		headerValue := request.Header.Get(
			tag,
		)
		if headerValue == "" {
			continue
		}

		if field.Kind() == reflect.String {
			field.SetString(headerValue)
		}
	}

	return validationErrors
}
