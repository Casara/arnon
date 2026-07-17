package openapi_test

import (
	"testing"

	"github.com/Casara/arnon/openapi"
)

// TestTagHelpers_CreateTagsOfTheExpectedKind covers the three
// constructor shortcuts in tag_helpers.go, each expected to set Name
// and the matching TagKind and nothing else.
func TestTagHelpers_CreateTagsOfTheExpectedKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		build    func(string) openapi.Tag
		wantKind openapi.TagKind
	}{
		{"NavigationTag", openapi.NavigationTag, openapi.TagKindNavigation},
		{"BadgeTag", openapi.BadgeTag, openapi.TagKindBadge},
		{"AudienceTag", openapi.AudienceTag, openapi.TagKindAudience},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tag := testCase.build("payments")

			if tag.Name != "payments" {
				t.Errorf("expected Name %q, got %q", "payments", tag.Name)
			}

			if tag.Kind != testCase.wantKind {
				t.Errorf("expected Kind %q, got %q", testCase.wantKind, tag.Kind)
			}
		})
	}
}
