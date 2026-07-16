package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/observability"
)

func TestRecover_RecoversPanicAndRespondsWithSafeProblem(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	handler := middleware.Recover()(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		panic("sensitive stack trace details")
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(
		observability.WithLogger(request.Context(), logger),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	if got := recorder.Header().
		Get("Content-Type"); got != "application/problem+json; charset=utf-8" {
		t.Errorf("expected a Problem Details content type, got %q", got)
	}

	body := recorder.Body.String()

	if strings.Contains(body, "sensitive stack trace details") {
		t.Errorf("expected the raw panic value not to leak into the response body, got %q", body)
	}

	if !strings.Contains(body, "An unexpected error occurred") {
		t.Errorf("expected the safe default detail in the response body, got %q", body)
	}

	if !strings.Contains(logBuffer.String(), "sensitive stack trace details") {
		t.Errorf(
			"expected the panic value to be logged server-side, got %q",
			logBuffer.String(),
		)
	}
}

func TestRecover_PassesThroughWhenNoPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.Recover()(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestRecover_UsesDefaultLoggerWhenNoneInContext(t *testing.T) {
	t.Parallel()

	handler := middleware.Recover()(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		panic("boom")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}
