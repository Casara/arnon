package openapi_test

import (
	"testing"

	"github.com/casara/arnon/openapi"
)

// TestGenerateSchema_UnexportedFieldIsSkipped confirms an unexported
// struct field is skipped entirely (parseField returns nil for it, per
// reflect.StructField.IsExported), rather than causing a panic or
// appearing in Properties.
func TestGenerateSchema_UnexportedFieldIsSkipped(t *testing.T) {
	t.Parallel()

	type request struct {
		Name     string `json:"name"`
		internal string // deliberately unexported, see comment above
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{internal: "secret"})

	if _, ok := schema.Properties["internal"]; ok {
		t.Error("expected the unexported field not to appear in Properties")
	}

	if len(schema.Properties) != 1 {
		t.Errorf(
			"expected exactly 1 property, got %d: %+v",
			len(schema.Properties),
			schema.Properties,
		)
	}
}

// TestGenerateSchema_NonBodyFieldsAreExcludedFromProperties confirms
// generateStruct's `result.Location != "body"` guard: fields tagged
// path/query/header describe parameters (handled separately by
// Generator.addParameters), not the JSON body schema, so
// GenerateSchema must not put them in Properties.
func TestGenerateSchema_NonBodyFieldsAreExcludedFromProperties(t *testing.T) {
	t.Parallel()

	type request struct {
		ID     string `json:"-"    path:"id"`
		Limit  int    `json:"-"              query:"limit"`
		APIKey string `json:"-"                            header:"X-Api-Key"`
		Name   string `json:"name"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	for _, field := range []string{"id", "limit", "X-Api-Key"} {
		if _, ok := schema.Properties[field]; ok {
			t.Errorf(
				"expected %q not to appear in body Properties, got %+v",
				field,
				schema.Properties,
			)
		}
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Errorf(
			"expected body field %q to appear in Properties, got %+v",
			"name",
			schema.Properties,
		)
	}
}

// TestGenerateSchema_UnsupportedKindFallsBackToString covers
// generateType's default arm: a Go kind with no explicit mapping (a
// channel, here) is represented as an opaque "string" schema rather
// than panicking.
func TestGenerateSchema_UnsupportedKindFallsBackToString(t *testing.T) {
	t.Parallel()

	type request struct {
		Signal chan int `json:"signal"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if got := schema.Properties["signal"].Type; got != "string" {
		t.Errorf("expected unsupported kind to fall back to type %q, got %q", "string", got)
	}
}

// TestGenerateSchema_NilTypeReturnsEmptySchema covers generateType's
// nil-Type guard, reached when GenerateSchema is called with a nil
// interface value (reflect.TypeOf(nil) is nil).
func TestGenerateSchema_NilTypeReturnsEmptySchema(t *testing.T) {
	t.Parallel()

	schema := openapi.NewSchemaGenerator().GenerateSchema(nil)

	if schema.Type != "" {
		t.Errorf("expected an empty Schema for a nil value, got %+v", schema)
	}
}

// TestGetJSONName_TagVariants covers getJSONName's remaining branches
// beyond the plain `json:"name"` case already exercised elsewhere:
// `json:"-"` (field excluded, handled by getFieldLocation returning an
// empty name), no json tag at all (falls back to the lowercased Go
// field name), and a tag that starts with a comma (e.g.
// `json:",omitempty"`, whose split-by-comma name is empty and falls
// back the same way).
func TestGetJSONName_TagVariants(t *testing.T) {
	t.Parallel()

	type request struct {
		Excluded string `json:"-"`
		NoTag    string
		// tagliatelle checks tag case against the Go field name when the
		// tag has no explicit name portion; the mismatch is the point of
		// this field (it exercises getJSONName's comma-only fallback).
		OmitemptyOnly string `json:",omitempty"` //nolint:tagliatelle // see comment above
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if _, ok := schema.Properties["Excluded"]; ok {
		t.Error("expected json:\"-\" field to be excluded from Properties")
	}

	if _, ok := schema.Properties["-"]; ok {
		t.Error("expected json:\"-\" field not to be registered under the literal tag value")
	}

	if _, ok := schema.Properties["notag"]; !ok {
		t.Errorf(
			"expected the untagged field to fall back to its lowercased name, got %+v",
			schema.Properties,
		)
	}

	if _, ok := schema.Properties["omitemptyonly"]; !ok {
		t.Errorf(
			"expected a `json:\",omitempty\"` field to fall back to its lowercased name, got %+v",
			schema.Properties,
		)
	}
}
