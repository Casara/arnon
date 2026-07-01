package validation

import (
	"reflect"
	"strings"

	"github.com/Casara/arnon/problem"
)

func resolveValidationSource(
	field reflect.StructField,
) problem.ValidationSource {
	switch {
	case field.Tag.Get("path") != "":
		return problem.ValidationSource{
			In:    problem.ValidationLocationPath,
			Field: field.Tag.Get("path"),
		}

	case field.Tag.Get("query") != "":
		return problem.ValidationSource{
			In:    problem.ValidationLocationQuery,
			Field: field.Tag.Get("query"),
		}

	case field.Tag.Get("header") != "":
		return problem.ValidationSource{
			In:    problem.ValidationLocationHeader,
			Field: field.Tag.Get("header"),
		}

	default:
		return problem.ValidationSource{
			In: problem.ValidationLocationBody,
			Field: "/" + normalizeJSONField(
				field.Tag.Get("json"),
			),
		}
	}
}

func normalizeJSONField(
	tagValue string,
) string {
	fieldName := strings.Split(
		tagValue,
		",",
	)[0]

	if fieldName == "" ||
		fieldName == "-" {
		return ""
	}

	return fieldName
}
