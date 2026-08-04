package openapi

// NavigationTag creates a navigation tag.
func NavigationTag(name string) Tag {
	return Tag{
		Name: name,
		Kind: TagKindNavigation,
	}
}

// BadgeTag creates a badge tag.
func BadgeTag(name string) Tag {
	return Tag{
		Name: name,
		Kind: TagKindBadge,
	}
}

// AudienceTag creates an audience tag.
func AudienceTag(name string) Tag {
	return Tag{
		Name: name,
		Kind: TagKindAudience,
	}
}
