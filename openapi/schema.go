package openapi

// Schema represents an OpenAPI schema.
type Schema struct {
	Ref string `json:"$ref,omitempty"`

	Type string `json:"type,omitempty"`

	Format string `json:"format,omitempty"`

	Description string `json:"description,omitempty"`

	Nullable bool `json:"nullable,omitempty"`

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
