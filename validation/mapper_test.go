package validation_test

import (
	"testing"

	"github.com/casara/arnon/problem"
	"github.com/casara/arnon/validation"
)

func TestPlaygroundValidator_MapsBuiltinTags(t *testing.T) {
	t.Parallel()

	type request struct {
		Required string `json:"required" validate:"required"`
		Min      string `json:"min"      validate:"omitempty,min=3"`
		Max      string `json:"max"      validate:"omitempty,max=2"`
		GT       int    `json:"gt"       validate:"omitempty,gt=10"`
		GTE      int    `json:"gte"      validate:"omitempty,gte=10"`
		LT       int    `json:"lt"       validate:"omitempty,lt=10"`
		LTE      int    `json:"lte"      validate:"omitempty,lte=10"`
		Len      string `json:"len"      validate:"omitempty,len=3"`
		OneOf    string `json:"oneof"    validate:"omitempty,oneof=a b c"`
		Eq       int    `json:"eq"       validate:"omitempty,eq=1"`
		// Ne intentionally has no omitempty: its zero value (0) is the
		// value under test, and omitempty would skip validation for it.
		Ne    int    `json:"ne"    validate:"ne=0"`
		UUID  string `json:"uuid"  validate:"omitempty,uuid"`
		Email string `json:"email" validate:"omitempty,email"`
		URL   string `json:"url"   validate:"omitempty,url"`
	}

	invalid := request{
		Required: "",
		Min:      "ab",
		Max:      "abc",
		GT:       5,
		GTE:      5,
		LT:       20,
		LTE:      20,
		Len:      "ab",
		OneOf:    "z",
		Eq:       2,
		Ne:       0,
		UUID:     "not-a-uuid",
		Email:    "not-an-email",
		URL:      "not-a-url",
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(invalid)

	byField := make(map[string]problem.ValidationError, len(errs))
	for _, validationErr := range errs {
		byField[validationErr.Source.Field] = validationErr
	}

	cases := []struct {
		field string
		code  problem.ValidationErrorCode
	}{
		{"/required", problem.ValidationCodeRequired},
		{"/min", problem.ValidationCodeMinLength},
		{"/max", problem.ValidationCodeMaxLength},
		{"/gt", problem.ValidationCodeGreaterThan},
		{"/gte", problem.ValidationCodeGreaterThanOrEqual},
		{"/lt", problem.ValidationCodeLessThan},
		{"/lte", problem.ValidationCodeLessThanOrEqual},
		{"/len", problem.ValidationCodeExactLength},
		{"/oneof", problem.ValidationCodeInvalidValue},
		{"/eq", problem.ValidationCodeEqual},
		{"/ne", problem.ValidationCodeNotEqual},
		{"/uuid", problem.ValidationCodeInvalidUUID},
		{"/email", problem.ValidationCodeInvalidEmail},
		{"/url", problem.ValidationCodeInvalidURL},
	}

	for _, testCase := range cases {
		got, ok := byField[testCase.field]
		if !ok {
			t.Errorf("missing validation error for field %q", testCase.field)

			continue
		}

		if got.Code != testCase.code {
			t.Errorf(
				"field %q: expected code %q, got %q",
				testCase.field,
				testCase.code,
				got.Code,
			)
		}

		if got.Source.In != problem.ValidationLocationBody {
			t.Errorf(
				"field %q: expected location %q, got %q",
				testCase.field,
				problem.ValidationLocationBody,
				got.Source.In,
			)
		}
	}
}

func TestPlaygroundValidator_ValidRequestHasNoErrors(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `json:"name" validate:"required"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{Name: "ok"})
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %+v", errs)
	}
}

// TestPlaygroundValidator_EscapesJSONPointerSpecialCharacters covers
// RFC 6901 §3: a JSON field name containing "/" or "~" must have those
// characters escaped ("~1"/"~0") in the pointer, or the resulting
// Source.Field would be indistinguishable from a nested path.
func TestPlaygroundValidator_EscapesJSONPointerSpecialCharacters(t *testing.T) {
	t.Parallel()

	type request struct {
		Slash string `json:"a/b" validate:"required"`
		Tilde string `json:"a~b" validate:"required"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{})

	byField := make(map[string]problem.ValidationError, len(errs))
	for _, validationErr := range errs {
		byField[validationErr.Source.Field] = validationErr
	}

	if _, ok := byField["/a~1b"]; !ok {
		t.Errorf(`expected escaped pointer "/a~1b" for json tag "a/b", got %+v`, byField)
	}

	if _, ok := byField["/a~0b"]; !ok {
		t.Errorf(`expected escaped pointer "/a~0b" for json tag "a~b", got %+v`, byField)
	}
}

// TestPlaygroundValidator_ResolvesNestedBodyFields confirms a field
// inside a nested struct (Address.City) produces the composed RFC
// 6901 pointer "/address/city", not just "/city" - buildFieldMap
// recurses into nested body structs and mapper.go matches errors
// against the full Go-name namespace (StructNamespace, with the
// leading root type name stripped), not just the leaf field name.
func TestPlaygroundValidator_ResolvesNestedBodyFields(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `json:"city" validate:"required"`
	}

	type request struct {
		Name    string  `json:"name"    validate:"required"`
		Address address `json:"address"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{Name: "ok"})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/address/city" {
		t.Errorf(`expected Source.Field "/address/city", got %q`, errs[0].Source.Field)
	}

	if errs[0].Source.In != problem.ValidationLocationBody {
		t.Errorf("expected body location, got %q", errs[0].Source.In)
	}
}

// TestPlaygroundValidator_ResolvesDeeplyNestedBodyFields covers more
// than one level of nesting (Building.Address.City), confirming the
// pointer composes across every level instead of only one.
func TestPlaygroundValidator_ResolvesDeeplyNestedBodyFields(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `json:"city" validate:"required"`
	}

	type building struct {
		Address address `json:"address"`
	}

	type request struct {
		Building building `json:"building"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/building/address/city" {
		t.Errorf(`expected Source.Field "/building/address/city", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_ResolvesSliceOfStructBodyFields confirms a
// field inside a struct element of a slice (Items[1].Name) produces
// the composed RFC 6901 pointer "/items/1/name", with the runtime
// index of the failing element - not the type-only "/items/name" a
// naive extension of struct-nesting support would produce.
func TestPlaygroundValidator_ResolvesSliceOfStructBodyFields(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `json:"name" validate:"required"`
	}

	type request struct {
		Items []item `json:"items" validate:"dive"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Items: []item{{Name: "ok"}, {Name: ""}},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/items/1/name" {
		t.Errorf(`expected Source.Field "/items/1/name", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_ResolvesSliceOfPrimitiveBodyFields covers a
// dive-validated slice of primitives (Tags[1]): validator/v10 reports
// an error with no field segment after the index
// (StructNamespace() == "...Tags[1]"), so the pointer must resolve to
// "/tags/1", not fall back to a single "/tags" or "/tags1" segment.
func TestPlaygroundValidator_ResolvesSliceOfPrimitiveBodyFields(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `json:"tags" validate:"dive,min=3"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Tags: []string{"valid", "ab"},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/tags/1" {
		t.Errorf(`expected Source.Field "/tags/1", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_ResolvesNestedSliceInsideStructBodyFields
// combines both mechanisms: a slice field inside a nested struct
// (Building.Items[0].Name), confirming the pointer composes through a
// struct level, then an index, then another field.
func TestPlaygroundValidator_ResolvesNestedSliceInsideStructBodyFields(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `json:"name" validate:"required"`
	}

	type building struct {
		Items []item `json:"items" validate:"dive"`
	}

	type request struct {
		Building building `json:"building"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Building: building{Items: []item{{Name: ""}}},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/building/items/0/name" {
		t.Errorf(`expected Source.Field "/building/items/0/name", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_ResolvesMapOfStructBodyFields confirms a
// field inside a struct value of a string-keyed map (Items["b"].Name)
// produces the composed RFC 6901 pointer "/items/b/name", with the
// actual failing key.
func TestPlaygroundValidator_ResolvesMapOfStructBodyFields(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `json:"name" validate:"required"`
	}

	type request struct {
		Items map[string]item `json:"items" validate:"dive"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Items: map[string]item{"a": {Name: "ok"}, "b": {Name: ""}},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/items/b/name" {
		t.Errorf(`expected Source.Field "/items/b/name", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_ResolvesMapOfPrimitiveBodyFields covers a
// dive-validated map of primitives: validator/v10 reports an error
// with no field segment after the key (StructNamespace() ends in
// `Tags["short"]`), so the pointer must resolve to "/tags/short", not
// fall back to a single "/tags" segment.
func TestPlaygroundValidator_ResolvesMapOfPrimitiveBodyFields(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags map[string]string `json:"tags" validate:"dive,min=3"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Tags: map[string]string{"long": "valid", "short": "ab"},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/tags/short" {
		t.Errorf(`expected Source.Field "/tags/short", got %q`, errs[0].Source.Field)
	}
}

// TestPlaygroundValidator_EscapesMapKeySpecialCharacters covers RFC
// 6901 §3 for a map key specifically (not a JSON field name): unlike
// a slice index (always digits), a map key can contain "~"/"/" and
// must be escaped the same way a field name is, or the pointer would
// be indistinguishable from a deeper nested path.
func TestPlaygroundValidator_EscapesMapKeySpecialCharacters(t *testing.T) {
	t.Parallel()

	type request struct {
		Meta map[string]string `json:"meta" validate:"dive,min=100"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{
		Meta: map[string]string{"a/b": "short"},
	})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Source.Field != "/meta/a~1b" {
		t.Errorf(`expected Source.Field "/meta/a~1b", got %q`, errs[0].Source.Field)
	}
}

func TestPlaygroundValidator_UnknownBuiltinTagFallsBackToGenericError(t *testing.T) {
	t.Parallel()

	type request struct {
		// alpha is a real validator/v10 tag with no framework mapping
		// and no registered CustomRule, so it must hit the generic
		// fallback branch in mapFieldError.
		Value string `json:"value" validate:"alpha"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{Value: "123"})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Code != problem.ValidationCodeValidationFailed {
		t.Errorf("expected generic fallback code, got %q", errs[0].Code)
	}
}

func TestPlaygroundValidator_NonStructValueProducesInvalidTypeError(t *testing.T) {
	t.Parallel()

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate("not a struct")

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Code != problem.ValidationCodeInvalidType {
		t.Errorf("expected invalid type code, got %q", errs[0].Code)
	}
}

func TestPlaygroundValidator_ResolvesPathQueryHeaderSources(t *testing.T) {
	t.Parallel()

	type request struct {
		ID     string `path:"id" validate:"required"`
		Filter string `          validate:"required" query:"filter"`
		Token  string `          validate:"required"                header:"X-Token"`
		Name   string `          validate:"required"                                 json:"name"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{})

	bySource := make(map[string]problem.ValidationSource, len(errs))
	for _, validationErr := range errs {
		bySource[validationErr.Source.Field] = *validationErr.Source
	}

	expectations := map[string]problem.ValidationLocation{
		"id":      problem.ValidationLocationPath,
		"filter":  problem.ValidationLocationQuery,
		"X-Token": problem.ValidationLocationHeader,
		"/name":   problem.ValidationLocationBody,
	}

	for field, location := range expectations {
		source, ok := bySource[field]
		if !ok {
			t.Errorf("missing validation error sourced from field %q", field)

			continue
		}

		if source.In != location {
			t.Errorf("field %q: expected location %q, got %q", field, location, source.In)
		}
	}
}
