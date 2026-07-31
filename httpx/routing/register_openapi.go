package routing

import "net/http"

func (router *Router) registerOpenAPI(
	pattern string,
	handler http.Handler,
) {
	if router.openAPIRegistry == nil {
		return
	}

	// Walks the middleware chain rather than asserting on the outermost
	// handler: wrapping an endpoint by hand would otherwise drop it out of the
	// document with no error. See Wrap.
	provider, ok := unwrapToProvider(handler)

	if !ok {
		return
	}

	operation := provider.OpenAPIOperation()

	if operation == nil {
		return
	}

	method, path, err := splitPattern(pattern)
	if err != nil {
		panic(err)
	}

	router.
		openAPIRegistry.
		RegisterTypes(
			method,
			path,
			*operation,
			provider.
				RequestType(),
			provider.
				ResponseType(),
		)
}
