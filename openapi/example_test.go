package openapi_test

import (
	"encoding/json"
	"fmt"

	"github.com/casara/arnon/openapi"
)

type exampleUserRequest struct {
	Name string `json:"name" validate:"required"`
}

type exampleUserResponse struct {
	ID string `json:"id"`
}

// A Generator collects operations and produces the document. Register is
// normally called for you, by the router, for every endpoint that opted into
// documentation.
func ExampleNewGenerator() {
	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	generator.Register(
		"POST",
		"/users",
		openapi.Operation{Summary: "Create a user"},
		exampleUserRequest{},
		exampleUserResponse{},
	)

	document := generator.Generate()

	fmt.Println(document.OpenAPI)
	fmt.Println(document.Paths["/users"].Post.Summary)
	// Output:
	// 3.2.0
	// Create a user
}

// The generated document declares OpenAPI 3.2.0 by default. WithSpecVersion
// switches it to 3.1.1 for tooling that has not caught up - nothing generated
// here uses a 3.2-only construct, so only the declaration changes.
func ExampleWithSpecVersion() {
	generator := openapi.NewGenerator(
		openapi.Info{Title: "Example API", Version: "1.0.0"},
		openapi.WithSpecVersion(openapi.SpecVersion31),
	)

	fmt.Println(generator.Generate().OpenAPI)
	// Output:
	// 3.1.1
}

// Nullability is expressed the way OpenAPI 3.1 replaced the 3.0 "nullable"
// keyword: as a type array. A pointer field is what triggers it.
func ExampleSchema() {
	encoded, err := json.Marshal(openapi.Schema{
		Type:     "string",
		Nullable: true,
	})
	if err != nil {
		fmt.Println("marshal:", err)

		return
	}

	fmt.Println(string(encoded))
	// Output:
	// {"type":["string","null"]}
}
