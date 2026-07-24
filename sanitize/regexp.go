package sanitize

import "regexp"

// FromRegexp returns a sanitizer that removes every substring
// matching pattern, e.g. FromRegexp(regexp.MustCompile(`[^0-9]`))
// keeps only digits.
//
// pattern must be compiled once by the caller - typically a
// package-level var - and reused across calls: compiling a pattern
// from a struct tag string on every request would be the same
// per-call recompilation cost that made mrz1836/go-sanitize deprecate
// its own Custom function in favor of CustomCompiled. Register the
// result with RegisterFunc to use it from a `sanitize` tag.
func FromRegexp(pattern *regexp.Regexp) func(string) string {
	return func(value string) string {
		return pattern.ReplaceAllString(value, "")
	}
}
