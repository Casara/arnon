<!-- markdownlint-disable-next-line MD041 -- The @import has to come first; the H1 lives in AGENTS.md. -->
@AGENTS.md

<!--
AGENTS.md holds the actual instructions, so that Codex, Cursor and anything
else following the AGENTS.md convention read the same file. Claude Code reads
CLAUDE.md and not AGENTS.md, so this import is what bridges the two; a symlink
would work too, but requires Developer Mode on Windows.

Keep Claude-specific content below the import, and everything tool-neutral in
AGENTS.md.
-->

## Claude Code

Package-scoped rules live in `CLAUDE.md` files next to the code
(`httpx/`, `httpx/middleware/`, `httpx/patch/`, `httpx/precondition/`,
`examples/`). They load on demand when you read files in those directories.

They are **not** re-injected after `/compact` — only this root file is. If a
compaction happens mid-task, re-read the relevant package file before relying on
its invariants.
