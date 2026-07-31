package routing

import "net/http"

// Wrap builds the handler a middleware returns, keeping the handler it
// decorates reachable.
//
// A middleware normally returns a closure, and a closure hides everything
// about what it wraps. That matters here for one reason: the router discovers
// which routes to document by asking the handler whether it implements
// OpenAPIProvider, and a closure never does. Wrapping a typed endpoint by hand
// would therefore drop it out of the generated document, silently, while the
// route kept working.
//
// Using this instead of a bare http.HandlerFunc keeps that from happening:
//
//	func Example() routing.Middleware {
//		return func(next http.Handler) http.Handler {
//			return routing.Wrap(next, http.HandlerFunc(func(
//				writer http.ResponseWriter,
//				request *http.Request,
//			) {
//				next.ServeHTTP(writer, request)
//			}))
//		}
//	}
//
// The convention mirrors errors.Unwrap: the result exposes Unwrap()
// http.Handler, and the router follows that chain to find the endpoint
// underneath, however many middleware deep it is. Third-party middleware that
// returns a plain closure still hides what it wraps — nothing can be done
// about that from here, and it is why installing per-route middleware on a
// Group remains the more robust option.
func Wrap(next, handler http.Handler) http.Handler {
	return wrappedHandler{
		Handler: handler,

		next: next,
	}
}

// wrappedHandler serves handler while keeping next reachable through Unwrap.
type wrappedHandler struct {
	http.Handler

	next http.Handler
}

// Unwrap returns the handler this one decorates.
func (wrapped wrappedHandler) Unwrap() http.Handler {
	return wrapped.next
}

// unwrapToProvider walks a chain of Wrap-built handlers looking for one that
// describes itself.
//
//nolint:ireturn // returning the interface is the point here
func unwrapToProvider(handler http.Handler) (OpenAPIProvider, bool) {
	// Bounded so a middleware that somehow unwraps to itself cannot spin
	// forever during route registration.
	const maxDepth = 32

	for depth := 0; handler != nil && depth < maxDepth; depth++ {
		if provider, ok := handler.(OpenAPIProvider); ok {
			return provider, true
		}

		unwrapper, ok := handler.(interface{ Unwrap() http.Handler })
		if !ok {
			return nil, false
		}

		handler = unwrapper.Unwrap()
	}

	return nil, false
}
