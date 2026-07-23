package openapi_test

import (
	"net/http"
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestGenerator_PutPatchDeleteWireOperationIntoPathItem completes the
// method-to-PathItem-field switch in Generator.Register: POST and GET
// already have coverage elsewhere, this rounds out PUT, PATCH and
// DELETE.
func TestGenerator_PutPatchDeleteWireOperationIntoPathItem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		method string
		get    func(item openapi.PathItem) *openapi.Operation
	}{
		{"PUT", http.MethodPut, func(item openapi.PathItem) *openapi.Operation { return item.Put }},
		{
			"PATCH",
			http.MethodPatch,
			func(item openapi.PathItem) *openapi.Operation { return item.Patch },
		},
		{
			"DELETE",
			http.MethodDelete,
			func(item openapi.PathItem) *openapi.Operation { return item.Delete },
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

			generator.Register(
				testCase.method,
				"/users/{id}",
				openapi.Operation{},
				createUserRequest{},
				createUserResponse{},
			)

			document := generator.Generate()

			pathItem, ok := document.Paths["/users/{id}"]
			if !ok {
				t.Fatalf("expected path %q to be registered", "/users/{id}")
			}

			if testCase.get(pathItem) == nil {
				t.Errorf("expected a %s operation to be wired into the PathItem", testCase.name)
			}
		})
	}
}

// TestGenerator_AllPathItemMethodsWireIntoTheOperation covers every
// method Generator.Register's switch knows how to wire into a
// PathItem: GET/PUT/POST/PATCH/DELETE plus HEAD/OPTIONS/TRACE/QUERY,
// all eight fields PathItem exposes.
func TestGenerator_AllPathItemMethodsWireIntoTheOperation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		method string
		get    func(item openapi.PathItem) *openapi.Operation
	}{
		{http.MethodGet, func(item openapi.PathItem) *openapi.Operation { return item.Get }},
		{http.MethodPut, func(item openapi.PathItem) *openapi.Operation { return item.Put }},
		{http.MethodPost, func(item openapi.PathItem) *openapi.Operation { return item.Post }},
		{http.MethodPatch, func(item openapi.PathItem) *openapi.Operation { return item.Patch }},
		{http.MethodDelete, func(item openapi.PathItem) *openapi.Operation { return item.Delete }},
		{http.MethodHead, func(item openapi.PathItem) *openapi.Operation { return item.Head }},
		{
			http.MethodOptions,
			func(item openapi.PathItem) *openapi.Operation { return item.Options },
		},
		{http.MethodTrace, func(item openapi.PathItem) *openapi.Operation { return item.Trace }},
		{"QUERY", func(item openapi.PathItem) *openapi.Operation { return item.Query }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.method, func(t *testing.T) {
			t.Parallel()

			generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

			generator.Register(
				testCase.method,
				"/users",
				openapi.Operation{Summary: "op"},
				createUserRequest{},
				createUserResponse{},
			)

			document := generator.Generate()

			pathItem, ok := document.Paths["/users"]
			if !ok {
				t.Fatalf("expected path %q to get a map entry", "/users")
			}

			operation := testCase.get(pathItem)
			if operation == nil || operation.Summary != "op" {
				t.Errorf(
					"expected %s to wire the operation into its PathItem field, got %+v",
					testCase.method,
					pathItem,
				)
			}
		})
	}
}

// TestGenerator_UnrepresentableMethodRegistersSchemasButNoPathOperation
// covers CONNECT: OpenAPI's Path Item Object has no field for it at
// all (unlike HEAD/OPTIONS/TRACE/QUERY above), so it's expected -not a
// gap- that registering a CONNECT route still registers its
// request/response schemas but never wires an operation into the
// generated PathItem.
func TestGenerator_UnrepresentableMethodRegistersSchemasButNoPathOperation(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodConnect,
		"/users",
		openapi.Operation{},
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	pathItem, ok := document.Paths["/users"]
	if !ok {
		t.Fatalf("expected path %q to still get a map entry", "/users")
	}

	if pathItem.Get != nil || pathItem.Put != nil || pathItem.Post != nil ||
		pathItem.Patch != nil || pathItem.Delete != nil || pathItem.Head != nil ||
		pathItem.Options != nil || pathItem.Trace != nil || pathItem.Query != nil {
		t.Errorf(
			"expected no operation wired into any PathItem field for CONNECT, got %+v",
			pathItem,
		)
	}

	if _, ok := document.Components.Schemas["createUserRequest"]; !ok {
		t.Error("expected the request schema to still be registered even though CONNECT is dropped")
	}
}

// TestGenerator_AddParametersSkipsUnexportedAndBodyFields confirms
// Generator.addParameters (distinct from generateStruct's own body-vs-
// parameter split, tested in reflection_extra_test.go) also skips
// unexported fields (parseField returns nil) and body-located fields
// (they belong in the request schema, not operation.Parameters).
func TestGenerator_AddParametersSkipsUnexportedAndBodyFields(t *testing.T) {
	t.Parallel()

	type mixedRequest struct {
		Limit    int    `query:"limit"`
		Name     string `              json:"name"`
		internal string // exercises the unexported-field skip in addParameters
	}

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/widgets",
		openapi.Operation{},
		mixedRequest{internal: "unused"},
		createUserResponse{},
	)

	document := generator.Generate()

	parameters := document.Paths["/widgets"].Post.Parameters

	if len(parameters) != 1 {
		t.Fatalf(
			"expected exactly 1 parameter (query:limit; name is body-only, internal is unexported), got %+v",
			parameters,
		)
	}

	if parameters[0].Name != "limit" || parameters[0].In != "query" {
		t.Errorf("expected query parameter %q, got %+v", "limit", parameters[0])
	}
}

// TestGenerator_ShouldGenerateBodyFalseWhenRequestHasNoBodyFields
// covers the branch of shouldGenerateBody where the method isn't GET
// but every field of the request is a path/query/header parameter -
// there is nothing left to put in a JSON request body, so no
// RequestBody should be generated.
func TestGenerator_ShouldGenerateBodyFalseWhenRequestHasNoBodyFields(t *testing.T) {
	t.Parallel()

	type deleteRequest struct {
		ID string `path:"id"`
	}

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodDelete,
		"/widgets/{id}",
		openapi.Operation{},
		deleteRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	if document.Paths["/widgets/{id}"].Delete.RequestBody != nil {
		t.Error("expected no request body for a DELETE request with only path parameters")
	}
}

// TestGenerator_PointerRequestAndResponseAreDereferenced confirms
// addParameters, shouldGenerateBody and registerSchema all unwrap a
// pointer request/response down to the underlying struct before
// inspecting it - registering with *createUserRequest{} instead of
// createUserRequest{} must produce the exact same parameters, request
// body and schema name.
func TestGenerator_PointerRequestAndResponseAreDereferenced(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{},
		&createUserRequest{},
		&createUserResponse{},
	)

	document := generator.Generate()

	operation := document.Paths["/users"].Post

	if operation.RequestBody == nil {
		t.Fatal("expected a request body for a pointer request with a body field")
	}

	if _, ok := document.Components.Schemas["createUserRequest"]; !ok {
		t.Errorf(
			"expected schema %q registered under its dereferenced name, got %v",
			"createUserRequest",
			document.Components.Schemas,
		)
	}

	if _, ok := document.Components.Schemas["createUserResponse"]; !ok {
		t.Errorf(
			"expected schema %q registered under its dereferenced name, got %v",
			"createUserResponse",
			document.Components.Schemas,
		)
	}
}

// TestGenerator_ShouldGenerateBodyFalseForNonStructRequest covers the
// requestType.Kind() != reflect.Struct branch: a non-struct request
// (here, a bare string) can't carry body fields, so no request body is
// generated even for a method that normally would (POST).
func TestGenerator_ShouldGenerateBodyFalseForNonStructRequest(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/widgets",
		openapi.Operation{},
		"not-a-struct",
		createUserResponse{},
	)

	document := generator.Generate()

	if document.Paths["/widgets"].Post.RequestBody != nil {
		t.Error("expected no request body when the request value is not a struct")
	}
}

// TestGenerator_NilRequestSkipsParametersAndBody covers the
// requestType == nil guards shared by addParameters and
// shouldGenerateBody: passing a nil request (reflect.TypeOf(nil) is
// nil) short-circuits both before they try to inspect a Kind, leaving
// no parameters and no request body - without these guards, calling
// reflect.TypeOf(nil).Kind() would panic on a nil pointer dereference.
func TestGenerator_NilRequestSkipsParametersAndBody(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/widgets",
		openapi.Operation{},
		nil,
		createUserResponse{},
	)

	document := generator.Generate()

	operation := document.Paths["/widgets"].Post

	if len(operation.Parameters) != 0 {
		t.Errorf("expected no parameters for a nil request, got %+v", operation.Parameters)
	}

	if operation.RequestBody != nil {
		t.Error("expected no request body for a nil request")
	}
}

// TestGenerator_NilResponseSkipsSchemaRegistration covers
// registerSchema's own valueType == nil guard (reached here through
// the response value, since Generator.buildOperation always calls
// registerSchema(response) regardless of method) - the response
// schema ref ends up empty instead of panicking on a nil reflect.Type.
func TestGenerator_NilResponseSkipsSchemaRegistration(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodGet,
		"/widgets",
		openapi.Operation{},
		struct{}{},
		nil,
	)

	document := generator.Generate()

	successResponse := document.Paths["/widgets"].Get.Responses["200"]

	if successResponse.Content["application/json"].Schema.Ref != "" {
		t.Errorf(
			"expected an empty schema ref for a nil response, got %+v",
			successResponse.Content["application/json"].Schema,
		)
	}
}

// TestGenerator_TagMergeFillsInPreviouslyEmptySummary complements the
// existing TestGenerator_TagsAreMergedByName (generator_test.go),
// which merges a tag with a Summary already set on the first
// registration - so the "existing.Summary == ”" branch of
// registerTags's merge logic is never taken there. This test
// registers the empty-Summary tag first, filling it in from a later
// registration, to cover that branch too.
func TestGenerator_TagMergeFillsInPreviouslyEmptySummary(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{
			Tags: []openapi.Tag{{Name: "users", Description: "User management"}},
		},
		createUserRequest{},
		createUserResponse{},
	)

	generator.Register(
		http.MethodGet,
		"/users",
		openapi.Operation{
			Tags: []openapi.Tag{{Name: "users", Summary: "Users"}},
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

// TestGenerator_RegisterTagSkipsEmptyName covers registerTags' guard
// against a zero-value Tag (empty Name) ending up in the merged tag
// map - Operation.Tags entries with no Name are simply skipped.
func TestGenerator_RegisterTagSkipsEmptyName(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})

	generator.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{
			Tags: []openapi.Tag{{Name: ""}, {Name: "users"}},
		},
		createUserRequest{},
		createUserResponse{},
	)

	document := generator.Generate()

	if len(document.Tags) != 1 {
		t.Fatalf("expected only the named tag to be registered, got %+v", document.Tags)
	}

	if document.Tags[0].Name != "users" {
		t.Errorf("expected tag %q, got %+v", "users", document.Tags[0])
	}
}
