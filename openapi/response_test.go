package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestHeader_RequiredMarshalsAsBoolean confirms Header.Required is a
// bool per the spec, not []string (Schema.Required's different "list
// of required property names" semantics) - Header is never
// constructed anywhere else in the codebase, so a wrong field type
// here would otherwise go uncaught.
func TestHeader_RequiredMarshalsAsBoolean(t *testing.T) {
	t.Parallel()

	header := openapi.Header{
		Description: "Rate limit ceiling for the current window",
		Required:    true,
		Schema: &openapi.Schema{
			Type: "integer",
		},
	}

	encoded, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]any

	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	required, ok := decoded["required"].(bool)
	if !ok {
		t.Fatalf(
			"expected \"required\" to decode as a JSON boolean, got %+v (%s)",
			decoded["required"],
			encoded,
		)
	}

	if !required {
		t.Errorf("expected required=true, got %+v", decoded["required"])
	}

	schema, ok := decoded["schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected \"schema\" to be present, got %s", encoded)
	}

	if schema["type"] != "integer" {
		t.Errorf(`expected schema.type "integer", got %+v`, schema["type"])
	}
}
