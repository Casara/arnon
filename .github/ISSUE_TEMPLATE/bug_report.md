---
name: Bug report
about: Something in arnon isn't behaving as documented
title: ''
labels: bug
assignees: ''
---

**Version**
`arnon` version/commit, `go version`.

**What happened**
A clear description of the behavior you saw.

**What you expected**
What you expected instead, ideally pointing at the doc/RFC that says so
(`docs/architecture/project-context.md`, `docs/architecture/rfc-compliance.md`)
if applicable.

**Minimal reproduction**
The smallest `httpx.Endpoint`/router/middleware setup that reproduces
it. A failing test case or `.hurl` request is even better than prose.

```go
// minimal repro here
```

**Additional context**
Logs, stack trace, anything else relevant.
