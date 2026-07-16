// Package users provides the sample typed handlers shared by every
// runnable example under examples/cmd - the same handlers are mounted
// whether wrapped in no middleware at all (examples/cmd/basic) or the
// full stack (examples/cmd/middleware).
package users

import (
	"context"
	"log/slog"
)

// CreateUserRequest is the request body for CreateUser.
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,notblank"`
	Email string `json:"email" validate:"required,email"`
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
