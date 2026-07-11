package validation_test

import (
	"testing"

	"github.com/Casara/arnon/problem"
	"github.com/Casara/arnon/validation"
)

func TestPlaygroundValidator_MapsBuiltinTags(t *testing.T) {
	t.Parallel()

	type request struct {
		Required string `json:"required" validate:"required"`
		Min      string `json:"min"      validate:"omitempty,min=3"`
		Max      string `json:"max"      validate:"omitempty,max=2"`
		GT       int    `json:"gt"       validate:"omitempty,gt=10"`
		GTE      int    `json:"gte"      validate:"omitempty,gte=10"`
		LT       int    `json:"lt"       validate:"omitempty,lt=10"`
		LTE      int    `json:"lte"      validate:"omitempty,lte=10"`
		Len      string `json:"len"      validate:"omitempty,len=3"`
		OneOf    string `json:"oneof"    validate:"omitempty,oneof=a b c"`
		Eq       int    `json:"eq"       validate:"omitempty,eq=1"`
		// Ne intentionally has no omitempty: its zero value (0) is the
		// value under test, and omitempty would skip validation for it.
		Ne    int    `json:"ne"    validate:"ne=0"`
		UUID  string `json:"uuid"  validate:"omitempty,uuid"`
		Email string `json:"email" validate:"omitempty,email"`
		URL   string `json:"url"   validate:"omitempty,url"`
	}

	invalid := request{
		Required: "",
		Min:      "ab",
		Max:      "abc",
		GT:       5,
		GTE:      5,
		LT:       20,
		LTE:      20,
		Len:      "ab",
		OneOf:    "z",
		Eq:       2,
		Ne:       0,
		UUID:     "not-a-uuid",
		Email:    "not-an-email",
		URL:      "not-a-url",
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(invalid)

	byField := make(map[string]problem.ValidationError, len(errs))
	for _, validationErr := range errs {
		byField[validationErr.Source.Field] = validationErr
	}

	cases := []struct {
		field string
		code  problem.ValidationErrorCode
	}{
		{"/required", problem.ValidationCodeRequired},
		{"/min", problem.ValidationCodeMinLength},
		{"/max", problem.ValidationCodeMaxLength},
		{"/gt", problem.ValidationCodeGreaterThan},
		{"/gte", problem.ValidationCodeGreaterThanOrEqual},
		{"/lt", problem.ValidationCodeLessThan},
		{"/lte", problem.ValidationCodeLessThanOrEqual},
		{"/len", problem.ValidationCodeExactLength},
		{"/oneof", problem.ValidationCodeInvalidValue},
		{"/eq", problem.ValidationCodeEqual},
		{"/ne", problem.ValidationCodeNotEqual},
		{"/uuid", problem.ValidationCodeInvalidUUID},
		{"/email", problem.ValidationCodeInvalidEmail},
		{"/url", problem.ValidationCodeInvalidURL},
	}

	for _, testCase := range cases {
		got, ok := byField[testCase.field]
		if !ok {
			t.Errorf("missing validation error for field %q", testCase.field)

			continue
		}

		if got.Code != testCase.code {
			t.Errorf(
				"field %q: expected code %q, got %q",
				testCase.field,
				testCase.code,
				got.Code,
			)
		}

		if got.Source.In != problem.ValidationLocationBody {
			t.Errorf(
				"field %q: expected location %q, got %q",
				testCase.field,
				problem.ValidationLocationBody,
				got.Source.In,
			)
		}
	}
}

func TestPlaygroundValidator_ValidRequestHasNoErrors(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `json:"name" validate:"required"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{Name: "ok"})
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %+v", errs)
	}
}

func TestPlaygroundValidator_UnknownBuiltinTagFallsBackToGenericError(t *testing.T) {
	t.Parallel()

	type request struct {
		// alpha is a real validator/v10 tag with no framework mapping
		// and no registered CustomRule, so it must hit the generic
		// fallback branch in mapFieldError.
		Value string `json:"value" validate:"alpha"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{Value: "123"})

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Code != problem.ValidationCodeValidationFailed {
		t.Errorf("expected generic fallback code, got %q", errs[0].Code)
	}
}

func TestPlaygroundValidator_NonStructValueProducesInvalidTypeError(t *testing.T) {
	t.Parallel()

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate("not a struct")

	if len(errs) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(errs), errs)
	}

	if errs[0].Code != problem.ValidationCodeInvalidType {
		t.Errorf("expected invalid type code, got %q", errs[0].Code)
	}
}

func TestPlaygroundValidator_ResolvesPathQueryHeaderSources(t *testing.T) {
	t.Parallel()

	type request struct {
		ID     string `path:"id" validate:"required"`
		Filter string `          validate:"required" query:"filter"`
		Token  string `          validate:"required"                header:"X-Token"`
		Name   string `          validate:"required"                                 json:"name"`
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	errs := validator.Validate(request{})

	bySource := make(map[string]problem.ValidationSource, len(errs))
	for _, validationErr := range errs {
		bySource[validationErr.Source.Field] = *validationErr.Source
	}

	expectations := map[string]problem.ValidationLocation{
		"id":      problem.ValidationLocationPath,
		"filter":  problem.ValidationLocationQuery,
		"X-Token": problem.ValidationLocationHeader,
		"/name":   problem.ValidationLocationBody,
	}

	for field, location := range expectations {
		source, ok := bySource[field]
		if !ok {
			t.Errorf("missing validation error sourced from field %q", field)

			continue
		}

		if source.In != location {
			t.Errorf("field %q: expected location %q, got %q", field, location, source.In)
		}
	}
}
