// Package binding decodes an HTTP request into a typed struct.
//
// Decode reads four sources, each selected by a struct tag on the destination
// field:
//
//	Field string `path:"id"`               // net/http path parameter
//	Field string `query:"page"`            // URL query
//	Field string `header:"X-Tenant"`       // request header
//	Field string `json:"name"`             // JSON request body
//
// Conversion to the field's Go type is handled for the usual scalars, their
// pointers, and slices of them. A value that cannot be converted does not stop
// the decode: it becomes a problem.ValidationError and the remaining fields
// are still processed, so one round trip reports every problem the request
// has rather than the first one found.
//
// That is also why Decode returns []problem.ValidationError rather than an
// error. The slice is empty on success. This deliberately does not fit the
// usual `if err != nil` shape, because "the first failure" is the wrong answer
// for request binding - a client fixing one field at a time is a bad API.
//
// # Multi-value fields
//
// A []string bound from a header treats repeated lines and a single
// comma-separated value as equivalent, because RFC 9110 §5.3 says they are.
// A []string bound from the query does not: it collects a repeated key
// (?tag=a&tag=b) and never splits on commas, since no RFC defines that for
// query strings and splitting would corrupt a legitimate value like
// ?q=cats,dogs.
//
// httpx.Endpoint calls this package for you; use it directly only when
// writing a plain http.Handler that wants the same binding rules.
package binding
