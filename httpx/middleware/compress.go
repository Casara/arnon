package middleware

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Casara/arnon/httpx/routing"
)

// defaultCompressibleTypes lists the response Content-Type values
// Compress compresses by default, when called without an explicit
// types list. Adapted from chi's middleware.Compress default list,
// with arnon's own "application/problem+json" added since every
// error response uses it.
//
//nolint:gochecknoglobals // read-only reference data, not per-request state
var defaultCompressibleTypes = []string{
	"application/json",
	"application/problem+json",
	"text/html",
	"text/css",
	"text/plain",
	"text/javascript",
	"application/javascript",
	"application/xml",
	"text/xml",
	"image/svg+xml",
}

// Compress gzip-compresses the response body when the client sends
// "Accept-Encoding: gzip" and the handler's response Content-Type
// matches one of types, or the default list of common compressible
// types when types is empty. A trailing "/*" in a type matches any
// subtype, e.g. "text/*".
//
// Checking the response Content-Type (rather than always compressing)
// avoids wasting CPU on content that gains little or nothing from
// gzip, such as images.
//
// level is the gzip compression level, as defined by compress/gzip
// (gzip.DefaultCompression is a reasonable default). An invalid level
// panics immediately, since that is a configuration error rather than
// something that can happen at request time.
//
// A request carrying a Range header runs next unwrapped, for a
// related reason confirmed empirically (see examples/cmd/staticfiles):
// Compress deletes Content-Length once it decides to gzip (correct for
// a full body), but never touches Content-Range, which
// http.ServeContent had already set to describe byte positions in the
// *uncompressed* resource. Left wrapped, a 206 response would end up
// advertising e.g. "Content-Range: bytes 0-99/10000" while the bytes
// actually sent are a self-contained gzip stream of a different
// length - internally decodable, but not what Content-Range means,
// and not something a real Range-aware client (resuming a download by
// byte offset) can use correctly. Stepping aside for Range requests
// means Compress can safely sit in front of a Range-capable handler
// (http.FileServer, http.ServeContent): a plain GET still gets
// gzipped when applicable, a Range GET is left exactly as
// http.ServeContent produced it.
func Compress(
	level int,
	types ...string,
) routing.Middleware {
	_, err := gzip.NewWriterLevel(nil, level)
	if err != nil {
		panic(fmt.Errorf("middleware: invalid compress level %d: %w", level, err))
	}

	if len(types) == 0 {
		types = defaultCompressibleTypes
	}

	allowedTypes, allowedWildcards := splitCompressibleTypes(types)

	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if !acceptsGzip(request.Header.Get("Accept-Encoding")) {
				next.ServeHTTP(
					writer,
					request,
				)

				return
			}

			if request.Header.Get("Range") != "" {
				next.ServeHTTP(
					writer,
					request,
				)

				return
			}

			compressWriter := &compressResponseWriter{
				ResponseWriter:   writer,
				level:            level,
				allowedTypes:     allowedTypes,
				allowedWildcards: allowedWildcards,
			}
			defer func() { _ = compressWriter.Close() }()

			next.ServeHTTP(
				compressWriter,
				request,
			)
		})
	}
}

func splitCompressibleTypes(
	types []string,
) (map[string]struct{}, map[string]struct{}) {
	allowedTypes := make(map[string]struct{})
	allowedWildcards := make(map[string]struct{})

	for _, contentType := range types {
		prefix, ok := strings.CutSuffix(
			contentType,
			"/*",
		)
		if ok {
			allowedWildcards[prefix] = struct{}{}

			continue
		}

		allowedTypes[contentType] = struct{}{}
	}

	return allowedTypes, allowedWildcards
}

// compressResponseWriter decides whether to compress once the
// handler's status/headers are written (so it can inspect the
// response Content-Type), then redirects writes through a gzip.Writer
// when compression applies.
type compressResponseWriter struct {
	http.ResponseWriter

	level            int
	allowedTypes     map[string]struct{}
	allowedWildcards map[string]struct{}

	gzipWriter   *gzip.Writer
	wroteHeader  bool
	compressible bool
}

func (writer *compressResponseWriter) WriteHeader(
	statusCode int,
) {
	if writer.wroteHeader {
		writer.ResponseWriter.WriteHeader(
			statusCode,
		)

		return
	}

	writer.wroteHeader = true

	if writer.isCompressible() {
		writer.compressible = true

		header := writer.Header()

		header.Set(
			"Content-Encoding",
			"gzip",
		)

		header.Add(
			"Vary",
			"Accept-Encoding",
		)

		header.Del("Content-Length")

		// level was already validated in Compress, so this can't fail.
		writer.gzipWriter, _ = gzip.NewWriterLevel(
			writer.ResponseWriter,
			writer.level,
		)
	}

	writer.ResponseWriter.WriteHeader(
		statusCode,
	)
}

func (writer *compressResponseWriter) Write(
	data []byte,
) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}

	if !writer.compressible {
		written, err := writer.ResponseWriter.Write(
			data,
		)
		if err != nil {
			return written, fmt.Errorf("write response: %w", err)
		}

		return written, nil
	}

	written, err := writer.gzipWriter.Write(
		data,
	)
	if err != nil {
		return written, fmt.Errorf("write gzip response: %w", err)
	}

	return written, nil
}

// Close flushes and closes the underlying gzip.Writer, if compression
// was used for this response. It must be called (via defer) once the
// handler returns.
func (writer *compressResponseWriter) Close() error {
	if writer.gzipWriter == nil {
		return nil
	}

	err := writer.gzipWriter.Close()
	if err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	return nil
}

func (writer *compressResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *compressResponseWriter) isCompressible() bool {
	contentType, _, _ := strings.Cut(
		writer.Header().Get("Content-Type"),
		";",
	)

	if _, ok := writer.allowedTypes[contentType]; ok {
		return true
	}

	prefix, _, ok := strings.Cut(
		contentType,
		"/",
	)
	if !ok {
		return false
	}

	_, ok = writer.allowedWildcards[prefix]

	return ok
}

// acceptsGzip reports whether an Accept-Encoding header value
// indicates the client accepts a gzip-encoded response.
//
// Unlike Accept (RFC 9110 §12.5.1), an absent Accept-Encoding does
// NOT imply "any encoding is fine" as far as this function is
// concerned - a client that never asked for gzip doesn't get it
// compressed, the same conservative default this middleware always
// had. What changes is precision: matching is done per RFC 9110
// §12.5.3 (parsing "gzip"/"*" tokens with their "q" parameter, most
// specific match wins) instead of a plain substring search, so
// "gzip;q=0" - a client explicitly declining gzip - is correctly
// treated as not accepting it, which a bare strings.Contains check
// would have missed entirely.
func acceptsGzip(acceptEncoding string) bool {
	if acceptEncoding == "" {
		return false
	}

	bestSpecificity := -1

	bestQuality := 0.0

	for entry := range strings.SplitSeq(acceptEncoding, ",") {
		coding, quality, ok := parseEncodingRange(entry)
		if !ok {
			continue
		}

		specificity, matches := codingSpecificity(coding)
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

// parseEncodingRange parses one Accept-Encoding entry
// ("gzip;q=0.5") into its coding name and quality value, defaulting
// to 1 when the "q" parameter is absent or malformed.
func parseEncodingRange(entry string) (string, float64, bool) {
	parameters := strings.Split(entry, ";")

	coding := strings.ToLower(strings.TrimSpace(parameters[0]))
	if coding == "" {
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

	return coding, quality, true
}

// codingSpecificity reports whether coding is relevant to gzip
// acceptance, and how specific it is ("gzip" itself outranks the "*"
// wildcard), so the most specific applicable entry's quality value
// is what decides acceptability when both appear in the same header.
func codingSpecificity(coding string) (int, bool) {
	switch coding {
	case "gzip":
		return 1, true
	case "*":
		return 0, true
	default:
		return 0, false
	}
}
