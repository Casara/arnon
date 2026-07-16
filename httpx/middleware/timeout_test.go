package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Casara/arnon/httpx/middleware"
)

func TestTimeout_PassesThroughWhenHandlerFinishesInTime(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(time.Second)(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.Header().Set("X-Custom", "value")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("ok"))
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if got := recorder.Header().Get("X-Custom"); got != "value" {
		t.Errorf("expected header X-Custom=value, got %q", got)
	}

	if got := recorder.Body.String(); got != "ok" {
		t.Errorf("expected body %q, got %q", "ok", got)
	}
}

func TestTimeout_RespondsWithProblemDetailsWhenHandlerIsTooSlow(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release)

	handler := middleware.Timeout(10 * time.Millisecond)(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		select {
		case <-release:
		case <-request.Context().Done():
		}

		// Simulates a handler that keeps writing after the deadline has
		// already fired - must not panic or race with the timeout
		// response (run with -race to verify).
		_, _ = writer.Write([]byte("too late"))
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			recorder.Code,
		)
	}

	if got := recorder.Header().
		Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Errorf("expected a Problem Details content type, got %q", got)
	}

	var body map[string]any

	err := json.Unmarshal(recorder.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body["title"] != http.StatusText(http.StatusServiceUnavailable) {
		t.Errorf(
			"expected title %q, got %v",
			http.StatusText(http.StatusServiceUnavailable),
			body["title"],
		)
	}

	if body["detail"] != "the request took too long to process" {
		t.Errorf("unexpected detail: %v", body["detail"])
	}
}

func TestTimeout_PropagatesHandlerPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(time.Second)(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		panic("boom")
	}))

	defer func() {
		recovered := recover()
		if recovered != "boom" {
			t.Errorf("expected panic %q to propagate, got %v", "boom", recovered)
		}
	}()

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	t.Error("expected ServeHTTP to panic")
}
