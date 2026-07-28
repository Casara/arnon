package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Casara/arnon/httpx/middleware"
	"github.com/Casara/arnon/problem"
)

func TestRateLimit_AllowsUpToLimitThenRejectsWithinWindow(t *testing.T) {
	t.Parallel()

	// A one-hour window is long enough that the test can't cross a
	// window boundary, so the assertions don't depend on timing.
	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 2,
		WindowLength: time.Hour,
		KeyFunc: func(*http.Request) string {
			return "same-client"
		},
	})(newNoopHandler())

	for i := range 2 {
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: expected status %d, got %d", i, http.StatusOK, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"expected status %d after exhausting the limit, got %d",
			http.StatusTooManyRequests,
			recorder.Code,
		)
	}

	if got := recorder.Header().Get("Retry-After"); got == "" {
		t.Error("expected a Retry-After header on the 429 response")
	}
}

func TestRateLimit_SetsRateLimitHeaders(t *testing.T) {
	t.Parallel()

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 5,
		WindowLength: time.Hour,
		KeyFunc: func(*http.Request) string {
			return "same-client"
		},
	})(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := recorder.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Errorf("expected X-RateLimit-Limit %q, got %q", "5", got)
	}

	if got := recorder.Header().Get("X-RateLimit-Remaining"); got != "4" {
		t.Errorf("expected X-RateLimit-Remaining %q, got %q", "4", got)
	}

	if got := recorder.Header().Get("X-RateLimit-Reset"); got == "" {
		t.Error("expected X-RateLimit-Reset to be set")
	}
}

func TestRateLimit_TracksDistinctKeysIndependently(t *testing.T) {
	t.Parallel()

	key := "client-a"

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: time.Hour,
		KeyFunc: func(*http.Request) string {
			return key
		},
	})(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d for client-a's first request, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected client-a's second request to be limited, got %d", recorder.Code)
	}

	key = "client-b"

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"expected client-b's first request to be allowed independently, got %d",
			recorder.Code,
		)
	}
}

func TestRateLimit_AllowsAgainAfterWindowFullyRolls(t *testing.T) {
	t.Parallel()

	const windowLength = 30 * time.Millisecond

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: windowLength,
		KeyFunc: func(*http.Request) string {
			return "same-client"
		},
	})(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected first request status %d, got %d", http.StatusOK, recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request (same window) to be limited, got %d", recorder.Code)
	}

	// Comfortably more than two window lengths, so the old window's
	// contribution to the sliding-window weight has fully decayed to
	// zero (the "far future" full-clear path in localLimitCounter.evict),
	// regardless of scheduling jitter.
	time.Sleep(windowLength * 4)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("expected a request in a fresh window to be allowed again, got %d", recorder.Code)
	}
}

func TestRateLimit_DefaultKeyFuncUsesRealIPThenRemoteAddr(t *testing.T) {
	t.Parallel()

	handler := middleware.RealIP()(
		middleware.RateLimit(middleware.RateLimitConfig{
			RequestLimit: 1,
			WindowLength: time.Hour,
		})(newNoopHandler()),
	)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Real-IP", "203.0.113.9")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected first request status %d, got %d", http.StatusOK, recorder.Code)
	}

	// Same client IP, fresh request: should hit the same bucket and be
	// limited.
	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Real-IP", "203.0.113.9")

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf("expected second request from the same IP to be limited, got %d", recorder.Code)
	}
}

// fakeLimitCounter is a minimal, non-concurrent-safe LimitCounter test
// double, proving RateLimitConfig.Counter is genuinely pluggable and
// not just wired to the local implementation.
type fakeLimitCounter struct {
	configured bool

	getErr         error
	incrementErr   error
	incrementCalls int
	counts         map[string]int
}

func (counter *fakeLimitCounter) Config(int, time.Duration) {
	counter.configured = true
}

func (counter *fakeLimitCounter) Increment(key string, currentWindow time.Time) error {
	return counter.IncrementBy(key, currentWindow, 1)
}

func (counter *fakeLimitCounter) IncrementBy(key string, _ time.Time, amount int) error {
	counter.incrementCalls++

	if counter.incrementErr != nil {
		return counter.incrementErr
	}

	if counter.counts == nil {
		counter.counts = map[string]int{}
	}

	counter.counts[key] += amount

	return nil
}

func (counter *fakeLimitCounter) Get(key string, _, _ time.Time) (int, int, error) {
	if counter.getErr != nil {
		return 0, 0, counter.getErr
	}

	return counter.counts[key], 0, nil
}

func TestRateLimit_UsesConfiguredCounter(t *testing.T) {
	t.Parallel()

	counter := &fakeLimitCounter{}

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: time.Hour,
		Counter:      counter,
		KeyFunc: func(*http.Request) string {
			return "same-client"
		},
	})(newNoopHandler())

	if !counter.configured {
		t.Fatal("expected RateLimit to call Counter.Config during setup")
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if counter.incrementCalls != 1 {
		t.Fatalf(
			"expected the custom counter to record 1 increment, got %d",
			counter.incrementCalls,
		)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf(
			"expected the custom counter's count to drive the limit, got status %d",
			recorder.Code,
		)
	}
}

func TestRateLimit_CounterGetErrorUsesOnCounterError(t *testing.T) {
	t.Parallel()

	counter := &fakeLimitCounter{getErr: errors.New("backend down")}

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: time.Hour,
		Counter:      counter,
	})(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("expected default status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
}

func TestRateLimit_CustomOnCounterError(t *testing.T) {
	t.Parallel()

	counter := &fakeLimitCounter{getErr: errors.New("backend down")}

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: time.Hour,
		Counter:      counter,
		OnCounterError: func(err error) *problem.Problem {
			return problem.New(http.StatusTeapot, "custom", err.Error())
		},
	})(newNoopHandler())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTeapot {
		t.Errorf("expected custom status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}

// previousWindowFakeCounter is a concurrent-safe LimitCounter test
// double that reports a fixed previousCount on every Get call,
// regardless of the window arguments it's called with - unlike
// fakeLimitCounter above and the built-in localLimitCounter, both of
// which effectively ignore the previousWindow argument (the local
// counter derives "previous" from its own eviction bookkeeping keyed
// only on currentWindow). A real external LimitCounter (Redis, etc.)
// is expected to honor previousWindow directly, so this is what lets
// a test control - and therefore verify - the sliding-window blend in
// checkRateLimit that the built-in counter alone never exercises.
type previousWindowFakeCounter struct {
	mu            sync.Mutex
	previousCount int
	currentCount  int

	// lastCurrentWindow/lastPreviousWindow record the exact arguments
	// checkRateLimit passed to the most recent Get call, so a test can
	// verify previousWindow was actually derived from currentWindow
	// (currentWindow.Add(-windowLength)), not just that some value was
	// passed.
	lastCurrentWindow  time.Time
	lastPreviousWindow time.Time
}

func (counter *previousWindowFakeCounter) Config(int, time.Duration) {}

func (counter *previousWindowFakeCounter) Increment(key string, currentWindow time.Time) error {
	return counter.IncrementBy(key, currentWindow, 1)
}

func (counter *previousWindowFakeCounter) IncrementBy(_ string, _ time.Time, amount int) error {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	counter.currentCount += amount

	return nil
}

func (counter *previousWindowFakeCounter) Get(
	_ string,
	currentWindow, previousWindow time.Time,
) (int, int, error) {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	counter.lastCurrentWindow = currentWindow
	counter.lastPreviousWindow = previousWindow

	return counter.currentCount, counter.previousCount, nil
}

// TestRateLimit_SlidingWindowWeightsPreviousWindowByElapsedFraction is
// the regression test for a gap mutation testing found: every other
// test in this file either uses an hour-long window (elapsed stays
// near zero throughout the test, so the blend weight stays near 1 and
// its own correctness is never actually observed) or a counter that
// reports previousCount as always 0 (multiplying the weight by zero,
// which hides a wrong weight just as effectively). This test uses
// previousWindowFakeCounter to hold previousCount fixed and nonzero,
// checks the request/deny decision at both ends of a real window -
// this is the "avoid burst at the boundary" behavior sliding-window
// weighting exists for in the first place (see RateLimit's doc
// comment) - and separately checks that previousWindow itself was
// derived from currentWindow (exactly one windowLength apart), which
// the request/deny assertions alone don't pin down since the fake
// counter's returned counts don't depend on the window values it
// receives.
func TestRateLimit_SlidingWindowWeightsPreviousWindowByElapsedFraction(t *testing.T) {
	t.Parallel()

	const (
		// Short enough to keep this test (and a mutation-testing run,
		// which reruns the whole suite once per mutant) fast, long
		// enough that the 5%/90%-of-window targets below stay well
		// clear of normal scheduling jitter.
		windowLength = 300 * time.Millisecond
		limit        = 10
	)

	counter := &previousWindowFakeCounter{previousCount: limit}

	handler := middleware.RateLimit(middleware.RateLimitConfig{
		RequestLimit: limit,
		WindowLength: windowLength,
		Counter:      counter,
		KeyFunc: func(*http.Request) string {
			return "same-client"
		},
	})(newNoopHandler())

	// Wait for a fresh window boundary, then a little further into it,
	// so elapsed is small and the blend weight is close to 1: the
	// previous window's full count of requests still weighs in almost
	// entirely, so a single additional request already meets the
	// limit - denied. A naive fixed window would instead treat this as
	// a brand new, empty bucket and allow it, which is exactly the
	// boundary-burst problem this algorithm exists to avoid.
	nextWindow := time.Now().Truncate(windowLength).Add(windowLength)
	time.Sleep(time.Until(nextWindow) + 15*time.Millisecond)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusTooManyRequests {
		t.Errorf(
			"expected a request just after a window boundary "+
				"(previous window's count still weighs in) to be limited, got status %d",
			recorder.Code,
		)
	}

	counter.mu.Lock()
	gotGap := counter.lastCurrentWindow.Sub(counter.lastPreviousWindow)
	counter.mu.Unlock()

	if gotGap != windowLength {
		t.Errorf(
			"expected previousWindow to be exactly one windowLength (%s) before currentWindow, got a %s gap",
			windowLength,
			gotGap,
		)
	}

	// Wait further into the same window, so elapsed is large and the
	// blend weight is close to 0: the same previous-window count has
	// now mostly decayed out of the blend, and the current window is
	// still empty (the denied request above was never counted), so a
	// request is allowed again.
	time.Sleep(time.Until(nextWindow.Add(270 * time.Millisecond)))

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"expected a request late in the window (previous window's count decayed) to be allowed, got status %d",
			recorder.Code,
		)
	}
}

func TestCanonicalizeIP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ip   string
		want string
	}{
		{"ipv4 unchanged", "203.0.113.9", "203.0.113.9"},
		{"ipv6 reduced to /64", "2001:db8::1", "2001:db8::"},
		{
			"different ipv6 host in same /64 collapses",
			"2001:db8::dead:beef",
			"2001:db8::",
		},
		{"empty string unchanged", "", ""},
		{"non-ip string unchanged", "not-an-ip", "not-an-ip"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := middleware.CanonicalizeIP(testCase.ip); got != testCase.want {
				t.Errorf("CanonicalizeIP(%q) = %q, want %q", testCase.ip, got, testCase.want)
			}
		})
	}
}

func TestNewLocalLimitCounter_TracksWindowedCounts(t *testing.T) {
	t.Parallel()

	const windowLength = time.Hour

	counter := middleware.NewLocalLimitCounter(windowLength)
	counter.Config(10, windowLength)

	now := time.Now().UTC().Truncate(windowLength)

	err := counter.IncrementBy("key", now, 3)
	if err != nil {
		t.Fatalf("IncrementBy failed: %v", err)
	}

	curr, prev, err := counter.Get("key", now, now.Add(-windowLength))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if curr != 3 || prev != 0 {
		t.Errorf("expected curr=3 prev=0, got curr=%d prev=%d", curr, prev)
	}

	otherCurr, _, err := counter.Get("other-key", now, now.Add(-windowLength))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if otherCurr != 0 {
		t.Errorf("expected an untouched key to read 0, got %d", otherCurr)
	}
}
