# httpx/patch

Loaded on demand, when files in this directory are read. The repo-wide rules
live in the root `AGENTS.md`.

* **`From` derives `PATCH` from `GET`+`PUT` by request replay, not by teaching
  `httpx.Endpoint` anything about partial updates.** It calls the given
  `get`/`put` `http.Handler`s **directly, never through a `Router`** — replaying
  through `Router.ServeHTTP` would re-run global middleware (`RequestID`,
  `RateLimit`, `Logging`, ...) a second time for the synthetic request.
  `request.Clone` is what makes this safe: confirmed against Go's stdlib source
  that `Clone` explicitly copies the internal `matches`/`otherValues` fields
  that `PathValue` reads (fixed for exactly this reuse-across-calls scenario,
  issue 61410), so the internal `GET`/`PUT` still resolve path parameters with
  no extra wiring.
* **No auto-discovery of the `GET`/`PUT` pair, unlike huma's
  `autopatch.AutoPatch(api)`.** `openapi.Generator`/`Registry` is write-only
  (`Register`/`RegisterTypes`, no lookup) and only tracks endpoints that opted
  into OpenAPI, so repurposing it as a routing registry would be a layering
  violation. Explicit `get`/`put` arguments avoid that and match the
  framework's "explicit over magic" pattern (`ChainConfig.Extra` over
  auto-ordering; `CORS` not intercepting every `OPTIONS`).
* **Preconditions are checked in two distinct places, and only one of them is
  `From`'s job.** `From` checks the *incoming* `PATCH` request's own
  `If-Match`/`If-Unmodified-Since` — the client's optimistic-concurrency
  intent, from an earlier `GET` — against the internal `GET`'s
  `ETag`/`Last-Modified` via `httpx/precondition.CheckRequest`, before applying
  the patch or calling `put`; a mismatch is `412`. It separately copies the
  internal `GET` response's `ETag`/`Last-Modified` onto the internal `PUT`'s
  `If-Match`/`If-Unmodified-Since` (mirroring huma), but that second
  propagation only closes the narrower internal `GET`-to-`PUT` race **if `put`
  itself** calls `precondition.Check`/`CheckRequest` with its own
  atomically-read current state. `From` cannot do that part: it doesn't know
  what "current state" means for an arbitrary resource.
