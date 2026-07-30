# httpx

Loaded on demand, when files in this directory are read. The repo-wide rules
live in the root `AGENTS.md`. Subdirectories carry their own file:
`httpx/middleware`, `httpx/patch`, `httpx/precondition`.

## Endpoint

* **`Endpoint()` always calls `EndpointConfig.WithDefaults()`.** Don't
  reintroduce the `if config.Validator != nil` / `if config.ProblemMapper !=
  nil` checks that used to exist — after `WithDefaults()`, those fields are
  never nil.
* **OpenAPI registration is opt-in per endpoint.** A route only appears in the
  generated document if `EndpointConfig.OpenAPI` is set, even to an empty
  `&openapi.Operation{}`.
* **`EndpointConfig.SuccessStatus` and `openapi.Operation.SuccessStatus` are
  independent fields.** Nothing syncs them; when you change one, check the
  other (see `examples/cmd/basic/main.go`).
* **`Endpoint` is JSON-only by design, but `Router` isn't.** `Endpoint()`
  checks `Accept` (`accept.go`, `acceptsJSON`) and returns `406` when the
  client explicitly excludes `application/json` — but that is not
  negotiation between multiple representations, and it shouldn't become one.
  Whoever needs XML, PDF or any other format mounts a plain `http.Handler` via
  `Router.GET`/`POST`/etc.; no middleware in the framework is coupled to JSON
  (`Compress`, `ETag` and the rest work with any `Content-Type`). Don't invent
  a second typed-endpoint abstraction for a "non-JSON endpoint" — the existing
  pattern is already a plain `http.Handler`.

## Errors

* **RFC 9457 is the only error format.** Every HTTP error becomes a
  `problem.Problem`. `WriteProblem`/`WriteJSON` always encode into a buffer
  *before* writing any header — don't reverse that order; it's what prevents a
  corrupted response when serialization fails.
* **Binding errors don't carry an HTTP status by default, but can override
  it.** `writeValidationProblem` defaults to 400 for any `binding.Decode`
  error, *except* when a `problem.ValidationError.Code` has `StatusOverride()
  != 0` (`problem/validation_code.go`) — today only
  `ValidationCodePayloadTooLarge` → 413. This is how `MaxBodyBytes` returns 413
  even when the body overflows mid-read (chunked, unknown size), without
  changing `binding.Decode`'s signature. New code implying a status other than
  400 adds a case to `StatusOverride()`; don't invent a parallel mechanism.
* **`WriteProblem` takes `*http.Request` and auto-populates
  `Problem.Instance`** with `request.URL.Path` when `Instance` is empty, never
  overwriting a value already set via `.WithInstance(...)`. `request_id` and
  `trace_id` can't be used here instead, because `httpx` may not depend on
  `httpx/middleware`/`observability` in the `.go-arch-lint.yml` graph — whoever
  wants that calls `.WithInstance(...)` in their own `ProblemMapper`. Every
  internal call site passes `request`, so a new one can't forget the parameter.

## Binding

* **Header binding (`[]string`) treats multiple lines and a single
  comma-separated value as equivalent** — RFC 9110 §5.3 says the two forms are
  semantically the same, so `binding/header.go` (`headerValues`) merges them.
  **Query binding does not**: it only collects a repeated key (`?tag=a&tag=b`)
  and never splits on commas, because no RFC defines that for query strings and
  splitting would break a legitimate `?q=cats,dogs`. Don't unify the two.
