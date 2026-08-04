# ADR 0001 — Module layout: three modules, not one

* **Status:** accepted
* **Date:** 2026-07-30
* **Applies to:** the first public release (v0.x)

## Context

arnon calls itself a minimalist HTTP foundation. Kept as a single module, its
`go.mod` would declare 12 direct requirements — among them two OTLP/gRPC
exporters, which drag in `google.golang.org/grpc`, `protobuf`, two `genproto`
modules and `grpc-gateway`. The question, settled here before the first tag,
was whether the OpenTelemetry wiring should live in its own module, since
module boundaries are the one thing a tag makes expensive to change.

The case for splitting is easy to overstate as "everyone who imports `problem`
pays for gRPC". **Measuring it first showed that is not what happens, and that
changed the decision's shape.**

### What was measured

Two throwaway consumer modules, each importing a different slice of arnon, then
`go mod tidy`:

| Consumer imports | `go list -m all` | indirect requires | `go.sum` content hashes |
| --- | --- | --- | --- |
| `problem` only | 34 modules | 0 | **none — no `go.sum` is created at all** |
| `httpx` + `httpx/routing` | 43 modules | 8 | 13 |

Module graph pruning (Go 1.17+) already prevented the concrete cost. A consumer
that imports only `problem` downloads nothing, compiles nothing extra, and
records no checksums. Even for the typical consumer, all 8 indirect requires
and all 13 content hashes came from the `go-playground/validator/v10` cluster —
**zero** from OpenTelemetry, gRPC or protobuf.

So the real cost of the single-module layout was never build time or binary
size. It was:

1. **Module-graph noise.** 33 third-party modules appear in `go list -m all`
   and `go mod graph` for every consumer regardless of what they import, which
   is also what vulnerability scanners and dependency dashboards read.
2. **MVS floor propagation.** arnon's chosen versions constrain a consumer's
   build list even for modules the consumer never links.
3. **Perception.** `go.mod` is the first file a Go reviewer opens, and two gRPC
   exporters in a "minimalist" framework is a visible contradiction.

Point 3 is not a rounding error for a project whose adoption depends on that
first impression, but it is a different — and smaller — claim than the one that
started the discussion.

### The cycle that forced a third module

Splitting only `observability/otel` does not work, because of the import graph:

```text
observability/otel/http_handler.go  ->  arnon/observability      (otel needs the root)
examples/cmd/observability/main.go  ->  arnon/observability/otel (the root needs otel)
```

With `examples/` in the root module, root and otel would require each other.
Before the first tag that is unresolvable cleanly: the root's `require` on the
otel module would need a published version that does not exist, and a committed
`replace` is ignored by consumers, so the published module would be broken for
anyone running `go get`.

What makes an acyclic layout possible is that **neither `observability` nor
`observability/otel` imports any other arnon package** — which
`.go-arch-lint.yml` already stated (`observability` has no `mayDependOn` at
all).

## Decision

Three modules:

| Module | Requires | Published |
| --- | --- | --- |
| `github.com/casara/arnon` | validator, uuid, json-patch, and the OTel **API** (`otel`, `metric`, `sdk/metric` for tests) | yes |
| `github.com/casara/arnon/observability/otel` | arnon + OTel SDK + OTLP exporters + contrib | yes |
| `github.com/casara/arnon/examples` | arnon + arnon/observability/otel + validator | **no** |

`observability` (the base package) stays in the root module: `httpx/middleware`
imports it from `recover.go` and `logging.go`, and it only uses the OTel API,
which is lightweight.

Two things were considered and rejected:

* **Splitting at `observability/` (base + otel together), to keep two modules.**
  `httpx/middleware` imports `observability`, so the root would require that
  module, and pruning would pull the SDK and exporters back into the graph of
  any consumer using middleware. It does not solve the problem.
* **Moving `PlaygroundValidator` out to drop `validator/v10` from the core.**
  This is the dependency a consumer actually pays for, so the saving is real —
  but it requires validation to become opt-in in `httpx.EndpointConfig`, which
  contradicts both "inference over configuration" and the framework's own
  headline (typed endpoints *with* validation). Not worth 8 indirect requires.
* **Splitting `httpx/patch`.** Removes exactly one module
  (`evanphx/json-patch/v5`). Does not pay for a fourth `go.mod`.

### Measured result

Same two probe consumers, measured against each candidate layout:

| Consumer imports | Layout | `go list -m all` | grpc/protobuf/genproto in graph |
| --- | --- | --- | --- |
| `problem` only | single module | 34 | yes |
| `problem` only | **three modules** | **20** | **no** |
| `httpx` + `routing` | single module | 43 | yes |
| `httpx` + `routing` | **three modules** | **30** | **no** |

The root module declares 6 direct requirements instead of the 12 a single-module layout would need.

## Consequences

### Tooling

Neither `go test ./...` nor `golangci-lint` crosses a module boundary, not even
inside a workspace, so `Makefile` targets iterate over `$(MODULES)` /
`$(LIB_MODULES)` instead of relying on `./...`.

`arch-lint` is the exception and needed **no** change: `go-arch-lint` resolves
components from the filesystem rather than the module graph, so one run from the
root still covers all three modules and `.go-arch-lint.yml` still describes the
whole package graph. This was verified by planting a deliberate
`observability/otel` → `problem` import and confirming it is reported.

`coverage` merges one profile per module, keeping a single `mode:` header.

### `go.work` is committed

`go.work` is not what makes the repository build — verified by removing it and
running `go build ./...` from `observability/otel` and `examples` standalone,
both succeed. For `examples/`, that stays true forever: it is never published,
so its `replace` directives are permanent. For `observability/otel`, it is only
true *before* the release checklist below runs — its own `replace` currently
points at the working tree, same as `examples/`'s. Once that `replace` is
deleted at release time, a standalone build of `observability/otel` resolves
`github.com/casara/arnon` from the last **published** tag, not from local
edits. That is the point at which `go.work` stops being a convenience and
starts being load-bearing for that module: it is what keeps local edits to the
root visible while developing `observability/otel` between releases, without
having to tag and republish just to see them.

Either way, it is not load-bearing for CI: `Makefile` targets `cd` into each
`$(MODULES)` entry explicitly rather than relying on the workspace (see
"Tooling" above). What it buys day to day is editor/tooling convenience — one
`go build`/`gopls` session across all three modules from the repo root, no
`cd`-ing into each one. Consumers never see it — `go.work` has no effect
outside the workspace root.

`gomoddirectives`' `replace-local` had to be enabled in `.golangci.yml` for the
same reason; the comment there records when it can be tightened again.

### Release ordering (this is the part that bites)

The submodule's dependency on the root cannot be satisfied until the root is
published, so releases are **ordered, not simultaneous**:

1. Tag the root module (`vX.Y.Z`) and **push the tag** — `go get` resolves
   against the remote, not the local repository, so an unpushed tag is
   invisible to the module proxy and to step 2.
2. In `observability/otel/go.mod`, replace
   `github.com/casara/arnon v0.0.0-00010101000000-000000000000` with the tag
   from step 1, and **delete the `replace` directive**. Then run `go mod tidy`
   in `observability/otel`: the local `replace` bypassed checksum verification
   entirely, so `observability/otel/go.sum` has **no entry at all** for
   `github.com/casara/arnon` yet (verified: `grep casara/arnon go.sum` is
   empty today) — without `tidy` regenerating it, the tagged module fails to
   build for anyone who fetches it.
3. Commit, then tag `observability/otel/vX.Y.Z` and push that tag too.

`examples/` is never published, so its `replace` directives are permanent.

Skipping step 2 (either half — the `replace` deletion or the `go mod tidy`) or
step 1's push publishes a module that cannot be fetched, and there is no
automated check for any of it. Worth a manual double-check before the first
tag, since this ordering has never been exercised end-to-end yet.

### When to revisit

Reopen this decision if any of the following becomes true:

* `observability/otel` needs to depend on something in the root beyond
  `observability`, which would make the split leakier than it is worth.
* The `validator/v10` cluster becomes the dominant complaint from consumers,
  which would reopen the rejected option above on new evidence.
* Go changes module graph pruning such that the noise in `go list -m all`
  disappears on its own, which would remove most of what is left of the
  rationale.
