# Contributing

*[Leia em português](CONTRIBUTING.pt-BR.md)*

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md). By
participating, you're expected to follow it.

## Before you start

Read, in this order:

1. [CLAUDE.md](CLAUDE.md) — commands, package dependency graph,
   non-obvious design decisions.
2. [docs/architecture/project-context.md](docs/architecture/project-context.md) —
   the project's functional/architectural spec and current state.
3. [docs/coding-style.md](docs/coding-style.md) — code conventions.
4. [docs/architecture/rfc-compliance.md](docs/architecture/rfc-compliance.md) —
   if your change touches error format, content negotiation, caching,
   or anything RFC-adjacent, the behavior already documented there is
   what can't regress.

## Environment

`make help` lists every command. Before opening a PR, run:

```sh
make check     # lint + arch-lint + tests with race detector
make lint-md   # markdown lint (requires Node.js >= 20)
```

That's the minimum CI runs on every PR
([.github/workflows/ci.yml](.github/workflows/ci.yml)) — `check` and
markdown lint run as separate jobs. If the change touches behavior —
not just refactoring — include a test covering the
new case (unit test in `_test.go`, or a case in
`examples/cmd/*/requests.hurl` if it's something observable end-to-end
over HTTP), and, when it makes sense, update
`docs/architecture/project-context.md`/`rfc-compliance.md`.

## Branch workflow

Rebase, not merge commits: update your branch with `git rebase`
against the base before opening/updating a PR, instead of merging the
base into your branch. Linear history, no merge commits.

The commit convention below is mandatory on `main`. A work branch that
will be squashed on merge doesn't need to follow it strictly — the
intermediate history isn't what stays.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/), in
English, imperative mood:

```text
<type>(<scope>): <description>
```

* `<description>` in the imperative, describing the action (`add`,
  `fix`, `remove`, `harden`, `resolve`, `guarantee`), not the
  resulting state (`added`) or history (`fixed`, "adds/added").
* `<scope>` is the affected package or area. It's not a closed list,
  but the most common ones match the components in
  [.go-arch-lint.yml](.go-arch-lint.yml): `problem`, `validation`,
  `openapi`, `httpx`, `binding`, `routing`, `middleware`,
  `observability`, `otel`, `examples` — plus the cross-cutting
  `release`, `docs`, `ci`, `deps`.

### Types

| Type | When to use |
| --- | --- |
| `feat` | New feature (semver MINOR). |
| `fix` | Bug fix (semver PATCH). |
| `docs` | Documentation only (README, `docs/`, comments) — no code change. |
| `test` | Test only (added, changed, or removed) — no production code change. |
| `refactor` | Changes how code is written/organized without changing observable behavior. |
| `perf` | A change whose purpose is performance. |
| `style` | Formatting, semicolons, whitespace, lint — no code change. |
| `build` | Build and dependencies (`go.mod`, `Makefile`, tooling). |
| `ci` | Continuous integration (`.github/workflows/`). |
| `chore` | Maintenance tasks that don't fit the types above (config, `.gitignore`, ...). |
| `cleanup` | Removes commented-out, dead, or unnecessary code — no behavior change. |
| `remove` | Removes an obsolete/unused file, directory, or feature. |
| `raw` | Change to a configuration/data/parameter file that doesn't fit the types above. |

### Examples (from the project's own history)

```text
feat(validation): resolve RFC 6901 pointers through array/slice indices
fix(routing): complete splitPattern's method whitelist
docs(readme): explain why arnon exists before comparing frameworks
test(openapi): cover dive-redirected schema constraints
```

## Pull requests

* `make check` passing is mandatory, not optional. If the change
  touches any Markdown file, `make lint-md` too.
* A small PR focused on one change is preferable to a large PR
  covering several unrelated things — but that's judgment, not a
  strict rule (e.g. a bug fix found while testing a new feature can go
  in the same PR, if clearly documented in the PR/commit description).
* If the change alters documented behavior, update the docs in the
  same PR — it's not acceptable for a PR to leave
  `project-context.md`/`rfc-compliance.md` stale "for later."
