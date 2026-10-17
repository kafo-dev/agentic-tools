// Command agentic-web-search searches the web through Exa's keyless MCP tier
// and prints one URL and one-line summary per result.
package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"agents.kafo.dev/tools/internal/skill"
	"agents.kafo.dev/tools/internal/websearch"
	"agents.kafo.dev/tools/pkg/exa"
)

// Exit codes shared by every agentic command.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

const (
	// command names the binary in diagnostics and usage.
	command = "agentic-web-search"
	// searchTimeout bounds the whole command, crawl time included.
	searchTimeout = 64 * time.Second
	// envEndpoint overrides the MCP endpoint.
	envEndpoint = "EXA_MCP_URL"
)

// skillTemplate is the skill document, with {{usage}} where the synopsis goes.
//
//go:embed skill.md
var skillTemplate string

func main() {
	os.Exit(do(os.Args[1:], os.Getenv, pipedStdin(), os.Stdout, os.Stderr))
}

// do executes the command and returns the process exit code. Every dependency
// arrives as an argument, so a test calls it without a process.
func do(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	out, errOut := &output{Writer: stdout}, &output{Writer: stderr}

	opts, err := parse(args, stdin)
	switch {
	case errors.Is(err, flag.ErrHelp):
		out.write(usage())
		return exitOK
	case err != nil:
		errOut.write(command + ": " + err.Error() + "\n")
		errOut.write(usage())
		return exitUsage
	}

	if opts.skill {
		out.write(skill.Render(usage, skillTemplate))
		return exitOf(out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()

	client := &exa.Client{Endpoint: getenv(envEndpoint)}
	request := websearch.Options{Query: opts.query, Results: opts.results, JSON: opts.json}
	if searchErr := websearch.Run(ctx, client, request, out); searchErr != nil {
		errOut.write(command + ": " + searchErr.Error() + "\n")
		return exitFail
	}
	return exitOf(out)
}

// output collects write errors, so each write site stays a single statement.
type output struct {
	io.Writer

	err error
}

// write writes s unless an earlier write already failed.
func (o *output) write(s string) {
	if o.err != nil {
		return
	}
	_, o.err = io.WriteString(o.Writer, s)
}

// exitOf reports a failed write, which is all that can be said about it.
func exitOf(out *output) int {
	if out.err != nil {
		return exitFail
	}
	return exitOK
}
