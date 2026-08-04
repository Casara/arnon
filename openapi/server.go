package openapi

// Server represents a server hosting the API, e.g. a base URL
// clients should send requests to.
//
// Example:
//
//	Server{
//		URL:         "https://api.example.com/v1",
//		Description: "Production",
//	}
type Server struct {
	// REQUIRED. A URL to the target host. This URL supports Server
	// Variables and MAY be relative, to indicate that the host location
	// is relative to the location where the OpenAPI document is being
	// served. Variable substitutions will be made when a variable is
	// named in `{brackets}`.
	URL string `json:"url"`

	// An optional string describing the host designated by the URL.
	// [CommonMark] syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`

	// A map between a variable name and its value. The value is used
	// for substitution in the server's URL template.
	Variables map[string]ServerVariable `json:"variables,omitempty"`
}

// ServerVariable describes a substitution value for a Server URL
// template.
type ServerVariable struct {
	// An enumeration of string values to be used if the substitution
	// options are from a limited set. The array MUST NOT be empty.
	Enum []string `json:"enum,omitempty"`

	// REQUIRED. The default value to use for substitution, which SHALL
	// be sent if an alternate value is not supplied. If Enum is
	// present, Default MUST be one of its values.
	Default string `json:"default"`

	// An optional description for the server variable. [CommonMark]
	// syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`
}
