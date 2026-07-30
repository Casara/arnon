package routing

import "net/http"

func (router *Router) registerOpenAPI(
	pattern string,
	handler http.Handler,
) {
	if router.openAPIRegistry == nil {
		return
	}

	provider, ok := handler.(OpenAPIProvider)

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
