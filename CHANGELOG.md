# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
the versioning follows [Semantic Versioning](https://semver.org/). While the
project is on 0.x, a minor release may break the public API — see the API
stability section in the README.

## [0.1.0] - 2026-07-30

First public release. Everything below is new: there is no previous version to
compare against.

### Added

- Typed HTTP endpoints (`httpx.Endpoint`): request/response binding
  (path, query, header, JSON body), validation and error mapping in
  one call, no hand-written `http.Handler` boilerplate.
- RFC 9457 (Problem Details) as the framework's single error format,
  including a validation-error extension (`errors`/`source`) that
  addresses into struct, slice/array and map bodies with RFC 6901
  (JSON Pointer).
- OpenAPI 3.2.0 document generation via reflection, opt-in per
  endpoint, with schema inference from `validate` struct tags
  (including `dive`-validated collections). `openapi.WithSpecVersion`
  emits 3.1.1 instead, for tooling that has not caught up; the document
  is otherwise identical, since nothing generated uses a 3.2-only
  construct. A route stays documented when its handler is wrapped in
  this project's middleware, which builds its result with
  `routing.Wrap` so the router can find the endpoint underneath.
- A router (`httpx/routing`) built directly on `net/http.ServeMux`,
  with route groups and per-route/per-group middleware.
- Guaranteed middleware ordering via `middleware.BuildChain`, with
  named anchor points (`ChainConfig.Extra`) to insert custom
  middleware relative to the built-in stack.
- A standard middleware suite: `Recover`, `Timeout`,
  `StripSlashes`/`RedirectSlashes`, `RealIP` (RFC 7239 `Forwarded`),
  `RequestID`, `SecureHeaders`, `RateLimit` (sliding window, pluggable
  storage via `LimitCounter`), `Throttle`, `ETag`/conditional GET
  (RFC 9111), `Compress`, `CORS`, `ServiceDesc` (RFC 8631), `Logging`,
  `AllowContentType`, `MaxBodyBytes`, `NoCache`. `ETag`/`Compress` both
  step aside for a request carrying a `Range` header, so either is
  safe to combine with a Range-capable handler (`http.FileServer`,
  `http.ServeContent`) without breaking its Range/conditional-GET
  support.
- Custom validation rules through a single registry
  (`validation.RegisterCustomRule`) that feeds runtime validation,
  error mapping and OpenAPI schema generation together.
- Struct-tag-driven request sanitization (`sanitize`), applied before
  validation runs (`trim`/`email` built in, `sanitize.RegisterFunc` for
  custom transforms, recursive into nested structs/slices/maps).
- OpenTelemetry integration (`observability`, `observability/otel`):
  automatic HTTP tracing/metrics, custom counters/histograms,
  trace-correlated logging.
- Interactive API docs via Stoplight Elements (`openapi.NewDocsHandler`).
- `PATCH` derived from an existing `GET`+`PUT` pair (`httpx/patch.From`):
  RFC 7386 (JSON Merge Patch) and RFC 6902 (JSON Patch), selected by
  the incoming request's Content-Type, via internal request replay -
  neither the `GET` nor the `PUT` handler needs to change. Setting
  `patch.Config.OpenAPI` documents the derived route, taking its
  schemas from the `PUT` endpoint so the two cannot disagree.
- Write preconditions (`httpx/precondition.Check`/`CheckRequest`): RFC
  9110 §13.1.1/§13.1.4 `If-Match`/`If-Unmodified-Since`, with opt-in
  `428 Precondition Required` (`Config.Require`). `httpx/patch.From`
  uses this to reject a `PATCH` whose `If-Match` no longer matches the
  resource, before ever applying the patch.
- Six runnable examples: `examples/cmd/basic`, `examples/cmd/middleware`,
  `examples/cmd/observability`, `examples/cmd/files` (non-JSON content:
  file download/upload as plain `http.Handler`s), `examples/cmd/staticfiles`
  (serving static assets via `http.FileServer`, with `ETag`/`Compress`
  applied globally alongside it), `examples/cmd/patch` (derived `PATCH`).
- A Claude Skill (`skills/using-arnon`) documenting how to consume
  arnon idiomatically in a project that depends on it.
- `docs/architecture/adr-0001-module-layout.md`, the project's first
  architecture decision record.
- 26 runnable `Example` functions across `problem`, `sanitize`,
  `validation`, `httpx`, `httpx/routing`, `httpx/middleware`,
  `httpx/patch` and `httpx/precondition`. They render on pkg.go.dev, and
  each asserts its own output, so `make test` fails if a status code,
  header or JSON shape ever stops matching what the docs show.
- `make verify-docs`, which compiles every self-contained `package main`
  program embedded in the project's Markdown against the working tree,
  and reports how many illustrative fragments it could not verify.
  Wired into `make check`, so CI runs it on every PR.
- `make vuln`, running `govulncheck` across all three modules, plus a
  `govulncheck` job in CI. It reports only vulnerabilities reachable
  from this code, so a finding is worth acting on rather than triaging.
- `.github/dependabot.yml`, watching all three modules and the GitHub
  Actions workflow. Dev tools pinned in the `Makefile`
  (golangci-lint, go-arch-lint, gremlins, markdownlint-cli2) are still
  updated by hand - Dependabot cannot see them.
- A stated API stability policy for v0.x in both READMEs: what counts as
  the public surface, how a breaking change is communicated, and what
  reaching v1.0.0 depends on.
- `AGENTS.md` plus per-package `CLAUDE.md` files, so coding agents and
  human contributors read the same conventions, and
  `docs/architecture/adr-0001-module-layout.md` recording why the
  repository is three modules and the release ordering that imposes.

[0.1.0]: https://github.com/casara/arnon/releases/tag/v0.1.0
