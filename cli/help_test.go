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

// Cobra's default template ends with a "Use [command] --help" tip. Ours must
// not: the help output already lists every command.
func TestUsageOmitsTrailingTip(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	if strings.Contains(a.Root().UsageString(), "for more information about a command") {
		t.Error("help output should not end with the [command] --help tip")
	}
}

// Subcommands have no template of their own — cobra walks up to the parent.
// ALIASES renders only on a subcommand, so it is covered here rather than
// against the root.
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

// A subcommand lists its own flags and the root's in one FLAGS section, rather
// than splitting the root's off under a second header.
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

// newTestSubcommand attaches a throwaway subcommand to the root for the length
// of the test, so it inherits the root's persistent flags.
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

// Tests do not run against a terminal, so headers must come back unstyled —
// the same path piped and redirected output takes.
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
