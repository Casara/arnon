package openapi

// Operation describes a single API operation on a path.
type Operation struct {
	Tags []Tag `json:"-"`

	// A list of tags for API documentation control. Tags can be used for logical grouping
	// of operations by resources or any other qualifier.
	TagNames []string `json:"tags,omitempty"`

	// A short summary of what the operation does.
	Summary string `json:"summary,omitempty"`

	// A verbose explanation of the operation behavior. [CommonMark] syntax MAY be used
	// for rich text representation.
	Description string `json:"description,omitempty"`

	// Unique string used to identify the operation. The id MUST be unique among all operations
	// described in the API. The operationId value is case-sensitive. Tools and libraries MAY use
	// the operationId to uniquely identify an operation, therefore, it is RECOMMENDED to follow
	// common programming naming conventions.
	OperationID string `json:"operationId,omitempty"`

	// A list of parameters that are applicable for this operation. If a parameter is already
	// defined at the Path Item, the new definition will override it but can never remove it.
	// The list MUST NOT include duplicated parameters. A unique parameter is defined by a
	// combination of a name and location. The list can use the Reference Object to link to
	// parameters that are defined in the OpenAPI Object's `components.parameters`.
	Parameters []Parameter `json:"parameters,omitempty"`

	// The request body applicable for this operation. The `requestBody` is fully supported in
	// HTTP methods where the HTTP specification [RFC9110] Section 9.3 has explicitly defined
	// semantics for request bodies. In other cases where the HTTP spec discourages message
	// content (such as GET and DELETE), `requestBody` is permitted but does not have well-defined
	// semantics and SHOULD be avoided if possible.
	RequestBody *RequestBody `json:"requestBody,omitempty"`

	// The list of possible responses as they are returned from executing this operation.
	Responses Responses `json:"responses,omitempty"`

	// Declares this operation to be deprecated. Consumers SHOULD refrain from usage of the
	// declared operation. Default value is `false`.
	Deprecated bool `json:"deprecated,omitempty"`

	// SuccessStatus is the status the success response is documented under.
	// Not part of the OpenAPI specification - it is how this package knows
	// which status to attach the response schema to, hence json:"-".
	//
	// httpx.Endpoint fills it in from EndpointConfig.SuccessStatus, so the
	// document describes the status the handler actually returns without you
	// stating it twice. Set it by hand only to document something other than
	// what the handler does, which is almost never what you want.
	SuccessStatus int `json:"-"`
}
