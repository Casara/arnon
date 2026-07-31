package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/problem"
)

const (
	ipv6PrefixBits = 64
	ipv6TotalBits  = 128
)

// LimitCounter is the storage behind RateLimit's sliding window: the algorithm
// lives in this package, the counts live wherever you put them. A nil
// RateLimitConfig.Counter uses NewLocalLimitCounter, which is in-memory and
// therefore correct only for a single instance.
//
// # It is the same interface as go-chi/httprate's
//
// This is deliberate, method for method. Go interfaces are structural, so an
// existing httprate backend already satisfies this one with no adapter:
//
//	counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
//		Host: "localhost",
//		Port: 6379,
//	})
//	if err != nil {
//		return err
//	}
//
//	router.Use(middleware.RateLimit(middleware.RateLimitConfig{
//		RequestLimit: 100,
//		WindowLength: time.Minute,
//		Counter:      counter,
//	}))
//
// That is why this package ships no Redis, Valkey or Memcached backend of its
// own: github.com/go-chi/httprate-redis already covers Redis and anything
// speaking its protocol, and duplicating it here would add a dependency, a
// release to coordinate and a CVE surface for no gain. The compatibility is
// pinned by a compile-time assertion in limitcounter_compat_test.go, so it
// cannot drift silently.
//
// Writing your own works the same way in reverse: implement these four methods
// and the result serves both projects.
type LimitCounter interface {
	// Config is called once, when the counter is installed, with the
	// configured request limit and window length.
	Config(requestLimit int, windowLength time.Duration)

	// Increment records one request for key in currentWindow.
	Increment(key string, currentWindow time.Time) error

	// IncrementBy records amount requests for key in currentWindow.
	IncrementBy(key string, currentWindow time.Time, amount int) error

	// Get returns the request counts for key in currentWindow and in
	// previousWindow.
	Get(
		key string,
		currentWindow time.Time,
		previousWindow time.Time,
	) (currentCount, previousCount int, err error)
}

// RateLimitConfig configures RateLimit. RequestLimit and WindowLength
// are the only required fields; everything else has a working default.
type RateLimitConfig struct {
	// RequestLimit is the maximum number of requests allowed per
	// client key within WindowLength.
	RequestLimit int

	// WindowLength is the duration of each rate-limit window, e.g. one
	// minute allows RequestLimit requests per client per minute. Must
	// be positive.
	WindowLength time.Duration

	// KeyFunc extracts the rate-limiting key (e.g. client IP) from a
	// request. Defaults to using RealIPFromContext (falling back to
	// request.RemoteAddr), canonicalized via CanonicalizeIP.
	//
	// That default is only as trustworthy as RealIP is in your deployment:
	// RealIP reads client-controlled headers and has no trusted-proxy list,
	// so on a directly-exposed server a caller can forge a new key per
	// request and slip past the limit entirely. See RealIP's documentation.
	// Without RealIP in the chain the fallback is request.RemoteAddr, which
	// cannot be forged over TCP.
	KeyFunc func(*http.Request) string

	// Counter is the storage backend. Defaults to an in-memory
	// NewLocalLimitCounter(WindowLength) when nil. Provide a shared
	// LimitCounter implementation (Redis, etc.) for multi-instance
	// deployments, where each instance must enforce the same limit.
	Counter LimitCounter

	// OnCounterError builds the problem response written to the
	// client when Counter.Get or Counter.IncrementBy returns an
	// error. Defaults to a generic 503 Service Unavailable problem
	// that does not leak err's message to the client.
	//
	// A LimitCounter that can transparently fall back to a local
	// counter on its own (as github.com/go-chi/httprate-redis does)
	// should rarely trigger this.
	OnCounterError func(err error) *problem.Problem
}

// RateLimit limits how many requests within RateLimitConfig.WindowLength
// a client (as identified by KeyFunc) may make.
//
// This limits rate, not concurrency: it does not cap how many requests
// from different clients may be in flight at once. For that, see
// Throttle.
//
// It uses a sliding-window-counter approximation, algorithm adapted
// from go-chi/httprate: two fixed windows (current and previous) are
// tracked per key, and a request's effective rate blends the previous
// window's count - weighted by how much it still overlaps the current
// sliding window - with the current window's own count. This avoids
// both the burst-at-boundary problem of a naive fixed window and the
// unbounded per-request memory of a true sliding log.
//
// The check-then-increment sequence for a request is serialized by a
// single mutex, same as upstream httprate: correct and simple, at the
// cost of serializing all keys through one lock rather than sharding
// by key. Revisit if this becomes a measured bottleneck.
func RateLimit(
	config RateLimitConfig,
) routing.Middleware {
	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = defaultRateLimitKey
	}

	counter := config.Counter
	if counter == nil {
		counter = NewLocalLimitCounter(config.WindowLength)
	}

	counter.Config(config.RequestLimit, config.WindowLength)

	onCounterError := config.OnCounterError
	if onCounterError == nil {
		onCounterError = defaultOnCounterError
	}

	var mu sync.Mutex

	return func(
		next http.Handler,
	) http.Handler {
		return routing.Wrap(next, http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			key := keyFunc(request)

			mu.Lock()
			result, err := checkRateLimit(
				counter,
				key,
				config.RequestLimit,
				config.WindowLength,
			)
			mu.Unlock()

			if err != nil {
				httpx.WriteProblem(
					writer,
					request,
					onCounterError(err),
				)

				return
			}

			header := writer.Header()

			header.Set(
				"X-RateLimit-Limit",
				strconv.Itoa(config.RequestLimit),
			)

			header.Set(
				"X-RateLimit-Reset",
				strconv.FormatInt(result.resetAt.Unix(), 10),
			)

			if !result.allowed {
				header.Set("X-RateLimit-Remaining", "0")

				header.Set(
					"Retry-After",
					strconv.Itoa(int(config.WindowLength.Seconds())),
				)

				httpx.WriteProblem(
					writer,
					request,
					problem.NewTooManyRequests(
						"rate limit exceeded, try again later",
					),
				)

				return
			}

			header.Set(
				"X-RateLimit-Remaining",
				strconv.Itoa(result.remaining),
			)

			next.ServeHTTP(
				writer,
				request,
			)
		}))
	}
}

func defaultOnCounterError(
	_ error,
) *problem.Problem {
	return problem.NewServiceUnavailable(
		"rate limit backend is temporarily unavailable",
	)
}

func defaultRateLimitKey(
	request *http.Request,
) string {
	if realIP := RealIPFromContext(
		request.Context(),
	); realIP != "" {
		return CanonicalizeIP(realIP)
	}

	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}

	return CanonicalizeIP(host)
}

// CanonicalizeIP normalizes a client IP string for use as a
// rate-limit key:
//
//   - IPv4 addresses are returned unchanged.
//   - IPv6 addresses are reduced to their /64 prefix. An IPv6 client
//     typically controls a whole /64 (2^64 addresses via SLAAC), so
//     keying on the full address would let it rotate within its own
//     /64 to win a fresh bucket per request and bypass the limit.
//   - Any other string, including "", is returned unchanged.
//
// Ported from go-chi/httprate.
func CanonicalizeIP(
	ip string,
) string {
	isIPv6 := false

	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return ip
		case ':':
			isIPv6 = true
		}
	}

	if !isIPv6 {
		return ip
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	return parsed.Mask(net.CIDRMask(ipv6PrefixBits, ipv6TotalBits)).String()
}

type rateLimitResult struct {
	allowed   bool
	remaining int
	resetAt   time.Time
}

// checkRateLimit computes whether key may make one more request within
// limit over windowLength, against counter, and records the request
// if so. It contains the sliding-window rate math; LimitCounter
// implementations only need to store and report per-window counts.
func checkRateLimit(
	counter LimitCounter,
	key string,
	limit int,
	windowLength time.Duration,
) (rateLimitResult, error) {
	now := time.Now().UTC()
	currentWindow := now.Truncate(windowLength)
	previousWindow := currentWindow.Add(-windowLength)
	resetAt := currentWindow.Add(windowLength)

	currentCount, previousCount, err := counter.Get(
		key,
		currentWindow,
		previousWindow,
	)
	if err != nil {
		return rateLimitResult{}, fmt.Errorf("get rate limit counts: %w", err)
	}

	elapsed := now.Sub(currentWindow)
	weight := float64(windowLength-elapsed) / float64(windowLength)
	rate := float64(previousCount)*weight + float64(currentCount)

	remaining := max(limit-int(math.Round(rate)), 0)

	if rate+1 > float64(limit) {
		return rateLimitResult{
			allowed:   false,
			remaining: remaining,
			resetAt:   resetAt,
		}, nil
	}

	err = counter.IncrementBy(key, currentWindow, 1)
	if err != nil {
		return rateLimitResult{}, fmt.Errorf("increment rate limit count: %w", err)
	}

	remaining = max(remaining-1, 0)

	return rateLimitResult{
		allowed:   true,
		remaining: remaining,
		resetAt:   resetAt,
	}, nil
}

// localLimitCounter is the default in-memory LimitCounter
// implementation, used when RateLimitConfig.Counter is nil. It tracks
// two adjacent fixed windows (current and previous) per key, evicting
// the older window in bulk whenever time moves into a new window.
// This bounds memory to the set of keys active within the last two
// windows, rather than every key ever seen.
type localLimitCounter struct {
	mu sync.Mutex

	windowLength   time.Duration
	trackedWindow  time.Time
	currentCounts  map[string]int
	previousCounts map[string]int
}

// NewLocalLimitCounter creates the in-memory LimitCounter used by
// RateLimit when no Counter is configured.
//
// It is exported so external LimitCounter implementations (e.g. a
// Redis-backed one) can use it as a fallback when their backing store
// is unavailable, the same way github.com/go-chi/httprate-redis falls
// back to httprate.NewLocalLimitCounter.
func NewLocalLimitCounter(
	windowLength time.Duration,
) LimitCounter {
	return &localLimitCounter{
		windowLength:   windowLength,
		trackedWindow:  time.Now().UTC().Truncate(windowLength),
		currentCounts:  make(map[string]int),
		previousCounts: make(map[string]int),
	}
}

func (counter *localLimitCounter) Config(
	_ int,
	windowLength time.Duration,
) {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	counter.windowLength = windowLength
}

func (counter *localLimitCounter) Increment(
	key string,
	currentWindow time.Time,
) error {
	return counter.IncrementBy(key, currentWindow, 1)
}

func (counter *localLimitCounter) IncrementBy(
	key string,
	currentWindow time.Time,
	amount int,
) error {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	counter.evict(currentWindow)

	counter.currentCounts[key] += amount

	return nil
}

func (counter *localLimitCounter) Get(
	key string,
	currentWindow time.Time,
	_ time.Time,
) (int, int, error) {
	counter.mu.Lock()
	defer counter.mu.Unlock()

	counter.evict(currentWindow)

	return counter.currentCounts[key], counter.previousCounts[key], nil
}

// evict rolls the tracked windows forward when window is newer than
// what's tracked, dropping the oldest window instead of letting
// either map grow without bound.
func (counter *localLimitCounter) evict(
	window time.Time,
) {
	if counter.trackedWindow.Equal(window) {
		return
	}

	if counter.trackedWindow.Equal(window.Add(-counter.windowLength)) {
		clear(counter.previousCounts)

		counter.currentCounts, counter.previousCounts = counter.previousCounts, counter.currentCounts
		counter.trackedWindow = window

		return
	}

	clear(counter.currentCounts)
	clear(counter.previousCounts)

	counter.trackedWindow = window
}
