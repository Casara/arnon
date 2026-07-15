// Package users provides the sample "create a user" endpoint shared
// by every runnable example under examples/cmd - the same typed
// handler is mounted whether it's wrapped in no middleware at all
// (examples/cmd/basic) or the full stack (examples/cmd/middleware).
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
