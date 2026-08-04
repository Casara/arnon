package sanitize

import "reflect"

// maxDepth bounds recursion into nested structs/slices/maps, mirroring
// validation's maxFieldMapDepth: real request DTOs never nest this
// deep, the limit only guarantees termination for a pathological
// self-referential struct (e.g. a tree node with a `Parent *Node`
// field) instead of recursing until the stack overflows.
const maxDepth = 16

// Apply sanitizes target in place. target must be a non-nil pointer
// to a struct; any other kind - including a nil pointer, or a struct
// passed by value - is a no-op, since there would be nothing
// addressable to write a transformed value back into.
//
// A field's `sanitize` tag is resolved against the registry fed by
// RegisterFunc. A struct field is always recursed into regardless of
// its own tag, mirroring how validator/v10 dives into a nested struct
// automatically; a slice/array/map field is only recursed into when
// its tag starts with "dive", mirroring validate's own convention. An
// unrecognized tag name is silently skipped here - call Prepare once,
// ahead of any request, to catch that instead.
func Apply(target any) {
	applyValue(reflect.ValueOf(target), 0)
}

func applyValue(value reflect.Value, depth int) {
	value = dereference(value)

	if !value.IsValid() || value.Kind() != reflect.Struct || depth >= maxDepth {
		return
	}

	structType := value.Type()

	for index := range structType.NumField() {
		field := structType.Field(index)
		fieldValue := value.Field(index)

		if !fieldValue.CanSet() {
			continue
		}

		tokens, dive := parseTag(field.Tag.Get("sanitize"))

		applyField(fieldValue, tokens, dive, depth)
	}
}

func applyField(fieldValue reflect.Value, tokens []string, dive bool, depth int) {
	dereferenced := dereference(fieldValue)

	if !dereferenced.IsValid() {
		return
	}

	//nolint:exhaustive // only String/Struct/Slice/Array/Map need special handling; everything else is a leaf
	switch dereferenced.Kind() {
	case reflect.String:
		applyTokens(dereferenced, tokens)

	case reflect.Struct:
		applyValue(dereferenced, depth+1)

	case reflect.Slice, reflect.Array:
		if !dive || depth >= maxDepth {
			return
		}

		for elementIndex := range dereferenced.Len() {
			applyField(dereferenced.Index(elementIndex), tokens, false, depth+1)
		}

	case reflect.Map:
		if !dive || depth >= maxDepth {
			return
		}

		for _, key := range dereferenced.MapKeys() {
			elementValue := dereferenced.MapIndex(key)

			// Map values aren't addressable/settable in place - copy
			// into a settable local, mutate that, then write it back.
			settable := reflect.New(elementValue.Type()).Elem()
			settable.Set(elementValue)

			applyField(settable, tokens, false, depth+1)

			dereferenced.SetMapIndex(key, settable)
		}
	}
}

func applyTokens(value reflect.Value, tokens []string) {
	if len(tokens) == 0 || !value.CanSet() {
		return
	}

	result := value.String()

	for _, tag := range tokens {
		fn, ok := lookupFunc(tag)
		if !ok {
			continue
		}

		result = fn(result)
	}

	value.SetString(result)
}

// dereference unwraps a (possibly nil) chain of pointers down to the
// underlying value. A nil pointer resolves to the zero Value, whose
// Kind() is reflect.Invalid - safe for every caller here, mirroring
// validation.dereference.
func dereference(value reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}

		value = value.Elem()
	}

	return value
}
