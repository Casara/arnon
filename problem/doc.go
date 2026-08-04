// Package problem implements RFC 9457 Problem Details, the single error
// format this framework produces.
//
// A Problem carries the standard members - type, title, status, detail,
// instance - and serializes with every zero-valued one omitted. It is also an
// error, so a handler can return one directly and callers can recover it with
// errors.As.
//
// The twelve named constructors (NewNotFound, NewConflict, ...) fill in the
// title from the status, which is what keeps the wording consistent across an
// API. New covers any other status, at the cost of writing the title yourself.
//
// # Extensions
//
// With adds an RFC 9457 extension member, serialized at the top level of the
// object rather than nested under a wrapper:
//
//	problem.NewTooManyRequests("slow down").With("retry_after_seconds", 30)
//
// It panics on an empty key or one colliding with a standard member, since
// both are written in the source rather than received from a client.
//
// # Validation errors
//
// The errors member carries a []ValidationError, one per offending field,
// each addressing the field with an RFC 6901 JSON Pointer and a stable
// machine-readable code. That pointer is what lets a client highlight the
// exact field - including inside a nested struct, a slice element or a map
// entry - instead of parsing a human sentence.
//
// This package depends on nothing else in the framework, so anything may
// import it.
package problem
