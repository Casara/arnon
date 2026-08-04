package binding

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/casara/arnon/problem"
)

func bindQuery[T any](
	request *http.Request,
	target *T,
) []problem.ValidationError {
	queryValues := request.URL.Query()

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

		tag := structField.Tag.Get("query")
		if tag == "" {
			continue
		}

		if field.Kind() == reflect.Slice {
			if field.Type().Elem().Kind() != reflect.String {
				continue
			}

			values := queryValues[tag]
			if len(values) == 0 {
				continue
			}

			field.Set(reflect.ValueOf(values))

			continue
		}

		value := queryValues.Get(tag)
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
					problem.NewQueryError(
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

		case reflect.Bool:
			parsedValue, err := strconv.ParseBool(
				value,
			)
			if err != nil {
				validationErrors = append(
					validationErrors,
					problem.NewQueryError(
						"invalid type",
						tag,
						problem.ValidationCodeInvalidType,
						map[string]any{
							"expected": "boolean",
							"actual":   value,
						},
					),
				)

				continue
			}

			field.SetBool(parsedValue)
		}
	}

	return validationErrors
}
