# httpx/middleware

Loaded on demand, when files in this directory are read. The repo-wide rules
live in the root `AGENTS.md`.

## Chain assembly

* **The relative order of global middleware matters, and `BuildChain` is how
  that's guaranteed by code, not discipline.** `chain.go`:
  `BuildChain(ChainConfig{...})` always assembles `Recover` → `Timeout` →
  `StripSlashes`/`RedirectSlashes` → `RealIP` → `RequestID` → `SecureHeaders`
  → `RateLimit`/`Throttle` → `ETag` → `Compress` → `CORS` → `ServiceDesc` →
  `Logging`, in that relative order, actually tested in `chain_test.go`
  (observable behavior, not just docs). Why each position exists is in
  `docs/architecture/project-context.md`, "Middleware order" — don't duplicate
  it here. A chain hand-assembled with `router.Use(mw1, mw2, ...)` is still the
  caller's responsibility; there's no static validation for that
  (`routing.Middleware` is just `func(http.Handler) http.Handler`, with no
  identity of its own at runtime) — which is exactly why `BuildChain` exists.
  `AllowContentType`/`MaxBodyBytes`/`NoCache` are deliberately left out:
  they're group middleware, not global.
* **Custom middleware with an ordering requirement uses `ChainConfig.Extra`
  (`ChainAnchor` + `ExtraMiddleware`), not a second parallel API.** Every
  position has an `AnchorXxx` constant; `ExtraMiddleware{Middleware: ...,
  Before: AnchorY}` or `{..., After: AnchorY}` inserts there. `BuildChain`
  panics if an entry sets neither or both, or names an unknown anchor — a typo
  must never silently drop the middleware. Equivalent alternative: split the
  `BuildChain` call in two (`Router.Use` accumulates across calls).
* **Adding a new global middleware needs three edits, or it becomes
  unreachable:** a field in `ChainConfig`, **and** a `ChainAnchor` registered
  in `validChainAnchors`, **and** an `appendStage(...)` call in the right
  place. Miss one and it can't be reached via `BuildChain`/`Extra`, and the
  order doc in `project-context.md` goes stale.

## How a middleware takes its configuration

Three shapes, and which one to use is decided by the configuration itself, not
by taste. Adding a middleware means picking from these, not inventing a fourth:

* **A required value goes positionally.** `Timeout(5*time.Second)`,
  `MaxBodyBytes(1<<20)`, `ServiceDesc("/openapi.json")`, `Logging(logger)`,
  `AllowContentType("application/json")`. There is nothing to default, so a
  config struct would only add ceremony.
* **Several settings, at least one required, go in a `XxxConfig` struct.**
  `RateLimit(RateLimitConfig{RequestLimit: …, WindowLength: …})`, `Throttle`,
  `SecureHeaders`, `CORS`. A struct is what can say "these two are required and
  the rest have defaults" in one place — see `RateLimitConfig`'s doc comment,
  which is the model to copy.
* **Optional tuning only goes in variadic options.** `RealIP()` and
  `RealIP(WithTrustedProxies(…))`. The bare call keeps working, and a new knob
  later is source-compatible for everyone who never passed one. This is why
  `RealIP` gained options rather than a config struct: making every existing
  `RealIP()` call site change to `RealIP(RealIPConfig{})` would have been a
  break with nothing gained.

`ChainConfig` mirrors the same split: a plain `bool` for a middleware with
nothing to configure, a `*XxxConfig` pointer for one that has settings, and
`RealIPOptions` alongside the `RealIP` bool for the one that takes options.

## Individual middleware

* **`RateLimit` uses a sliding-window counter (2 windows), not a per-key
  limiter with no eviction.** Adapted from `go-chi/httprate`, implemented from
  scratch in `rate_limit.go` with no external dependency (just `sync`/`time`).
  Old windows are discarded in bulk as time advances, so inactive keys are
  removed within at most two windows — **don't reintroduce the
  `golang.org/x/time/rate` + `sync.Map`-with-no-eviction version that briefly
  existed here; it had unbounded memory growth.**
* **`RateLimit` separates the algorithm from storage via the `LimitCounter`
  interface**, deliberately mirroring `go-chi/httprate`'s own interface so a
  backend written for httprate (e.g. `go-chi/httprate-redis`) ports over with
  trivial changes, and vice versa. A nil `RateLimitConfig.Counter` uses
  `NewLocalLimitCounter` (exported; correct only for a single instance).
  External backends belong in separate Go modules, never as a dependency of
  arnon itself — that's why only the interface and docs live here. A `Counter`
  error becomes a `problem.Problem` via `RateLimitConfig.OnCounterError`
  (default 503). Don't add a second storage mechanism parallel to
  `LimitCounter`.
* **`ETag` buffers the entire response before deciding 200 or 304.** Unlike
  `Compress` (which streams via `gzip.Writer`), hashing needs the complete
  body first — the same "buffer everything, decide at the end" pattern
  `Timeout` uses. Only acts on `GET`/`HEAD`, only on 2xx. Body suppression for
  `HEAD` and `Content-Length` happen in `net/http.Server`'s connection layer,
  *below* any middleware `ResponseWriter`, so `ETag` doesn't need to
  special-case `HEAD` — the buffer sees the full body either way (confirmed
  empirically). With `Compress`, install `ETag` first (more outer) so it hashes
  the already-compressed bytes, consistent with the `Vary: Accept-Encoding`
  that `Compress` sets.
* **`ETag` and `Compress` both step aside for any request carrying a `Range`
  header**, running `next` unwrapped. This is what makes either safe in front
  of a Range-capable handler (`http.FileServer`, `http.ServeContent`).
  Without it, `ETag` breaks `If-Range` and yields a validator that isn't stable
  across full vs. partial requests; `Compress` leaves `Content-Range`
  describing pre-compression byte positions while shipping a compressed body of
  a different length. A Range GET gets neither; a plain GET to the same route
  still gets both. See the doc comments on `ETag`/`Compress` for the mechanism.
* **`CORS` only intercepts `OPTIONS` when it's a genuine preflight** —
  `request.Method == http.MethodOptions && request.Header.Get(
  "Access-Control-Request-Method") != ""`, the exact definition in Fetch §4.1.
  An `OPTIONS` without that header falls through to the `mux`, which returns a
  real `405`+`Allow` or reaches an explicit user `OPTIONS` handler. Don't go
  back to intercepting every `OPTIONS` — that masked `ServeMux`'s real `Allow`
  and made `Router.OPTIONS(...)` unreachable.
* **`RealIP` checks `Forwarded` (RFC 7239) before
  `X-Forwarded-For`/`X-Real-IP`/`RemoteAddr`.** `parseForwardedFor` uses only
  the first hop (same "leftmost = original client" logic as `X-Forwarded-For`),
  treats `for=unknown` as no information (falls through), and keeps an
  obfuscated identifier (`for=_something`) as-is, since it still works as a
  stable rate-limit key even though it isn't an IP.
* **`Compress` parses `Accept-Encoding` for real, not `strings.Contains`.** It
  has to honor the `q` parameter (RFC 9110 §12.5.1) — an explicit `q=0` means
  refusal, which the old substring check couldn't see. Any new
  content-negotiation check follows this pattern (parse `;q=`, pick the most
  specific match), never a substring test. `httpx/accept.go` does the same for
  `Accept`.

## Interaction with observability

* **`Logging` only correlates `trace_id`/`span_id` if registered *after*
  `routing.WithInstrumentation`.** `router.register` applies
  `instrumentHandler` (which creates the span) on top of group/route
  middleware, but only *inside* the mux's dispatch — a **global** middleware
  (`router.Use`) runs before that dispatch. `Logging` builds its attributes
  (including `observability.TraceID`/`SpanID`) before calling `next`, so as a
  global it sees a context with no span yet. That's why
  `examples/cmd/observability` registers `Logging` via `api.Use(...)` (group),
  not `router.Use(...)`.
