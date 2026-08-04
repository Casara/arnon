# Contributing

*[Leia em português](CONTRIBUTING.pt-BR.md)*

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md). By
participating, you're expected to follow it.

## Before you start

Read, in this order:

1. [AGENTS.md](AGENTS.md) — commands, package dependency graph,
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
make check     # lint + arch-lint + verify-docs + tests with race detector
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

## Changing the public API

The project is on **v0.x**, so a breaking change is allowed — see
[API stability](README.md#api-stability) for what that means for users. It
still has to be deliberate and visible:

* Say so in the PR description, and add a `### Changed` or `### Removed`
  entry to `CHANGELOG.md` with the migration in the same entry.
* Where a rename can keep the old spelling compiling, keep it as a
  deprecated alias rather than deleting it outright.
* Prefer an additive change when one exists: a new field on a config struct,
  a variadic option, a new constructor alongside the old one.
* Everything in the doc-sync list of
  [AGENTS.md](AGENTS.md#documentation-synchronization) applies. An
  `Example` function is the cheapest way to prove the new shape works —
  `make check` runs them.

Error strings and log messages are explicitly *not* part of the public API;
matching on their text is not supported.

## How a change gets made

The history of this repository follows one loop, and it is worth stating
because it is not visible from the code: **explore, plan, implement, commit.**
Reading `project-context.md` and the package's own `CLAUDE.md` before touching
anything is the "explore" step, and it is where most of the cost is avoided —
several invariants here exist because a previous attempt got them wrong.

Two habits matter more than the loop itself:

* **Land a feature, its example and its documentation together.** The history
  shows this as triples: the package change, then `examples/cmd/*`, then the
  docs. A PR that leaves the third for later is the one that goes stale.
* **Prefer an `Example` over theory.** `make check` compiles and runs every
  `Example`, comparing its `// Output:` block, so it is the only documentation
  the toolchain can keep honest. Writing the ones in this repository caught
  three errors that had been sitting in the docs, compiling fine.

## Review

This project has a single maintainer, so "review" is not a gate a second
person opens — it is what the checks and the diff have to make obvious on
their own.

* **CI has to be green before merge.** `check` (across all three modules),
  `govulncheck` and `markdown lint`. A red run is not merged and then fixed.
* **The maintainer merges**, squashing the branch so `main` keeps one commit
  per change. That is why an intermediate commit on a work branch does not
  have to follow the convention strictly, while the squashed message does.
* **A change to an exported symbol gets read against the doc-sync list**
  in [AGENTS.md](AGENTS.md#documentation-synchronization), not just against
  the tests. `make doc-sync`
  prints what a diff touching the public surface has left behind.

## Contributing with an AI assistant

This project is built with one, and there is nothing to hide or disclose about
that: the code is judged the same either way, and a PR is not marked.

What is asked is the same thing asked of anyone:

* **Understand what you are submitting.** If you cannot explain why a change is
  correct, it is not ready — regardless of what wrote it.
* **Do not let a tool re-litigate a settled decision.** The `AGENTS.md` and the
  per-package `CLAUDE.md` files exist because several of these decisions were
  made once, reverted, and made again. A PR reintroducing one gets closed with
  a pointer, not a debate.
* **Run the checks locally.** `make check` and `make lint-md`, before opening
  the PR rather than after CI says so.

`AGENTS.md` is read by Codex and Cursor directly; `CLAUDE.md` imports it for
Claude Code. Anything you add for one of them belongs in `AGENTS.md`, so the
others get it too.

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
