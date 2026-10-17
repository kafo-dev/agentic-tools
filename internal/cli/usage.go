// Package cli holds the plumbing every agentic command shares.
package cli

import (
	"flag"
	"fmt"
	"strings"
)

// Usage returns the renderer for a command's help text, in the shape
// [flag.FlagSet.Usage] expects: the heading and the flag list the flag package
// prints, a blank line, then notes on the parts of the command line the flags
// leave out, so the help text documents the command on its own.
//
// The renderer writes to the flag set's output. A caller that needs the text
// itself, as `-skill` does, points that output at a buffer first.
func Usage(name string, flags *flag.FlagSet, notes string) func() {
	return func() {
		out := flags.Output()
		fmt.Fprintf(out, "Usage of %s:\n", name)
		flags.PrintDefaults()
		if notes == "" {
			return
		}
		fmt.Fprintf(out, "\n%s\n", strings.TrimRight(notes, "\n"))
	}
}
