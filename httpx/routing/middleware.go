package routing

import "net/http"

// Middleware decorates an HTTP handler.
type Middleware func(
	http.Handler,
) http.Handler
