package openapi_test

import (
	"net/http"
	"testing"

	"github.com/casara/arnon/openapi"
)

func TestGenerator_RegistersProblemSchemasOnceForDefaultResponses(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{},
		createUserRequest{},
		createUserResponse{},
	)

	generator.Register(
		http.MethodPost,
		"/admins",
		openapi.Operation{},
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	for _, name := range []string{"Problem", "ValidationError", "ValidationSource"} {
		if _, ok := document.Components.Schemas[name]; !ok {
			t.Errorf("expected schema %q to be registered exactly once", name)
		}
	}

	problemSchema := document.Components.Schemas["Problem"]

	for _, field := range []string{"type", "title", "status", "detail", "instance", "errors"} {
		if _, ok := problemSchema.Properties[field]; !ok {
			t.Errorf("expected Problem schema to have property %q", field)
		}
	}

	if len(problemSchema.Required) == 0 {
		t.Error("expected Problem schema to declare required fields")
	}
}
