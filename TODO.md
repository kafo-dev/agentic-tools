# TODO

## `agentic-web-search`

- **Line filters against `-json`.** Under `-json` the command emits one line of JSON, so `| head -n 1` returns the entire payload rather than the first hit, and `| head -n 5` cuts the JSON mid-document. Revisit the skill document to teach the two shapes side by side: `head` against the plain-text mode, which holds one hit per line, and `jq` against the JSON mode, where `.[0].url` selects the first hit. State that the JSON mode never splits a hit across lines.

- **Repeated identical queries.** An agent reads a result list, then runs the same query again to pipe it into `cut` or `jq`, spending a second request from the per-IP Exa rate limit for an answer already in hand. Cache the response instead: key the entry on the arguments and the endpoint, write it under the system temporary directory, and expire it after 64 minutes. The temporary directory inside an agent sandbox clears on restart, so a cache in that location covers one session at most.

## `agentic-web-fetch`

- **Repeated fetches of one address.** An agent fetches a page, reads one part, then fetches it again for another. Each run repeats every request the page needed: the page, its Markdown alternate, the `.md` variant, and one `llms.txt` per directory up to the root. Cache the result — the address, the branch, and the content — keyed on the address and on the version of the command, since another build carries another set of site rules. Write the entry under the system temporary directory and expire it after 64 minutes. The temporary directory inside an agent sandbox clears on restart, so a cache in that location covers one session at most.
