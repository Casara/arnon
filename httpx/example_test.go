package httpx_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
	"github.com/casara/arnon/problem"
)

type createUserRequest struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email" sanitize:"email"`
}

type createUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func createUser(
	_ context.Context,
	request createUserRequest,
) (createUserResponse, error) {
	return createUserResponse{ID: "usr_123", Email: request.Email}, nil
}

// Endpoint turns a typed handler into an http.Handler: it binds the request,
// sanitizes it, validates it, calls the handler and writes the result. Note
// that the email arrives with surrounding whitespace and mixed case and reaches
// the handler already normalized, because of the sanitize tag.
func ExampleEndpoint() {
	handler := httpx.Endpoint(createUser, httpx.EndpointConfig{
		SuccessStatus: http.StatusCreated,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":"Ada Lovelace","email":"  ADA@Example.COM "}`),
	)

	handler.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Println(recorder.Header().Get("Content-Type"))
	fmt.Print(recorder.Body.String())
	// Output:
	// 201
	// application/json; charset=utf-8
	// {"id":"usr_123","email":"ada@example.com"}
}

// A validation failure never reaches the handler. The response is an RFC 9457
// problem document whose errors extension points at the offending field with an
// RFC 6901 JSON Pointer.
func ExampleEndpoint_validationFailure() {
	handler := httpx.Endpoint(createUser, httpx.EndpointConfig{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":"","email":"not-an-email"}`),
	)

	handler.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Println(recorder.Header().Get("Content-Type"))
	// Output:
	// 400
	// application/problem+json; charset=utf-8
}

// A handler that returns a *problem.Problem gets that exact status and body;
// any other error becomes a 500, so an internal failure never leaks its message
// to the client.
func ExampleEndpoint_handlerError() {
	handler := httpx.Endpoint(
		func(_ context.Context, _ createUserRequest) (createUserResponse, error) {
			return createUserResponse{}, problem.NewConflict(
				"an account with this email already exists",
			)
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`),
	)

	handler.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Print(recorder.Body.String())
	// Output:
	// 409
	// {"title":"Conflict","status":409,"detail":"an account with this email already exists","instance":"/users"}
}

// WriteProblem fills Instance from the request path when it is empty, so a
// handwritten http.Handler produces the same shape as a typed Endpoint.
func ExampleWriteProblem() {
	handler := http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		httpx.WriteProblem(writer, request, problem.NewNotFound("no such user"))
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/users/42", nil),
	)

	fmt.Println(recorder.Code)
	fmt.Print(recorder.Body.String())
	// Output:
	// 404
	// {"title":"Not Found","status":404,"detail":"no such user","instance":"/users/42"}
}

// SuccessStatus is declared once. The generated OpenAPI document describes the
// success response under the same status the handler returns, because Endpoint
// copies it onto the operation.
func ExampleEndpointConfig() {
	operation := &openapi.Operation{Summary: "Create a user"}

	handler := httpx.Endpoint(createUser, httpx.EndpointConfig{
		SuccessStatus: http.StatusCreated,
		OpenAPI:       operation,
	})

	provider, ok := handler.(routing.OpenAPIProvider)
	if !ok {
		fmt.Println("handler does not describe itself")

		return
	}

	fmt.Println(provider.OpenAPIOperation().SuccessStatus)

	// The caller's own operation is untouched, so it stays reusable.
	fmt.Println(operation.SuccessStatus)
	// Output:
	// 201
	// 0
}
