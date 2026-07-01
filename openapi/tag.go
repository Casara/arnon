package openapi

// TagKind defines the visual role of a tag.
type TagKind string

const (
	// TagKindAudience tags with `kind: audience` indicate the intended audience for an operation.
	TagKindAudience TagKind = "audience"

	// TagKindBadge tags with `kind: badge` are applied as visible badges in documentation.
	TagKindBadge TagKind = "badge"

	// TagKindNavigation tags with `kind: nav` are used in documentation to group operations
	// into sections.
	TagKindNavigation TagKind = "nav"
)

// Tag adds metadata to a single tag that is used by the Operation Object. It is not
// mandatory to have a Tag Object per tag defined in the Operation Object instances.
//
// Example:
//
//	Tags: []openapi.Tag{
//		{
//			Name:        "account-updates",
//			Summary:     "Account Updates",
//			Description: "Account update operations",
//			Kind:        openapi.TagKindNavigation,
//		},
//		{
//			Name:        "partner",
//			Summary:     "Partner",
//			Description: "Operations available to the partners network",
//			Parent:      "external",
//			Kind:        openapi.TagKindAudience,
//		},
//		{
//			Name:        "external",
//			Summary:     "External",
//			Description: "Operations available to external consumers",
//			Kind:        openapi.TagKindAudience,
//		},
//	}
type Tag struct {
	// REQUIRED. The name of the tag. Use this value in the `tags` array of an Operation.
	Name string `json:"name"`

	// A short summary of the tag, used for display purposes.
	Summary string `json:"summary,omitempty"`

	// A description for the tag. [CommonMark] syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`

	// Additional external documentation for this tag.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`

	// The `name` of a tag that this tag is nested under. The named tag MUST exist in the
	// API description, and circular references between parent and child tags MUST NOT be used.
	Parent string `json:"parent,omitempty"`

	// A machine-readable string to categorize what sort of tag it is. Any string value can be
	// used; common uses are `nav` for Navigation, `badge` for visible badges, `audience` for
	// APIs used by different groups.
	Kind TagKind `json:"kind,omitempty"`
}

// WithSummary sets the tag summary.
func (tag Tag) WithSummary(summary string) Tag {
	tag.Summary = summary

	return tag
}

// WithDescription sets the tag description.
func (tag Tag) WithDescription(description string) Tag {
	tag.Description = description

	return tag
}

// WithParent sets the tag parent.
func (tag Tag) WithParent(parent string) Tag {
	tag.Parent = parent

	return tag
}

// WithExternalDocs sets the tag external docs.
func (tag Tag) WithExternalDocs(docs *ExternalDocs) Tag {
	tag.ExternalDocs = docs

	return tag
}

// WithKind sets the tag kind.
func (tag Tag) WithKind(kind TagKind) Tag {
	tag.Kind = kind

	return tag
}
