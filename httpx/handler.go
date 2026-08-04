package httpx

import "context"

// HandlerFunc handles an HTTP request DTO.
type HandlerFunc[TRequest any, TResponse any] func(
	context.Context,
	TRequest,
) (TResponse, error)
