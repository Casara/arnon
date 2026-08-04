package sanitize_test

import (
	"errors"
	"testing"

	"github.com/casara/arnon/sanitize"
)

func TestRegisterFunc_RejectsEmptyTag(t *testing.T) {
	t.Parallel()

	err := sanitize.RegisterFunc("", func(s string) string { return s })

	if !errors.Is(err, sanitize.ErrSanitizerTagEmpty) {
		t.Fatalf("expected ErrSanitizerTagEmpty, got %v", err)
	}
}

func TestRegisterFunc_RejectsNilFunc(t *testing.T) {
	t.Parallel()

	err := sanitize.RegisterFunc("nilfunc", nil)

	if !errors.Is(err, sanitize.ErrSanitizerFuncNil) {
		t.Fatalf("expected ErrSanitizerFuncNil, got %v", err)
	}
}

func TestRegisterFunc_OverridesExistingTag(t *testing.T) {
	t.Parallel()

	err := sanitize.RegisterFunc("registertest_override", func(s string) string { return "first" })
	if err != nil {
		t.Fatalf("first RegisterFunc failed: %v", err)
	}

	err = sanitize.RegisterFunc("registertest_override", func(s string) string { return "second" })
	if err != nil {
		t.Fatalf("second RegisterFunc failed: %v", err)
	}

	type target struct {
		Value string `sanitize:"registertest_override"`
	}

	value := target{Value: "anything"}

	sanitize.Apply(&value)

	if value.Value != "second" {
		t.Fatalf("expected the later registration to win, got %q", value.Value)
	}
}

func TestRegisterFunc_BuiltinsAvailableWithoutRegistration(t *testing.T) {
	t.Parallel()

	type target struct {
		Name  string `sanitize:"trim"`
		Email string `sanitize:"email"`
	}

	value := target{Name: "  Ada  ", Email: "  ADA@Example.com "}

	sanitize.Apply(&value)

	if value.Name != "Ada" {
		t.Fatalf("expected trim to strip whitespace, got %q", value.Name)
	}

	if value.Email != "ada@example.com" {
		t.Fatalf("expected email to be trimmed and lowercased, got %q", value.Email)
	}
}
