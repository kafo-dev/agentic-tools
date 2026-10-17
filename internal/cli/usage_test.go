package cli

import (
	"flag"
	"strings"
	"testing"
)

// newFlags builds a flag set with one flag, for the usage tests.
func newFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.Int("n", 4, "number of results to request")
	return flags
}

func TestUsagePrintsTheFlagListThenTheNotes(t *testing.T) {
	t.Parallel()

	flags := newFlags()
	var text strings.Builder
	flags.SetOutput(&text)

	Usage("demo", flags, "The arguments join into one query.")()

	got := text.String()
	if !strings.HasPrefix(got, "Usage of demo:\n") {
		t.Errorf("text = %q, want the usage heading", got)
	}
	if !strings.Contains(got, "-n int") {
		t.Errorf("text = %q, want the flag list", got)
	}
	if !strings.Contains(got, "\n\nThe arguments join into one query.\n") {
		t.Errorf("text = %q, want a blank line then the notes", got)
	}
}

func TestUsageWithoutNotesEndsOnTheFlagList(t *testing.T) {
	t.Parallel()

	flags := newFlags()
	var text strings.Builder
	flags.SetOutput(&text)

	Usage("demo", flags, "")()

	if got := text.String(); !strings.HasSuffix(got, "(default 4)\n") {
		t.Errorf("text = %q, want no blank line after the flag list", got)
	}
}
