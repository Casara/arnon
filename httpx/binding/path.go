package binding

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/Casara/arnon/problem"
)

func bindPath[T any](
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

		tag := structField.Tag.Get("path")
		if tag == "" {
			continue
		}

		value := request.PathValue(tag)
		if value == "" {
			continue
		}

		//nolint:exhaustive // Only supported kinds are mapped.
		switch field.Kind() {
		case reflect.String:
			field.SetString(value)

		case reflect.Int:
			parsedValue, err := strconv.Atoi(
				value,
			)
			if err != nil {
				validationErrors = append(
					validationErrors,
					problem.NewPathError(
						"invalid type",
						tag,
						problem.ValidationCodeInvalidType,
						map[string]any{
							"expected": "integer",
							"actual":   value,
						},
					),
				)

				continue
			}

			field.SetInt(int64(parsedValue))
		}
	}

	return validationErrors
}
