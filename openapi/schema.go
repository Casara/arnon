package openapi

import (
	"encoding/json"
	"fmt"
)

// Schema represents an OpenAPI schema.
type Schema struct {
	Ref string `json:"$ref,omitempty"`

	Type string `json:"type,omitempty"`

	Format string `json:"format,omitempty"`

	Description string `json:"description,omitempty"`

	// Nullable marks the value as accepting null in addition to Type.
	//
	// It is not serialized under this name: OpenAPI 3.1 removed the 3.0
	// "nullable" keyword, and a 3.1+ parser silently ignores it, which would
	// lose the information entirely. It is emitted the way the current
	// specification expects instead - as a type array, `"type": ["string",
	// "null"]` - by MarshalJSON below.
	Nullable bool `json:"-"`

	Deprecated bool `json:"deprecated,omitempty"`

	ReadOnly bool `json:"readOnly,omitempty"`

	WriteOnly bool `json:"writeOnly,omitempty"`

	Default any `json:"default,omitempty"`

	Example any `json:"example,omitempty"`

	Enum []any `json:"enum,omitempty"`

	Required []string `json:"required,omitempty"`

	Properties map[string]*Schema `json:"properties,omitempty"`

	AdditionalProperties *Schema `json:"additionalProperties,omitempty"`

	Items *Schema `json:"items,omitempty"`

	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`

	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`

	Minimum *float64 `json:"minimum,omitempty"`

	Maximum *float64 `json:"maximum,omitempty"`

	MinLength *int `json:"minLength,omitempty"`

	MaxLength *int `json:"maxLength,omitempty"`

	MinItems *int `json:"minItems,omitempty"`

	MaxItems *int `json:"maxItems,omitempty"`

	Pattern string `json:"pattern,omitempty"`
}

// MarshalJSON renders the schema, expressing Nullable as an OpenAPI 3.1+ type
// array rather than the 3.0 "nullable" keyword the specification removed.
//
// A schema with no Type - a bare $ref, most commonly - has nothing to widen,
// so Nullable is dropped rather than emitting a meaningless `["null"]`.
func (schema Schema) MarshalJSON() ([]byte, error) {
	// Sheds the MarshalJSON method, so the json package encodes the fields
	// instead of recursing into this function forever.
	type plainSchema Schema

	if !schema.Nullable || schema.Type == "" {
		encoded, err := json.Marshal(plainSchema(schema))
		if err != nil {
			return nil, fmt.Errorf("marshal schema: %w", err)
		}

		return encoded, nil
	}

	widened := plainSchema(schema)
	widened.Type = ""

	// The outer Type shadows the embedded one regardless of declaration order
	// (encoding/json prefers the shallower field), and the embedded one is
	// blanked above so omitempty drops it either way.
	encoded, err := json.Marshal(struct {
		plainSchema

		Type []string `json:"type,omitempty"`
	}{
		plainSchema: widened,

		Type: []string{schema.Type, "null"},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal nullable schema: %w", err)
	}

	return encoded, nil
}
