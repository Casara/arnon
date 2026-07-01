package validation

import (
	"reflect"

	"github.com/Casara/arnon/problem"
)

func buildFieldMap(
	value any,
) map[string]problem.ValidationSource {
	valueType := reflect.TypeOf(
		value,
	)

	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	fieldMap := make(
		map[string]problem.ValidationSource,
	)

	for field := range valueType.Fields() {
		fieldMap[field.Name] = resolveValidationSource(
			field,
		)
	}

	return fieldMap
}
