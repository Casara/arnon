package sanitize_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Casara/arnon/sanitize"
)

func TestPrepare_AcceptsKnownTags(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `sanitize:"trim,email"`
	}

	err := sanitize.Prepare(reflect.TypeFor[request]())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPrepare_RejectsUnknownTopLevelTag(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `sanitize:"preparetest_bogus"`
	}

	err := sanitize.Prepare(reflect.TypeFor[request]())
	if !errors.Is(err, sanitize.ErrUnknownSanitizer) {
		t.Fatalf("expected ErrUnknownSanitizer, got %v", err)
	}

	if !strings.Contains(err.Error(), "Name") {
		t.Fatalf("expected error to mention the field name, got %v", err)
	}
}

func TestPrepare_CatchesUnknownTagInsideNilablePointerField(t *testing.T) {
	t.Parallel()

	type inner struct {
		Name string `sanitize:"preparetest_alsobogus"`
	}

	type request struct {
		// Inner is nil in the zero value: Prepare must still catch a
		// bad tag inside it, since Apply would never reach a nil
		// pointer field to do so at request time.
		Inner *inner
	}

	err := sanitize.Prepare(reflect.TypeFor[request]())
	if !errors.Is(err, sanitize.ErrUnknownSanitizer) {
		t.Fatalf("expected ErrUnknownSanitizer for a field behind a nil pointer, got %v", err)
	}

	if !strings.Contains(err.Error(), "Inner.Name") {
		t.Fatalf("expected error to mention the nested field path, got %v", err)
	}
}

func TestPrepare_RejectsUnknownTagInsideDivedSlice(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `sanitize:"dive,preparetest_slicebogus"`
	}

	err := sanitize.Prepare(reflect.TypeFor[request]())
	if !errors.Is(err, sanitize.ErrUnknownSanitizer) {
		t.Fatalf("expected ErrUnknownSanitizer, got %v", err)
	}
}

func TestPrepare_SelfReferentialStructDoesNotRecurseForever(t *testing.T) {
	t.Parallel()

	type node struct {
		Name  string `sanitize:"trim"`
		Child *node
	}

	// A self-referential type has no finite depth to walk to
	// completion - maxDepth must stop this instead of recursing until
	// the stack overflows.
	err := sanitize.Prepare(reflect.TypeFor[node]())
	if err != nil {
		t.Fatalf("expected no error (valid tags all the way down), got %v", err)
	}
}

func TestPrepare_IgnoresTagOnSliceWithoutDive(t *testing.T) {
	t.Parallel()

	type request struct {
		// Without "dive", this tag is never applied to anything (see
		// Apply), so Prepare must not flag it either - both must agree
		// on what a tag means.
		Tags []string `sanitize:"preparetest_ignored"`
	}

	err := sanitize.Prepare(reflect.TypeFor[request]())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
