package openapi_test

import (
	"testing"

	"github.com/casara/arnon/openapi"
)

// TestTag_WithMethodsSetFieldsAndReturnCopies confirms every Tag
// builder method sets the expected field and returns a new value
// rather than mutating the receiver - Tag's WithXxx methods have
// value (not pointer) receivers specifically so a shared base Tag can
// be reused across variants without aliasing.
func TestTag_WithMethodsSetFieldsAndReturnCopies(t *testing.T) {
	t.Parallel()

	base := openapi.Tag{Name: "users"}

	withSummary := base.WithSummary("Users")

	if withSummary.Summary != "Users" {
		t.Errorf("expected Summary %q, got %q", "Users", withSummary.Summary)
	}

	if base.Summary != "" {
		t.Errorf("expected WithSummary not to mutate the receiver, got %q", base.Summary)
	}

	withDescription := base.WithDescription("User management")

	if withDescription.Description != "User management" {
		t.Errorf(
			"expected Description %q, got %q",
			"User management",
			withDescription.Description,
		)
	}

	withParent := base.WithParent("accounts")

	if withParent.Parent != "accounts" {
		t.Errorf("expected Parent %q, got %q", "accounts", withParent.Parent)
	}

	docs := &openapi.ExternalDocs{URL: "https://example.com"}

	withExternalDocs := base.WithExternalDocs(docs)

	if withExternalDocs.ExternalDocs != docs {
		t.Errorf("expected ExternalDocs %v, got %v", docs, withExternalDocs.ExternalDocs)
	}

	withKind := base.WithKind(openapi.TagKindBadge)

	if withKind.Kind != openapi.TagKindBadge {
		t.Errorf("expected Kind %q, got %q", openapi.TagKindBadge, withKind.Kind)
	}

	// Chaining should compose, since each With method returns a Tag
	// (not *Tag) that already carries every prior change.
	chained := base.
		WithSummary("Users").
		WithDescription("User management").
		WithParent("accounts").
		WithKind(openapi.TagKindNavigation)

	if chained.Summary != "Users" || chained.Description != "User management" ||
		chained.Parent != "accounts" || chained.Kind != openapi.TagKindNavigation {
		t.Errorf("expected all chained fields to be set, got %+v", chained)
	}
}
