package binding_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Casara/arnon/httpx/binding"
	"github.com/Casara/arnon/problem"
)

type decodeRequest struct {
	ID     int    `path:"id"`
	Limit  int    `          query:"limit"`
	Active bool   `          query:"active"`
	Filter string `          query:"filter"`
	Token  string `                         header:"X-Token"`
	Name   string `                                          json:"name"`
}

func newDecodeRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost,
		"/items/42?limit=10&active=true&filter=x",
		strings.NewReader(body),
	)
	request.SetPathValue("id", "42")
	request.Header.Set("X-Token", "secret")
	request.Header.Set("Content-Type", "application/json")

	return request
}

func TestDecode_BindsPathQueryHeaderAndJSON(t *testing.T) {
	t.Parallel()

	request := newDecodeRequest(t, `{"name":"widget"}`)

	dto, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	want := decodeRequest{
		ID:     42,
		Limit:  10,
		Active: true,
		Filter: "x",
		Token:  "secret",
		Name:   "widget",
	}
	if dto != want {
		t.Errorf("expected %+v, got %+v", want, dto)
	}
}

func TestDecode_EmptyBodyIsNotAnError(t *testing.T) {
	t.Parallel()

	request := newDecodeRequest(t, "")

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 0 {
		t.Errorf("expected empty body to be treated as no body, got %+v", validationErrors)
	}
}

func TestDecode_InvalidPathIntegerProducesPathError(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(
		http.MethodPost,
		"/items/abc?limit=10&active=true",
		strings.NewReader(""),
	)
	request.SetPathValue("id", "abc")

	_, validationErrors := binding.Decode[decodeRequest](request)

	found := false

	for _, validationErr := range validationErrors {
		if validationErr.Source.In == problem.ValidationLocationPath &&
			validationErr.Code == problem.ValidationCodeInvalidType {
			found = true
		}
	}

	if !found {
		t.Errorf("expected a path invalid-type error, got %+v", validationErrors)
	}
}

func TestDecode_InvalidQueryBooleanProducesQueryError(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(
		http.MethodPost,
		"/items/1?limit=10&active=not-a-bool",
		strings.NewReader(""),
	)
	request.SetPathValue("id", "1")

	_, validationErrors := binding.Decode[decodeRequest](request)

	found := false

	for _, validationErr := range validationErrors {
		if validationErr.Source.In == problem.ValidationLocationQuery &&
			validationErr.Source.Field == "active" {
			found = true
		}
	}

	if !found {
		t.Errorf("expected a query invalid-type error for 'active', got %+v", validationErrors)
	}
}

func TestDecode_MalformedJSONProducesBodyError(t *testing.T) {
	t.Parallel()

	// An unquoted key is a genuine JSON syntax error, as opposed to a
	// truncated body (which decodes as io.ErrUnexpectedEOF, a
	// different failure mode entirely - see
	// TestDecode_TruncatedJSONProducesInvalidTypeError below).
	request := newDecodeRequest(t, `{name: "widget"}`)

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 1 ||
		validationErrors[0].Code != problem.ValidationCodeMalformedJSON {
		t.Errorf("expected a malformed JSON error, got %+v", validationErrors)
	}
}

func TestDecode_TruncatedJSONProducesInvalidTypeError(t *testing.T) {
	t.Parallel()

	// A body that ends mid-value decodes as io.ErrUnexpectedEOF, which
	// is neither a json.SyntaxError nor io.EOF, so it falls through to
	// bindJSON's generic error branch (invalid_type) rather than
	// malformed_json. This documents that existing behavior rather
	// than asserting it is ideal.
	request := newDecodeRequest(t, `{"name":`)

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 1 || validationErrors[0].Code != problem.ValidationCodeInvalidType {
		t.Errorf("expected an invalid-type fallback error, got %+v", validationErrors)
	}
}

func TestDecode_OversizedBodyProducesPayloadTooLargeError(t *testing.T) {
	t.Parallel()

	request := newDecodeRequest(t, `{"name":"a much longer value than allowed"}`)
	request.Body = http.MaxBytesReader(nil, request.Body, 4)

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 1 ||
		validationErrors[0].Code != problem.ValidationCodePayloadTooLarge {
		t.Errorf("expected a payload-too-large error, got %+v", validationErrors)
	}
}

func TestDecode_UnknownJSONFieldIsRejected(t *testing.T) {
	t.Parallel()

	request := newDecodeRequest(t, `{"name":"widget","unknown":true}`)

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 1 {
		t.Fatalf("expected 1 validation error for the unknown field, got %+v", validationErrors)
	}
}

func TestDecode_WrongJSONTypeProducesBodyError(t *testing.T) {
	t.Parallel()

	request := newDecodeRequest(t, `{"name": 123}`)

	_, validationErrors := binding.Decode[decodeRequest](request)

	if len(validationErrors) != 1 || validationErrors[0].Code != problem.ValidationCodeInvalidType {
		t.Errorf("expected an invalid-type body error, got %+v", validationErrors)
	}

	if validationErrors[0].Source.Field != "/name" {
		t.Errorf("expected error sourced from /name, got %q", validationErrors[0].Source.Field)
	}
}

type multiQueryRequest struct {
	Tags []string `query:"tag"`
}

func TestDecode_QuerySliceCollectsRepeatedKeys(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/?tag=a&tag=b", nil)

	dto, validationErrors := binding.Decode[multiQueryRequest](request)
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	want := []string{"a", "b"}
	if len(dto.Tags) != len(want) || dto.Tags[0] != want[0] || dto.Tags[1] != want[1] {
		t.Errorf("expected Tags %v, got %v", want, dto.Tags)
	}
}

func TestDecode_QuerySliceIsEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)

	dto, validationErrors := binding.Decode[multiQueryRequest](request)
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	if len(dto.Tags) != 0 {
		t.Errorf("expected no Tags, got %v", dto.Tags)
	}
}

type multiHeaderRequest struct {
	Tags []string `header:"X-Tags"`
}

func TestDecode_HeaderSliceCollectsRepeatedLines(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Add("X-Tags", "a")
	request.Header.Add("X-Tags", "b")

	dto, validationErrors := binding.Decode[multiHeaderRequest](request)
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	want := []string{"a", "b"}
	if len(dto.Tags) != len(want) || dto.Tags[0] != want[0] || dto.Tags[1] != want[1] {
		t.Errorf("expected Tags %v, got %v", want, dto.Tags)
	}
}

func TestDecode_HeaderSliceSplitsCommaJoinedSingleLine(t *testing.T) {
	t.Parallel()

	// RFC 9110 §5.3: a single line with comma-joined values is
	// equivalent to repeating the header once per value.
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Tags", "a, b , c")

	dto, validationErrors := binding.Decode[multiHeaderRequest](request)
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	want := []string{"a", "b", "c"}
	if len(dto.Tags) != len(want) {
		t.Fatalf("expected Tags %v, got %v", want, dto.Tags)
	}

	for i, tag := range want {
		if dto.Tags[i] != tag {
			t.Errorf("expected Tags %v, got %v", want, dto.Tags)
		}
	}
}

func TestDecode_HeaderSliceIsEmptyWhenHeaderAbsent(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)

	dto, validationErrors := binding.Decode[multiHeaderRequest](request)
	if len(validationErrors) != 0 {
		t.Fatalf("expected no validation errors, got %+v", validationErrors)
	}

	if len(dto.Tags) != 0 {
		t.Errorf("expected no Tags, got %v", dto.Tags)
	}
}
