package validation

import (
	"fmt"
	"reflect"

	"github.com/Casara/arnon/problem"
)

// maxFieldMapDepth bounds recursion into nested body structs/
// slices/maps while building the field map. Real request DTOs never
// nest this deep; the limit exists to guarantee termination for a
// pathological self-referential struct (e.g. a tree node with a
// `Parent *Node` field) instead of recursing until the stack
// overflows - fields past the limit still validate, they just fall
// back to the single-segment pointer built from mapper.go's lookup
// miss path.
const maxFieldMapDepth = 16

// buildFieldMap walks value's actual struct value (not just its type
// - a slice/array/map's runtime size is only known at the value
// level) and returns a map from the Go namespace path (matching the
// tail of a validator/v10 FieldError.StructNamespace(), once its
// leading "TypeName." is stripped - see mapper.go) to the
// ValidationSource that field resolves to. Body fields recurse into
// nested structs, slice/array elements and string-keyed map entries,
// so a field like Address.City produces the RFC 6901 pointer
// "/address/city" instead of a single "/city" segment, Items[2].Name
// produces "/items/2/name", and Meta["x/y"] produces "/meta/x~1y"
// (the map key gets RFC 6901 §3 escaping same as a JSON field name -
// a slice index never needs it, since it's always digits).
//
// Because this walks the actual value, its cost scales with the size
// of any slice/array/map reachable from value, not just the number of
// fields in its type - only relevant on the validation-failure path
// (mapValidationErrors only calls this once validator/v10 has already
// found at least one error), so it doesn't affect successful requests.
func buildFieldMap(
	value any,
) map[string]problem.ValidationSource {
	fieldMap := make(
		map[string]problem.ValidationSource,
	)

	populateFieldMap(
		fieldMap,
		reflect.ValueOf(value),
		"",
		"",
		0,
	)

	return fieldMap
}

func populateFieldMap(
	fieldMap map[string]problem.ValidationSource,
	structValue reflect.Value,
	namespacePrefix string,
	pointerPrefix string,
	depth int,
) {
	structValue = dereference(structValue)

	if structValue.Kind() != reflect.Struct || depth >= maxFieldMapDepth {
		return
	}

	structType := structValue.Type()

	for index := range structType.NumField() {
		field := structType.Field(index)

		namespace := field.Name
		if namespacePrefix != "" {
			namespace = namespacePrefix + "." + field.Name
		}

		source := resolveValidationSource(
			field,
			pointerPrefix,
		)

		fieldMap[namespace] = source

		if source.In == problem.ValidationLocationBody {
			populateNestedFieldMap(
				fieldMap,
				structValue.Field(index),
				namespace,
				source.Field,
				depth,
			)
		}
	}
}

// populateNestedFieldMap descends into a body field's value: directly
// if it's a (possibly pointer) struct, per-element - with the
// element's runtime index spliced into both the Go namespace and the
// RFC 6901 pointer - if it's a slice/array, or per-key if it's a
// string-keyed map. A slice/array element or map entry gets a map
// entry even when it isn't a struct (e.g. `Tags []string`), since
// validator/v10 reports a per-element error for a `dive`-validated
// primitive collection with no further field segment
// (FieldError.StructNamespace() ends in "Tags[0]" or "Meta[key]", not
// "Tags[0].something"). Map keys, unlike a slice index, can contain
// "~"/"/" and so need the same RFC 6901 §3 escaping as a JSON field
// name; validator/v10's namespace itself uses the raw, unescaped key
// (confirmed empirically), so only the pointer half is escaped - the
// namespace segment used as this function's own map key must match
// validator/v10's raw format exactly. Non-string map keys are the one
// remaining gap: skipped here, falling back to mapper.go's
// single-segment lookup-miss path, since JSON object keys (and so a
// meaningful RFC 6901 segment) are always strings.
func populateNestedFieldMap(
	fieldMap map[string]problem.ValidationSource,
	fieldValue reflect.Value,
	namespace string,
	pointerPrefix string,
	depth int,
) {
	fieldValue = dereference(fieldValue)

	//nolint:exhaustive // only Struct/Slice/Array/Map need special handling; everything else is a leaf
	switch fieldValue.Kind() {
	case reflect.Struct:
		populateFieldMap(
			fieldMap,
			fieldValue,
			namespace,
			pointerPrefix,
			depth+1,
		)

	case reflect.Slice, reflect.Array:
		for elementIndex := range fieldValue.Len() {
			elementNamespace := fmt.Sprintf("%s[%d]", namespace, elementIndex)
			elementPointer := fmt.Sprintf("%s/%d", pointerPrefix, elementIndex)

			fieldMap[elementNamespace] = problem.ValidationSource{
				In:    problem.ValidationLocationBody,
				Field: elementPointer,
			}

			populateNestedFieldMap(
				fieldMap,
				fieldValue.Index(elementIndex),
				elementNamespace,
				elementPointer,
				depth+1,
			)
		}

	case reflect.Map:
		for _, key := range fieldValue.MapKeys() {
			if key.Kind() != reflect.String {
				continue
			}

			keyString := key.String()

			elementNamespace := fmt.Sprintf("%s[%s]", namespace, keyString)
			elementPointer := pointerPrefix + "/" + escapeJSONPointerToken(keyString)

			fieldMap[elementNamespace] = problem.ValidationSource{
				In:    problem.ValidationLocationBody,
				Field: elementPointer,
			}

			populateNestedFieldMap(
				fieldMap,
				fieldValue.MapIndex(key),
				elementNamespace,
				elementPointer,
				depth+1,
			)
		}
	}
}

// dereference unwraps a (possibly nil) chain of pointers down to the
// underlying value. A nil pointer resolves to the zero Value, whose
// Kind() is reflect.Invalid - safe for every caller here, since
// neither populateFieldMap nor populateNestedFieldMap's switch match
// that kind, so a nil pointer field simply contributes no nested
// entries instead of panicking.
func dereference(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}

		value = value.Elem()
	}

	return value
}
