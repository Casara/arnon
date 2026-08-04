package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/problem"
)

func TestWriteProblem_PopulatesInstanceFromRequestPathWhenEmpty(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)

	httpx.WriteProblem(
		recorder,
		request,
		problem.New(http.StatusNotFound, "Not Found", "user not found"),
	)

	var body map[string]any

	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body["instance"] != "/api/users/42" {
		t.Errorf("expected instance %q, got %v", "/api/users/42", body["instance"])
	}
}

func TestWriteProblem_PreservesExplicitlySetInstance(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)

	problemInstance := problem.
		New(http.StatusNotFound, "Not Found", "user not found").
		WithInstance("urn:request:custom-id")

	httpx.WriteProblem(recorder, request, problemInstance)

	var body map[string]any

	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body["instance"] != "urn:request:custom-id" {
		t.Errorf("expected the explicitly set instance to be preserved, got %v", body["instance"])
	}
}
