package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Casara/arnon/httpx/middleware"
)

func writeBody(body string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(body))
	})
}

func TestETag_FirstRequestGetsFullBodyAndETagHeader(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Body.String() != `{"id":"usr_123"}` {
		t.Errorf("unexpected body: %q", recorder.Body.String())
	}

	etag := recorder.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected an ETag header to be set")
	}

	if etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Errorf("expected a quoted strong ETag, got %q", etag)
	}
}

func TestETag_MatchingIfNoneMatchReturnsNotModifiedWithEmptyBody(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	etag := first.Header().Get("ETag")

	second := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	request.Header.Set("If-None-Match", etag)

	handler.ServeHTTP(second, request)

	if second.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, second.Code)
	}

	if second.Body.Len() != 0 {
		t.Errorf("expected an empty body on 304, got %q", second.Body.String())
	}

	if got := second.Header().Get("ETag"); got != etag {
		t.Errorf("expected the 304 to repeat ETag %q, got %q", etag, got)
	}
}

func TestETag_IfNoneMatchWildcardMatchesAnyRepresentation(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	request := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	request.Header.Set("If-None-Match", "*")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, recorder.Code)
	}
}

func TestETag_DifferentBodyProducesDifferentETagAndNoMatch(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	staleETag := first.Header().Get("ETag")

	changedHandler := middleware.ETag()(writeBody(`{"id":"usr_456"}`))

	request := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	request.Header.Set("If-None-Match", staleETag)

	recorder := httptest.NewRecorder()
	changedHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d for a changed body, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Body.String() != `{"id":"usr_456"}` {
		t.Errorf("unexpected body: %q", recorder.Body.String())
	}

	if got := recorder.Header().Get("ETag"); got == staleETag {
		t.Error("expected a different ETag for a different body")
	}
}

func TestETag_WeakComparisonIgnoresWPrefix(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	etag := first.Header().Get("ETag")

	request := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	request.Header.Set("If-None-Match", "W/"+etag)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotModified {
		t.Fatalf(
			"expected weak comparison to still match, got status %d",
			recorder.Code,
		)
	}
}

func TestETag_NonGetHeadMethodsPassThroughUnbuffered(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/users", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for a non-GET/HEAD request")
	}
}

func TestETag_RangeRequestPassesThroughUnbuffered(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	request := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	request.Header.Set("Range", "bytes=0-3")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for a Range request")
	}
}

// TestETag_DoesNotBreakRangeSupportOnAWrappedFileServer is the
// regression test for a real interaction confirmed empirically (see
// examples/cmd/staticfiles): wrapping a Range-capable handler in ETag
// used to make it fall back to a full 200 instead of the requested
// 206, because ETag only set its header after the handler returned,
// too late for http.ServeContent's own If-Range check to see it.
func TestETag_DoesNotBreakRangeSupportOnAWrappedFileServer(t *testing.T) {
	t.Parallel()

	content := strings.NewReader("0123456789")

	fileHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeContent(writer, request, "numbers.txt", time.Time{}, content)
	})

	handler := middleware.ETag()(fileHandler)

	request := httptest.NewRequest(http.MethodGet, "/numbers.txt", nil)
	request.Header.Set("Range", "bytes=0-3")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("expected status %d, got %d", http.StatusPartialContent, recorder.Code)
	}

	if recorder.Body.String() != "0123" {
		t.Errorf("expected partial body %q, got %q", "0123", recorder.Body.String())
	}

	if got := recorder.Header().Get("Content-Range"); got != "bytes 0-3/10" {
		t.Errorf("expected Content-Range %q, got %q", "bytes 0-3/10", got)
	}
}

func TestETag_NonSuccessResponsesAreNotGivenAnETag(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte(`{"title":"Not Found"}`))
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	if recorder.Header().Get("ETag") != "" {
		t.Error("expected no ETag header for a non-2xx response")
	}

	if recorder.Body.String() != `{"title":"Not Found"}` {
		t.Errorf(
			"expected the error body to pass through unchanged, got %q",
			recorder.Body.String(),
		)
	}
}

// TestETag_HeadRequestGetsCorrectContentLengthAndNoBody runs the
// middleware behind a real net/http.Server (not just a bare
// httptest.ResponseRecorder): the body-suppression-for-HEAD and
// Content-Length computation both happen in net/http.Server's own
// connection-level response writer, one layer further out than any
// middleware-level http.ResponseWriter wrapping (confirmed
// empirically while designing this middleware) - so this needs a real
// server round trip to actually exercise, not just a Recorder.
func TestETag_HeadRequestGetsCorrectContentLengthAndNoBody(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(writeBody(`{"id":"usr_123"}`))

	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.Head(server.URL + "/users/123")
	if err != nil {
		t.Fatalf("HEAD request failed: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	if response.Header.Get("ETag") == "" {
		t.Error("expected an ETag header on the HEAD response")
	}

	if got := response.Header.Get("Content-Length"); got != "16" {
		t.Errorf(`expected Content-Length "16" (len of {"id":"usr_123"}), got %q`, got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("expected an empty body for HEAD, got %q", body)
	}
}

// TestETag_WorksWithNonJSONContentTypes proves ETag is content-type
// agnostic: it works the same on a PDF/XML/binary response as on a
// JSON one, since arnon's typed httpx.Endpoint being JSON-only is a
// separate layer above this middleware, not something ETag itself
// assumes.
func TestETag_WorksWithNonJSONContentTypes(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/pdf")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("%PDF-1.4 fake report"))
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/report.pdf", nil))

	if first.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, first.Code)
	}

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected an ETag header for a non-JSON response")
	}

	request := httptest.NewRequest(http.MethodGet, "/report.pdf", nil)
	request.Header.Set("If-None-Match", etag)

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)

	if second.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, second.Code)
	}
}

func TestETag_RespectsHandlerProvidedETag(t *testing.T) {
	t.Parallel()

	handler := middleware.ETag()(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("ETag", `"custom-etag"`)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"id":"usr_123"}`))
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/123", nil))

	if got := recorder.Header().Get("ETag"); got != `"custom-etag"` {
		t.Errorf("expected the handler-provided ETag to be preserved, got %q", got)
	}
}
