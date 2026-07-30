package routing_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
)

// fakeOpenAPIHandler implements httpx.OpenAPIProvider so tests can
// exercise Router.registerOpenAPI (invoked from register) without
// depending on httpx.Endpoint itself.
type fakeOpenAPIHandler struct {
	operation *openapi.Operation
}

func (handler *fakeOpenAPIHandler) ServeHTTP(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.WriteHeader(http.StatusOK)
}

func (handler *fakeOpenAPIHandler) OpenAPIOperation() *openapi.Operation {
	return handler.operation
}

func (handler *fakeOpenAPIHandler) RequestType() reflect.Type {
	return reflect.TypeFor[struct{}]()
}

func (handler *fakeOpenAPIHandler) ResponseType() reflect.Type {
	return reflect.TypeFor[struct{}]()
}

// TestWithOpenAPI_RegistersOperationsWithGenerator confirms
// WithOpenAPI wires the registry into the router so that routes whose
// handler implements httpx.OpenAPIProvider end up in the generated
// document.
func TestWithOpenAPI_RegistersOperationsWithGenerator(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})
	registry := generator

	router := routing.NewRouter(routing.WithOpenAPI(registry))

	router.GET("/widgets", &fakeOpenAPIHandler{operation: &openapi.Operation{}})

	document := generator.Generate()

	pathItem, ok := document.Paths["/widgets"]
	if !ok {
		t.Fatalf("expected path %q to be registered in the OpenAPI document", "/widgets")
	}

	if pathItem.Get == nil {
		t.Error("expected a GET operation to be registered")
	}
}

// TestWithOpenAPI_NilOperationSkipsRegistration confirms a handler
// that implements OpenAPIProvider but opts out of documentation
// (returns a nil *openapi.Operation) is not registered - OpenAPI
// registration is opt-in per endpoint (see httpx/CLAUDE.md).
func TestWithOpenAPI_NilOperationSkipsRegistration(t *testing.T) {
	t.Parallel()

	generator := openapi.NewGenerator(openapi.Info{Title: "Test API", Version: "1.0.0"})
	registry := generator

	router := routing.NewRouter(routing.WithOpenAPI(registry))

	router.GET("/widgets", &fakeOpenAPIHandler{operation: nil})

	document := generator.Generate()

	if _, ok := document.Paths["/widgets"]; ok {
		t.Error("expected no path to be registered when OpenAPIOperation returns nil")
	}
}

// TestWithInstrumentation_WrapsRegisteredHandlers confirms
// WithInstrumentation's function is applied to every registered route,
// receiving the final wrapped handler and the route pattern.
func TestWithInstrumentation_WrapsRegisteredHandlers(t *testing.T) {
	t.Parallel()

	var instrumentedPatterns []string

	instrument := func(next http.Handler, pattern string) http.Handler {
		instrumentedPatterns = append(instrumentedPatterns, pattern)

		return next
	}

	router := routing.NewRouter(routing.WithInstrumentation(instrument))

	router.GET("/widgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	router.POST("/gadgets", http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusCreated)
	}))

	want := []string{"GET /widgets", "POST /gadgets"}

	if len(instrumentedPatterns) != len(want) {
		t.Fatalf("expected instrumented patterns %v, got %v", want, instrumentedPatterns)
	}

	for i, pattern := range want {
		if instrumentedPatterns[i] != pattern {
			t.Errorf("expected instrumented patterns %v, got %v", want, instrumentedPatterns)

			break
		}
	}

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/widgets", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected the instrumented handler to still serve the request, got status %d",
			recorder.Code,
		)
	}
}
