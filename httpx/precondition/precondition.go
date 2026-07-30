package precondition

import (
	"net/http"
	"strings"
	"time"

	"github.com/casara/arnon/problem"
)

// Config configures Check/CheckRequest.
type Config struct {
	// Require, when true, rejects a request carrying neither If-Match
	// nor If-Unmodified-Since with 428 Precondition Required (RFC
	// 6585 §3) instead of letting it through unconditionally. Default
	// false: most APIs don't want every write to mandate a
	// precondition.
	Require bool
}

// Check validates the write preconditions of RFC 9110 §13.1.1
// (If-Match) and §13.1.4 (If-Unmodified-Since) against a resource's
// actual current state - etag and lastModified, both supplied by the
// caller, since only the caller (having already loaded the resource
// to apply the write) knows it. Meant to be called from inside a
// typed httpx.HandlerFunc, using header values already bound onto the
// request DTO via `header:"If-Match"`/`header:"If-Unmodified-Since"`
// tags (see httpx/binding) - HandlerFunc never receives *http.Request,
// so Check can't read headers itself. CheckRequest below is the
// equivalent for a plain http.Handler.
//
// If-Match takes precedence when both are present (RFC 9110 §13.1.4:
// a server evaluating If-Unmodified-Since MUST ignore it when
// If-Match is also present), and uses strong comparison (RFC 9110
// §13.1.1) - unlike If-None-Match's weak comparison for GET/HEAD
// (middleware.ETag), a weak ETag (W/"...") can never satisfy If-Match
// on either side. "*" matches any existing resource (etag != ""); an
// empty etag (the caller's resource has no ETag at all) never
// satisfies any If-Match value, "*" included.
//
// An unparseable If-Unmodified-Since date, or a zero lastModified (the
// caller doesn't track modification time), is treated as unverifiable
// and ignored (the request proceeds) rather than rejected - the same
// "can't evaluate, don't block" default RFC 9110 leaves to server
// discretion.
//
// Returns nil when the request may proceed, a 412 Precondition Failed
// problem.Problem when a precondition present fails, or (if
// config.Require) a 428 Precondition Required problem.Problem when
// neither header is present.
func Check(
	ifMatch string,
	ifUnmodifiedSince string,
	etag string,
	lastModified time.Time,
	config Config,
) *problem.Problem {
	ifMatch = strings.TrimSpace(ifMatch)
	ifUnmodifiedSince = strings.TrimSpace(ifUnmodifiedSince)

	if ifMatch == "" && ifUnmodifiedSince == "" {
		if config.Require {
			return problem.NewPreconditionRequired(
				"this request requires an If-Match or If-Unmodified-Since header",
			)
		}

		return nil
	}

	if ifMatch != "" {
		if !ifMatchMatches(ifMatch, etag) {
			return problem.NewPreconditionFailed(
				"the resource's current state does not match If-Match",
			)
		}

		return nil
	}

	if !unmodifiedSince(ifUnmodifiedSince, lastModified) {
		return problem.NewPreconditionFailed(
			"the resource has been modified since If-Unmodified-Since",
		)
	}

	return nil
}

// CheckRequest reads If-Match/If-Unmodified-Since directly from
// request and calls Check - for a plain http.Handler or
// framework-internal code (httpx/patch.From) that already holds
// *http.Request, instead of a typed HandlerFunc's bound DTO.
func CheckRequest(
	request *http.Request,
	etag string,
	lastModified time.Time,
	config Config,
) *problem.Problem {
	return Check(
		request.Header.Get("If-Match"),
		request.Header.Get("If-Unmodified-Since"),
		etag,
		lastModified,
		config,
	)
}

// ifMatchMatches reports whether etag satisfies the client's If-Match
// precondition. Comparison is strong (RFC 9110 §13.1.1): a "W/" prefix
// on either side means that side can never satisfy the match, even if
// the opaque values are otherwise equal.
func ifMatchMatches(header, etag string) bool {
	if header == "*" {
		return etag != ""
	}

	if etag == "" || strings.HasPrefix(etag, "W/") {
		return false
	}

	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(candidate, "W/") {
			continue
		}

		if candidate == etag {
			return true
		}
	}

	return false
}

// unmodifiedSince reports whether lastModified satisfies the client's
// If-Unmodified-Since precondition. An unparseable header or a zero
// lastModified is treated as unverifiable and reported as satisfied
// (see Check's doc comment). HTTP-date has second precision, so
// lastModified is truncated before comparing.
func unmodifiedSince(header string, lastModified time.Time) bool {
	if lastModified.IsZero() {
		return true
	}

	parsed, err := http.ParseTime(header)
	if err != nil {
		return true
	}

	return !lastModified.Truncate(time.Second).After(parsed)
}
