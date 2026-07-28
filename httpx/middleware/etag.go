package middleware

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// ETag adds RFC 9111 conditional GET support: it buffers a GET/HEAD
// response, computes a strong ETag from its exact bytes (unless the
// handler already set one itself), and - when the incoming
// If-None-Match header already names that ETag - replaces the
// response with an empty-bodied 304 Not Modified instead of resending
// bytes the client already has.
//
// Only GET/HEAD requests are considered (next runs unwrapped for
// anything else, with no buffering overhead): conditional requests
// for other methods are a different mechanism (If-Match-based
// optimistic concurrency for PUT/PATCH/DELETE) not attempted here.
// Only a 2xx response gets an ETag; redirects and Problem Details
// error responses are forwarded unchanged.
//
// Install this outside (before) Compress in the middleware chain if
// both are used together, so the ETag reflects the bytes actually
// sent to the client (e.g. gzip-compressed) rather than the
// pre-compression body - matching Compress's own Vary: Accept-Encoding,
// a cache correctly ends up with one validator per encoding.
//
// Pairing this with NoCache on the same route defeats the purpose of
// both: NoCache tells clients/caches never to store the response, so
// there is nothing for a future request to send an If-None-Match
// against.
//
// A request carrying a Range header is also passed to next
// unbuffered, for a reason confirmed empirically (see
// examples/cmd/staticfiles): wrapping a Range-capable handler
// (http.FileServer, http.ServeContent, or anything else serving
// resumable downloads or media seeking) otherwise breaks its own
// If-Range/Content-Range handling. ETag only sets its own header
// after next returns, so when http.ServeContent checks an incoming
// If-Range against the response headers it can see so far, no ETag
// exists yet for it to compare against, and If-Range silently fails
// to match - the request falls back to a full 200 instead of the
// expected 206. The ETag that does get set afterward is also unstable
// across requests to the same resource: it hashes whatever bytes were
// actually written for that specific request, so a 206 (partial body)
// and a 200 (full body) for the same file produce two different
// ETags. Stepping aside for Range requests means ETag can safely sit
// in front of a Range-capable handler: a plain GET still gets a
// validator, a Range GET gets neither - just http.ServeContent's own,
// untouched Range/conditional-GET handling via Last-Modified.
func ETag() routing.Middleware {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if request.Method != http.MethodGet && request.Method != http.MethodHead {
				next.ServeHTTP(writer, request)

				return
			}

			if request.Header.Get("Range") != "" {
				next.ServeHTTP(writer, request)

				return
			}

			buffered := &etagResponseWriter{
				header: make(http.Header),
			}

			next.ServeHTTP(buffered, request)

			buffered.flush(writer, request)
		})
	}
}

type etagResponseWriter struct {
	header      http.Header
	buffer      bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (writer *etagResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *etagResponseWriter) Write(data []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}

	n, err := writer.buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("buffer response body: %w", err)
	}

	return n, nil
}

func (writer *etagResponseWriter) WriteHeader(statusCode int) {
	if writer.wroteHeader {
		return
	}

	writer.wroteHeader = true
	writer.statusCode = statusCode
}

// flush replays the buffered response onto destination, the actual
// http.ResponseWriter, either as-is (non-2xx statuses, or a 2xx that
// the request's If-None-Match doesn't match) or as an empty-bodied
// 304 Not Modified (a 2xx response whose ETag the client already has).
func (writer *etagResponseWriter) flush(
	destination http.ResponseWriter,
	request *http.Request,
) {
	if !writer.wroteHeader {
		writer.statusCode = http.StatusOK
	}

	if writer.statusCode < http.StatusOK || writer.statusCode >= http.StatusMultipleChoices {
		copyHeader(destination.Header(), writer.header)
		destination.WriteHeader(writer.statusCode)
		_, _ = destination.Write(writer.buffer.Bytes())

		return
	}

	etag := writer.header.Get("ETag")
	if etag == "" {
		etag = computeETag(writer.buffer.Bytes())
	}

	copyHeader(destination.Header(), writer.header)
	destination.Header().Set("ETag", etag)

	if ifNoneMatchMatches(request.Header.Get("If-None-Match"), etag) {
		destination.WriteHeader(http.StatusNotModified)

		return
	}

	destination.WriteHeader(writer.statusCode)
	_, _ = destination.Write(writer.buffer.Bytes())
}

func copyHeader(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

// computeETag builds a strong ETag (RFC 9110 §8.8.3) from body: a
// quoted-string wrapping a hex-encoded FNV-1a 64-bit hash. FNV isn't
// cryptographically strong, but an ETag is a change-detection
// validator, not a security token, and FNV is stdlib-only (hash/fnv),
// consistent with the project's preference for no new dependencies.
func computeETag(body []byte) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write(body)

	return `"` + hex.EncodeToString(hasher.Sum(nil)) + `"`
}

// ifNoneMatchMatches reports whether etag satisfies the client's
// If-None-Match precondition. Comparison is weak (RFC 9110 §13.1.2:
// GET/HEAD conditional requests SHOULD use weak comparison), so a
// "W/" prefix on either side is ignored - only the opaque quoted
// value has to match.
func ifNoneMatchMatches(header, etag string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}

	if header == "*" {
		return true
	}

	target := strings.TrimPrefix(etag, "W/")

	for candidate := range strings.SplitSeq(header, ",") {
		if strings.TrimPrefix(strings.TrimSpace(candidate), "W/") == target {
			return true
		}
	}

	return false
}
