package validation

import (
	"reflect"

	"github.com/Casara/arnon/problem"
)

// maxFieldMapDepth bounds recursion into nested body structs while
// building the field map. Real request DTOs never nest this deep;
// the limit exists to guarantee termination for a pathological
// self-referential struct (e.g. a tree node with a `Parent *Node`
// field) instead of recursing until the stack overflows - fields
// past the limit still validate, they just fall back to the
// single-segment pointer built from mapper.go's lookup miss path.
const maxFieldMapDepth = 16

// buildFieldMap walks value's struct type and returns a map from the
// dot-joined Go field name path (matching the tail of a
// validator/v10 FieldError.StructNamespace(), once its leading
// "TypeName." is stripped - see mapper.go) to the ValidationSource
// that field resolves to. Body fields recurse into nested structs, so
// a field like Address.City produces the RFC 6901 pointer
// "/address/city" instead of a single "/city" segment.
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

	populateFieldMap(
		fieldMap,
		valueType,
		"",
		"",
		0,
	)

	return fieldMap
}

func populateFieldMap(
	fieldMap map[string]problem.ValidationSource,
	valueType reflect.Type,
	namespacePrefix string,
	pointerPrefix string,
	depth int,
) {
	if depth >= maxFieldMapDepth {
		return
	}

	for field := range valueType.Fields() {
		namespace := field.Name
		if namespacePrefix != "" {
			namespace = namespacePrefix + "." + field.Name
		}

		source := resolveValidationSource(
			field,
			pointerPrefix,
		)

		fieldMap[namespace] = source

		nestedType := field.Type
		for nestedType.Kind() == reflect.Pointer {
			nestedType = nestedType.Elem()
		}

		if nestedType.Kind() == reflect.Struct &&
			source.In == problem.ValidationLocationBody {
			populateFieldMap(
				fieldMap,
				nestedType,
				namespace,
				source.Field,
				depth+1,
			)
		}
	}
}
