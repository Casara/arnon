package openapi

// ExternalDocs allows referencing an external resource for extended documentation.
//
// Example:
//
//	ExternalDocs: &openapi.ExternalDocs{
//		Description: "Find more info here",
//		URL:  "https://example.com"
//	}
type ExternalDocs struct {
	// A description of the target documentation. [CommonMark] syntax MAY be used for
	// rich text representation.
	Description string `json:"description,omitempty"`

	// REQUIRED. The URI for the target documentation. This MUST be in the form of a URI.
	URL string `json:"url"`
}
