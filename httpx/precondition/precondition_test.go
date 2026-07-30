package precondition_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/casara/arnon/httpx/precondition"
)

func TestCheck_BothAbsentAllowsRequestByDefault(t *testing.T) {
	t.Parallel()

	got := precondition.Check("", "", `"abc"`, time.Time{}, precondition.Config{})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_BothAbsentRequiredReturns428(t *testing.T) {
	t.Parallel()

	got := precondition.Check("", "", `"abc"`, time.Time{}, precondition.Config{Require: true})

	if got == nil || got.StatusCode() != http.StatusPreconditionRequired {
		t.Fatalf("expected 428, got %+v", got)
	}
}

func TestCheck_IfMatchExactMatchAllowsRequest(t *testing.T) {
	t.Parallel()

	got := precondition.Check(`"abc"`, "", `"abc"`, time.Time{}, precondition.Config{})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_IfMatchMismatchReturns412(t *testing.T) {
	t.Parallel()

	got := precondition.Check(`"abc"`, "", `"def"`, time.Time{}, precondition.Config{})

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheck_IfMatchMatchesOneOfSeveralCandidates(t *testing.T) {
	t.Parallel()

	got := precondition.Check(`"abc", "def"`, "", `"def"`, time.Time{}, precondition.Config{})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_IfMatchWildcardMatchesAnyExistingResource(t *testing.T) {
	t.Parallel()

	got := precondition.Check("*", "", `"abc"`, time.Time{}, precondition.Config{})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_IfMatchWildcardFailsWhenResourceHasNoETag(t *testing.T) {
	t.Parallel()

	got := precondition.Check("*", "", "", time.Time{}, precondition.Config{})

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheck_IfMatchNeverMatchesWhenResourceETagIsWeak(t *testing.T) {
	t.Parallel()

	got := precondition.Check(`"abc"`, "", `W/"abc"`, time.Time{}, precondition.Config{})

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheck_IfMatchNeverMatchesAWeakCandidate(t *testing.T) {
	t.Parallel()

	got := precondition.Check(`W/"abc"`, "", `"abc"`, time.Time{}, precondition.Config{})

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheck_IfMatchTakesPrecedenceOverIfUnmodifiedSince(t *testing.T) {
	t.Parallel()

	// If-Match matches, but If-Unmodified-Since would fail on its own -
	// RFC 9110 §13.1.4 says a server MUST ignore If-Unmodified-Since
	// when If-Match is present, so this must still succeed.
	got := precondition.Check(
		`"abc"`,
		"Wed, 21 Oct 2015 07:28:00 GMT",
		`"abc"`,
		time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		precondition.Config{},
	)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_IfUnmodifiedSinceSatisfiedWhenNotModifiedAfter(t *testing.T) {
	t.Parallel()

	got := precondition.Check(
		"",
		"Wed, 21 Oct 2026 07:28:00 GMT",
		"",
		time.Date(2026, time.October, 21, 7, 28, 0, 0, time.UTC),
		precondition.Config{},
	)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_IfUnmodifiedSinceFailsWhenModifiedAfter(t *testing.T) {
	t.Parallel()

	got := precondition.Check(
		"",
		"Wed, 21 Oct 2026 07:28:00 GMT",
		"",
		time.Date(2026, time.October, 21, 7, 28, 1, 0, time.UTC),
		precondition.Config{},
	)

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheck_UnparseableIfUnmodifiedSinceIsIgnored(t *testing.T) {
	t.Parallel()

	got := precondition.Check(
		"",
		"not a date",
		"",
		time.Date(2026, time.October, 21, 7, 28, 1, 0, time.UTC),
		precondition.Config{},
	)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheck_ZeroLastModifiedIsIgnored(t *testing.T) {
	t.Parallel()

	got := precondition.Check(
		"",
		"Wed, 21 Oct 2015 07:28:00 GMT",
		"",
		time.Time{},
		precondition.Config{},
	)
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestCheckRequest_ReadsHeadersFromRealRequest(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPut, "/", nil)
	request.Header.Set("If-Match", `"abc"`)

	got := precondition.CheckRequest(request, `"def"`, time.Time{}, precondition.Config{})

	if got == nil || got.StatusCode() != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %+v", got)
	}
}

func TestCheckRequest_NoConditionalHeadersAllowsRequest(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPut, "/", nil)

	got := precondition.CheckRequest(request, `"abc"`, time.Time{}, precondition.Config{})
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
