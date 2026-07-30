package problem_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/casara/arnon/problem"
)

// A Problem serializes to RFC 9457 JSON, omitting every standard member left
// at its zero value.
func ExampleNew() {
	details := problem.New(
		http.StatusConflict,
		"Conflict",
		"an account with this email already exists",
	)

	encoded, err := json.Marshal(details)
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// {"title":"Conflict","status":409,"detail":"an account with this email already exists"}
}

// The named constructors fill in the title from the status, so the wording
// stays consistent across an API.
func ExampleNewNotFound() {
	details := problem.NewNotFound("no user with that id")

	fmt.Println(details.StatusCode(), details.Title)
	fmt.Println(details.Error())
	// Output:
	// 404 Not Found
	// 404 Not Found: no user with that id
}

// With adds an RFC 9457 extension member, serialized at the top level of the
// object rather than nested.
func ExampleProblem_With() {
	details := problem.NewTooManyRequests("slow down").
		With("retry_after_seconds", 30)

	encoded, err := json.Marshal(details)
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// {"title":"Too Many Requests","status":429,"detail":"slow down","retry_after_seconds":30}
}

// A Problem is an error, so a handler can return it directly and callers can
// recover it with errors.As. WithError keeps the underlying cause available to
// errors.Is without leaking it into the response body.
func ExampleProblem_WithError() {
	cause := errors.New("dial tcp: connection refused")

	details := problem.NewServiceUnavailable("the account service is down").
		WithError(cause)

	var recovered *problem.Problem

	fmt.Println(errors.As(error(details), &recovered))
	fmt.Println(errors.Is(details, cause))

	encoded, err := json.Marshal(details)
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// true
	// true
	// {"title":"Service Unavailable","status":503,"detail":"the account service is down"}
}

// Validation failures are carried in the errors extension, each one pointing at
// the offending field with an RFC 6901 JSON Pointer.
func ExampleProblem_AddError() {
	details := problem.New(
		http.StatusBadRequest,
		"Request validation failed",
		"Request validation failed",
	)

	details = details.AddError(problem.NewBodyError(
		"must be a valid email address",
		"/email",
		problem.ValidationCodeInvalidEmail,
		nil,
	))

	encoded, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// {
	//   "title": "Request validation failed",
	//   "status": 400,
	//   "detail": "Request validation failed",
	//   "errors": [
	//     {
	//       "detail": "must be a valid email address",
	//       "code": "invalid_email",
	//       "source": {
	//         "in": "body",
	//         "field": "/email"
	//       }
	//     }
	//   ]
	// }
}

// Extension members serialize in a stable, alphabetical order, so an error
// response is byte-for-byte reproducible - which is what lets a golden test
// assert on one.
func ExampleProblem_With_multiple() {
	details := problem.NewConflict("insufficient funds").
		With("currency", "BRL").
		With("balance", 100).
		With("account", "acc_1")

	encoded, err := json.Marshal(details)
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// {"title":"Conflict","status":409,"detail":"insufficient funds","account":"acc_1","balance":100,"currency":"BRL"}
}

// A Problem declared once at package level is safe to derive from: every WithX
// method returns a copy, so two concurrent requests never overwrite each
// other's instance.
func ExampleProblem_WithInstance() {
	shared := problem.NewNotFound("no such user")

	first := shared.WithInstance("/users/1")
	second := shared.WithInstance("/users/2")

	fmt.Printf("shared: %q\n", shared.Instance)
	fmt.Printf("first:  %q\n", first.Instance)
	fmt.Printf("second: %q\n", second.Instance)
	// Output:
	// shared: ""
	// first:  "/users/1"
	// second: "/users/2"
}

// A Go client decoding an arnon error response gets the extension members back
// as well as the standard ones.
func ExampleProblem_UnmarshalJSON() {
	response := []byte(`{
		"title": "Too Many Requests",
		"status": 429,
		"detail": "slow down",
		"retry_after_seconds": 30
	}`)

	var details problem.Problem

	err := json.Unmarshal(response, &details)
	if err != nil {
		fmt.Println("unmarshal:", err)

		return
	}

	fmt.Println(details.StatusCode(), details.Detail)

	retryAfter, found := details.Extension("retry_after_seconds")

	fmt.Println(found, retryAfter)
	// Output:
	// 429 slow down
	// true 30
}
