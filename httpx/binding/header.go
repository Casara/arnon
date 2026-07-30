package binding

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/casara/arnon/problem"
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

		//nolint:exhaustive // only string and []string are supported today
		switch field.Kind() {
		case reflect.String:
			headerValue := request.Header.Get(tag)
			if headerValue == "" {
				continue
			}

			field.SetString(headerValue)

		case reflect.Slice:
			if field.Type().Elem().Kind() != reflect.String {
				continue
			}

			values := headerValues(request, tag)
			if len(values) == 0 {
				continue
			}

			field.Set(reflect.ValueOf(values))
		}
	}

	return validationErrors
}

// headerValues returns every value of a possibly-repeated header,
// flattened into a single slice: RFC 9110 §5.3 treats N header lines
// with the same name as equivalent to one line with the values joined
// by commas, so a client may use either form and both are collected
// the same way here.
func headerValues(
	request *http.Request,
	tag string,
) []string {
	values := make([]string, 0, 1)

	for _, line := range request.Header.Values(tag) {
		for part := range strings.SplitSeq(line, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
	}

	return values
}
