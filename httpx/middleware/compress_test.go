package middleware_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Casara/arnon/httpx/middleware"
)

func writeWithContentType(contentType, body string) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		if contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}

		_, _ = writer.Write([]byte(body))
	})
}

func TestCompress_CompressesAllowedContentType(t *testing.T) {
	t.Parallel()

	const body = "hello, compressed world"

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("application/json", body),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip, deflate")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf(
			"expected Content-Encoding: gzip, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}

	reader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatalf("response body is not valid gzip: %v", err)
	}
	defer func() { _ = reader.Close() }()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress response body: %v", err)
	}

	if string(decompressed) != body {
		t.Errorf("expected decompressed body %q, got %q", body, string(decompressed))
	}
}

func TestCompress_PassesThroughWithoutAcceptEncoding(t *testing.T) {
	t.Parallel()

	const body = "plain response"

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("application/json", body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding, got %q", recorder.Header().Get("Content-Encoding"))
	}

	if recorder.Body.String() != body {
		t.Errorf("expected plain body %q, got %q", body, recorder.Body.String())
	}
}

func TestCompress_RangeRequestPassesThroughUnwrapped(t *testing.T) {
	t.Parallel()

	const body = "plain response"

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("application/json", body),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("Range", "bytes=0-3")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf(
			"expected no Content-Encoding for a Range request, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}

	if recorder.Body.String() != body {
		t.Errorf("expected uncompressed body %q, got %q", body, recorder.Body.String())
	}
}

// TestCompress_DoesNotCorruptContentRangeOnAWrappedFileServer is the
// regression test for a real interaction confirmed empirically (see
// examples/cmd/staticfiles): wrapping a Range-capable handler in
// Compress used to gzip the partial body while leaving Content-Range
// describing byte positions in the uncompressed resource, so the two
// no longer agreed.
func TestCompress_DoesNotCorruptContentRangeOnAWrappedFileServer(t *testing.T) {
	t.Parallel()

	content := strings.NewReader("0123456789")

	fileHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, "numbers.txt", time.Time{}, content)
	})

	handler := middleware.Compress(gzip.DefaultCompression)(fileHandler)

	request := httptest.NewRequest(http.MethodGet, "/numbers.txt", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("Range", "bytes=0-3")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("expected status %d, got %d", http.StatusPartialContent, recorder.Code)
	}

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf(
			"expected no Content-Encoding on a Range response, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}

	if got := recorder.Header().Get("Content-Range"); got != "bytes 0-3/10" {
		t.Errorf("expected Content-Range %q, got %q", "bytes 0-3/10", got)
	}

	if recorder.Body.String() != "0123" {
		t.Errorf("expected uncompressed partial body %q, got %q", "0123", recorder.Body.String())
	}
}

func TestCompress_ExplicitQZeroDeclinesGzip(t *testing.T) {
	t.Parallel()

	const body = "plain response"

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("application/json", body),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	// A plain strings.Contains(header, "gzip") check would wrongly
	// treat this as accepting gzip; "q=0" explicitly declines it
	// (RFC 9110 §12.5.3).
	request.Header.Set("Accept-Encoding", "gzip;q=0, deflate")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf(
			"expected no Content-Encoding when gzip;q=0, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}

	if recorder.Body.String() != body {
		t.Errorf("expected plain body %q, got %q", body, recorder.Body.String())
	}
}

func TestCompress_WildcardAcceptEncodingMatchesGzip(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("application/json", "plain response"),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "*")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf(
			"expected gzip via wildcard Accept-Encoding, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}
}

func TestCompress_SkipsDisallowedContentType(t *testing.T) {
	t.Parallel()

	const body = "binary-ish data"

	handler := middleware.Compress(gzip.DefaultCompression)(
		writeWithContentType("image/png", body),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf(
			"expected no Content-Encoding for an unlisted type, got %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}

	if recorder.Body.String() != body {
		t.Errorf("expected plain body %q, got %q", body, recorder.Body.String())
	}
}

func TestCompress_CustomTypesReplaceRatherThanExtendDefaults(t *testing.T) {
	t.Parallel()

	// application/json is in the default list but not in this custom,
	// narrower list, so it must not be compressed here.
	handler := middleware.Compress(gzip.DefaultCompression, "text/plain")(
		writeWithContentType("application/json", "{}"),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "" {
		t.Errorf(
			"expected custom types to replace the default list, got Content-Encoding %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}
}

func TestCompress_WildcardTypeMatchesSubtypes(t *testing.T) {
	t.Parallel()

	handler := middleware.Compress(gzip.DefaultCompression, "text/*")(
		writeWithContentType("text/plain", "plain text body"),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf(
			"expected text/* to match text/plain, got Content-Encoding %q",
			recorder.Header().Get("Content-Encoding"),
		)
	}
}

func TestCompress_RemovesContentLengthWhenCompressing(t *testing.T) {
	t.Parallel()

	const body = "hello, compressed world"

	handler := middleware.Compress(gzip.DefaultCompression)(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = writer.Write([]byte(body))
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("Content-Length") != "" {
		t.Errorf(
			"expected Content-Length to be removed once compressing, got %q",
			recorder.Header().Get("Content-Length"),
		)
	}
}

func TestCompress_InvalidLevelPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Error("expected an invalid compression level to panic")
		}
	}()

	middleware.Compress(999)
}
