package openapi

import (
	"reflect"
	"strings"
)

func getJSONName(
	field reflect.StructField,
) string {
	tagValue := field.Tag.Get(
		"json",
	)

	if tagValue == "-" {
		return ""
	}

	if tagValue == "" {
		return strings.ToLower(
			field.Name,
		)
	}

	name := strings.Split(
		tagValue,
		",",
	)[0]

	if name == "" {
		return strings.ToLower(
			field.Name,
		)
	}

	return name
}
