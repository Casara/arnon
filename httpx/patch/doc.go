// Package patch derives a PATCH http.Handler from an existing GET and
// PUT handler for the same resource, supporting RFC 7386 (JSON Merge
// Patch) and RFC 6902 (JSON Patch), selected by the incoming
// request's Content-Type.
//
// It works by request replay, not by teaching the resource's typed
// request/response structs anything about partial updates: an
// internal GET fetches the resource's current JSON representation,
// the patch is applied directly to those raw bytes
// (github.com/evanphx/json-patch/v5), and the result is replayed as
// an internal PUT - reusing that handler's existing binding,
// sanitization, and validation entirely unchanged. Neither the GET
// nor the PUT handler needs to know PATCH exists.
//
// Operating on raw JSON bytes, rather than decoding the patch body
// straight into the resource's Go struct, sidesteps a real Go
// limitation: a plain struct field can't distinguish "omitted" from
// "explicit null" (json.Unmarshal into a *string produces nil for
// both `{}` and `{"name":null}`), which RFC 7386 requires (omitted =
// leave alone, null = delete). It also means RFC 6902 fits the same
// mechanism for free: its body (an operation list) was never going to
// decode into the resource's struct in the first place.
package patch
