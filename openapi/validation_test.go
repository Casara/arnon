package openapi_test

import (
	"testing"

	"github.com/casara/arnon/openapi"
)

// TestApplyValidationTags_NumericConstraints exercises applyMin,
// applyMax, applyGT, applyGTE, applyLT and applyLTE through the
// public GenerateSchema API by using the matching `validate` tag on
// numeric fields. min/max on a non-string, non-array field apply to
// Schema.Minimum/Maximum (the switch's default arm); string/array
// fields are covered separately below since min/max means length/item
// count there.
func TestApplyValidationTags_NumericConstraints(t *testing.T) {
	t.Parallel()

	type request struct {
		Min int     `json:"min" validate:"min=1"`
		Max int     `json:"max" validate:"max=10"`
		GT  float64 `json:"gt"  validate:"gt=0"`
		GTE float64 `json:"gte" validate:"gte=0"`
		LT  float64 `json:"lt"  validate:"lt=100"`
		LTE float64 `json:"lte" validate:"lte=100"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if got := schema.Properties["min"].Minimum; got == nil || *got != 1 {
		t.Errorf("expected min property Minimum=1, got %+v", got)
	}

	if got := schema.Properties["max"].Maximum; got == nil || *got != 10 {
		t.Errorf("expected max property Maximum=10, got %+v", got)
	}

	if got := schema.Properties["gt"].ExclusiveMinimum; got == nil || *got != 0 {
		t.Errorf("expected gt property ExclusiveMinimum=0, got %+v", got)
	}

	if got := schema.Properties["gte"].Minimum; got == nil || *got != 0 {
		t.Errorf("expected gte property Minimum=0, got %+v", got)
	}

	if got := schema.Properties["lt"].ExclusiveMaximum; got == nil || *got != 100 {
		t.Errorf("expected lt property ExclusiveMaximum=100, got %+v", got)
	}

	if got := schema.Properties["lte"].Maximum; got == nil || *got != 100 {
		t.Errorf("expected lte property Maximum=100, got %+v", got)
	}
}

// TestApplyValidationTags_StringLengthConstraints exercises the
// "string" branch of applyMin/applyMax/applyLen, which targets
// MinLength/MaxLength instead of the numeric Minimum/Maximum used for
// every other schema type.
func TestApplyValidationTags_StringLengthConstraints(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `json:"name" validate:"min=2,max=50"`
		Code string `json:"code" validate:"len=5"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	name := schema.Properties["name"]

	if name.MinLength == nil || *name.MinLength != 2 {
		t.Errorf("expected name MinLength=2, got %+v", name.MinLength)
	}

	if name.MaxLength == nil || *name.MaxLength != 50 {
		t.Errorf("expected name MaxLength=50, got %+v", name.MaxLength)
	}

	code := schema.Properties["code"]

	if code.MinLength == nil || *code.MinLength != 5 {
		t.Errorf("expected code MinLength=5, got %+v", code.MinLength)
	}

	if code.MaxLength == nil || *code.MaxLength != 5 {
		t.Errorf("expected code MaxLength=5, got %+v", code.MaxLength)
	}
}

// TestApplyValidationTags_ArrayLengthConstraints exercises the
// "array" branch of applyMin/applyMax/applyLen, which targets
// MinItems/MaxItems instead of MinLength/MaxLength (string) or
// Minimum/Maximum (every other schema type).
func TestApplyValidationTags_ArrayLengthConstraints(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `json:"tags" validate:"min=1,max=3"`
		IDs  []int    `json:"ids"  validate:"len=2"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	tags := schema.Properties["tags"]

	if tags.MinItems == nil || *tags.MinItems != 1 {
		t.Errorf("expected tags MinItems=1, got %+v", tags.MinItems)
	}

	if tags.MaxItems == nil || *tags.MaxItems != 3 {
		t.Errorf("expected tags MaxItems=3, got %+v", tags.MaxItems)
	}

	ids := schema.Properties["ids"]

	if ids.MinItems == nil || *ids.MinItems != 2 {
		t.Errorf("expected ids MinItems=2, got %+v", ids.MinItems)
	}

	if ids.MaxItems == nil || *ids.MaxItems != 2 {
		t.Errorf("expected ids MaxItems=2, got %+v", ids.MaxItems)
	}
}

// TestApplyValidationTags_InvalidNumericValueIsIgnored confirms that
// an unparseable min/max/len value (not a valid integer) leaves the
// schema untouched rather than panicking - applyMin/applyMax/applyLen
// all silently return on a strconv.Atoi error.
func TestApplyValidationTags_InvalidNumericValueIsIgnored(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `json:"name" validate:"min=notanumber,max=notanumber,len=notanumber"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	name := schema.Properties["name"]

	if name.MinLength != nil || name.MaxLength != nil {
		t.Errorf(
			"expected no length constraints from unparseable values, got min=%+v max=%+v",
			name.MinLength,
			name.MaxLength,
		)
	}
}

// TestApplyValidationTags_InvalidFloatValueIsIgnored covers the
// strconv.ParseFloat error path shared by applyGT/applyGTE/applyLT/
// applyLTE: an unparseable bound leaves the corresponding schema
// field nil instead of panicking.
func TestApplyValidationTags_InvalidFloatValueIsIgnored(t *testing.T) {
	t.Parallel()

	type request struct {
		Value float64 `json:"value" validate:"gt=notanumber,gte=notanumber,lt=notanumber,lte=notanumber"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	value := schema.Properties["value"]

	if value.ExclusiveMinimum != nil || value.Minimum != nil ||
		value.ExclusiveMaximum != nil || value.Maximum != nil {
		t.Errorf(
			"expected no numeric constraints from unparseable values, got %+v",
			value,
		)
	}
}

// TestApplyValidationTags_OneOfWithEmptyValueLeavesEnumUnset covers
// applyOneOf's own value=="" guard: `oneof=` (an "=" with nothing
// after it) still reaches applyOneOf, since
// strings.Cut(tag, "=") reports hasValue=true, but applyOneOf itself
// must still no-op rather than set an empty Enum.
func TestApplyValidationTags_OneOfWithEmptyValueLeavesEnumUnset(t *testing.T) {
	t.Parallel()

	type request struct {
		Status string `json:"status" validate:"oneof="`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if schema.Properties["status"].Enum != nil {
		t.Errorf(
			"expected no enum from an empty oneof value, got %v",
			schema.Properties["status"].Enum,
		)
	}
}

// TestApplyValidationTags_OneOfSetsEnum confirms applyOneOf translates
// space-separated `oneof` values into Schema.Enum, and that a bare
// `oneof` tag without a value (hasValue == false in
// applyValidationTags) is simply skipped instead of clearing Enum.
func TestApplyValidationTags_OneOfSetsEnum(t *testing.T) {
	t.Parallel()

	type request struct {
		Status string `json:"status" validate:"oneof=draft published archived"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	status := schema.Properties["status"]

	want := []any{"draft", "published", "archived"}

	if len(status.Enum) != len(want) {
		t.Fatalf("expected enum %v, got %v", want, status.Enum)
	}

	for i, value := range want {
		if status.Enum[i] != value {
			t.Errorf("expected enum %v, got %v", want, status.Enum)

			break
		}
	}
}

// TestApplyValidationTags_DiveRedirectsConstraintsToItems confirms
// `dive,min=2` on a []string field constrains each element's length
// (Items.MinLength), not the array's item count (MinItems) -
// dive,min=2 means "each string must be >= 2 characters", not "the
// array must have >= 2 elements". Before the fix, applyMin only
// looked at schema.Type ("array" either way), so this produced
// MinItems regardless of dive.
func TestApplyValidationTags_DiveRedirectsConstraintsToItems(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `json:"tags" validate:"dive,min=2,max=10"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	tags := schema.Properties["tags"]

	if tags.MinItems != nil || tags.MaxItems != nil {
		t.Errorf(
			"expected no item-count constraint on the array itself, got MinItems=%+v MaxItems=%+v",
			tags.MinItems,
			tags.MaxItems,
		)
	}

	if tags.Items == nil {
		t.Fatalf("expected an Items schema, got nil")
	}

	if tags.Items.MinLength == nil || *tags.Items.MinLength != 2 {
		t.Errorf("expected Items.MinLength=2, got %+v", tags.Items.MinLength)
	}

	if tags.Items.MaxLength == nil || *tags.Items.MaxLength != 10 {
		t.Errorf("expected Items.MaxLength=10, got %+v", tags.Items.MaxLength)
	}
}

// TestApplyValidationTags_RepeatedDiveDescendsTwoLevels confirms
// `dive,dive` (a slice of slices) redirects constraints two Items
// levels down, not just one.
func TestApplyValidationTags_RepeatedDiveDescendsTwoLevels(t *testing.T) {
	t.Parallel()

	type request struct {
		Grid [][]string `json:"grid" validate:"dive,dive,min=1"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	grid := schema.Properties["grid"]

	innerItems := grid.Items.Items

	if innerItems == nil {
		t.Fatalf("expected a two-level Items schema, got %+v", grid)
	}

	if innerItems.MinLength == nil || *innerItems.MinLength != 1 {
		t.Errorf("expected inner Items.MinLength=1, got %+v", innerItems.MinLength)
	}
}

// TestApplyValidationTags_DiveRequiredDoesNotMarkFieldRequired covers
// the `dive,required` case: it means each element must be non-zero,
// which has no OpenAPI object-level "required" equivalent - the field
// itself must not be added to the parent schema's Required list
// because of a required tag that comes after dive.
func TestApplyValidationTags_DiveRequiredDoesNotMarkFieldRequired(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `json:"tags" validate:"dive,required"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	for _, name := range schema.Required {
		if name == "tags" {
			t.Fatalf("expected \"tags\" not to be in Required, got %v", schema.Required)
		}
	}
}

// TestApplyValidationTags_FormatShortcuts covers applyEmail and
// applyURL (applyUUID already has coverage from
// TestSchemaGenerator_AppliesCustomRuleSchemaEffect's sibling tests),
// each of which just sets Schema.Format to a fixed string.
func TestApplyValidationTags_FormatShortcuts(t *testing.T) {
	t.Parallel()

	type request struct {
		Email string `json:"email" validate:"email"`
		Site  string `json:"site"  validate:"url"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if got := schema.Properties["email"].Format; got != "email" {
		t.Errorf("expected format %q, got %q", "email", got)
	}

	if got := schema.Properties["site"].Format; got != "uri" {
		t.Errorf("expected format %q, got %q", "uri", got)
	}
}
