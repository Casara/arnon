package sanitize

import (
	"errors"
	"fmt"
	"reflect"
)

// Prepare validates that every `sanitize` tag reachable from
// structType references a registered function, walking the type
// structurally rather than a live value - so a nested pointer-to-
// struct field is checked even when it would be nil at runtime,
// unlike Apply's value-based walk. Intended to be called once when an
// endpoint is constructed (see httpx.Endpoint), not per request: a
// typo in a tag then fails at startup instead of silently being
// skipped on a real request.
func Prepare(structType reflect.Type) error {
	var errs []error

	prepareType(structType, "", 0, &errs)

	return errors.Join(errs...)
}

func prepareType(structType reflect.Type, path string, depth int, errs *[]error) {
	structType = elemType(structType)

	if structType == nil || structType.Kind() != reflect.Struct || depth >= maxDepth {
		return
	}

	for field := range structType.Fields() {
		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		tokens, dive := parseTag(field.Tag.Get("sanitize"))

		prepareField(field.Type, fieldPath, tokens, dive, depth, errs)
	}
}

func prepareField(
	fieldType reflect.Type,
	path string,
	tokens []string,
	dive bool,
	depth int,
	errs *[]error,
) {
	fieldType = elemType(fieldType)
	if fieldType == nil {
		return
	}

	//nolint:exhaustive // only String/Struct/Slice/Array/Map need special handling; everything else is a leaf
	switch fieldType.Kind() {
	case reflect.String:
		checkTokens(path, tokens, errs)

	case reflect.Struct:
		prepareType(fieldType, path, depth+1, errs)

	case reflect.Slice, reflect.Array:
		if !dive || depth >= maxDepth {
			return
		}

		prepareField(fieldType.Elem(), path+"[]", tokens, false, depth+1, errs)

	case reflect.Map:
		if !dive || depth >= maxDepth {
			return
		}

		prepareField(fieldType.Elem(), path+"[]", tokens, false, depth+1, errs)
	}
}

func checkTokens(path string, tokens []string, errs *[]error) {
	for _, tag := range tokens {
		_, ok := lookupFunc(tag)
		if !ok {
			*errs = append(
				*errs,
				fmt.Errorf("%s: sanitizer %q: %w", path, tag, ErrUnknownSanitizer),
			)
		}
	}
}

// elemType unwraps a chain of pointer types down to the underlying
// type, returning nil if fieldType itself is nil (a nested field's
// type can recurse down to nil through Elem(), unlike the top-level
// type Prepare is first called with).
func elemType(fieldType reflect.Type) reflect.Type {
	if fieldType == nil {
		return nil
	}

	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	return fieldType
}
