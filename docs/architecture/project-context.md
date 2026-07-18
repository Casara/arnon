# Foundation Go - Project Context

*[Leia em português](project-context.pt-BR.md)*

## Overview

The goal of this project is to build a modern foundation for APIs and microservices in Go, focused on:

* Excellent developer experience.
* Strong OpenAPI integration.
* First-class observability.
* Compatibility with Clean Architecture, DDD, and Hexagonal Architecture.
* Little boilerplate.
* Sensible conventions.
* Independent, decoupled components.
* Ease of testing.
* Production readiness.

The future intent is for the foundation to be distributed as an open source library for the Go community.

---

# Architectural Principles

## Simplicity before abstraction

Abstractions should only be added when there is real gain.

Avoid over-engineering.

---

## Convention over configuration

The framework should infer as much as possible through:

* reflection
* tags
* validators
* Go types

Explicit configuration should exist only to override behavior.

---

## Hybrid OpenAPI

OpenAPI documentation should be generated automatically whenever possible.

The developer can supplement or override metadata manually.

Example:

* request body generated automatically
* default responses generated automatically
* schemas generated automatically
* customized operation when necessary

---

## RFC 9457 as the error standard

All HTTP errors must converge to Problem Details.

The framework uses RFC 9457 as the official error representation standard.

---

## OpenAPI 3.2.0

The adopted version is OpenAPI 3.2.0.

Reasons:

* The project is not yet public.
* It's possible to adopt more modern features of the specification.
* By the time the foundation matures, the version should be more broadly supported.

---

# Current State

## HTTP

Implemented:

* Router
* Route Groups
* Middleware Chain
* Endpoint helper
* Request binding
* Request validation
* JSON responses
* Problem Details
* Problem Mapper

---

## Endpoint Helper

Endpoints use a typed signature.

Conceptual example:

```go
func(
    context.Context,
    RequestDTO,
) (
    ResponseDTO,
    error,
)
```

The endpoint automatically performs:

* binding
* validation
* serialization
* error handling
* mapping to Problem Details

The defaults (default validator, `DefaultProblemMapper`, status 200) come from
`EndpointConfig.WithDefaults()`, called internally by `Endpoint()`.
None of them need to be configured manually for the common case.

### OpenAPI registration is opt-in per endpoint

Unlike binding/validation, a route only enters the generated OpenAPI
document if `EndpointConfig.OpenAPI` is populated (even with an empty
`&openapi.Operation{}`). This is intentional: the developer explicitly
decides which routes are public in the documentation.

### `SuccessStatus` exists in two places

`EndpointConfig.SuccessStatus` (the HTTP status the handler actually
returns) and `openapi.Operation.SuccessStatus` (the status the generated
OpenAPI document describes as the success response) are independent
fields. Today it's the developer's responsibility to keep them
synchronized manually; see `examples/cmd/basic/main.go`. A future
unification of these two fields is a candidate for improvement.

---

## Validation

Validation happens through validators. The default validator
(`validation.Default()`) uses `github.com/go-playground/validator/v10`
under the hood.

Validator information is reused in OpenAPI generation.

### Custom Validators

Custom validation rules (tags that `validator/v10` doesn't know
natively) are registered once, via `validation.RegisterCustomRule(rule)`,
typically during application bootstrap. A single registration feeds
three points at once, which used to be disconnected:

1. **Runtime**: the rule's `Func` is automatically applied to every
   validator created by `validation.New()`/`validation.Default()` from
   the moment of registration onward.
2. **Error mapping**: the rule's `Code` and `Message` define the
   code/detail returned in `problem.ValidationError` when the rule
   fails, instead of the generic `validation_failed` fallback.
3. **OpenAPI**: `Schema` (a `*validation.SchemaEffect` with `Format`
   and/or `Pattern`) enriches the generated schema for fields that use
   the tag, the same way it already happens today for
   `email`/`uuid`/`url`.

Before this change, there was no access to the internal
`validator.Validate` instance used by `PlaygroundValidator`, so there
was no way to register a custom rule at the application level; and even
if there were, OpenAPI generation (which reprocesses the `validate` tag
independently) would have no way to know about the new rule.

### Why `ValidationError` has `detail` + `code` + `source` + `meta`

Each field has a deliberately different role, it's not redundancy:

* `detail` — human-readable English text. It is not a stable contract:
  API consumers should not parse it.
* `code` — stable, i18n-friendly vocabulary (`required`, `min_length`,
  ...). It is deliberately decoupled from `validator/v10`'s internal tag
  names, so as not to leak implementation detail or break the contract
  if the underlying validation library is ever swapped out.
* `meta` — structured rule values (e.g. `min`) so consumers can build
  their own localized message without having to parse `detail`.

### `min`/`max` is length, not numeric value

`validate:"min=1,max=100"` on an `int` is a common semantic mistake:
`validator/v10`'s `min`/`max` always mean string/slice/map length, never
the numeric value itself — `validation/mapper.go` maps both
unconditionally to `ValidationCodeMinLength`/`MaxLength`, with the
message "must contain at least/most N characters", even when applied to
a numeric field. To constrain the *value* of a number, the correct tag
is `gt`/`gte`/`lt`/`lte`.

### Validator error mapping: explicit rules + fallback

`mapFieldError` (`validation/mapper.go`) explicitly maps a fixed set of
known tags; any tag not covered (a `validator/v10` built-in without a
dedicated mapping, or a custom rule registered directly on the
underlying `*validator.Validate` instead of via
`validation.RegisterCustomRule`) falls back to a generic
(`ValidationCodeValidationFailed`, with `meta.rule`/`meta.param`). The
fallback exists deliberately so as to never expose `validator/v10`'s raw
error string (format like `Key: 'Foo.Bar' Error:Field validation...`)
as `detail` — that would leak implementation detail and break the
guarantee that `code` is a stable vocabulary.

---

# OpenAPI

## Current State

Implemented:

* automatic schema generation
* automatic request body generation
* automatic response generation
* automatic query parameter generation
* automatic path parameter generation
* automatic header parameter generation
* automatic error schema generation

---

## Schema Features

Implemented:

* type
* format
* description
* nullable
* deprecated
* readOnly
* writeOnly
* default
* example
* enum
* required
* properties
* additionalProperties
* items
* minimum
* maximum
* exclusiveMinimum
* exclusiveMaximum
* minLength
* maxLength
* minItems
* maxItems
* pattern

`required` in the schema comes exclusively from the `validate:"required"`
tag — never from `json:"...,omitempty"`. These are independent
concerns: `omitempty` only controls JSON serialization (omitting a
zero-value field), it is not used as a proxy for "optional field" in the
generated schema (unlike some other Go frameworks).

---

## Automatic Inference

The framework automatically infers information from validators.

Currently:

### email

```go
validate:"email"
```

↓

```yaml
format: email
```

---

### uuid

```go
validate:"uuid"
```

↓

```yaml
format: uuid
```

---

### url

```go
validate:"url"
```

↓

```yaml
format: uri
```

---

### `dive` redirects the constraint to the element's schema

```go
Tags []string `validate:"dive,min=2"`
```

↓

```yaml
type: array
items:
  type: string
  minLength: 2   # not minItems
```

`applyValidationTags` (`openapi/validation.go`) tracks whether it has
already passed through a `dive` in the `validate` tag; from that point
on, `min`/`max`/`len`/`gt`/`gte`/`lt`/`lte`/`oneof`/`email`/`uuid`/`url`/
custom rule redirect to `schema.Items` instead of the field's schema —
without this, `applyMin`/`applyMax` only look at `schema.Type`
(`"array"` with or without `dive`), so `dive,min=2` would turn into
`minItems: 2` (array with 2+ elements) instead of `minLength: 2` on each
element (the schema would lie about its own contract: the runtime
already validated correctly, only the documented schema was wrong).
`dive,dive` (slice of slice) descends two `Items` levels, and a
`required` after `dive` does not mark the field as required in the
schema (there's no OpenAPI equivalent for "no element may be
zero-value").

---

## Examples

Examples are converted to the correct type.

Examples:

```go
example:"1"
```

↓

```yaml
example: 1
```

---

```go
example:"true"
```

↓

```yaml
example: true
```

---

```go
example:"1.5"
```

↓

```yaml
example: 1.5
```

---

## Default

Defaults are also converted to the correct type.

Examples:

```go
default:"20"
```

↓

```yaml
default: 20
```

---

## OpenAPI Tags

The OpenAPI 3.2 model was adopted.

Supported fields:

* name
* summary
* description
* externalDocs
* parent
* kind

Supported kinds:

* nav
* badge
* audience

---

## Documentation UI

Adopted tool:

Stoplight Elements

Reasons:

* better visual experience
* modern OpenAPI support
* more advanced support than Swagger UI

---

### Current features

* customizable title
* customizable logo
* customizable favicon
* embed mode
* CDN mode

---

# Problem Details

## Standard

RFC 9457

---

## Schemas

Implemented:

### Problem

Represents an HTTP error.

---

### ValidationError

Represents a single validation error.

---

### ValidationSource

Represents the origin of the error.

Example:

```json
{
  "in": "body",
  "field": "/name"
}
```

`field` only uses JSON Pointer syntax (RFC 6901: `/name`,
`/address/city`, `/items/0/name`, `/tags/1`, `/meta/x~1y`) when `in` is
`"body"` — it's the only hierarchical `in`. Each segment's name (the
`json` tag, not the Go field name) is escaped via `~0`/`~1` per RFC
6901 §3 (`validation.escapeJSONPointerToken`) — without this, a field
literally named `"a/b"` would turn into `/a/b`, indistinguishable from
two segments; the same applies to a map key (`Meta["x/y"]` →
`/meta/x~1y`, unlike a slice index, which never needs escaping since
it's always a digit). `buildFieldMap` (`validation/field_map.go`) walks
the *actual value* of the request (not just the type — it needs the
real size of a slice/array/map), recursing into a nested struct (value
or pointer), a slice/array element (struct or primitive), and a map
entry with a string key (struct or primitive), composing the pointer
level by level — including a slice/map of primitives with `dive`
(`Tags[1]`, `Meta["x"]`), since `validator/v10` reports an element error
without a field segment after the index/key, so it needs its own entry
in the map instead of just recursion. The cross-reference with
`validator/v10`'s error uses `FieldError.StructNamespace()` with the
root type name removed (`validation.structFieldNamespace`), not
`StructField()` (which only gives the leaf field name, no path); the
format of `StructNamespace()` for a slice/map element (raw key, no
escaping, in the namespace — only the final pointer is escaped) was
confirmed empirically, not assumed. Limited to `maxFieldMapDepth` (16)
levels (struct, index, and key all count toward the same limit), just
to guarantee termination even with a self-referencing struct. Since it
now walks the actual value, the cost scales with the size of any
slice/map reachable in the request — but only on the error path
(`mapValidationErrors` only runs after there's already at least one
error), it does not affect a successful request. **Current
limitation**: a non-string map key (`map[int]T`) falls back to the
field-name fallback — JSON only has string keys anyway, a rare case in
a request DTO. For
`path`/`query`/`header`
(`NewPathError`/`NewQueryError`/`NewHeaderError` in `problem/validation.go`),
`field` is always the raw field name (`id`, `page`, `Authorization`),
with no `/` prefix (it's not a JSON Pointer, so there's no reason to
escape it). RFC details in
[docs/architecture/rfc-compliance.md](rfc-compliance.md).

---

## Automatic responses

Endpoints automatically receive:

### 400

Bad Request

```http
application/problem+json
```

---

### 500

Internal Server Error

```http
application/problem+json
```

---

## Examples

Each response has its own example.

Example:

400 → validation error

500 → internal error

---

# Middleware

All in `httpx/middleware`, built as `routing.Middleware`
(`func(http.Handler) http.Handler`), applied via `Router.Use`
(global, runs before routing) or `Group.Use` (per-group).

## Implemented

* **CORS** — configurable (`CORSConfig.AllowedOrigins`, etc). Only
  intercepts `OPTIONS` with `204` when it's a real preflight
  (`Access-Control-Request-Method` present, Fetch spec §4.1); a "bare"
  `OPTIONS` falls through to `next`, reaching the `mux` (which returns a
  real `405`+`Allow` reflecting the methods registered for the path, or
  triggers an explicit user `OPTIONS` handler, if any). Always adds
  `Vary: Origin` (the response always depends on the request's
  `Origin`, since `Access-Control-Allow-Origin` echoes the received
  value instead of using a literal `*` — necessary to support
  `AllowCredentials`).
* **Logging** — structured logger (`slog`), enriched with
  `request_id`/`real_ip`/`trace_id`/`span_id` when the corresponding
  middlewares are installed.
* **RealIP** — extracts the client IP, checking in this order:
  `Forwarded` (RFC 7239, the IETF standard) → `X-Forwarded-For` →
  `X-Real-IP` → `RemoteAddr`. Available via `RealIPFromContext`.
* **RequestID** — generates/propagates `X-Request-Id`, available via
  `RequestIDFromContext`.
* **Recover** — recovers from panics, converts them into a 500 Problem
  Details (`problem.NewInternal("")`, generic detail) and logs the
  panic value via `observability.LoggerFromContext` — never includes
  the raw panic value in the response (RFC 9457 §3.1.5).
* **Timeout** — request timeout; a custom implementation (no longer
  uses stdlib's `http.TimeoutHandler`) that responds with Problem
  Details instead of plain text on timeout.
* **StripSlashes** / **RedirectSlashes** — two ways of handling a
  trailing slash in the path: `StripSlashes` normalizes silently (no
  round trip), `RedirectSlashes` redirects (308, preserves method and
  body). Both need to run as *global* middleware (pre-routing) to work
  — see the note in "Important Decisions" about `Router.ServeHTTP`. Do
  not install both at the same time.
* **Compress** — `Compress(level int, types ...string)`, ported from
  chi's `middleware.Compress`. Only compresses when the *response's*
  `Content-Type` (not the request's) matches `types` (or the default
  list of textual/JSON types when `types` is empty; a `/*` suffix
  matches subtypes, e.g. `text/*`) — this avoids spending CPU
  compressing content that wouldn't benefit (images, etc). An invalid
  `level` panics at middleware creation (a configuration error, not a
  runtime one). Removes `Content-Length` from the response when
  compression is applied. `Accept-Encoding` is parsed for real
  (`acceptsGzip`, parsing `;q=` and the `*` wildcard, RFC 9110 §12.5.3),
  not with a simple `strings.Contains` — an explicit `gzip;q=0` is a
  refusal, not acceptance. Absence of the header still means "don't
  compress" (the conservative default that already existed, unchanged
  by this precision).
* **NoCache** — ported from chi's `middleware.NoCache`: besides the
  response headers (full `Cache-Control`, `Pragma`,
  `X-Accel-Expires`, `Expires` at the Unix epoch), it also removes the
  conditional headers from the *request* (`ETag`, `If-Modified-Since`,
  `If-Match`, `If-None-Match`, `If-Range`, `If-Unmodified-Since`)
  before calling the handler — this prevents any downstream code from
  responding conditionally/cached, which would contradict the
  middleware's intent.
* **AllowContentType** — allow-list of accepted `Content-Type` on the
  request, 415 otherwise. Requests without `Content-Type` pass
  (binding already tolerates a missing body).
* **MaxBodyBytes** — request body size limit. When `Content-Length` is
  known and already exceeds the limit, rejects immediately with 413.
  When it isn't (chunked, or a client lying about the size), it uses
  `http.MaxBytesReader` as a second line of defense; the overflow is
  only noticed during reading (inside JSON binding), but it still
  correctly becomes a 413, via
  `problem.ValidationErrorCode.StatusOverride()` — see "Binding errors
  do not carry an HTTP status by default" in "Important Decisions".
* **SecureHeaders** — `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy` always; `Strict-Transport-Security` only if
  explicitly configured (HSTS breaks local development over plain HTTP
  if enabled by default).
* **Throttle** — limit on *concurrent* requests (semaphore), with
  optional backlog (`BacklogLimit`/`BacklogTimeout`) to queue instead
  of rejecting immediately. Not time-based rate limiting — see
  `RateLimit` for that.
* **RateLimit** — real rate limiting
  (`RequestLimit`/`WindowLength` per client key, `KeyFunc` defaulting
  to `RealIPFromContext` → `RemoteAddr`, canonicalized via
  `CanonicalizeIP`). A sliding-window-counter algorithm adapted from
  `go-chi/httprate`: two fixed windows (current and previous) per key,
  with the previous window's count weighted by its overlap with the
  current sliding window. The algorithm (`checkRateLimit`) is separated
  from storage via the `LimitCounter` interface
  (`Config`/`Increment`/`IncrementBy`/`Get`), deliberately mirroring the
  interface of the same name in `go-chi/httprate` — a backend already
  written for httprate (e.g. `go-chi/httprate-redis`) needs only
  trivial changes to serve arnon. A nil `RateLimitConfig.Counter` uses
  the in-memory default (`NewLocalLimitCounter`, exported): memory is
  self-limited, old windows are discarded in bulk (not key by key)
  whenever time advances into a new window, so inactive keys are
  automatically removed within two windows, with no need for manual
  eviction/TTL — but this is only correct for a single instance;
  deployments with multiple instances need a `LimitCounter` with shared
  storage (Redis, Valkey, Memcached, ...), implemented as a separate Go
  module (arnon's core never depends on a specific storage backend).
  A `Counter` error (`Get`/`IncrementBy`) becomes a `problem.Problem`
  via `RateLimitConfig.OnCounterError` (default: 503 Service
  Unavailable, without leaking the error message; configurable).
  `CanonicalizeIP` reduces IPv6 addresses to their /64 prefix (an IPv6
  client controls an entire /64 via SLAAC; without this it could
  rotate addresses within its own block to dodge the limit). The
  response always includes
  `X-RateLimit-Limit`/`X-RateLimit-Remaining`/`X-RateLimit-Reset`, and
  `Retry-After` (RFC 6585) on 429. Implemented in
  `httpx/middleware/rate_limit.go`, with no external dependency (only
  `sync`/`time`/`net`/`math` from the stdlib in the core; external
  storage adapters live outside the module).
* **ETag** — conditional GET (RFC 9111/9110 §13). Only acts on
  `GET`/`HEAD` and on 2xx responses; buffers the handler's entire body
  (needs the complete body to hash it), computes a strong ETag via
  FNV-1a 64-bit (stdlib `hash/fnv`) and compares it against
  `If-None-Match` using weak comparison (ignores a `W/` prefix on
  either side, per RFC 9110 §13.1.2). On a match (or
  `If-None-Match: *`), responds `304 Not Modified` with no body;
  otherwise, responds with the full body plus the `ETag` header. An
  `ETag` already set by the handler is respected instead of
  recalculated. When combined with `Compress`: install `ETag` first
  (more external), to hash the already-compressed bytes, consistent
  with the `Vary: Accept-Encoding` that `Compress` already sets.
  Combining with `NoCache` on the same route defeats the purpose of
  both. Implemented in `httpx/middleware/etag.go`.
* **ServiceDesc** — adds a `Link: <path>; rel="service-desc"`
  (RFC 8631) header to every response, pointing to the OpenAPI document
  (e.g. `/openapi.json`), enabling automatic discovery by a
  generic client/tool that already understands `Link` headers. Uses
  `header.Add`, not `Set`, so it adds to any other existing `Link`
  headers instead of replacing them. Implemented in
  `httpx/middleware/service_desc.go`.

## Middleware order

The relative order of global middlewares (`Router.Use`) matters —
several have real dependencies on each other (context that one
populates and another reads, bytes that one needs to see before
another transforms them). Two ways to enforce this, in order of
preference:

### `middleware.BuildChain` — order guaranteed by code, not by discipline

`middleware.BuildChain(config middleware.ChainConfig) []routing.Middleware`
(`httpx/middleware/chain.go`) always assembles the recommended global
chain in the right order — each `ChainConfig` field is
optional/independent (nil or `false` = "not mentioned", not
"disabled"), but the relative position of whichever ones are included
never changes, because the order is decided by `BuildChain`'s code, not
by whoever calls `router.Use(...)`. Usage:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover:       true,
    RealIP:        true,
    RequestID:     true,
    SecureHeaders: &middleware.SecureHeadersConfig{},
    RateLimit:     &middleware.RateLimitConfig{ /* ... */ },
    ETag:          true,
    Compress:      &middleware.CompressConfig{},
    CORS:          &middleware.CORSConfig{ /* ... */ },
    ServiceDescPath: "/openapi.json",
    Logger:        logger,
})...)
```

`BuildChain` also **prevents at runtime** the only mutually exclusive
combination that exists today: setting `StripSlashes` and
`RedirectSlashes` together causes an immediate panic (at chain
creation, not in the middle of a request).

What `BuildChain` does and does not guarantee: any call with the same
subset of fields populated always produces the same relative order
between them — this is genuinely tested in
`httpx/middleware/chain_test.go` (not just documented), verifying
observable behavior (`RequestID` appearing in `Logging`'s log, `ETag`
hashing bytes already compressed by `Compress`, `Recover` catching a
panic from anywhere in the chain, `SecureHeaders` appearing even on a
`429` response from `RateLimit`). What is not guaranteed: a chain
assembled manually with `router.Use(mw1, mw2, ...)`, entirely outside
`BuildChain`, remains the responsibility of whoever writes it — there
is no (nor would it be reasonable to build, given that
`routing.Middleware` is just `func(http.Handler) http.Handler`, with no
runtime identity of its own) static validation that blocks a malformed
manual call. `BuildChain` (including `Extra`, below) is the recommended
path precisely so this isn't needed in most cases.

Group middlewares (`AllowContentType`, `MaxBodyBytes`, `NoCache`) are
deliberately left out of `BuildChain`: they're scoped to a specific
group (e.g. only `/api`, not `/openapi.json`/`/docs`) by design, and
don't make sense as part of the global chain. There's no relevant order
between them (they're independent), so they don't need their own
builder — use `group.Use(...)` directly.

### Custom middleware with an order requirement — `ChainConfig.Extra`

`BuildChain` only knows about arnon's built-in middlewares — if a
custom or third-party middleware needs to run at a specific position
relative to a built-in one (e.g. "after `RateLimit`, before `ETag`"),
that can be done in two ways:

**1. Split the call.** Since `Router.Use(...)` accumulates on every
call (call order is preserved) and each `ChainConfig` field is
independent of the others, you can call `BuildChain` twice with
complementary subsets of fields, with the custom middleware in
between:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover: true, RealIP: true, RequestID: true, RateLimit: &cfg,
})...)
router.Use(xpto.Middleware()) // precisa vir depois do RateLimit, antes do ETag
router.Use(middleware.BuildChain(middleware.ChainConfig{
    ETag: true, Compress: &cCfg, CORS: &corsCfg, Logger: logger,
})...)
```

**2. `ChainConfig.Extra` — same result, in a single call.** Each
position in `BuildChain` has a named `ChainAnchor`
(`AnchorRecover`, `AnchorTimeout`, `AnchorStripSlashes`,
`AnchorRedirectSlashes`, `AnchorRealIP`, `AnchorRequestID`,
`AnchorSecureHeaders`, `AnchorRateLimit`, `AnchorThrottle`,
`AnchorETag`, `AnchorCompress`, `AnchorCORS`, `AnchorServiceDesc`,
`AnchorLogging`, in the same order as the list below). An
`ExtraMiddleware{Middleware: ..., Before: Anchor...}` or `{...,
After: Anchor...}` inserts the custom middleware right before/after
that point:

```go
router.Use(middleware.BuildChain(middleware.ChainConfig{
    Recover: true, RealIP: true, RequestID: true,
    RateLimit: &cfg,
    Extra: []middleware.ExtraMiddleware{
        {Middleware: xpto.Middleware(), After: middleware.AnchorRateLimit},
    },
    ETag: true, Compress: &cCfg, CORS: &corsCfg, Logger: logger,
})...)
```

A `ChainAnchor` names a *position*, not the presence of a specific
middleware — `Extra` anchored at `AnchorETag` still lands in the right
place even if `ChainConfig.ETag` is `false` on that call. `BuildChain`
validates each `ExtraMiddleware` and panics (at chain creation, not in
the middle of a request) if: neither `Before` nor `After` is set, both
are set at the same time, or the referenced anchor isn't one of the
`AnchorXxx` constants — this last case exists because a typo in the
anchor name, without this validation, would simply drop the custom
middleware from the chain silently. Multiple `Extra` entries anchored
at the same point stack in the order they appear in the slice.

Both approaches produce the same result; `Extra` just avoids having to
split the call and remember which fields go in each half. Both remain,
ultimately, "where in the code the middleware gets called" — `Extra`
adds no verification beyond "this anchor exists and is well-formed",
it does not validate whether the custom middleware itself is safe to
run at that position (that remains the judgment of whoever writes it,
as in any other language without a type system that can model
"execution order").

### The order itself, and why

From the most external (runs first, wraps everything) to the most
internal (runs last, closest to the handler):

1. **`Recover`** — needs to wrap literally everything below it to
   catch a panic from any middleware, not just the final handler.
   Accepted trade-off: since it runs before `RequestID`/`Logging`, it
   doesn't have `request_id`/`trace_id` in the panic log, unless
   repositioned to after those two (see the note under `Recover`,
   below).
2. **`Timeout`** — the deadline must apply to the entire chain below
   it, and `Timeout` itself re-raises (`panic`) the handler's panic
   outward, expecting a more external `Recover` to catch it.
3. **`StripSlashes`/`RedirectSlashes`** (mutually exclusive) — needs
   to normalize the path before anything that depends on it,
   including the `mux`'s own routing.
4. **`RealIP`** — populates context that `RateLimit` (keyed by IP) and
   `Logging` (`real_ip` in the log) read later.
5. **`RequestID`** — populates context that `Logging` (`request_id` in
   the log) reads later.
6. **`SecureHeaders`** — cheap, wants to appear on every response,
   including errors generated by any middleware below it (a `429`
   from `RateLimit`, a `404` from the `mux`).
7. **`RateLimit`**/**`Throttle`** — reject early, before any real work
   (including before `ETag`/`Compress` spend CPU on a response that
   won't even be accepted).
8. **`ETag`** — needs to come before `Compress` to hash the bytes that
   actually go out on the wire (already compressed), not the
   pre-compression version — consistent with the
   `Vary: Accept-Encoding` that `Compress` sets.
9. **`Compress`**.
10. **`CORS`** — intercepts preflight (`OPTIONS` with
    `Access-Control-Request-Method`) before the `mux`; an `OPTIONS`
    that isn't a preflight falls through to the `mux`, so this
    position doesn't block the real `405`+`Allow` discussed in
    `docs/architecture/rfc-compliance.md`.
11. **`ServiceDesc`** — only adds a header, with no strong position
    dependency; sits near the end by convention.
12. **`Logging`** — deliberately the most internal of the group above:
    it only assembles its log attributes (including what
    `RealIP`/`RequestID` populated) once, before calling `next`, so it
    needs to be last to already see everything the others left in the
    context.

Note on `Recover` + correlation: since it's the most external (item
1), it *cannot* see the `request_id`/`trace_id` that
`RequestID`/`Logging` (items 5 and 12) only populate after it has
already run its pre-processing logic. Anyone who needs this must give
up `BuildChain` for this specific part and assemble `Recover` manually
after `RequestID` — a real trade-off (loses the guarantee of catching
a panic from anywhere, gains correlation in the panic log), not a
configuration that can have it both ways at once.

**Maintenance**: every new global middleware needs to gain a field in
`ChainConfig`, a corresponding `ChainAnchor` (also added to
`validChainAnchors`), and an `appendStage(...)` call at the right
position inside `BuildChain` — otherwise it becomes inaccessible via
`BuildChain`/`Extra` and this doc section becomes outdated. Group
middleware (`AllowContentType`-like) doesn't need this.

## Planned / deferred

* **Authentication (Bearer/Basic)** — deferred, see "Security" below.

---

# Important Decisions

## Global middleware wraps the entire mux, not each route

`Router.Use` (global middleware) is applied in `Router.ServeHTTP`,
wrapping the entire `mux` — not in `router.register`, per route. This
is what allows pre-routing middleware (`StripSlashes`,
`RedirectSlashes`) to work, and makes routes not found (404) also pass
through `RequestID`/`Logging`/`RateLimit`/etc. Group middleware
(`Group.Use`) continues to be applied per-route in `router.register`,
since `net/http.ServeMux` has no notion of prefix. Do not go back to
merging `router.middlewares` inside `register()` — that would
duplicate execution.

## Binding errors do not carry an HTTP status by default

`httpx.Endpoint` maps every `binding.Decode` error to 400
(`writeValidationProblem`, in `httpx/endpoint.go`), regardless of the
specific `problem.ValidationError` code. The exception is
`problem.ValidationErrorCode.StatusOverride()`
(`problem/validation_code.go`): if any error has a code with an
override (today only `ValidationCodePayloadTooLarge` → 413), that
status replaces the default 400. This is what lets `MaxBodyBytes`
return 413 even when the body overflows during reading (chunked),
without needing to change `binding.Decode`'s signature. When adding a
new validation code that should imply a status other than 400, add the
case in `StatusOverride()` instead of inventing another mechanism.

## `WriteProblem` requires `*http.Request` to auto-populate `Problem.Instance`

`httpx.WriteProblem(writer, request, problemInstance)` has taken the
request since 2026-07-15 (a signature change — acceptable because the
framework hasn't had a public release yet). If
`problemInstance.Instance` is empty, it gets filled with
`request.URL.Path` before serializing, never overwriting a value
already set via `.WithInstance(...)`. `httpx` cannot depend on
`httpx/middleware`/`observability` (see the dependency graph above),
so `request_id`/`trace_id` can't be used here — path is what's
achievable without widening that boundary. Every new `WriteProblem`
call site must pass the request.

## `httpx.Endpoint` is JSON-only by design; `Accept` negotiation formalizes this

`Endpoint()` checks the `Accept` header (`httpx/accept.go`,
`acceptsJSON`) before doing any binding and responds `406 Not
Acceptable` (Problem Details) when the client explicitly excludes
`application/json` (e.g. `Accept: application/xml` alone, or
`application/json;q=0`). A missing, empty `Accept`, or one that
includes `application/json`/`application/*`/`*/*` with `q > 0` passes
normally — RFC 9110 §12.5.1 says a missing header means "accepts
anything". The parser follows the "most specific match wins" rule: an
exact entry beats `application/*`, which beats `*/*`.

This is not (and should not become) negotiation of multiple
representations of the same endpoint — `Endpoint()` still only ever
produces JSON. Anyone who needs to serve XML, PDF, CSV, or any other
format/file builds a plain `http.Handler` via `Router.GET`/`POST`/etc,
exactly like any other route; no framework middleware (`Compress`,
`ETag`, `SecureHeaders`, ...) is coupled to JSON. Don't create a second
"format-generic" typed endpoint abstraction to cover this case — the
pattern is already to use a plain `http.Handler`.

## Pointers in Schemas

Properties use pointers.

Example:

```go
Properties map[string]*Schema
```

Reason:

Avoid unnecessary copies and allow recursive structures.

---

## AdditionalProperties

Uses:

```go
AdditionalProperties *Schema
```

---

## Receivers

Preference for pointer receivers.

Reasons:

* avoid copies
* consistency
* compatibility with the recvcheck linter

Deliberate exception: small, immutable value-types with no identity of
their own (e.g. `openapi.Tag`, which is fluent and returns new values
on every `With*`; `problem.ValidationErrorCode`, an enum) use a
**value** receiver on purpose — this is not an inconsistency to fix.
The heuristic: a type with identity/mutation/builder → pointer; a
small, immutable type that behaves like a value → value.

---

## Stoplight

Stoplight Elements was chosen over Swagger UI — it fits better with the
"lightly opinionated foundation" premise (more neutral visuals,
presented as a doc/portal rather than a test console).

Point revisited on 2026-07-16: Swagger UI (starting with
`swagger-ui-dist@5.32.0`, Feb/2026) gained support for OpenAPI 3.2.0;
Stoplight Elements, as of the same date, documents official support
only up to 3.1. Since `arnon` generates documents with
`"openapi": "3.2.0"` (`openapi.OpenAPIVersion3_2`), this may mean
Stoplight Elements doesn't recognize new 3.2 features (most of the 3.2
changes over 3.1 are additive, so overall rendering should keep
working). Not yet verified empirically in a real browser — before
switching the default or exposing the UI as configurable (`NOTES.md`),
this validation is worth doing.

---

## Hybrid OpenAPI

Automatic generation remains the primary strategy.

Customizations should complement automatic generation, never require
repeating configuration.

---

# Planned Features

## Observability (Highest Priority)

### OpenTelemetry

Tracing:

* HTTP Server Tracing
* Trace Propagation
* Route Attribution
* Error Attribution

Metrics:

* Request Count
* Request Duration
* Active Requests

Context:

* Trace ID
* Span ID
* Request ID

**Note**: despite the section title, everything above is already
implemented (`observability`/`observability/otel`), it's no longer
"planned" — `routing.WithInstrumentation(otel.NewHandler)` gives
automatic HTTP tracing (via `otelhttp`, with route attribution and
context propagation), `otel.Initialize` with `MetricsEnabled: true`
enables automatic HTTP metrics plus Go runtime metrics, and
`observability.TraceID`/`SpanID` correlate trace_id/span_id in
structured logs (`httpx/middleware/logging.go`). Demonstrated and
validated end to end (exported trace matching the application log,
custom metrics with exemplars pointing to the exact trace) in
`examples/cmd/observability`, including a local OTel Collector via
Docker Compose. What's genuinely still missing is automated test
coverage for `observability`/`observability/otel` (0% today, see
NOTES.md), not the functionality itself.

---

## Health Endpoints

* /health
* /ready
* /live

Kubernetes-compatible.

---

## Security

Authentication:

* Bearer Token
* Basic Auth

Authorization:

* policy abstraction

---

## Configuration

* env var reading
* defaults
* configuration validation

---

## Tests

Implemented:

### Unit

* coverage of the core packages: `validation`, `openapi`, `problem`,
  `httpx` and `httpx/binding`.
* `httpx`: end-to-end tests via `httptest`, covering binding,
  validation, error mapping (default and custom), and the success
  path.

Planned:

### OpenAPI (Golden Tests)

* Golden Tests

### Unit (pending)

* coverage of `httpx/middleware` and `observability`/`observability/otel`

### Mutation

* robustness validation

---

## Quality and Tooling

Implemented:

* `.golangci.yml`: a curated set of linters (not `--enable-all`),
  tuned to the project's style (e.g. `funlen`/`cyclop` with limits
  compatible with the adopted vertical format; `ireturn` allowing the
  interface returns that are a design decision, like
  `validation.Validator`).
* `.go-arch-lint.yml`: models arnon's actual dependency graph between
  packages and fails the build if a disallowed dependency is
  introduced.
* `examples/`: executable examples organized as `cmd`+`internal`.
  `examples/cmd/basic` (`go run ./examples/cmd/basic`) is the bare
  minimum — typed endpoint, validation, OpenAPI, zero middleware.
  `examples/cmd/middleware` (`go run ./examples/cmd/middleware`) is
  the same endpoint with the full middleware stack (CORS, rate limit,
  compression, security headers, etc).
  `examples/cmd/observability` (`go run ./examples/cmd/observability`)
  is the same endpoint with `routing.WithInstrumentation(otel.NewHandler)`,
  custom metrics (`observability.Counter`/`Histogram`) and logs
  correlated by trace_id/span_id, genuinely exporting via OTLP/gRPC to
  a local OTel Collector brought up by
  `docker compose -f examples/cmd/observability/docker-compose.yml up`
  (config in `otel-collector-config.yaml`, `debug` exporter — prints
  every trace/metric received right in the collector's own log,
  without needing Jaeger/Prometheus to validate the integration). It
  also explicitly handles graceful shutdown (`SIGINT`/`SIGTERM`),
  unlike the other two examples: that's what guarantees the flush of
  pending spans/metrics in the SDK before the process exits. Code
  shared across all three examples (logger, custom validator
  registration, the example handler) lives in `examples/internal/*`,
  which cannot be imported from outside `examples/` per Go's rule. All
  are built and exercised via `hurl --test` as part of the project's
  validation; `examples/cmd/observability` was also validated with a
  real collector running (exported trace matching bit-for-bit the
  trace_id/span_id logged by the application, custom metrics with
  exemplars pointing to the exact trace).
* [docs/architecture/rfc-compliance.md](rfc-compliance.md): a full
  review (2026-07-15) of compliance with the RFCs relevant to an HTTP
  foundation (RFC 9457, RFC 9110, RFC 9111, RFC 7239, RFC 6585,
  RFC 8288/8631/8615, RFC 8259), separating what's already compliant,
  what's a deliberate scope decision, and what's a real gap —
  including two findings that break the "RFC 9457 is the only error
  format" guarantee itself (`Recover()` leaking panic detail,
  `Timeout` responding in plain text) and a multi-hop
  `X-Forwarded-For` parsing bug found along the way.

---

# Future OpenAPI Improvements

Not yet a priority.

* operationId
* multiple examples
* discriminator
* automatic pattern
* security schemes
* callbacks
* webhooks
* links
* XML
* const
* automatic tags

---

# Short-Term Goal

Implement OpenTelemetry-based observability.

Initial scope:

* HTTP tracing
* context propagation
* trace id
* span id
* automatic route attribution
* automatic error marking

After tracing:

* metrics
* health endpoints

---

# Long-Term Goal

Make the foundation a modern alternative for building APIs and
microservices in Go, offering:

* first-class OpenAPI
* native observability
* low boilerplate
* excellent developer experience
* independent components
* strong integration with modern architectures
* production readiness from the start
  """
