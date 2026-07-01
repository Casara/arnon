package openapi

// Responses describes endpoint responses.
type Responses map[string]Response

// Response describes a single response from an API operation, including
// design-time, static `links` to operations based on the response.
type Response struct {
	// A short summary of the meaning of the response.
	Summary string `json:"summary,omitempty"`

	// A description of the response. [CommonMark] syntax MAY be used for
	// rich text representation.
	Description string `json:"description"`

	// Maps a header name to its definition. [RFC9110] Section 5.1 states
	// header names are case-insensitive. If a response header is defined
	// with the name `"Content-Type"`, it SHALL be ignored.
	Headers map[string]Header `json:"headers,omitempty"`

	// A map containing descriptions of potential response payloads. The key
	// is a media type or media type range and the value describes it. For
	// responses that match multiple keys, only the most specific key is
	// applicable. e.g. `"text/plain"` overrides `"text/*"`.
	Content map[string]MediaType `json:"content,omitempty"`

	// A map of operations links that can be followed from the response.
	// The key of the map is a short name for the link, following the naming
	// constraints of the names for Component Objects.
	Links map[string]Link `json:"links,omitempty"`
}

// Header describes a single header for HTTP responses.
type Header struct {
	// A brief description of the header. This could contain examples of use.
	// [CommonMark] syntax MAY be used for rich text representation.
	Description string `json:"description"`

	// Determines whether this header is mandatory. The default value is `false`.
	Required []string `json:"required,omitempty"`

	// Specifies that the header is deprecated and SHOULD be transitioned out of
	// usage. Default value is `false`.
	Deprecated bool `json:"deprecated,omitempty"`
}

// Link represents a possible design-time link for a response.
type Link struct {
	// A URI reference to an OAS operation. This field is mutually exclusive of
	// the `operationId` field, and MUST point to an Operation Object. Relative
	// `operationRef` values MAY be used to locate an existing Operation Object
	// in the OpenAPI Description.
	OperationRef string `json:"operationRef,omitempty"`

	// The name of an existing, resolvable OAS operation, as defined with a unique
	// `operationId`. This field is mutually exclusive of the `operationRef` field.
	OperationID string `json:"operationId,omitempty"`

	// A description of the link. [CommonMark] syntax MAY be used for rich
	// text representation.
	Description string `json:"description"`
}
