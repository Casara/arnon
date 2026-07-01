package routing

import (
	"net/http"

	"github.com/Casara/arnon/httpx"
)

func (router *Router) registerOpenAPI(
	pattern string,
	handler http.Handler,
) {
	if router.openAPIRegistry == nil {
		return
	}

	provider, ok := handler.(httpx.OpenAPIProvider)

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
		Register(
			method,
			path,
			*operation,
			provider.
				RequestType(),
			provider.
				ResponseType(),
		)
}
