// Package httpx turns a typed Go function into an http.Handler.
//
// Endpoint is the entry point. Given a handler with the signature
// func(context.Context, TRequest) (TResponse, error), it produces an
// http.Handler that, on every request:
//
//  1. binds the request into TRequest - path, query and header values from
//     struct tags, the JSON body from the json tags (see httpx/binding);
//  2. sanitizes it, applying the transforms named in each sanitize tag
//     (see the sanitize package);
//  3. validates it against the validate tags (see the validation package);
//  4. calls the handler;
//  5. writes TResponse as JSON on success, or an RFC 9457 problem document
//     on any failure.
//
// Every error path produces a problem.Problem: a binding or validation
// failure becomes a 400 - or whatever status the failure's own code implies -
// listing each offending field with an RFC 6901 JSON Pointer; an error from
// the handler goes through EndpointConfig.ProblemMapper, which returns the
// handler's own *problem.Problem unchanged and turns anything else into a 500
// without leaking its message.
//
// The zero EndpointConfig works: it installs the default validator, the
// default problem mapper and a 200 success status. Setting
// EndpointConfig.OpenAPI - even to an empty operation - is what opts the route
// into the generated OpenAPI document; registration is per endpoint, never
// automatic.
//
// # Scope
//
// Endpoint is JSON-only by design. It answers 406 when a client explicitly
// excludes application/json, but it does not negotiate between
// representations, and it is not meant to grow that. Anything serving XML, a
// PDF or a file is a plain http.Handler mounted on the router like any other
// route - no middleware in this framework is coupled to JSON.
//
// Routing lives in httpx/routing, middleware in httpx/middleware.
package httpx
