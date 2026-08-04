# examples

Loaded on demand, when files in this directory are read. The repo-wide rules
live in the root `AGENTS.md`.

`examples/` is its own Go module — see
`docs/architecture/adr-0001-module-layout.md` for why (it imports
`observability/otel`, which imports the root module; keeping the examples in
the root module would make the two require each other).

* **`cmd/`+`internal/` pattern.** Each runnable example is `examples/cmd/<name>`:
  `basic` (typed endpoint + validation + OpenAPI, zero middleware),
  `middleware` (the same endpoint with the full stack), `observability` (the
  same endpoint with tracing/metrics via OpenTelemetry), `files`
  (download/upload as plain `http.Handler`s), `staticfiles` (`http.FileServer`
  with `ETag`/`Compress` applied globally), `patch` (`PATCH` derived from
  `GET`+`PUT` via `httpx/patch.From`), `custom-validator` (go-playground/
  validator swapped for ozzo-validation via `EndpointConfig.Validator`),
  `custom-errors` (errors answered outside RFC 9457, by reusing
  `binding`/`sanitize`/`validation` directly instead of `httpx.Endpoint` -
  see that example's own doc comment for why this one isn't a config field).
  Shared code — logger, custom validator registration, the sample handler —
  lives in `examples/internal/*`, not importable from outside `examples/` per
  Go's own rule. That rule is also why `examples` lists **itself** in its own
  `mayDependOn` in `.go-arch-lint.yml`: without it, the `cmd/* ->
  internal/*` cross-import is blocked even though both sides are the same
  component. New example: create `examples/cmd/<name>`, reuse what's in
  `examples/internal`, duplicate only what's specific to it.
* **A vendor import new to `examples/` needs an entry in `.go-arch-lint.yml`.**
  `custom-validator` and `ozzovalidator` needed
  `vendors.ozzo-validation` plus `examples.canUse` — `examples` may freely use
  any internal component (see the comment on the `examples` entry under
  `components`), but an external module still has to be declared, the same as
  `validator` (go-playground) already was.
* **`cmd/observability` needs graceful shutdown to make sense.** It's the only
  example handling `SIGINT`/`SIGTERM` explicitly (`signal.NotifyContext` +
  `server.Shutdown` + the shutdown function `otel.Initialize` returns).
  Without it, spans and metrics still sitting in the SDK's buffers — the trace
  batch processor, the metric periodic reader — are lost when the process dies.
  `otel.Initialize` only returns an OTLP/gRPC exporter (no stdout option), so
  the example spins up a local OTel Collector via `docker compose` with the
  `debug` exporter.
* Each example is tested end-to-end with `requests.hurl` against a real running
  server, not with `go test` — which is why `examples` is excluded from
  `LIB_MODULES` in the `Makefile`.
