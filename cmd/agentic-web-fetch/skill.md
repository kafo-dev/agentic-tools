---
name: agentic-web-fetch
description: Use when you need a web page's text as Markdown.
---

# agentic-web-fetch

```text
{{usage}}
```

## Output

Standard output carries the content alone. `-json` prints one line in this
shape:

```json
{"url": "https://example.com/docs/guide", "method": "md-variant", "content": "# Guide\n\nText.\n"}
```

`url` holds the address the content came from, which differs from the requested
URL after a redirect or a negotiation. `content` holds the Markdown, with its
line breaks escaped onto the JSON line. `method` names the branch that produced
the content, tried in this order:

- `negotiated` — the site answered the request for Markdown with Markdown;
- `alternate` — a `rel="alternate" type="text/markdown"` link, in the page or in a `Link:` header;
- `md-variant` — the `.md` variant of the URL, the convention of [the llms.txt proposal](https://llmstxt.org/);
- `llms.txt` — an `llms.txt` file that covers the page, or an entry from one;
- `rule:<name>` — a site rule;
- `generic` — HTML with its furniture stripped.

A page the site's `robots.txt` file disallows fails, and exit code 1 reports
it.

## Examples

A page arrives as one chunk on stdout, so move a long one into a file and read
the part that matters:

```sh
agentic-web-fetch https://go.dev/doc/go1.27 > page.md
rg -n -i 'generic method' page.md
head -40 page.md
```

Fetch the page a search returns first:

```sh
agentic-web-search -json "Go 1.27 release notes" | jq -r '.[0].url' |
  xargs -I{} agentic-web-fetch {} > page.md
```

Read the content out of the JSON payload:

```sh
agentic-web-fetch -json "$url" | jq -r .content | wc -l
```
