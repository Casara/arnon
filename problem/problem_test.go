package problem_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/casara/arnon/problem"
)

func TestValidationErrorCode_StatusOverride(t *testing.T) {
	t.Parallel()

	if got := problem.ValidationCodePayloadTooLarge.StatusOverride(); got != http.StatusRequestEntityTooLarge {
		t.Errorf("expected %d, got %d", http.StatusRequestEntityTooLarge, got)
	}

	if got := problem.ValidationCodeRequired.StatusOverride(); got != 0 {
		t.Errorf("expected 0 (no override) for a plain validation code, got %d", got)
	}
}

func TestProblem_StatusCodeDefaultsToInternalServerError(t *testing.T) {
	t.Parallel()

	problemInstance := &problem.Problem{}

	if got := problemInstance.StatusCode(); got != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, got)
	}
}

func TestProblem_ErrorFormatsStatusTitleAndDetail(t *testing.T) {
	t.Parallel()

	withDetail := problem.New(http.StatusBadRequest, "Bad Request", "name is required")
	if got := withDetail.Error(); got != "400 Bad Request: name is required" {
		t.Errorf("unexpected Error() output: %q", got)
	}

	withoutDetail := problem.New(http.StatusBadRequest, "Bad Request", "")
	if got := withoutDetail.Error(); got != "400 Bad Request" {
		t.Errorf("unexpected Error() output: %q", got)
	}
}

func TestProblem_UnwrapReturnsWrappedError(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")

	problemInstance := problem.New(http.StatusInternalServerError, "", "").WithError(cause)

	if !errors.Is(problemInstance, cause) {
		t.Error("expected errors.Is to find the wrapped cause via Unwrap")
	}
}

func TestProblem_WithAttachesExtensionAtTopLevel(t *testing.T) {
	t.Parallel()

	problemInstance := problem.New(http.StatusBadRequest, "Bad Request", "detail").
		With("traceId", "abc-123")

	encoded, err := json.Marshal(problemInstance)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]any

	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded["traceId"] != "abc-123" {
		t.Errorf("expected extension traceId at top level, got %v", decoded)
	}
}

func TestProblem_WithPanicsOnEmptyKey(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected panic for empty extension key")
		}
	}()

	_ = problem.New(http.StatusBadRequest, "", "").With("   ", "value")
}

func TestProblem_WithPanicsOnReservedKey(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected panic for reserved extension key")
		}
	}()

	_ = problem.New(http.StatusBadRequest, "", "").With("status", 123)
}

func TestProblem_MarshalJSONOmitsZeroFields(t *testing.T) {
	t.Parallel()

	problemInstance := problem.New(http.StatusBadRequest, "Bad Request", "")

	encoded, err := json.Marshal(problemInstance)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]any

	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	for _, absent := range []string{"detail", "instance", "errors", "type"} {
		if _, ok := decoded[absent]; ok {
			t.Errorf("expected field %q to be omitted, got %v", absent, decoded)
		}
	}

	if decoded["title"] != "Bad Request" {
		t.Errorf("expected title to be present, got %v", decoded)
	}
}

func TestProblem_AddErrorAppendsValidationErrors(t *testing.T) {
	t.Parallel()

	problemInstance := problem.New(http.StatusBadRequest, "Bad Request", "invalid request").
		AddError(problem.NewBodyError("field is required", "/name", problem.ValidationCodeRequired, nil))

	if len(problemInstance.Errors) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(problemInstance.Errors))
	}

	if problemInstance.Errors[0].Source.Field != "/name" {
		t.Errorf("unexpected validation error source: %+v", problemInstance.Errors[0].Source)
	}
}

func TestNewXxx_ProduceExpectedStatusCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		build      func(string) *problem.Problem
		wantStatus int
	}{
		{"BadRequest", problem.NewBadRequest, http.StatusBadRequest},
		{"Unauthorized", problem.NewUnauthorized, http.StatusUnauthorized},
		{"Forbidden", problem.NewForbidden, http.StatusForbidden},
		{"NotFound", problem.NewNotFound, http.StatusNotFound},
		{"Conflict", problem.NewConflict, http.StatusConflict},
		{"UnprocessableEntity", problem.NewUnprocessableEntity, http.StatusUnprocessableEntity},
		{"PreconditionFailed", problem.NewPreconditionFailed, http.StatusPreconditionFailed},
		{"PreconditionRequired", problem.NewPreconditionRequired, http.StatusPreconditionRequired},
		{"TooManyRequests", problem.NewTooManyRequests, http.StatusTooManyRequests},
		{"Internal", problem.NewInternal, http.StatusInternalServerError},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.build("detail")
			if got.StatusCode() != testCase.wantStatus {
				t.Errorf("expected status %d, got %d", testCase.wantStatus, got.StatusCode())
			}
		})
	}
}

func TestNewInternal_DefaultsDetailWhenEmpty(t *testing.T) {
	t.Parallel()

	got := problem.NewInternal("")

	if got.Detail == "" {
		t.Error("expected a default detail message when none is provided")
	}
}

func TestValidationErrorBuilders_SetExpectedSource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		build        func(string, string, problem.ValidationErrorCode, map[string]any) problem.ValidationError
		wantLocation problem.ValidationLocation
	}{
		{"Body", problem.NewBodyError, problem.ValidationLocationBody},
		{"Path", problem.NewPathError, problem.ValidationLocationPath},
		{"Query", problem.NewQueryError, problem.ValidationLocationQuery},
		{"Header", problem.NewHeaderError, problem.ValidationLocationHeader},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.build("detail", "field", problem.ValidationCodeRequired, nil)

			if got.Source.In != testCase.wantLocation {
				t.Errorf("expected location %q, got %q", testCase.wantLocation, got.Source.In)
			}

			if got.Source.Field != "field" {
				t.Errorf("expected field %q, got %q", "field", got.Source.Field)
			}

			if got.Code != problem.ValidationCodeRequired {
				t.Errorf("expected code %q, got %q", problem.ValidationCodeRequired, got.Code)
			}
		})
	}
}

// Regression test: extension members were serialized by ranging over the
// extensions map directly, and Go randomizes map iteration order, so the same
// Problem produced a different byte sequence on each call.
func TestProblem_MarshalJSON_ExtensionOrderIsStable(t *testing.T) {
	t.Parallel()

	build := func() *problem.Problem {
		return problem.New(http.StatusConflict, "Conflict", "duplicate").
			With("zulu", 1).
			With("alpha", 2).
			With("mike", 3)
	}

	first, err := json.Marshal(build())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for range 100 {
		again, err := json.Marshal(build())
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		if !bytes.Equal(first, again) {
			t.Fatalf("unstable extension order:\n%s\n%s", first, again)
		}
	}

	if !bytes.Contains(first, []byte(`"alpha":2,"mike":3,"zulu":1`)) {
		t.Errorf("extensions not in alphabetical order: %s", first)
	}
}
