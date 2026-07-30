#!/usr/bin/env bash
#
# Reports the documentation a change to the public API has left behind.
#
# `make check` proves the code is correct and `verify-docs` compiles the
# snippets that can be compiled, but neither can tell that examples/cmd/basic
# still demonstrates a pattern that changed, or that SKILL.md describes a
# signature that no longer exists. Those are fragments and prose; nothing
# compiles them. This is the residue, and this script names it.
#
# It is a reminder, not a gate: it always exits 0. A diff can legitimately
# touch an exported symbol without any of these needing an edit - a doc comment
# fix, for instance - and a check that cried wolf would be turned off within a
# week.
#
# Compares against HEAD by default, or against the ref given as $1.
set -euo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

base="${1:-HEAD}"

changed=$(git diff --name-only "$base" -- '*.go' ':!*_test.go' ':!examples/*')

if [ -z "$changed" ]; then
    exit 0
fi

# A line added or removed at the start of an exported declaration: a top-level
# func/type/const/var, or an exported struct field (one tab, capital letter).
# Deliberately a heuristic over the diff rather than a real API diff - the
# point is to prompt a human, and `go doc` on two worktrees would cost more
# than the reminder is worth.
surface_changed=$(
    git diff -U0 "$base" -- '*.go' ':!*_test.go' ':!examples/*' |
        grep -E '^[+-](func|type|const|var) [A-Z]|^[+-]\t[A-Z][A-Za-z0-9_]* ' |
        grep -vE '^[+-](func|type|const|var) [A-Z][A-Za-z0-9_]*Test' |
        head -1 ||
        true
)

if [ -z "$surface_changed" ]; then
    exit 0
fi

# Everything the doc-sync list in AGENTS.md names, plus the reason each one
# cannot be checked automatically.
declare -a targets=(
    "skills/using-arnon/SKILL.md|its snippets are fragments, so nothing compiles them"
    "examples/cmd|a change that still compiles can leave an example on a stale pattern"
    "README.md|the prose around the Quick start, which verify-docs does not read"
    "README.pt-BR.md|same, and it has to stay in sync with README.md"
    "docs/architecture/project-context.md|the behavior description"
    "docs/architecture/project-context.pt-BR.md|the pt-BR twin"
    "AGENTS.md|if an invariant changed"
    "CHANGELOG.md|every public change belongs here"
)

stale=()

for entry in "${targets[@]}"; do
    path="${entry%%|*}"
    reason="${entry#*|}"

    if ! git diff --quiet "$base" -- "$path" 2>/dev/null; then
        continue
    fi

    stale+=("  $path
      $reason")
done

if [ ${#stale[@]} -eq 0 ]; then
    exit 0
fi

printf '\nThis diff changes the exported API. Untouched documentation:\n\n'
printf '%s\n' "${stale[@]}"
printf '\nPer-package CLAUDE.md files are not listed - check the one for the package
you changed. If none of the above needs an edit, nothing to do.\n\n'
