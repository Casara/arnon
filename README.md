<!-- markdownlint-disable-next-line MD041 -- deliberately no H1: the logo image is the title. -->
<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./static/arnon-dark.svg" />
    <img src="./static/arnon-light.svg" height="105" alt="Arnon Logo" />
  </picture>
</div>

<div align="center">
  <strong>
    A minimalist, conceptual, standards-based foundation for the edge.
  </strong>
</div>

<div align="center">

[![CI](https://github.com/casara/arnon/actions/workflows/ci.yml/badge.svg)](https://github.com/casara/arnon/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/casara/arnon.svg)](https://pkg.go.dev/github.com/casara/arnon)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

*[Leia em português](README.pt-BR.md)*

## Why Arnon

> "...for Arnon is the border of Moab, between Moab and the Amorites."
>
> — [Numbers 21:13](https://www.bibliaonline.com.br/kjv/nm/21/13+) (KJV)

The Arnon is a boundary river: the line where one territory ends and
the next begins. An HTTP API sits on that same kind of line — the edge
where untyped bytes from outside become typed values inside, and where
a failure has to be answered in a format the other side already
understands. That's where the name comes from, what the tagline means,
and why the logo is two banks and the crossing between them.

`arnon` didn't start as "let's build a framework." It started as an
API for a real financial system, on top of fuego. In that project,
pointing at the exact field of a validation error via RFC 6901 (JSON
Pointer) — the way RFC 9457 (Problem Details) expects — didn't fit
fuego's own error-field convention (`err.StructNamespace()`, Go
type/field names, not built for pointer-style paths). That was a
problem specific to what this project needed, not a verdict on fuego
in general. The fix was to drop down to the stdlib (`net/http`)
directly instead of trying another framework. The resulting structure
grew reusable enough to become a lib, then a framework — published in
case others run into the same need; the tradeoffs below are for you to
judge.

The stated goal is to be as unopinionated as possible — whoever uses
it should be able to keep `arnon`'s default choices or swap them for
their own without much complexity. We don't consider that fully
achievable: the moment any composition decision is made (e.g.
`Endpoint()` bundling binding, validation, error, and OpenAPI into one
piece), that's already an opinion. The criterion that remains, then,
isn't "opinionated or not," it's **where each opinion comes from** and
**how cheap it is to change**:

* **Opinion comes from an RFC when one exists, not from taste.** The
  field that points at the origin of a validation error uses real RFC
  6901 (JSON Pointer), `~`/`/` escaping included (RFC 6901 §3) — not a
  notation invented for `arnon`. Compliance doesn't stop at the error
  format (RFC 9457) either: `Forwarded` (RFC 7239) instead of just
  `X-Forwarded-For`, real ETag/conditional GET (RFC 9111), real
  `q=`-parsing `Accept`/`Accept-Encoding` negotiation (RFC 9110), exact
  preflight semantics in `CORS`, OpenAPI 3.2.0 since the initial
  design. Full detail, RFC by RFC — including what isn't 100% compliant
  yet — in
  [docs/architecture/rfc-compliance.md](docs/architecture/rfc-compliance.md).
* **Where no RFC exists, still the language's own standard — not
  something reinvented.** `arnon` runs directly on top of
  `net/http.Server`: `Router` implements `http.Handler`, so it's just
  `&http.Server{Handler: router}` (see "Quick start" above) — no
  custom HTTP engine replacing the stdlib's. That inherits, for free,
  maintenance/security patches from the Go team itself, compatibility
  with any existing `http.Handler` middleware, `httptest`,
  `net/http/pprof`, automatic HTTP/2 with TLS.
* **Every built-in opinion comes with a cheap way to change it.**
  Middleware order: `middleware.BuildChain` guarantees the recommended
  order by code, but `ChainConfig.Extra`/`ChainAnchor` lets you insert
  custom middleware at a specific position without editing the
  built-in chain (detail in
  [docs/architecture/project-context.md](docs/architecture/project-context.md),
  "Middleware order" section). Validation:
  `validation.RegisterCustomRule` is the only mechanism — there's no
  second, parallel path to register a custom rule. Rate limit: the
  algorithm (sliding window) is separated from storage via the
  `LimitCounter` interface, swappable for a shared backend (Redis
  etc.) without touching the algorithm.

### Compared to huma and fuego, specifically

huma and fuego **are not the most-used Go frameworks** — Gin dominates
real-world adoption (~48%, ~88k stars), followed by Echo and Fiber,
orders of magnitude ahead of huma (~4.3k stars) and fuego (~1.8k). The
comparison below isn't with them because they're the most popular,
it's because they're the only two with the same core pitch as `arnon`
(typed handler → OpenAPI generated automatically by reflection, no
annotation or separate generation step). Gin/Echo/Fiber solve a
different problem — low-level routing and middleware, manual binding
(`c.ShouldBindJSON`), no native RFC 9457, and OpenAPI (when used) comes
from comment annotations in the code via `swaggo/swag`, not from the
Go type itself — so a direct comparison on the points below wouldn't
be fair or informative for them; comparing `arnon` to Gin on these
terms would be like comparing a bicycle and a car by their engine.

| Aspect | [huma](https://github.com/danielgtaylor/huma) | [fuego](https://github.com/go-fuego/fuego) | `arnon` |
| --- | --- | --- | --- |
| Error format | Native RFC 9457 | Native RFC 9457 | Native RFC 9457 — **not a differentiator**, it's today's expected floor |
| Validation error field | Ad-hoc dot notation (`"body.title"`) | `validator/v10`'s internal namespace (`err.StructNamespace()`, Go type/field name, not the `json` tag) | RFC 6901 (JSON Pointer), with escaping |
| Sanitization before validation | Not built in | `InTransform`/`OutTransform` interface methods — explicit code per type, not recursive into nested structs | `sanitize` struct tag (`sanitize:"trim"`), recursive into nested structs/slices/maps, custom transforms through the same single-registry pattern as validation |
| Wire format beyond JSON | Opt-in CBOR (RFC 8949) via the `formats/cbor` subpackage — isolates its `fxamacker/cbor/v2` dependency there, registers into the same `huma.Format`/`Accept` negotiation JSON already uses | Built-in JSON/XML/YAML/HTML/plain text via `Accept`; other formats need a hand-written `WithContentTypeSerDes` — no CBOR out of the box | JSON only by design — no per-endpoint negotiation between representations; anything else is a plain `http.Handler` (see `examples/cmd/files`), not `httpx.Endpoint` |
| `PATCH` from `GET`+`PUT` | `autopatch.AutoPatch(api)` — auto-discovers the pair via huma's own operation registry, RFC 7386/RFC 6902 both supported | Not built in | `httpx/patch.From(get, put, ...)` — explicit, no route introspection (`arnon` has none to reuse without a layering violation); same two RFCs |
| Write preconditions / optimistic concurrency | `conditional.Params` embedded in the input struct + `input.PreconditionFailed(etag, modified)`, called by the handler with its own current state | Not built in | `httpx/precondition.Check`/`CheckRequest` — same "handler supplies current state" shape as huma, `412`/opt-in `428`; `httpx/patch.From` calls it on the client's own `If-Match` before ever applying a patch |
| Generated OpenAPI version | 3.1 (`kin-openapi`, no 3.2 yet) | ~3.0 (3.1/3.2 unconfirmed) | 3.2.0 — an advantage with a short shelf life, it's a matter of time until the rest of the ecosystem catches up |
| Server/router | Bring-your-own — `net/http` via `humago`, but also `fasthttp` via `humafiber` (which loses compatibility with the `net/http` ecosystem) | `net/http` directly, same choice as `arnon` | `net/http` directly, but its own router — not pluggable into an existing `gin.Engine`/`echo.Echo` |

### Where `arnon` isn't the right choice today

* **No production track record.** It's a new project, no track
  record — unlike huma (more mature, bigger community) or fuego. If
  maturity/battle-testing outweighs the differences above, consider
  the alternatives.
* **Its own router is today's most expensive opinion to change.**
  Unlike the other opinions in this section, this one didn't come from
  any RFC (it's pure architecture) and doesn't yet have a low-friction
  way around it — see the table above.

## Installation

```sh
go get github.com/casara/arnon
```

Requires Go 1.26 or later.

The OpenTelemetry wiring is a **separate module**, so the OTel SDK and the
OTLP/gRPC exporters stay out of your dependency graph unless you ask for them:

```sh
go get github.com/casara/arnon/observability/otel
```

You only need this to export traces and metrics. `observability` itself — the
counters, histograms and attributes the middleware records — ships with the
main module. See
[docs/architecture/adr-0001-module-layout.md](docs/architecture/adr-0001-module-layout.md).

## Quick start

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/casara/arnon/httpx"
	"github.com/casara/arnon/httpx/routing"
	"github.com/casara/arnon/openapi"
)

const readHeaderTimeout = 5 * time.Second

type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

func createUser(
	_ context.Context,
	request CreateUserRequest,
) (CreateUserResponse, error) {
	return CreateUserResponse{ID: "usr_123"}, nil
}

func main() {
	generator := openapi.NewGenerator(openapi.Info{
		Title:   "Example API",
		Version: "1.0.0",
	})

	router := routing.NewRouter(
		routing.WithOpenAPI(generator),
	)

	router.POST("/users", httpx.Endpoint(
		createUser,
		httpx.EndpointConfig{
			SuccessStatus: http.StatusCreated,
			OpenAPI: &openapi.Operation{
				Summary:       "Create a user",
			},
		},
	))

	document := generator.Generate()

	router.GET("/openapi.json", openapi.NewHandler(&document))
	router.GET("/docs", openapi.NewDocsHandler(openapi.DocsConfig{}))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	log.Fatal(server.ListenAndServe())
}
```

With no extra code, this handler automatically gets:

* path, query, header, and JSON body binding;
* request validation (`validate:"required,email"`) with a
  `400 application/problem+json` response in RFC 9457 format;
* handler error mapping to Problem Details;
* an OpenAPI 3.2 schema generated by reflection for request and
  response;
* default `400`/`500` responses documented automatically;
* a documentation UI (Stoplight Elements) at `/docs`.

Complete, runnable examples live in `examples/cmd`:

* [examples/cmd/basic](examples/cmd/basic/main.go) — exactly the
  handler above, with no middleware at all.
* [examples/cmd/middleware](examples/cmd/middleware/main.go) — the
  same handler with arnon's full middleware stack (CORS, rate
  limiting, compression, security headers, etc).
* [examples/cmd/observability](examples/cmd/observability/main.go) —
  the same handler with tracing and metrics via OpenTelemetry,
  actually exporting to a local OTel Collector (started with
  `docker compose`).
* [examples/cmd/files](examples/cmd/files/main.go) — file
  download/upload (PDF, CSV, XML): plain `http.Handler`s mounted
  directly on the router, since `httpx.Endpoint` is JSON-only by
  design.
* [examples/cmd/staticfiles](examples/cmd/staticfiles/main.go) —
  serving static assets via `http.FileServer`, with `ETag`/`Compress`
  applied globally: both step aside for any request carrying a
  `Range` header, so `http.FileServer`'s own Range/conditional-GET
  support keeps working untouched - confirmed empirically, see the
  doc comments on `middleware.ETag`/`middleware.Compress`.
* [examples/cmd/patch](examples/cmd/patch/main.go) — `PATCH` derived
  from an existing `GET`+`PUT` pair via `httpx/patch.From` (RFC 7386
  JSON Merge Patch and RFC 6902 JSON Patch, selected by
  `Content-Type`), with neither handler changed to support it.

```sh
go run ./examples/cmd/basic
# or
go run ./examples/cmd/middleware
# or (requires docker compose -f examples/cmd/observability/docker-compose.yml up)
go run ./examples/cmd/observability
# or
go run ./examples/cmd/files
# or
go run ./examples/cmd/staticfiles
# or
go run ./examples/cmd/patch
```

## Overview

`arnon` is made up of independent packages, each with a single
responsibility:

| Package                | Responsibility                                                                                                       |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `problem`             | HTTP errors in RFC 9457 (Problem Details) format.                                                                      |
| `sanitize`            | Struct-tag-driven data transforms (trim, custom funcs), applied before validation.                                     |
| `validation`          | Request validation, with custom rule registration.                                                            |
| `openapi`             | Schema and OpenAPI document generation from Go types.                                                        |
| `httpx`               | Typed endpoint: binding, validation, serialization, and errors.                                                             |
| `httpx/binding`       | Path, query, header, and JSON body binding.                                                                            |
| `httpx/routing`       | Router based on `net/http.ServeMux`, with groups and middleware.                                                        |
| `httpx/middleware`    | Standard middleware (CORS, recovery, request ID, logging, rate limiting, throttle, compression, security headers, etc). |
| `httpx/patch`         | Derives a `PATCH` handler from an existing `GET`+`PUT` pair (RFC 6902 / RFC 7386).                                     |
| `httpx/precondition`  | Write preconditions (`If-Match`/`If-Unmodified-Since`, RFC 9110 §13.1.1/§13.1.4), opt-in `428`.                        |
| `observability`       | Thin abstractions over the OpenTelemetry API.                                                                         |
| `observability/otel`  | OpenTelemetry SDK configuration and initialization.                                                                  |

The allowed dependency graph between these packages is documented in
[.go-arch-lint.yml](.go-arch-lint.yml) (diagram in
[AGENTS.md](AGENTS.md#package-dependency-graph)) and is checked in CI.

More context on architectural decisions is in
[docs/architecture/project-context.md](docs/architecture/project-context.md).

## Custom validation

Custom validation rules are registered once, at any point during
application bootstrap, and take effect for every validator created
from then on — both at runtime and in OpenAPI schema generation:

```go
validation.RegisterCustomRule(validation.CustomRule{
	Tag:  "cpf",
	Func: validateCPF,
	Schema: &validation.SchemaEffect{
		Format:  "cpf",
		Pattern: `^\d{11}$`,
	},
	Code: "invalid_cpf",
	Message: func(param string) string {
		return "invalid CPF"
	},
})

func validateCPF(field validation.FieldContext) bool {
	return cpfPattern.MatchString(field.Field().String())
}
```

`FieldContext` is arnon's own interface, so a rule never names a type from the
validation library underneath and keeps compiling if that implementation is
ever replaced. It exposes the field value, the tag parameter, and the parent
and top-level structs for a cross-field check.

## Sanitization

Struct tags transform request data before validation runs, so a check
like `required`/`min` sees the value a client intends, not raw bytes
that happen to satisfy it without meaning to (e.g. `"C "` passing
`min=2` on its untrimmed length):

```go
type CreateUserRequest struct {
	Email string `json:"email" validate:"required,email" sanitize:"email"`
}
```

Built-in: `trim` and `email` (trim + lowercase). Register your own the
same way as a custom validation rule:

```go
sanitize.RegisterFunc(
	"digitsOnly",
	sanitize.FromRegexp(regexp.MustCompile(`[^0-9]`)),
)
```

A struct field is always recursed into; a slice/array/map field needs
its tag to start with `dive` (`sanitize:"dive,trim"` on a `[]string`),
matching `validate`'s own convention.

## API stability

`arnon` is released as **v0.x**, and in Semantic Versioning that carries a
specific meaning worth stating plainly rather than leaving you to infer it:
**a minor release may break the public API.** Pin a version, read the
[CHANGELOG](CHANGELOG.md) before upgrading, and expect to make small
mechanical changes when you do.

What counts as the public API: every exported symbol in the published
modules — `problem`, `sanitize`, `validation`, `openapi`, `httpx` and its
subpackages, `observability`, and `github.com/casara/arnon/observability/otel`.
Not public: `examples/` (a separate, unpublished module), anything under an
`internal/` directory, and the exact wording of error strings and log messages.

Breaking changes will be listed under `### Changed` or `### Removed` in the
changelog, with the migration in the same entry. Where a rename can keep the
old spelling compiling, it will — as a deprecated alias, removed no earlier
than the following minor.

The road to v1.0.0 is the point at which the surface stops moving, not a
feature milestone. One question is still open, and it is the kind that gets
expensive to change afterwards: whether `validation.WithConfigure` — the one
remaining place a `go-playground/validator/v10` type appears in an exported
signature — should stay as an escape hatch or move behind a subpackage. Once
that settles, v1 follows.

## Rate limit storage

`RateLimit` keeps the sliding-window algorithm in `arnon` and leaves the counts
to a `LimitCounter` you supply. The default is in-memory, so it is correct for
a single instance only; anything with more than one replica needs shared
storage.

**`arnon`'s `LimitCounter` is the same interface as
[`go-chi/httprate`](https://github.com/go-chi/httprate)'s**, method for method
and on purpose. Go interfaces are structural, so an existing httprate backend
works here with no adapter:

```go
counter, err := httprateredis.NewRedisLimitCounter(&httprateredis.Config{
	Host: "localhost",
	Port: 6379,
})
if err != nil {
	return err
}

router.Use(middleware.RateLimit(middleware.RateLimitConfig{
	RequestLimit: 100,
	WindowLength: time.Minute,
	Counter:      counter,
}))
```

That covers Redis, and anything speaking its protocol (Valkey, for one), today.
It is also why this project ships no storage backend of its own: duplicating a
maintained package would add a dependency, a release to coordinate and a CVE
surface, for nothing a user gains. A compile-time assertion in
`httpx/middleware/limitcounter_compat_test.go` keeps the two interfaces from
drifting apart.

Writing your own works the same way in reverse — implement the four methods and
the result serves both projects.

## Skill for AI assistants

[skills/using-arnon/SKILL.md](skills/using-arnon/SKILL.md) documents,
in a format AI tools can load on demand (Claude Skill), how to use
`arnon` idiomatically: endpoint pattern, binding, validation, RFC 9457
errors, and middleware order. To use it in a project that depends on
`arnon`, copy the directory into it:

```sh
cp -r skills/using-arnon <your-project>/.claude/skills/using-arnon
```

## Development

```sh
make help              # lists every command
make test              # tests
make test-race         # tests with the race detector
make coverage          # generates coverage.html
make lint              # golangci-lint (v2, version pinned)
make arch-lint         # validates the package dependency graph
make test-mutation     # mutation testing (gremlins), writes mutation.json
make check             # lint + arch-lint + tests with race (minimum expected before a PR)
```

Tool versions (`golangci-lint`, `go-arch-lint`, `gremlins`) are pinned
in the `Makefile` via `go run pkg@version`, so there's no global
install needed and no dev-only dependencies polluting the module's
`go.mod`.

Style conventions are documented in
[docs/coding-style.md](docs/coding-style.md); branch workflow and
commit convention (Conventional Commits) in
[CONTRIBUTING.md](CONTRIBUTING.md).

Change history in [CHANGELOG.md](CHANGELOG.md). To report a
vulnerability, follow [SECURITY.md](SECURITY.md) (don't open a public
issue).
