# Contributing

## Naming

Every binary carries the `agentic-` prefix and names what it wraps or the job it does.

## Layout

- `cmd/agentic-<name>/` — the `main` package. It carries the shape of the command line: the flags, the positional arguments, and the parsing of both.
- `cmd/agentic-<name>/skill.md` — the skill document, with `{{usage}}` standing where the usage line goes.
- `internal/<name>/` — the implementation: process handling, rendering, and the work the command does. The compiler keeps it inside this module.
- `pkg/<name>/` — the implementation a third party may import: shared output helpers, protocol handling, anything reusable outside this module.

Shared plumbing belongs in `internal/cli`; shared output helpers belong in `pkg/agentio`. Reach for either before copying code between two commands.

## The Command Shape

A command puts its work in `do`, so a test calls it without a process:

```go
func do(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int
```

`main` passes `os.Args[1:]`, `os.Getenv`, the piped standard input, and the two streams, then exits with the code `do` returns.

The help text comes from [`internal/cli`](internal/cli/usage.go):

```go
flags.Usage = cli.Usage(command, flags, notes)
```

`Usage(name, flags, notes)` returns the renderer that `flags.Usage` takes. It prints what the flag package prints, a blank line, then `notes`, which describes the parts of the command line the flags leave out: the positional arguments and standard input. The help text documents the command on its own, and `-skill` carries the same text. Setup that only a person needs, an environment override for instance, belongs in the README instead.

## The Output Contract

An agent reads standard output, cannot answer a prompt, and pays per token. Every command holds to these rules:

- Results go to standard output, diagnostics to standard error.
- Plain text by default, one line of JSON under `-json`.
- Exit 0 on success, 1 on failure, 2 on a malformed invocation.
- Mirror the wrapped tool's status code when that code carries meaning.

## Skill Files

A command an agent has to discover ships a skill document, printed by `-skill`. The document sits beside the command at `cmd/agentic-<name>/skill.md`, copied from [`cmd/skill.md.tmpl`](cmd/skill.md.tmpl), which holds the shape and one comment per section.

The `description` line opens with `Use when` and names the situation alone: `Use when you need to do a web search.` It never describes the command, its internals, or the way it works.

`{{usage}}` marks the help text. [`internal/skill`](internal/skill/skill.go) fills it with what the command's own usage function renders, so the document, the flags, and the notes never drift:

```go
//go:embed skill.md
var skillTemplate string

document := skill.Render(usage, skillTemplate)
```

`Render(usage, template)` calls `usage` and returns `template` with the result where `{{usage}}` stood, leaving every other byte alone.

The prose lints in `scripts/check-markdown` read a skill document as the author's own prose, so `skip_path` in that script covers every skill file.

## Checks

`scripts/check` runs every check, and the `pre-commit` hook runs the same script. Run it before you commit. `scripts/check-go` and its siblings each run one check on their own.

Commit subjects read `<scope>: <description>`, with the scope naming the area touched: `cli: reject an empty command line`.
