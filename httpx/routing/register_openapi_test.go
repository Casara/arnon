package routing_test

import (
	"net/http"
	"testing"

	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
)

// TestRegisterOpenAPI_NoRegistryIsANoOp confirms that routes register
// fine, and no schema is generated, when no OpenAPI registry was
// configured via WithOpenAPI - the common case for routers that don't
// expose documentation at all.
func TestRegisterOpenAPI_NoRegistryIsANoOp(t *testing.T) {
	t.Parallel()

	router := routing.NewRouter()

	// Registering a handler that does implement OpenAPIProvider must
	// not panic even though there is no registry to receive it.
	router.GET("/widgets", &fakeOpenAPIHandler{operation: &openapi.Operation{}})
}

// TestRegisterOpenAPI_HandlerWithoutProviderIsSkipped confirms a plain
// http.Handler (not implementing httpx.OpenAPIProvider) is silently
// skipped by registerOpenAPI even when a registry is configured -
// OpenAPI documentation stays strictly opt-in per handler.
func TestRegisterOpenAPI_HandlerWithoutProviderIsSkipped(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})
	registry := generator

	router := routing.NewRouter(routing.WithOpenAPI(registry))

	router.GET("/widgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	document := generator.Generate()

	if _, ok := document.Paths["/widgets"]; ok {
		t.Error("expected no path to be registered for a handler that isn't an OpenAPIProvider")
	}
}

// TestRegisterOpenAPI_ConnectRegistersRouteWithoutPathOperation covers
// registering a CONNECT route through Router with both an OpenAPI
// registry and an httpx.OpenAPIProvider handler present. CONNECT
// doesn't panic (splitPattern's whitelist covers it, see pattern.go),
// but OpenAPI's Path Item Object has no field to represent a CONNECT
// operation, so registerOpenAPI reaches the registry and the route
// still gets a (empty) path entry - same behavior openapi.Generator
// documents in TestGenerator_UnrepresentableMethodRegistersSchemasButNoPathOperation.
func TestRegisterOpenAPI_ConnectRegistersRouteWithoutPathOperation(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})
	registry := generator

	router := routing.NewRouter(routing.WithOpenAPI(registry))

	router.CONNECT("/widgets", &fakeOpenAPIHandler{operation: &openapi.Operation{}})

	document := generator.Generate()

	pathItem, ok := document.Paths["/widgets"]
	if !ok {
		t.Fatal("expected /widgets to still get a path entry")
	}

	if pathItem.Get != nil || pathItem.Options != nil || pathItem.Trace != nil {
		t.Errorf("expected no operation wired into PathItem for CONNECT, got %+v", pathItem)
	}
}
