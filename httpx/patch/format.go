package patch

import (
	"errors"
	"fmt"
	"mime"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

// ErrUnsupportedContentType indicates that the incoming PATCH
// request's Content-Type is neither JSON Merge Patch (RFC 7386) nor
// JSON Patch (RFC 6902), nor left absent/plain JSON (which defaults
// to Merge Patch).
var ErrUnsupportedContentType = errors.New("unsupported patch content type")

const (
	contentTypeJSON       = "application/json"
	contentTypeMergePatch = "application/merge-patch+json"
	contentTypeJSONPatch  = "application/json-patch+json"
)

// apply applies patchBody to original according to the format
// selected by contentType:
//
//   - Absent, "application/json", or "application/merge-patch+json"
//     ("" is what an absent Content-Type header parses to): RFC 7386
//     JSON Merge Patch.
//   - "application/json-patch+json": RFC 6902 JSON Patch.
//   - Anything else: ErrUnsupportedContentType.
func apply(contentType string, original, patchBody []byte) ([]byte, error) {
	// mime.ParseMediaType still returns a best-effort main type even
	// when a parameter fails to parse (confirmed empirically, e.g.
	// "application/json; =" still yields mediaType "application/json"
	// alongside the error) - so the parsed value is used regardless of
	// err, never the raw header string, which would otherwise defeat
	// the switch below for any Content-Type with a slightly malformed
	// parameter.
	mediaType, _, _ := mime.ParseMediaType(contentType)

	switch mediaType {
	case "", contentTypeJSON, contentTypeMergePatch:
		merged, mergeErr := jsonpatch.MergePatch(original, patchBody)
		if mergeErr != nil {
			return nil, fmt.Errorf("apply merge patch: %w", mergeErr)
		}

		return merged, nil

	case contentTypeJSONPatch:
		operations, decodeErr := jsonpatch.DecodePatch(patchBody)
		if decodeErr != nil {
			return nil, fmt.Errorf("decode json patch: %w", decodeErr)
		}

		merged, applyErr := operations.Apply(original)
		if applyErr != nil {
			return nil, fmt.Errorf("apply json patch: %w", applyErr)
		}

		return merged, nil

	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedContentType, contentType)
	}
}
