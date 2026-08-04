// Package users provides the sample typed handlers shared by every
// runnable example under examples/cmd - the same handlers are mounted
// whether wrapped in no middleware at all (examples/cmd/basic) or the
// full stack (examples/cmd/middleware).
package users

import (
	"context"
	"log/slog"
)

// Address is a nested request field, used only to demonstrate that a
// validation error inside a nested struct resolves to a full RFC 6901
// pointer (e.g. "/address/city"), not just "/city" -
// validation.buildFieldMap recurses into nested body structs to
// compose that path.
type Address struct {
	City string `json:"city" validate:"required"`
}

// CreateUserRequest is the request body for CreateUser. Address is a
// pointer: go-playground/validator only dives into a nested struct
// field when it's non-nil, so a request that omits "address" entirely
// isn't forced through its validation. Tags demonstrates the other
// nesting case, `dive` into a slice of primitives: a validation error
// on a specific element (e.g. Tags[1]) resolves to the RFC 6901
// pointer "/tags/1", with the failing element's actual index, not a
// single "/tags" segment. Email demonstrates sanitize:"email" (trim +
// lowercase, applied before validation runs): validator's own "email"
// tag rejects a value with surrounding whitespace outright (confirmed
// empirically - a very common data-entry mistake, e.g. a copy-pasted
// address), so without the sanitize tag " ada@example.com " would
// fail validation instead of being accepted like any other email.
type CreateUserRequest struct {
	Name    string   `json:"name"    validate:"required,notblank"`
	Email   string   `json:"email"   validate:"required,email"       sanitize:"email"`
	Address *Address `json:"address"`
	Tags    []string `json:"tags"    validate:"omitempty,dive,min=2"`
}

// CreateUserResponse is the response body for CreateUser.
type CreateUserResponse struct {
	ID string `json:"id"`
}

// CreateUser is a minimal typed handler: it only demonstrates binding
// and validation, so it always succeeds once CreateUserRequest
// passes validation.
func CreateUser(
	_ context.Context,
	request CreateUserRequest,
) (CreateUserResponse, error) {
	slog.Debug("creating user", "name", request.Name, "email", request.Email)

	return CreateUserResponse{ID: "usr_123"}, nil
}

// GetUserRequest is the request for GetUser. It demonstrates path
// binding (ID), and []string binding from both query (Tags - repeated
// keys only, e.g. "?tags=a&tags=b") and header (Locales -
// Accept-Language, which arrives either as repeated header lines or a
// single comma-joined one; both are collected the same way, per
// RFC 9110 §5.3). Mixes single-purpose tags on purpose, same as the
// _test.go structs excluded from tagalign in .golangci.yml.
//
//nolint:tagalign // see doc comment above
type GetUserRequest struct {
	ID      string   `path:"id"`
	Tags    []string `          query:"tags"`
	Locales []string `                       header:"Accept-Language"`
}

// GetUserResponse is the response body for GetUser.
type GetUserResponse struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Locales []string `json:"locales"`
}

// GetUser is a minimal typed handler demonstrating path/query/header
// binding (including the []string case): it just echoes back whatever
// was bound, so it always succeeds.
func GetUser(
	_ context.Context,
	request GetUserRequest,
) (GetUserResponse, error) {
	return GetUserResponse(request), nil
}
