package httpx

import (
	"strconv"
	"strings"
)

// Specificity of a media-range match, used to pick which range's
// quality value applies when several in the same Accept header match
// the same target media type - the most specific one wins (RFC 9110
// §12.5.1).
const (
	anyWildcardSpecificity  = 0
	typeWildcardSpecificity = 1
	exactMatchSpecificity   = 2
)

// acceptsJSON reports whether an Accept header value indicates the
// client can accept an application/json response.
//
// A missing or blank Accept header means the client accepts any
// media type (RFC 9110 §12.5.1), so this returns true in that case -
// only an Accept header that explicitly excludes application/json
// (every applicable media range has q=0, or none matches at all)
// makes it return false.
func acceptsJSON(accept string) bool {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return true
	}

	bestSpecificity := -1

	bestQuality := 0.0

	for entry := range strings.SplitSeq(accept, ",") {
		mediaType, quality, ok := parseMediaRange(entry)
		if !ok {
			continue
		}

		specificity, matches := matchSpecificity(mediaType, "application/json")
		if !matches {
			continue
		}

		if specificity > bestSpecificity {
			bestSpecificity = specificity
			bestQuality = quality
		}
	}

	return bestSpecificity >= 0 && bestQuality > 0
}

// parseMediaRange parses one Accept entry ("type/subtype;q=0.5;
// other=x") into its media type and quality value, defaulting to 1
// when the "q" parameter is absent or malformed.
func parseMediaRange(entry string) (string, float64, bool) {
	parameters := strings.Split(entry, ";")

	mediaType := strings.ToLower(strings.TrimSpace(parameters[0]))
	if mediaType == "" {
		return "", 0, false
	}

	quality := 1.0

	for _, parameter := range parameters[1:] {
		name, value, found := strings.Cut(parameter, "=")
		if !found {
			continue
		}

		if strings.TrimSpace(strings.ToLower(name)) != "q" {
			continue
		}

		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			quality = parsed
		}
	}

	return mediaType, quality, true
}

// matchSpecificity reports whether mediaRange matches target, and how
// specific the match is - the most specific applicable range's
// quality value is what decides acceptability when several ranges in
// the same header match.
func matchSpecificity(mediaRange, target string) (int, bool) {
	if mediaRange == target {
		return exactMatchSpecificity, true
	}

	if mediaRange == "*/*" {
		return anyWildcardSpecificity, true
	}

	targetType, _, found := strings.Cut(target, "/")
	if !found {
		return anyWildcardSpecificity, false
	}

	if mediaRange == targetType+"/*" {
		return typeWildcardSpecificity, true
	}

	return anyWildcardSpecificity, false
}
