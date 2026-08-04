package openapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/casara/arnon/openapi"
)

type createUserRequest struct {
	Name string `json:"name" validate:"required"`
}

type createUserResponse struct {
	ID string `json:"id"`
}

type listUsersRequest struct {
	Limit int `query:"limit"`
}

func TestGenerator_PostRegistersRequestBodyAndDefaultResponses(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{},
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	pathItem, ok := document.Paths["/users"]
	if !ok {
		t.Fatalf("expected path %q to be registered", "/users")
	}

	if pathItem.Post == nil {
		t.Fatal("expected a POST operation")
	}

	if pathItem.Post.RequestBody == nil {
		t.Fatal("expected a request body to be generated for POST")
	}

	if _, ok := document.Components.Schemas["createUserRequest"]; !ok {
		t.Errorf("expected schema %q to be registered", "createUserRequest")
	}

	if _, ok := document.Components.Schemas["createUserResponse"]; !ok {
		t.Errorf("expected schema %q to be registered", "createUserResponse")
	}

	successResponse, ok := pathItem.Post.Responses["200"]
	if !ok {
		t.Fatalf("expected a default 200 response, got %+v", pathItem.Post.Responses)
	}

	if successResponse.Content["application/json"].Schema.Ref == "" {
		t.Error("expected the 200 response schema to reference the response schema")
	}

	if _, ok := pathItem.Post.Responses["400"]; !ok {
		t.Error("expected a default 400 problem response")
	}

	if _, ok := pathItem.Post.Responses["500"]; !ok {
		t.Error("expected a default 500 problem response")
	}
}

func TestGenerator_GetDoesNotRegisterRequestBody(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodGet,
		"/users",
		openapi.Operation{},
		listUsersRequest{},
		[]createUserResponse{},
	)

	document := generator.Generate()

	pathItem := document.Paths["/users"]
	if pathItem.Get == nil {
		t.Fatal("expected a GET operation")
	}

	if pathItem.Get.RequestBody != nil {
		t.Error("expected no request body for GET")
	}
}

func TestGenerator_QueryParameterIsAddedAsParameterNotSchema(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodGet,
		"/users",
		openapi.Operation{},
		listUsersRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	pathItem := document.Paths["/users"]

	if len(pathItem.Get.Parameters) != 1 {
		t.Fatalf(
			"expected 1 parameter, got %d: %+v",
			len(pathItem.Get.Parameters),
			pathItem.Get.Parameters,
		)
	}

	parameter := pathItem.Get.Parameters[0]

	if parameter.Name != "limit" || parameter.In != "query" {
		t.Errorf("expected query parameter %q, got %+v", "limit", parameter)
	}
}

func TestGenerator_SchemaIsRegisteredOnlyOnce(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	// Custom responses on both operations keep registerProblemSchema
	// out of the picture, so the only schemas that can appear are the
	// request/response types shared by both routes.
	customResponses := openapi.Operation{
		Responses: openapi.Responses{
			"201": openapi.Response{Description: "Created"},
		},
	}

	generator.Register(
		http.MethodPost,
		"/users",
		customResponses,
		createUserRequest{},
		createUserResponse{},
	)

	generator.Register(
		http.MethodPost,
		"/admins",
		customResponses,
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	if len(document.Components.Schemas) != 2 {
		t.Errorf(
			"expected exactly 2 registered schemas (no duplicates across routes), got %d: %v",
			len(document.Components.Schemas),
			document.Components.Schemas,
		)
	}
}

func TestGenerator_CustomResponsesAreNotOverwrittenByDefaults(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{
			Responses: openapi.Responses{
				"201": openapi.Response{Description: "Created"},
			},
		},
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	responses := document.Paths["/users"].Post.Responses

	if _, ok := responses["201"]; !ok {
		t.Fatal("expected the custom 201 response to be preserved")
	}

	if _, ok := responses["200"]; ok {
		t.Error("expected no default 200 response when custom responses were provided")
	}

	if _, ok := responses["400"]; ok {
		t.Error("expected no default problem responses when custom responses were provided")
	}
}

func TestGenerator_RegisterTypesBuildsZeroValuesFromReflectType(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.RegisterTypes(
		http.MethodPost,
		"/users",
		openapi.Operation{},
		reflect.TypeFor[createUserRequest](),
		reflect.TypeFor[createUserResponse](),
	)

	document := generator.Generate()

	if document.Paths["/users"].Post == nil {
		t.Fatal("expected a POST operation registered via RegisterTypes")
	}

	if _, ok := document.Components.Schemas["createUserRequest"]; !ok {
		t.Error("expected request schema to be registered via RegisterTypes")
	}
}

func TestGenerator_TagsAreMergedByName(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{
			Tags: []openapi.Tag{{Name: "users", Summary: "Users"}},
		},
		createUserRequest{},
		createUserResponse{},
	)

	generator.Register(
		http.MethodGet,
		"/users",
		openapi.Operation{
			Tags: []openapi.Tag{{Name: "users", Description: "User management"}},
		},
		listUsersRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	if len(document.Tags) != 1 {
		t.Fatalf("expected tags to be merged into 1 entry, got %+v", document.Tags)
	}

	merged := document.Tags[0]

	if merged.Summary != "Users" || merged.Description != "User management" {
		t.Errorf("expected merged tag metadata, got %+v", merged)
	}
}
