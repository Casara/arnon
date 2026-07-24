package sanitize_test

import (
	"regexp"
	"testing"

	"github.com/Casara/arnon/sanitize"
)

func TestFromRegexp_KeepsOnlyDigits(t *testing.T) {
	t.Parallel()

	digitsOnly := sanitize.FromRegexp(regexp.MustCompile(`[^0-9]`))

	result := digitsOnly("+55 (11) 91234-5678")

	if result != "5511912345678" {
		t.Fatalf("expected only digits to remain, got %q", result)
	}
}

func TestFromRegexp_UsableThroughRegisterFunc(t *testing.T) {
	t.Parallel()

	err := sanitize.RegisterFunc(
		"regexptest_digitsonly",
		sanitize.FromRegexp(regexp.MustCompile(`[^0-9]`)),
	)
	if err != nil {
		t.Fatalf("RegisterFunc failed: %v", err)
	}

	type request struct {
		Phone string `sanitize:"regexptest_digitsonly"`
	}

	value := request{Phone: "+55 (11) 91234-5678"}

	sanitize.Apply(&value)

	if value.Phone != "5511912345678" {
		t.Fatalf("expected only digits to remain, got %q", value.Phone)
	}
}
