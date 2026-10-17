---
name: agentic-web-search
description: Use when you need to do a web search.
---

# agentic-web-search

```text
{{usage}}
```

## JSON

Under `-json` the command prints one line in this shape:

```json
[{"url": "https://go.dev/doc/go1.27", "summary": "Generic methods land."}]
```

`url` is the page and `summary` its one-line description, empty when the page offers none. A search with no hits prints `[]`.

## Examples

Describe the page you want rather than the keywords: `agentic-web-search "official Go 1.27 release notes"` reaches the Go site, while `agentic-web-search go 1.27` returns blog posts and mirrors.

Read the URLs alone:

```sh
agentic-web-search -n 10 "golang generics tutorial" | cut -f1
```

Fetch the first hit:

```sh
agentic-web-search -json "official Go 1.27 release notes" | jq -r '.[].url' | head -n 1 | xargs curl -fsSL
```

Search a question held in a file:

```sh
cat question.md | agentic-web-search
```
