package validation

import (
	"reflect"
	"strings"

	"github.com/casara/arnon/problem"
)

// resolveValidationSource determines where a field's value comes from
// and, for a body field, its RFC 6901 pointer. pointerPrefix is the
// already-resolved pointer of the enclosing struct field ("" at the
// root, "/address" when resolving a field inside an Address struct
// nested under an "address" JSON field) - it has no meaning outside
// the body case, since path/query/header are never hierarchical.
func resolveValidationSource(
	field reflect.StructField,
	pointerPrefix string,
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
			Field: pointerPrefix + "/" + escapeJSONPointerToken(
				normalizeJSONField(
					field.Tag.Get("json"),
				),
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

// escapeJSONPointerToken escapes a single JSON Pointer (RFC 6901 §3)
// reference token: "~" becomes "~0" and "/" becomes "~1", in that
// order (escaping "/" first would double-escape the "~" it
// introduces). Without this, a JSON field literally named e.g. "a/b"
// would produce "/a/b" - indistinguishable from two path segments
// ("a" then "b") instead of the single segment "a/b" the pointer
// actually means.
func escapeJSONPointerToken(token string) string {
	replacer := strings.NewReplacer(
		"~", "~0",
		"/", "~1",
	)

	return replacer.Replace(token)
}
