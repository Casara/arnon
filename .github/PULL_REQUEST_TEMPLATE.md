<!-- markdownlint-disable-next-line MD041 -- GitHub PR templates aren't rendered as a titled page, no H1 needed. -->
## What and why

<!-- What changes, and the motivation - link an issue if there is one. -->

## Checklist

- [ ] `make check` passes locally (lint + arch-lint + verify-docs +
      test-race, across all three modules).
- [ ] `make lint-md` passes, if any Markdown file changed.
- [ ] New/changed behavior has a test (`_test.go`, or a case in
      `examples/cmd/*/requests.hurl` if it's observable end-to-end
      over HTTP). For a new exported symbol, an `Example` is the
      cheapest way to cover it, and it renders on pkg.go.dev.
- [ ] Commit messages follow [CONTRIBUTING.md](../CONTRIBUTING.md)'s
      Conventional Commits convention.

### If an exported symbol changed

<!-- Delete this section if the change is internal only. -->

See the doc-sync list in [AGENTS.md](../AGENTS.md). `make check` already covers
the Quick start in both READMEs and every `Example`'s output; the rest is
manual:

- [ ] `skills/using-arnon/SKILL.md` — its snippets are fragments, so nothing
      compiles them.
- [ ] `examples/cmd/*` — a change that still compiles can leave an example
      demonstrating a stale pattern.
- [ ] The prose around the snippets in `README.md` / `README.pt-BR.md`.
- [ ] `docs/architecture/project-context.md` and its `.pt-BR` twin, plus
      `rfc-compliance.md` if documented RFC behavior changed.
- [ ] `AGENTS.md` and the per-package `CLAUDE.md`, if an invariant changed.
- [ ] `CHANGELOG.md` — and if this is a breaking change, read
      [Changing the public API](../CONTRIBUTING.md#changing-the-public-api).
