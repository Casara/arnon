package sanitize

import "strings"

// parseTag splits a `sanitize` tag value into comma-separated tokens,
// and reports whether the first token is "dive" - the marker used on
// slice/array/map fields to apply the remaining tokens to each
// element instead of to the field itself, mirroring validate's own
// dive convention.
func parseTag(tag string) ([]string, bool) {
	if tag == "" {
		return nil, false
	}

	tokens := strings.Split(tag, ",")

	if tokens[0] == "dive" {
		return tokens[1:], true
	}

	return tokens, false
}
