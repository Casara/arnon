# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
this project follows [Semantic Versioning](https://semver.org/) once the
first tag is cut.

## [Unreleased]

Initial public release.

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
  (including `dive`-validated collections).
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
  `AllowContentType`, `MaxBodyBytes`, `NoCache`.
- Custom validation rules through a single registry
  (`validation.RegisterCustomRule`) that feeds runtime validation,
  error mapping and OpenAPI schema generation together.
- OpenTelemetry integration (`observability`, `observability/otel`):
  automatic HTTP tracing/metrics, custom counters/histograms,
  trace-correlated logging.
- Interactive API docs via Stoplight Elements (`openapi.NewDocsHandler`).
- Three runnable examples: `examples/cmd/basic`, `examples/cmd/middleware`,
  `examples/cmd/observability`.
- A Claude Skill (`skills/using-arnon`) documenting how to consume
  arnon idiomatically in a project that depends on it.

[Unreleased]: https://github.com/Casara/arnon/commits/main
