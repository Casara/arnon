package httpx

import (
	"reflect"

	"github.com/Casara/arnon/openapi"
)

// OpenAPIProvider exposes OpenAPI operations that can be registered
// in the generated API specification.
//
// Handlers may implement this interface to contribute their own
// OpenAPI definitions without coupling the router to a specific
// documentation implementation.
type OpenAPIProvider interface {
	OpenAPIOperation() *openapi.Operation

	RequestType() reflect.Type

	ResponseType() reflect.Type
}
