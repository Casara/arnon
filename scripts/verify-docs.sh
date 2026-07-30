#!/usr/bin/env bash
#
# Compiles every self-contained Go program embedded in the project's Markdown,
# against the arnon in this working tree.
#
# Why this exists: `make check` proves the code is correct, not that the
# documentation still describes it. A README snippet that stopped compiling
# three commits ago looks exactly like one that still works. This closes that
# gap for the blocks that *can* be closed.
#
# What it verifies: every ```go block whose first line is `package main`. Those
# are complete programs (today: the Quick start in README.md and
# README.pt-BR.md).
#
# What it deliberately does NOT verify: illustrative fragments - a bare type
# declaration, a statement, a struct literal with a literal `...` inside. Those
# are pedagogically right as fragments and can never compile standalone. They
# are counted and reported so the unverified surface stays visible instead of
# being silently assumed correct.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

compiled=0
skipped=0
failed=0

# Markdown files to scan. NOTES.md is gitignored and never lint-clean by design.
mapfile -t markdown_files < <(
    git ls-files '*.md' | sort
)

for markdown_file in "${markdown_files[@]}"; do
    # Split the file into ```go blocks, one numbered file per block.
    block_dir="$work_dir/$(echo "$markdown_file" | tr '/' '_')"
    mkdir -p "$block_dir"

    awk -v out="$block_dir" '
        /^```go$/ { inblock = 1; n++; next }
        /^```$/   { inblock = 0; next }
        inblock   { print > (out "/" n ".go") }
    ' "$markdown_file"

    shopt -s nullglob
    for block in "$block_dir"/*.go; do
        if [ "$(head -n 1 "$block")" != "package main" ]; then
            skipped=$((skipped + 1))
            continue
        fi

        program_dir="$block.d"
        mkdir -p "$program_dir"
        mv "$block" "$program_dir/main.go"

        cat > "$program_dir/go.mod" <<EOF
module docsnippet

go 1.26

require github.com/casara/arnon v0.0.0

replace github.com/casara/arnon => $repo_root
EOF

        block_name="$markdown_file block $(basename "$block" .go)"

        if (
            cd "$program_dir" &&
                GOFLAGS=-mod=mod GOWORK=off go mod tidy >/dev/null 2>&1 &&
                GOWORK=off go build ./... >/dev/null
        ); then
            compiled=$((compiled + 1))
            printf '  ok       %s\n' "$block_name"
        else
            failed=$((failed + 1))
            printf '  FAILED   %s\n' "$block_name"
            (cd "$program_dir" && GOWORK=off go build ./... 2>&1 | sed 's/^/           /')
        fi
    done
    shopt -u nullglob
done

printf '\n%d compiled, %d fragments not verifiable, %d failed\n' \
    "$compiled" "$skipped" "$failed"

if [ "$failed" -gt 0 ]; then
    exit 1
fi

if [ "$compiled" -eq 0 ]; then
    echo "no complete Go program found in the documentation - did the extraction break?" >&2
    exit 1
fi
