// Package precondition validates the write preconditions of RFC 9110
// §13.1.1 (If-Match) and §13.1.4 (If-Unmodified-Since) against a
// resource's actual current state, returning a 412 Precondition
// Failed (or, opt-in, a 428 Precondition Required) problem.Problem
// when the check fails.
//
// Unlike middleware.ETag - which buffers a GET/HEAD response and
// handles If-None-Match entirely on its own, since a response's bytes
// are all it needs - a write precondition can't be evaluated by
// generic middleware: only the resource layer (whatever loads the
// current row/record to apply a PUT/PATCH/DELETE) knows the
// resource's current ETag/modification time at the moment of the
// write. Check/CheckRequest are meant to be called by that layer
// directly, mirroring huma's danielgtaylor/huma/v2/conditional
// package (its PreconditionFailed(etag, modified) is likewise called
// by the handler, not computed by the framework).
package precondition
