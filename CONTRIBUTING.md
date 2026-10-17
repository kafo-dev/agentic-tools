# Contributing

## Naming

Every binary carries the `agentic-` prefix and names what it wraps or the job it does.

## Layout

- `cmd/agentic-<name>/main.go` — the `main` package, holding little beyond the call into `internal/`.
- `internal/<name>/` — flag parsing, process handling, and rendering for that command. The compiler keeps it inside this module.
- `pkg/<name>/` — code a third party may import: shared output helpers, protocol handling, anything reusable outside this module.

Shared plumbing belongs in `internal/cli`; shared output helpers belong in `pkg/agentio`. Reach for either before copying code between two commands.

## The Output Contract

An agent reads standard output, cannot answer a prompt, and pays per token. Every command holds to these rules:

- Results go to standard output, diagnostics to standard error.
- Plain text by default, one line of JSON under `-json`.
- Exit 0 on success, 1 on failure, 2 on a malformed invocation.
- Mirror the wrapped tool's status code when that code carries meaning.

## Checks

`scripts/check` runs every check, and the `pre-commit` hook runs the same script. Run it before you commit. `scripts/check-go` and its siblings each run one check on their own.

Commit subjects read `<scope>: <description>`, with the scope naming the area touched: `cli: reject an empty command line`.
