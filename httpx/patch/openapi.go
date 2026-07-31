package patch

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/casara/arnon/openapi"
)

// typedHandler is what an httpx.Endpoint exposes about itself. Declared here
// rather than imported from httpx/routing so that this package gains no
// dependency on the router: Go interfaces are structural, so an Endpoint
// satisfies both without knowing either exists.
type typedHandler interface {
	OpenAPIOperation() *openapi.Operation
	RequestType() reflect.Type
	ResponseType() reflect.Type
}

// documentedHandler is the handler From returns when Config.OpenAPI is set. It
// serves the PATCH and, separately, describes it well enough for the router to
// register the operation.
//
// Without this, a derived PATCH route could never appear in the generated
// document: From returns a plain http.Handler, the router discovers operations
// by asking the handler, and a plain handler has nothing to answer.
type documentedHandler struct {
	http.Handler

	operation    *openapi.Operation
	requestType  reflect.Type
	responseType reflect.Type
}

func (handler documentedHandler) OpenAPIOperation() *openapi.Operation {
	return handler.operation
}

func (handler documentedHandler) RequestType() reflect.Type {
	return handler.requestType
}

func (handler documentedHandler) ResponseType() reflect.Type {
	return handler.responseType
}

// describe wraps handler so the router can document it, taking the request and
// response types from put.
//
// The types come from put rather than from the caller because they are already
// there and cannot disagree: a merge patch carries the same fields as the
// replacement body, and the response is whatever put returns. Asking for them
// again would be a second source of truth for something already stated once.
//
// It panics when put cannot supply them, which means put is not an
// httpx.Endpoint - a middleware-wrapped one still works, since arnon's
// middleware keeps the endpoint reachable through routing.Wrap. That is a
// startup-time wiring mistake, and the alternative - registering nothing - is
// precisely the silent disappearance this exists to prevent.
func describe(
	handler http.Handler,
	put http.Handler,
	operation *openapi.Operation,
) http.Handler {
	typed, ok := unwrapToTyped(put)
	if !ok {
		panic(fmt.Sprintf(
			"patch: From cannot document the route: put is %T, which does not "+
				"expose its request and response types. Pass an httpx.Endpoint "+
				"(wrapped in arnon middleware is fine), or leave Config.OpenAPI "+
				"nil to register the route without documenting it.",
			put,
		))
	}

	return documentedHandler{
		Handler: handler,

		operation:    operation,
		requestType:  typed.RequestType(),
		responseType: typed.ResponseType(),
	}
}

// unwrapToTyped finds the typed endpoint under put, walking the Unwrap chain
// that arnon's middleware leaves behind (see routing.Wrap) so that a wrapped
// put still supplies its schemas.
//
//nolint:ireturn // returning the interface is the point here
func unwrapToTyped(handler http.Handler) (typedHandler, bool) {
	const maxDepth = 32

	for depth := 0; handler != nil && depth < maxDepth; depth++ {
		if typed, ok := handler.(typedHandler); ok {
			return typed, true
		}

		unwrapper, ok := handler.(interface{ Unwrap() http.Handler })
		if !ok {
			return nil, false
		}

		handler = unwrapper.Unwrap()
	}

	return nil, false
}
