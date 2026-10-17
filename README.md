# `agentic-tools`

> **Warning: Large Language Model-Generated Code**
>
> This code functions as of the time of writing, yet it has received little to no human review. It must not be regarded as reliable: bugs and stability issues are to be expected.

Wrappers that reshape existing tools and services for an agent.

## Scope

A command in this module wraps something that already runs — a command-line program, an HTTP endpoint, a git repository — and re-emits the result for a caller that has no terminal, cannot answer a prompt, and pays per token. Every `agentic-*` command holds to the same contract:

- Plain text by default, one line of JSON under `-json`;
- A status code that separates success, failure, and a malformed invocation;

## Install

```sh
go install agents.kafo.dev/tools/cmd/...@latest
```

## License

A copy of the license is available [here](LICENSE.txt).

`agentic-tools` is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

`agentic-tools` is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with `agentic-tools`. If not, see <https://www.gnu.org/licenses/>.
