// Command agentic-web-fetch reads a URL as markdown for an agent: it asks the
// site for a form meant for agents, follows the llms.txt conventions, and
// falls back to the readable part of the HTML.
package main

import (
	_ "embed"
	"errors"
	"flag"
	"io"
	"os"

	"agents.kafo.dev/tools/internal/skill"
	"agents.kafo.dev/tools/pkg/webfetch"
)

// Exit codes shared by every agentic command.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// command names the binary in diagnostics and usage.
const command = "agentic-web-fetch"

// skillTemplate is the skill document, with {{usage}} where the synopsis goes.
//
//go:embed skill.md
var skillTemplate string

func main() {
	os.Exit(do(os.Args[1:], pipedStdin(), os.Stdout, os.Stderr))
}

// do executes the command and returns the process exit code. Every dependency
// arrives as an argument, so a test calls it without a process.
func do(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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

	fetcher := webfetch.New(webfetch.Options{Timeout: opts.timeout})
	result, err := webfetch.Read(fetcher, siteRules(), opts.target)
	if err != nil {
		errOut.write(command + ": " + err.Error() + "\n")
		return exitFail
	}
	if renderErr := render(stdout, result, opts.json); renderErr != nil {
		errOut.write(command + ": " + renderErr.Error() + "\n")
		return exitFail
	}
	return exitOf(out)
}

// siteRules returns the site rules the command reads with. The registry holds
// no rule: every page goes through the generic pass. A site that needs its own
// selector, its own furniture list, or a mirror takes a [webfetch.Rule] here.
func siteRules() *webfetch.Registry {
	registry := webfetch.NewRegistry()
	return registry
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
