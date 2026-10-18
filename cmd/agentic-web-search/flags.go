package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"agentic.kafo.dev/tools/internal/cli"
)

const (
	// defaultResults keeps a result set small enough to read in one screen.
	defaultResults = 4
	// queryBytes caps a query read from standard input.
	queryBytes = 8 << 10
)

// notes describes the parts of the command line the flag list leaves out, so
// the help text documents the command on its own.
const notes = `The positional arguments join into one query, so quoting is optional. With no
positional argument the query comes from standard input, folded onto one line;
a terminal there carries no query, so the command exits 2 instead of blocking
on a prompt.`

// options holds the parsed command line.
type options struct {
	query   string
	results int
	json    bool
	skill   bool
}

// newFlags registers the command's flags and installs the help text, so parse
// and usage share one list.
func newFlags(opts *options) *flag.FlagSet {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.IntVar(&opts.results, "n", defaultResults, "number of results to request")
	flags.BoolVar(&opts.json, "json", false, "print the results as one line of JSON")
	flags.BoolVar(&opts.skill, "skill", false, "print the skill document and exit")
	flags.Usage = cli.Usage(command, flags, notes)
	return flags
}

// usage returns the help text the command prints.
func usage() string {
	var text strings.Builder

	flags := newFlags(&options{})
	flags.SetOutput(&text)
	flags.Usage()

	return text.String()
}

// parse reads the flags and resolves the query from the positional arguments,
// or from standard input when the command line carries none.
func parse(args []string, stdin io.Reader) (options, error) {
	opts := options{}
	flags := newFlags(&opts)

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if opts.skill {
		return opts, nil
	}
	opts.query = strings.Join(flags.Args(), " ")
	if opts.query == "" {
		query, err := readQuery(stdin)
		if err != nil {
			return options{}, err
		}
		opts.query = query
	}
	if opts.query == "" {
		return options{}, errors.New("no query given: pass the search terms as arguments or on standard input")
	}
	if opts.results < 1 {
		return options{}, fmt.Errorf("invalid -n %d: want a positive count", opts.results)
	}
	return opts, nil
}

// readQuery folds the query on standard input onto one line. A nil reader, an
// empty pipe, and a terminal that [pipedStdin] rejected all yield no query.
func readQuery(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", nil
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, queryBytes+1))
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	if len(raw) > queryBytes {
		return "", fmt.Errorf("standard input holds more than %d bytes: pipe one query", queryBytes)
	}
	return strings.Join(strings.Fields(string(raw)), " "), nil
}

// pipedStdin returns standard input when it carries piped data, and nil when it
// is a terminal: reading a terminal blocks a caller that cannot answer.
func pipedStdin() io.Reader {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil
	}
	return os.Stdin
}
