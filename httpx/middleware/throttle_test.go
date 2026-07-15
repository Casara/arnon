package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Casara/arnon/httpx/middleware"
)

// blockingHandler signals on started when it begins executing, then
// blocks until release is closed before responding 200. This lets
// tests deterministically know a request is occupying a Throttle slot
// before making further assertions.
//
// started must be created with enough buffer to hold one value per
// request the test sends through the same handler instance: the
// signal send here is a plain (blocking) send, and only the first
// request's signal is ever read by these tests, so a too-small buffer
// would deadlock later invocations.
func blockingHandler(started chan<- struct{}, release <-chan struct{}) http.Handler {
	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		started <- struct{}{}

		<-release

		writer.WriteHeader(http.StatusOK)
	})
}

func TestThrottle_RejectsImmediatelyWhenLimitReachedWithNoBacklog(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 4)
	release := make(chan struct{})

	handler := middleware.Throttle(middleware.ThrottleConfig{
		Limit: 1,
	})(blockingHandler(started, release))

	firstDone := make(chan *httptest.ResponseRecorder, 1)

	go func() {
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		firstDone <- recorder
	}()

	<-started // first request now holds the only slot

	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, secondRecorder.Code)
	}

	close(release)

	first := <-firstDone
	if first.Code != http.StatusOK {
		t.Errorf("expected first request to complete with %d, got %d", http.StatusOK, first.Code)
	}
}

func TestThrottle_QueuesInBacklogUntilSlotFrees(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 4)
	release := make(chan struct{})

	handler := middleware.Throttle(middleware.ThrottleConfig{
		Limit:          1,
		BacklogLimit:   1,
		BacklogTimeout: time.Second,
	})(blockingHandler(started, release))

	firstDone := make(chan int, 1)

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		firstDone <- recorder.Code
	}()

	<-started // first request now holds the only slot

	secondDone := make(chan int, 1)

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		secondDone <- recorder.Code
	}()

	// Give the second request time to reach the backlog wait before we
	// free the slot; there is no cheap deterministic signal for
	// "now waiting in the backlog" without instrumenting the
	// middleware itself.
	time.Sleep(50 * time.Millisecond)

	close(release)

	if code := <-firstDone; code != http.StatusOK {
		t.Errorf("expected first request status %d, got %d", http.StatusOK, code)
	}

	select {
	case code := <-secondDone:
		if code != http.StatusOK {
			t.Errorf("expected queued request status %d, got %d", http.StatusOK, code)
		}
	case <-time.After(time.Second):
		t.Fatal("expected the queued request to complete after the slot freed")
	}
}

func TestThrottle_RejectsWhenBacklogIsFull(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 4)
	release := make(chan struct{})

	handler := middleware.Throttle(middleware.ThrottleConfig{
		Limit:          1,
		BacklogLimit:   1,
		BacklogTimeout: time.Second,
	})(blockingHandler(started, release))

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	<-started // first request holds the only slot

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	// Give the second request time to occupy the single backlog slot.
	time.Sleep(50 * time.Millisecond)

	thirdRecorder := httptest.NewRecorder()
	handler.ServeHTTP(thirdRecorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if thirdRecorder.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, thirdRecorder.Code)
	}

	close(release)
}

func TestThrottle_BacklogTimeoutRejectsQueuedRequest(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 4)

	release := make(chan struct{})
	defer close(release)

	handler := middleware.Throttle(middleware.ThrottleConfig{
		Limit:          1,
		BacklogLimit:   1,
		BacklogTimeout: 20 * time.Millisecond,
	})(blockingHandler(started, release))

	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	<-started // first request holds the only slot, and stays blocked

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf(
			"expected status %d after backlog timeout, got %d",
			http.StatusTooManyRequests,
			recorder.Code,
		)
	}
}
