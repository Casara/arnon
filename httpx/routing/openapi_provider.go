package routing

import (
	"reflect"

	"github.com/casara/arnon/openapi"
)

// OpenAPIProvider is implemented by a handler that can describe itself in the
// generated OpenAPI document. Handlers built with httpx.Endpoint implement it
// when EndpointConfig.OpenAPI is set.
//
// It is declared here, in the package that consumes it, rather than next to
// the handlers that satisfy it. Go interfaces are structural, so nothing needs
// to import this to implement it - which is what keeps httpx free of any
// dependency on the router.
type OpenAPIProvider interface {
	// OpenAPIOperation returns the operation to register, or nil to stay out
	// of the document. This is what makes documentation opt-in per route.
	OpenAPIOperation() *openapi.Operation

	// RequestType and ResponseType are the Go types the schemas are generated
	// from by reflection.
	RequestType() reflect.Type
	ResponseType() reflect.Type
}

// OpenAPIRegistrar receives the operations a Router discovers. *openapi.Generator
// satisfies it; the interface exists so the router depends on the shape it
// needs rather than on a concrete generator, which is what lets openapi stay
// usable with any other router.
type OpenAPIRegistrar interface {
	RegisterTypes(
		method string,
		path string,
		operation openapi.Operation,
		requestType reflect.Type,
		responseType reflect.Type,
	)
}
