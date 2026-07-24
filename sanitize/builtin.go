package sanitize

import "strings"

// trim removes leading and trailing whitespace, registered under the
// "trim" tag.
func trim(value string) string {
	return strings.TrimSpace(value)
}

// normalizeEmail trims and lowercases a value, registered under the
// "email" tag - email addresses are conventionally treated as
// case-insensitive, and leading/trailing whitespace is never
// significant.
func normalizeEmail(value string) string {
	return strings.ToLower(trim(value))
}
