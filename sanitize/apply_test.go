package sanitize_test

import (
	"testing"

	"github.com/Casara/arnon/sanitize"
)

func TestApply_TrimsTopLevelStringField(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `sanitize:"trim"`
	}

	value := request{Name: "  C  "}

	sanitize.Apply(&value)

	if value.Name != "C" {
		t.Fatalf("expected %q, got %q", "C", value.Name)
	}
}

func TestApply_LeavesUntaggedFieldUntouched(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string
	}

	value := request{Name: "  C  "}

	sanitize.Apply(&value)

	if value.Name != "  C  " {
		t.Fatalf("expected untagged field to be left as-is, got %q", value.Name)
	}
}

func TestApply_ChainsMultipleTokensInOrder(t *testing.T) {
	t.Parallel()

	err := sanitize.RegisterFunc("applytest_upper", func(s string) string { return s + "!" })
	if err != nil {
		t.Fatalf("RegisterFunc failed: %v", err)
	}

	type request struct {
		Name string `sanitize:"trim,applytest_upper"`
	}

	value := request{Name: "  C  "}

	sanitize.Apply(&value)

	if value.Name != "C!" {
		t.Fatalf("expected tokens applied in order, got %q", value.Name)
	}
}

func TestApply_SkipsUnknownTokenSilently(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `sanitize:"trim,applytest_doesnotexist"`
	}

	value := request{Name: "  C  "}

	sanitize.Apply(&value)

	if value.Name != "C" {
		t.Fatalf("expected known tokens still applied, got %q", value.Name)
	}
}

func TestApply_RecursesIntoPointerToStructAutomatically(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `sanitize:"trim"`
	}

	type request struct {
		Address *address
	}

	value := request{Address: &address{City: "  São Paulo  "}}

	sanitize.Apply(&value)

	if value.Address.City != "São Paulo" {
		t.Fatalf("expected nested pointer struct field to be trimmed, got %q", value.Address.City)
	}
}

func TestApply_NilPointerToStructIsNoOp(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `sanitize:"trim"`
	}

	type request struct {
		Address *address
	}

	value := request{}

	sanitize.Apply(&value)

	if value.Address != nil {
		t.Fatalf("expected nil pointer to remain nil, got %+v", value.Address)
	}
}

func TestApply_RecursesIntoValueStructAutomatically(t *testing.T) {
	t.Parallel()

	type address struct {
		City string `sanitize:"trim"`
	}

	type request struct {
		Address address
	}

	value := request{Address: address{City: "  Recife  "}}

	sanitize.Apply(&value)

	if value.Address.City != "Recife" {
		t.Fatalf("expected nested value struct field to be trimmed, got %q", value.Address.City)
	}
}

func TestApply_DiveAppliesToEachSliceOfStringElement(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `sanitize:"dive,trim"`
	}

	value := request{Tags: []string{" a ", " b "}}

	sanitize.Apply(&value)

	if value.Tags[0] != "a" || value.Tags[1] != "b" {
		t.Fatalf("expected every element trimmed, got %#v", value.Tags)
	}
}

func TestApply_WithoutDiveLeavesSliceOfStringUntouched(t *testing.T) {
	t.Parallel()

	type request struct {
		Tags []string `sanitize:"trim"`
	}

	value := request{Tags: []string{" a "}}

	sanitize.Apply(&value)

	if value.Tags[0] != " a " {
		t.Fatalf("expected slice left untouched without dive, got %#v", value.Tags)
	}
}

func TestApply_DiveRecursesIntoSliceOfStructElements(t *testing.T) {
	t.Parallel()

	type item struct {
		Name string `sanitize:"trim"`
	}

	type request struct {
		Items []item `sanitize:"dive"`
	}

	value := request{Items: []item{{Name: " a "}, {Name: " b "}}}

	sanitize.Apply(&value)

	if value.Items[0].Name != "a" || value.Items[1].Name != "b" {
		t.Fatalf("expected every element's field trimmed, got %#v", value.Items)
	}
}

func TestApply_DiveAppliesToEachMapOfStringValue(t *testing.T) {
	t.Parallel()

	type request struct {
		Labels map[string]string `sanitize:"dive,trim"`
	}

	value := request{Labels: map[string]string{"a": " x ", "b": " y "}}

	sanitize.Apply(&value)

	if value.Labels["a"] != "x" || value.Labels["b"] != "y" {
		t.Fatalf("expected every map value trimmed, got %#v", value.Labels)
	}
}

func TestApply_SelfReferentialStructDoesNotRecurseForever(t *testing.T) {
	t.Parallel()

	type node struct {
		Name  string `sanitize:"trim"`
		Child *node
	}

	root := &node{Name: " a "}
	current := root

	// One level past maxDepth is enough to prove the bound stops
	// recursion instead of overflowing the stack; it doesn't need to
	// build a cycle, just a chain deeper than the limit.
	for range 20 {
		current.Child = &node{Name: " b "}
		current = current.Child
	}

	sanitize.Apply(root)

	if root.Name != "a" {
		t.Fatalf("expected the root field to still be sanitized, got %q", root.Name)
	}
}

func TestApply_NonPointerTargetIsNoOp(t *testing.T) {
	t.Parallel()

	type request struct {
		Name string `sanitize:"trim"`
	}

	value := request{Name: "  C  "}

	sanitize.Apply(value)

	if value.Name != "  C  " {
		t.Fatalf("expected non-pointer target to be left as-is, got %q", value.Name)
	}
}
