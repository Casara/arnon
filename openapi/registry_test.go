package openapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestRegistry_RegisterDelegatesToGenerator confirms Registry.Register
// forwards to the underlying Generator's RegisterTypes, ending up with
// the operation and schemas present in the generated document -
// Registry exists purely as the thin seam routing.Router talks to
// (see httpx/routing/register_openapi.go), so this only needs to
// prove the delegation, not re-test Generator's own behavior.
func TestRegistry_RegisterDelegatesToGenerator(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})
	registry := openapi.NewRegistry(generator)

	registry.Register(
		http.MethodPost,
		"/users",
		openapi.Operation{},
		reflect.TypeFor[createUserRequest](),
		reflect.TypeFor[createUserResponse](),
	)

	document := generator.Generate()

	pathItem, ok := document.Paths["/users"]
	if !ok {
		t.Fatalf("expected path %q to be registered", "/users")
	}

	if pathItem.Post == nil {
		t.Fatal("expected a POST operation registered via Registry.Register")
	}

	if _, ok := document.Components.Schemas["createUserRequest"]; !ok {
		t.Error("expected the request schema to be registered via Registry.Register")
	}
}
