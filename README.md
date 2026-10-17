# `agentic-tools`

> **Warning: Large Language Model-Generated Code**
>
> This code functions as of the time of writing, yet it has received little to no human review. It must not be regarded as reliable: bugs and stability issues are to be expected.

Wrappers that reshape existing tools and services for an agent.

## Scope

A command in this module wraps something that already runs — a command-line program, an HTTP endpoint, a git repository — and re-emits the result for a caller that has no terminal, cannot answer a prompt, and pays per token. Every `agentic-*` command holds to the same contract:

- Plain text by default, one line of JSON under `-json`;
- A status code that separates success, failure, and a malformed invocation;

## Commands

- `agentic-web-search` — searches the web through Exa's keyless MCP tier and prints one URL and one-line summary per result. `-json` emits the same list as one line of JSON, and `-skill` prints the skill document that teaches an agent to use it.

### `agentic-web-search`

The command needs no account and no API key: it queries Exa's hosted MCP endpoint in keyless mode, inside a per-IP rate limit shared with every other Exa client behind the same address. The search terms arrive as arguments, or on standard input when the command line carries none:

```sh
agentic-web-search "official Go 1.27 release notes"
printf '%s' 'golang generics tutorial' | agentic-web-search
```

`EXA_MCP_URL` points the command at another MCP endpoint.

## Install

```sh
go install agents.kafo.dev/tools/cmd/...@latest
```

## License

A copy of the license is available [here](LICENSE.txt).

`agentic-tools` is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

`agentic-tools` is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with `agentic-tools`. If not, see <https://www.gnu.org/licenses/>.
