package validation_test

import (
	"errors"
	"testing"

	validatorv10 "github.com/go-playground/validator/v10"

	"github.com/casara/arnon/problem"
	"github.com/casara/arnon/validation"
)

func TestRegisterCustomRule_RejectsEmptyTag(t *testing.T) {
	t.Parallel()

	err := validation.RegisterCustomRule(validation.CustomRule{
		Func: func(validatorv10.FieldLevel) bool { return true },
	})

	if !errors.Is(err, validation.ErrCustomRuleTagEmpty) {
		t.Fatalf("expected ErrCustomRuleTagEmpty, got %v", err)
	}
}

func TestRegisterCustomRule_RejectsNilFunc(t *testing.T) {
	t.Parallel()

	err := validation.RegisterCustomRule(validation.CustomRule{
		Tag: "example_nil_func",
	})

	if !errors.Is(err, validation.ErrCustomRuleFuncNil) {
		t.Fatalf("expected ErrCustomRuleFuncNil, got %v", err)
	}
}

func TestCustomRule_AppliesAcrossValidationAndErrorMapping(t *testing.T) {
	t.Parallel()

	const tag = "even_digits"

	err := validation.RegisterCustomRule(validation.CustomRule{
		Tag: tag,

		Func: func(field validatorv10.FieldLevel) bool {
			return len(field.Field().String())%2 == 0
		},

		Schema: &validation.SchemaEffect{
			Format: "even-digits",
		},

		Code: problem.ValidationErrorCode("even_digits"),

		Message: func(string) string {
			return "must have an even number of characters"
		},
	})
	if err != nil {
		t.Fatalf("RegisterCustomRule failed: %v", err)
	}

	validator, err := validation.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	type request struct {
		Code string `json:"code" validate:"even_digits"`
	}

	validationErrors := validator.Validate(request{Code: "abc"})

	if len(validationErrors) != 1 {
		t.Fatalf(
			"expected 1 validation error, got %d: %+v",
			len(validationErrors),
			validationErrors,
		)
	}

	got := validationErrors[0]

	if got.Detail != "must have an even number of characters" {
		t.Errorf("unexpected detail: %q", got.Detail)
	}

	if got.Code != problem.ValidationErrorCode("even_digits") {
		t.Errorf("unexpected code: %q", got.Code)
	}

	if valid := validator.Validate(request{Code: "abcd"}); len(valid) != 0 {
		t.Errorf("expected no validation errors for even-length code, got %+v", valid)
	}
}

func TestLookupCustomRule_UnknownTagReturnsFalse(t *testing.T) {
	t.Parallel()

	_, ok := validation.LookupCustomRule("definitely_not_registered")
	if ok {
		t.Fatal("expected ok=false for an unregistered tag")
	}
}
