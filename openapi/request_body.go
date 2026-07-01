package openapi

// RequestBody describes an HTTP request body.
type RequestBody struct {
	// A brief description of the request body. This could contain examples of use.
	// [CommonMark] syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`

	// REQUIRED. The content of the request body. The key is a media type or media type
	// range and the value describes it. The map SHOULD have at least one entry; if it
	// does not, the behavior is implementation-defined. For requests that match multiple
	// keys, only the most specific key is applicable. e.g. `"text/plain"` overrides `"text/*"`.
	Content map[string]MediaType `json:"content"`

	// Determines if the request body is required in the request. Defaults to `false`.
	Required bool `json:"required,omitempty"`
}

// MediaType describes request or response media.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}
