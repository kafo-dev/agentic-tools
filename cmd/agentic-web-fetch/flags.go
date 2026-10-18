package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agentic.kafo.dev/tools/internal/cli"
)

const (
	// defaultTimeout bounds one request.
	defaultTimeout = 32 * time.Second
	// urlBytes caps a URL read from standard input.
	urlBytes = 4 << 10
	// schemeSep separates a URL's scheme from the address that follows it, and
	// httpsScheme is what a bare address takes.
	schemeSep   = "://"
	httpsScheme = "https"
)

// notes describes the parts of the command line the flag list leaves out, so
// the help text documents the command on its own.
const notes = `The URL arrives as the single positional argument, or on standard input when the
command line carries none. A terminal there carries no URL, so the command
exits 2 instead of blocking on a prompt, and an address without a scheme takes
the https one.

Standard output carries the content alone. -json adds the address the content
came from and the branch that produced it.`

// options holds the parsed command line.
type options struct {
	target  string
	timeout time.Duration
	json    bool
	skill   bool
}

// newFlags registers the command's flags and installs the help text, so parse
// and usage share one list.
func newFlags(opts *options) *flag.FlagSet {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.DurationVar(&opts.timeout, "t", defaultTimeout, "timeout of one request, as a duration")
	flags.BoolVar(&opts.json, "json", false, "print the URL, the method, and the content as one line of JSON")
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

// parse reads the flags and resolves the URL from the positional arguments, or
// from standard input when the command line carries none.
func parse(args []string, stdin io.Reader) (options, error) {
	opts := options{}
	flags := newFlags(&opts)

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if opts.skill {
		return opts, nil
	}
	target, err := readTarget(flags.Args(), stdin)
	if err != nil {
		return options{}, err
	}
	opts.target = target
	if opts.timeout < time.Second {
		return options{}, fmt.Errorf("invalid -t %s: want at least a second", opts.timeout)
	}
	return opts, nil
}

// readTarget resolves the URL: the single positional argument, or the one
// address on standard input when the command line carries none.
func readTarget(args []string, stdin io.Reader) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("want one URL, got %d: %s", len(args), strings.Join(args, " "))
	}
	target, err := targetFrom(args, stdin)
	if err != nil {
		return "", err
	}
	if target == "" {
		return "", errors.New("no URL given: pass it as the argument or on standard input")
	}
	if strings.Contains(target, schemeSep) {
		return target, nil
	}
	return httpsScheme + schemeSep + target, nil
}

// targetFrom returns what the command line carries: its single positional
// argument, or the address on standard input.
func targetFrom(args []string, stdin io.Reader) (string, error) {
	if len(args) == 1 {
		return strings.TrimSpace(args[0]), nil
	}
	return readPipedURL(stdin)
}

// readPipedURL reads one address from standard input. A nil reader and an
// empty pipe both yield no address.
func readPipedURL(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", nil
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, urlBytes+1))
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	if len(raw) > urlBytes {
		return "", fmt.Errorf("standard input holds more than %d bytes: pipe one URL", urlBytes)
	}
	words := strings.Fields(string(raw))
	switch len(words) {
	case 0:
		return "", nil
	case 1:
		return words[0], nil
	default:
		return "", fmt.Errorf("standard input holds %d words: pipe one URL", len(words))
	}
}

// pipedStdin returns standard input when it carries piped data, and nil when
// it is a terminal: reading a terminal blocks a caller that cannot answer.
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
