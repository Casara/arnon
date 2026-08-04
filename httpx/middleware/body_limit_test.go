package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casara/arnon/httpx/middleware"
)

func TestMaxBodyBytes_RejectsKnownOversizedContentLengthImmediately(t *testing.T) {
	t.Parallel()

	handlerCalled := false

	handler := middleware.MaxBodyBytes(4)(http.HandlerFunc(func(
		_ http.ResponseWriter,
		_ *http.Request,
	) {
		handlerCalled = true
	}))

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/", strings.NewReader("way too long")),
	)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, recorder.Code)
	}

	if handlerCalled {
		t.Error("expected the handler not to run for an oversized body")
	}
}

func TestMaxBodyBytes_AllowsBodyWithinLimit(t *testing.T) {
	t.Parallel()

	const body = "ok"

	var readBody string

	handler := middleware.MaxBodyBytes(int64(len(body)))(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		data, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}

		readBody = string(data)
	}))

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if readBody != body {
		t.Errorf("expected handler to read %q, got %q", body, readBody)
	}
}

func TestMaxBodyBytes_RejectsUnknownLengthOversizedBodyWhileReading(t *testing.T) {
	t.Parallel()

	var readErr error

	handler := middleware.MaxBodyBytes(4)(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		_, readErr = io.ReadAll(request.Body)
	}))

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("way too long"))
	// Simulate an unknown length (e.g. chunked transfer encoding),
	// where MaxBodyBytes can't reject upfront and has to rely on
	// http.MaxBytesReader kicking in during the read instead.
	request.ContentLength = -1

	handler.ServeHTTP(httptest.NewRecorder(), request)

	var maxBytesErr *http.MaxBytesError

	if !errors.As(readErr, &maxBytesErr) {
		t.Fatalf("expected an *http.MaxBytesError, got %v", readErr)
	}
}
