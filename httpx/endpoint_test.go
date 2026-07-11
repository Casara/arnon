package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Casara/arnon/httpx"
	"github.com/Casara/arnon/problem"
)

type greetRequest struct {
	Name string `json:"name" validate:"required"`
}

type greetResponse struct {
	Greeting string `json:"greeting"`
}

func newJSONRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/greet", strings.NewReader(body))
}

func TestEndpoint_SuccessPathReturnsDefaultStatus(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, req greetRequest) (greetResponse, error) {
			return greetResponse{Greeting: "hello " + req.Name}, nil
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":"world"}`))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response greetResponse

	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Greeting != "hello world" {
		t.Errorf("unexpected greeting: %q", response.Greeting)
	}
}

func TestEndpoint_CustomSuccessStatus(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, req greetRequest) (greetResponse, error) {
			return greetResponse{Greeting: req.Name}, nil
		},
		httpx.EndpointConfig{SuccessStatus: http.StatusCreated},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":"world"}`))

	if recorder.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestEndpoint_BindingErrorReturnsBadRequestProblem(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, req greetRequest) (greetResponse, error) {
			return greetResponse{Greeting: req.Name}, nil
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":`))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	assertProblemContentType(t, recorder)
}

func TestEndpoint_ValidationErrorReturnsBadRequestProblem(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, req greetRequest) (greetResponse, error) {
			return greetResponse{Greeting: req.Name}, nil
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{}`))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var decoded problem.Problem

	err := json.Unmarshal(recorder.Body.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("failed to decode problem: %v", err)
	}

	if len(decoded.Errors) != 1 || decoded.Errors[0].Source.Field != "/name" {
		t.Errorf("expected a single validation error for /name, got %+v", decoded.Errors)
	}
}

func TestEndpoint_HandlerErrorIsMappedByDefaultProblemMapper(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, _ greetRequest) (greetResponse, error) {
			return greetResponse{}, problem.NewConflict("already exists")
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":"world"}`))

	if recorder.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
}

func TestEndpoint_HandlerErrorIsMappedByCustomProblemMapper(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("user not found")

	handler := httpx.Endpoint(
		func(_ context.Context, _ greetRequest) (greetResponse, error) {
			return greetResponse{}, sentinelErr
		},
		httpx.EndpointConfig{
			ProblemMapper: httpx.ProblemMapperFunc(func(err error) *problem.Problem {
				if errors.Is(err, sentinelErr) {
					return problem.NewNotFound("user not found")
				}

				return problem.NewInternal("")
			}),
		},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":"world"}`))

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

// unencodableResponse cannot be marshaled by encoding/json (channels
// are unsupported), used to exercise the serialization-failure path.
type unencodableResponse struct {
	Ch chan int `json:"ch"`
}

func TestEndpoint_SerializationFailureReturnsCleanProblemResponse(t *testing.T) {
	t.Parallel()

	handler := httpx.Endpoint(
		func(_ context.Context, _ greetRequest) (unencodableResponse, error) {
			return unencodableResponse{Ch: make(chan int)}, nil
		},
		httpx.EndpointConfig{},
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, newJSONRequest(`{"name":"world"}`))

	// Regression test: WriteJSON used to write the status header before
	// encoding the body, so an encode failure here would previously
	// leave the recorder with a 200 status and a corrupted/appended
	// body instead of a clean 500 problem response.
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var decoded problem.Problem

	err := json.Unmarshal(recorder.Body.Bytes(), &decoded)
	if err != nil {
		t.Fatalf(
			"expected a single well-formed problem body, got %q: %v",
			recorder.Body.String(),
			err,
		)
	}

	if decoded.Status != http.StatusInternalServerError {
		t.Errorf(
			"expected problem status %d, got %d",
			http.StatusInternalServerError,
			decoded.Status,
		)
	}
}

func assertProblemContentType(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	const problemContentType = "application/problem+json; charset=utf-8"

	if got := recorder.Header().Get("Content-Type"); got != problemContentType {
		t.Errorf("expected Content-Type %q, got %q", problemContentType, got)
	}
}
