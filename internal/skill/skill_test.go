package skill

import "testing"

func TestRenderFillsTheUsageLine(t *testing.T) {
	t.Parallel()

	template := "# demo\n\n```text\n{{usage}}\n```\n"
	usage := func() string { return "Usage of demo:\n  -json\n" }

	got := Render(usage, template)
	want := "# demo\n\n```text\nUsage of demo:\n  -json\n\n```\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRenderLeavesOtherBracesAlone(t *testing.T) {
	t.Parallel()

	template := "{{usage}} keeps {{ .Name }} and {{}}\n"
	usage := func() string { return "usage" }

	got := Render(usage, template)
	want := "usage keeps {{ .Name }} and {{}}\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}
