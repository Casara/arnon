package openapi

// Parameter describes a single operation parameter.
//
// A unique parameter is defined by a combination of a name and location.
type Parameter struct {
	// REQUIRED. The name of the parameter. Parameter names are case-sensitive.
	Name string `json:"name"`

	// REQUIRED. The location of the parameter. Possible values are
	// `"body"`, `"path"`, `"query"`, `"header"`.
	In string `json:"in"`

	// A brief description of the parameter. This could contain examples of use.
	// [CommonMark] syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`

	// Determines whether this parameter is mandatory. If the parameter location
	// is `"path"`, this field is REQUIRED and its value MUST be `true`. Otherwise,
	// the field MAY be included and its default value is `false`.
	Required bool `json:"required,omitempty"`

	// Specifies that a parameter is deprecated and SHOULD be transitioned out of usage.
	// Default value is `false`.
	Deprecated bool `json:"deprecated,omitempty"`

	// The schema defining the type used for the parameter.
	Schema *Schema `json:"schema,omitempty"`
}
