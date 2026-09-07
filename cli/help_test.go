package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUsageTemplateSections(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	a.Product.RootExamples = []string{"neetodesk tickets list"}
	a.root.Example = a.examples()

	usage := a.Root().UsageString()
	for _, header := range []string{"USAGE", "COMMANDS", "FLAGS", "EXAMPLES"} {
		if !strings.Contains(usage, header) {
			t.Errorf("expected a %q section in the root help, got:\n%s", header, usage)
		}
	}
}

func TestUsageOmitsTrailingTip(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	if strings.Contains(a.Root().UsageString(), "for more information about a command") {
		t.Error("help output should not end with the [command] --help tip")
	}
}

func TestSubcommandInheritsUsageTemplate(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	child := newTestSubcommand(t, a)

	usage := child.UsageString()
	for _, header := range []string{"USAGE", "ALIASES", "FLAGS"} {
		if !strings.Contains(usage, header) {
			t.Errorf("expected a %q section in subcommand help, got:\n%s", header, usage)
		}
	}
	if strings.Contains(usage, "for more information about a command") {
		t.Error("subcommand help should not end with the [command] --help tip")
	}
}

func TestSubcommandFlagsAreOneSection(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	usage := newTestSubcommand(t, a).UsageString()

	if count := strings.Count(usage, "FLAGS"); count != 1 {
		t.Errorf("expected exactly one FLAGS section, found %d in:\n%s", count, usage)
	}
	for _, flag := range []string{"--dry-run", "--json"} {
		if !strings.Contains(usage, flag) {
			t.Errorf("expected %s listed under FLAGS, got:\n%s", flag, usage)
		}
	}
}

func TestRequiredMarkerIsAddedAtRenderTime(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	child := newTestSubcommand(t, a)
	MarkFlagsRequired(child, "dry-run")

	if !strings.Contains(child.UsageString(), "Report what would change (required)") {
		t.Errorf("expected the required marker in help, got:\n%s", child.UsageString())
	}
	if usage := child.Flags().Lookup("dry-run").Usage; usage != "Report what would change" {
		t.Errorf("the marker leaked into the flag usage: %q", usage)
	}
}

func newTestSubcommand(t *testing.T, a *App) *cobra.Command {
	t.Helper()
	child := &cobra.Command{
		Use:     "widgets",
		Short:   "Manage widgets",
		Aliases: []string{"w"},
		Run:     func(*cobra.Command, []string) {},
	}
	child.Flags().Bool("dry-run", false, "Report what would change")
	a.Root().AddCommand(child)
	t.Cleanup(func() { a.Root().RemoveCommand(child) })
	return child
}

func TestBoldIsPlainWhenNotATerminal(t *testing.T) {
	if got := bold("USAGE"); got != "USAGE" {
		t.Errorf("bold(%q) = %q, want it unstyled off a terminal", "USAGE", got)
	}
}

func TestBoldRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := bold("USAGE"); got != "USAGE" {
		t.Errorf("bold(%q) = %q, want it unstyled under NO_COLOR", "USAGE", got)
	}
}
