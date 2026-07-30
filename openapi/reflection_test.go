package openapi_test

import (
	"testing"

	"github.com/casara/arnon/openapi"
)

func TestGenerateSchema_Primitives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		value    any
		wantType string
	}{
		{"string", "", "string"},
		{"bool", false, "boolean"},
		{"int", 0, "integer"},
		{"float", 0.0, "number"},
	}

	generator := openapi.NewSchemaGenerator()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := generator.GenerateSchema(testCase.value)

			if schema.Type != testCase.wantType {
				t.Errorf("expected type %q, got %q", testCase.wantType, schema.Type)
			}
		})
	}
}

func TestGenerateSchema_SliceHasItemsSchema(t *testing.T) {
	t.Parallel()

	schema := openapi.NewSchemaGenerator().GenerateSchema([]string{})

	if schema.Type != "array" {
		t.Fatalf("expected type array, got %q", schema.Type)
	}

	if schema.Items == nil || schema.Items.Type != "string" {
		t.Fatalf("expected items schema of type string, got %+v", schema.Items)
	}
}

func TestGenerateSchema_MapHasAdditionalPropertiesSchema(t *testing.T) {
	t.Parallel()

	schema := openapi.NewSchemaGenerator().GenerateSchema(map[string]int{})

	if schema.Type != "object" {
		t.Fatalf("expected type object, got %q", schema.Type)
	}

	if schema.AdditionalProperties == nil || schema.AdditionalProperties.Type != "integer" {
		t.Fatalf(
			"expected additionalProperties schema of type integer, got %+v",
			schema.AdditionalProperties,
		)
	}
}

func TestGenerateSchema_NestedStruct(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `json:"city" validate:"required"`
	}

	type person struct {
		Name    string  `json:"name"    validate:"required"`
		Address address `json:"address"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(person{})

	if schema.Type != "object" {
		t.Fatalf("expected type object, got %q", schema.Type)
	}

	if _, ok := schema.Properties["name"]; !ok {
		t.Fatalf("expected property %q, got %+v", "name", schema.Properties)
	}

	addressSchema, ok := schema.Properties["address"]
	if !ok {
		t.Fatalf("expected property %q, got %+v", "address", schema.Properties)
	}

	if _, ok := addressSchema.Properties["city"]; !ok {
		t.Fatalf("expected nested property %q, got %+v", "city", addressSchema.Properties)
	}

	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("expected required=[name], got %v", schema.Required)
	}
}

func TestGenerateSchema_PointerFieldIsNullable(t *testing.T) {
	t.Parallel()

	type request struct {
		Nickname *string `json:"nickname"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	nickname, ok := schema.Properties["nickname"]
	if !ok {
		t.Fatalf("expected property %q, got %+v", "nickname", schema.Properties)
	}

	if !nickname.Nullable {
		t.Errorf("expected nickname schema to be nullable")
	}
}

func TestGenerateSchema_ExplicitFormatTagWinsOverInference(t *testing.T) {
	t.Parallel()

	type request struct {
		ID string `format:"custom-id" json:"id" validate:"uuid"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	id, ok := schema.Properties["id"]
	if !ok {
		t.Fatalf("expected property %q, got %+v", "id", schema.Properties)
	}

	if id.Format != "custom-id" {
		t.Errorf("expected explicit format to win, got %q", id.Format)
	}
}

func TestGenerateSchema_FormatInferredFromValidatorWhenTagAbsent(t *testing.T) {
	t.Parallel()

	type request struct {
		CreatedAt string `json:"createdAt" validate:"datetime"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	createdAt, ok := schema.Properties["createdAt"]
	if !ok {
		t.Fatalf("expected property %q, got %+v", "createdAt", schema.Properties)
	}

	if createdAt.Format != "date-time" {
		t.Errorf("expected inferred format date-time, got %q", createdAt.Format)
	}
}

func TestGenerateSchema_ExampleAndDefaultAreTypeConverted(t *testing.T) {
	t.Parallel()

	type request struct {
		Limit int `default:"20" example:"1" json:"limit"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	limit, ok := schema.Properties["limit"]
	if !ok {
		t.Fatalf("expected property %q, got %+v", "limit", schema.Properties)
	}

	if limit.Example != int64(1) {
		t.Errorf("expected example int64(1), got %#v", limit.Example)
	}

	if limit.Default != int64(20) {
		t.Errorf("expected default int64(20), got %#v", limit.Default)
	}
}

func TestGenerateSchema_DeprecatedReadOnlyWriteOnlyFlags(t *testing.T) {
	t.Parallel()

	type request struct {
		Legacy   string `json:"legacy"   deprecated:"true"`
		ServerID string `json:"serverId"                   readonly:"true"`
		Password string `json:"password"                                   writeonly:"true"`
	}

	schema := openapi.NewSchemaGenerator().GenerateSchema(request{})

	if !schema.Properties["legacy"].Deprecated {
		t.Error("expected legacy to be deprecated")
	}

	if !schema.Properties["serverId"].ReadOnly {
		t.Error("expected serverId to be readOnly")
	}

	if !schema.Properties["password"].WriteOnly {
		t.Error("expected password to be writeOnly")
	}
}
