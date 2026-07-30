# httpx/precondition

Loaded on demand, when files in this directory are read. The repo-wide rules
live in the root `AGENTS.md`.

* **`Check`/`CheckRequest` validate RFC 9110 §13.1.1/§13.1.4 write
  preconditions (`If-Match`/`If-Unmodified-Since`) against a resource's
  *current* state, supplied by the caller — not computed by the package.**
  Unlike `middleware.ETag` (which buffers a `GET`/`HEAD` response and handles
  `If-None-Match` entirely on its own, since the response bytes are all it
  needs), a write precondition can't be evaluated generically: only the
  resource layer — whatever loads the current row to apply a
  `PUT`/`PATCH`/`DELETE` — knows the resource's `ETag`/modification time at the
  moment of the write. This mirrors huma's `conditional` package exactly (its
  `PreconditionFailed(etag, modified)` is likewise called by the handler, not
  the framework), confirmed by reading huma's design before implementing this.
* **Two entry points because `HandlerFunc` never sees `*http.Request`.**
  `Check` takes header values already bound onto a request DTO via
  `header:"If-Match"`/`header:"If-Unmodified-Since"` tags; `CheckRequest` is
  the `*http.Request`-based equivalent, for a plain `http.Handler` or
  framework-internal code (`httpx/patch.From`).
* **`If-Match` uses strong comparison** (RFC 9110 §13.1.1) — a weak `ETag`
  (`W/"..."`) can never satisfy it, unlike `middleware.ETag`'s weak comparison
  for `If-None-Match` — and takes precedence over `If-Unmodified-Since` when
  both are present.
* **`Config.Require` (default `false`) opts into `428 Precondition Required`**
  when neither header is present. It's bundled with `If-Match` rather than
  shipped separately because one only makes sense alongside the other.
