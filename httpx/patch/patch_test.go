package patch_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Casara/arnon/httpx/patch"
	"github.com/Casara/arnon/httpx/routing"
	"github.com/Casara/arnon/problem"
)

// memoryResource is a tiny, thread-safe in-memory JSON document backing
// the GET/PUT test handlers below - just enough real state for From to
// fetch and replace.
type memoryResource struct {
	mu       sync.Mutex
	data     []byte
	existing bool

	responseHeader http.Header
}

func newMemoryResource(initial string) *memoryResource {
	return &memoryResource{
		data:           []byte(initial),
		existing:       true,
		responseHeader: make(http.Header),
	}
}

func (resource *memoryResource) getHandler() http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		resource.mu.Lock()
		defer resource.mu.Unlock()

		for name, values := range resource.responseHeader {
			for _, value := range values {
				writer.Header().Add(name, value)
			}
		}

		if !resource.existing {
			writer.Header().Set("Content-Type", problemContentType)
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte(`{"title":"Not Found","status":404}`))

			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(resource.data)
	})
}

// putHandler optionally records the request it received (headers,
// path value) into recorded, for tests that need to inspect what the
// internal PUT actually looked like.
func (resource *memoryResource) putHandler(recorded *http.Request) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if recorded != nil {
			*recorded = *request
		}

		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		resource.mu.Lock()
		resource.data = body
		resource.mu.Unlock()

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(body)
	})
}

const problemContentType = "application/problem+json; charset=utf-8"

func decodeJSON(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var value map[string]any

	err := json.Unmarshal(body, &value)
	if err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}

	return value
}

func TestFrom_MergePatchDefaultContentType(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	got := decodeJSON(t, recorder.Body.Bytes())

	if got["name"] != "Ada" {
		t.Errorf("expected name to be untouched, got %v", got["name"])
	}

	if got["age"] != float64(31) {
		t.Errorf("expected age to be updated to 31, got %v", got["age"])
	}
}

func TestFrom_MergePatchExplicitContentType(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("Content-Type", "application/merge-patch+json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestFrom_MergePatchNullDeletesField(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","nickname":"Countess"}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(
		http.MethodPatch,
		"/",
		bytes.NewBufferString(`{"nickname":null}`),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	got := decodeJSON(t, recorder.Body.Bytes())

	if _, ok := got["nickname"]; ok {
		t.Errorf("expected nickname to be deleted, got %v", got["nickname"])
	}

	if got["name"] != "Ada" {
		t.Errorf("expected name to be untouched, got %v", got["name"])
	}
}

func TestFrom_JSONPatchAddReplaceRemove(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","tags":["math"],"nickname":"Countess"}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	body := `[
		{"op":"replace","path":"/name","value":"Ada Lovelace"},
		{"op":"add","path":"/tags/-","value":"computing"},
		{"op":"remove","path":"/nickname"}
	]`

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json-patch+json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	got := decodeJSON(t, recorder.Body.Bytes())

	if got["name"] != "Ada Lovelace" {
		t.Errorf("expected name to be replaced, got %v", got["name"])
	}

	tags, ok := got["tags"].([]any)
	if !ok || len(tags) != 2 || tags[1] != "computing" {
		t.Errorf("expected tags to have computing appended, got %v", got["tags"])
	}

	if _, ok := got["nickname"]; ok {
		t.Errorf("expected nickname to be removed, got %v", got["nickname"])
	}
}

func TestFrom_NonExistentResourceForwardsGetResponseWithoutCallingPut(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{}`)
	resource.existing = false

	putCalled := false

	put := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		putCalled = true
	})

	handler := patch.From(resource.getHandler(), put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	if putCalled {
		t.Error("expected put to never be called when get reports the resource as missing")
	}

	if got := recorder.Header().Get("Content-Type"); got != problemContentType {
		t.Errorf("expected the get response's Content-Type to be forwarded, got %q", got)
	}
}

// TestFrom_GetStatusExactlyAtTheSuccessUpperBoundIsNotSuccess is the
// boundary case for isSuccess: 300 (http.StatusMultipleChoices) is
// the first status past the 2xx range, so a get that returns it must
// not be treated as success (put must not be called).
func TestFrom_GetStatusExactlyAtTheSuccessUpperBoundIsNotSuccess(t *testing.T) {
	t.Parallel()

	get := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusMultipleChoices)
	})

	putCalled := false

	put := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		putCalled = true
	})

	handler := patch.From(get, put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if putCalled {
		t.Error("expected put to never be called for a 300 get response")
	}

	if recorder.Code != http.StatusMultipleChoices {
		t.Errorf("expected status %d, got %d", http.StatusMultipleChoices, recorder.Code)
	}
}

// TestFrom_WriteBeforeWriteHeaderImpliesStatusOKAndIgnoresLaterCalls
// mirrors real net/http.ResponseWriter semantics (an explicit
// WriteHeader after bytes were already written is a no-op, the
// implicit status from the first Write already won) - get here writes
// its body first and only calls WriteHeader afterward, with a
// different status.
func TestFrom_WriteBeforeWriteHeaderImpliesStatusOKAndIgnoresLaterCalls(t *testing.T) {
	t.Parallel()

	get := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = writer.Write([]byte(`{"name":"Ada"}`))
		writer.WriteHeader(http.StatusNotFound)
	})

	var putCalled bool

	put := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		putCalled = true

		writer.WriteHeader(http.StatusOK)
	})

	handler := patch.From(get, put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if !putCalled {
		t.Fatal(
			"expected put to be called: the implicit status from the first Write is 200, not the later 404",
		)
	}
}

// TestFrom_GetHandlerWithNoExplicitWriteHeaderIsTreatedAs200 covers
// the same implicit-200 convention net/http itself uses (a handler
// that never calls WriteHeader still gets 200): get here writes
// nothing at all, relying on that default.
func TestFrom_GetHandlerWithNoExplicitWriteHeaderIsTreatedAs200(t *testing.T) {
	t.Parallel()

	get := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = writer.Write([]byte(`{"name":"Ada"}`))
	})

	var putCalled bool

	put := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		putCalled = true

		writer.WriteHeader(http.StatusOK)
	})

	handler := patch.From(get, put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if !putCalled {
		t.Fatal("expected put to be called: an implicit 200 from get should count as success")
	}

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

// TestFrom_ContentTypeWithMalformedParameterStillMatchesMainType
// proves a Content-Type with a broken parameter (a real client
// mistake, not just a missing header) still resolves by its main
// type instead of falling through to 415 - confirmed empirically that
// mime.ParseMediaType returns a usable type alongside its error for
// this shape of input (see the comment in format.go).
func TestFrom_ContentTypeWithMalformedParameterStillMatchesMainType(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("Content-Type", "application/merge-patch+json; =")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected the malformed parameter to be ignored and merge patch applied, got status %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestFrom_MalformedPatchBodyUsesDefaultOnApplyError(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`not json`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != problemContentType {
		t.Errorf("expected a Problem Details response, got Content-Type %q", got)
	}
}

func TestFrom_CustomOnApplyError(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{
		OnApplyError: func(err error) *problem.Problem {
			return problem.New(http.StatusTeapot, "custom", err.Error())
		},
	})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`not json`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTeapot {
		t.Errorf("expected custom status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}

func TestFrom_UnsupportedContentTypeReturns415(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)
	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`<name>Ada</name>`))
	request.Header.Set("Content-Type", "application/xml")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, recorder.Code)
	}
}

// TestFrom_PathValuePropagatesToGetAndPut proves the internal
// GET/PUT requests still resolve {id} correctly - Request.Clone
// explicitly preserves the internal fields PathValue reads from
// (confirmed against Go's stdlib source, fixed for exactly this
// reuse-across-calls scenario, issue 61410).
func TestFrom_PathValuePropagatesToGetAndPut(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)

	var gotGetID, gotPutID string

	get := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotGetID = request.PathValue("id")
		resource.getHandler().ServeHTTP(writer, request)
	})

	put := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPutID = request.PathValue("id")
		resource.putHandler(nil).ServeHTTP(writer, request)
	})

	router := routing.NewRouter()
	router.PATCH("/items/{id}", patch.From(get, put, patch.Config{}))

	request := httptest.NewRequest(
		http.MethodPatch,
		"/items/42",
		bytes.NewBufferString(`{"age":31}`),
	)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if gotGetID != "42" {
		t.Errorf(`expected the internal GET to see PathValue("id") == "42", got %q`, gotGetID)
	}

	if gotPutID != "42" {
		t.Errorf(`expected the internal PUT to see PathValue("id") == "42", got %q`, gotPutID)
	}
}

func TestFrom_HeadersPropagateToSyntheticRequests(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)

	var gotGetHeader, gotPutHeader string

	get := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotGetHeader = request.Header.Get("X-Trace-Id")
		resource.getHandler().ServeHTTP(writer, request)
	})

	put := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPutHeader = request.Header.Get("X-Trace-Id")
		resource.putHandler(nil).ServeHTTP(writer, request)
	})

	handler := patch.From(get, put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("X-Trace-Id", "trace-123")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if gotGetHeader != "trace-123" {
		t.Errorf("expected the internal GET to see X-Trace-Id, got %q", gotGetHeader)
	}

	if gotPutHeader != "trace-123" {
		t.Errorf("expected the internal PUT to see X-Trace-Id, got %q", gotPutHeader)
	}
}

func TestFrom_ETagAndLastModifiedPropagateToIfMatchAndIfUnmodifiedSince(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada"}`)
	resource.responseHeader.Set("ETag", `"abc123"`)
	resource.responseHeader.Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")

	var putRequest http.Request

	handler := patch.From(
		resource.getHandler(),
		resource.putHandler(&putRequest),
		patch.Config{},
	)

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if got := putRequest.Header.Get("If-Match"); got != `"abc123"` {
		t.Errorf(`expected If-Match %q, got %q`, `"abc123"`, got)
	}

	if got := putRequest.Header.Get("If-Unmodified-Since"); got != "Wed, 21 Oct 2026 07:28:00 GMT" {
		t.Errorf(
			`expected If-Unmodified-Since %q, got %q`,
			"Wed, 21 Oct 2026 07:28:00 GMT",
			got,
		)
	}
}

// TestFrom_MatchingIfMatchOnPatchRequestSucceeds proves a client's own
// optimistic-concurrency If-Match (from an earlier GET of the same
// resource) is honored: it's checked against the internal GET's ETag,
// and since it matches here, the patch applies normally.
func TestFrom_MatchingIfMatchOnPatchRequestSucceeds(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	resource.responseHeader.Set("ETag", `"abc123"`)

	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("If-Match", `"abc123"`)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// TestFrom_StaleIfMatchOnPatchRequestReturns412WithoutCallingPut is the
// case that actually closes the gap this package's doc comment used to
// call "inert": a client's If-Match that no longer matches the
// resource's current ETag must reject the whole PATCH with 412, before
// ever calling put.
func TestFrom_StaleIfMatchOnPatchRequestReturns412WithoutCallingPut(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	resource.responseHeader.Set("ETag", `"abc123"`)

	putCalled := false

	put := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		putCalled = true
	})

	handler := patch.From(resource.getHandler(), put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("If-Match", `"stale"`)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPreconditionFailed {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusPreconditionFailed,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if putCalled {
		t.Error("expected put to never be called when the original request's If-Match is stale")
	}

	if got := recorder.Header().Get("Content-Type"); got != problemContentType {
		t.Errorf("expected a Problem Details response, got Content-Type %q", got)
	}
}

// TestFrom_NoIfMatchOnPatchRequestBehavesAsBefore is a regression guard:
// a PATCH request that carries no conditional header at all must
// behave exactly as it did before this package checked one.
func TestFrom_NoIfMatchOnPatchRequestBehavesAsBefore(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	resource.responseHeader.Set("ETag", `"abc123"`)

	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// TestFrom_IfUnmodifiedSinceNotAfterInternalGetsLastModifiedSucceeds
// proves parseLastModified correctly parses the internal GET's
// Last-Modified into a real time.Time, not just a non-empty string:
// an If-Unmodified-Since exactly at the resource's Last-Modified must
// satisfy the precondition.
func TestFrom_IfUnmodifiedSinceNotAfterInternalGetsLastModifiedSucceeds(t *testing.T) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	resource.responseHeader.Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")

	handler := patch.From(resource.getHandler(), resource.putHandler(nil), patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("If-Unmodified-Since", "Wed, 21 Oct 2026 07:28:00 GMT")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// TestFrom_IfUnmodifiedSinceBeforeInternalGetsLastModifiedReturns412WithoutCallingPut
// is the mirror case: an If-Unmodified-Since earlier than the internal
// GET's real (parsed) Last-Modified must reject the request.
func TestFrom_IfUnmodifiedSinceBeforeInternalGetsLastModifiedReturns412WithoutCallingPut(
	t *testing.T,
) {
	t.Parallel()

	resource := newMemoryResource(`{"name":"Ada","age":30}`)
	resource.responseHeader.Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")

	putCalled := false

	put := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		putCalled = true
	})

	handler := patch.From(resource.getHandler(), put, patch.Config{})

	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"age":31}`))
	request.Header.Set("If-Unmodified-Since", "Tue, 20 Oct 2026 00:00:00 GMT")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPreconditionFailed {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusPreconditionFailed,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if putCalled {
		t.Error("expected put to never be called when If-Unmodified-Since is not satisfied")
	}
}
