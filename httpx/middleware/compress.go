package middleware

import (
	"compress/gzip"
	"fmt"
	"net/http"
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
			if !strings.Contains(
				request.Header.Get("Accept-Encoding"),
				"gzip",
			) {
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
