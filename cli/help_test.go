package cli

import (
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/output"
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

func TestUsageOmitsUseLineForGroupCommands(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	group := &cobra.Command{Use: "products", Short: "Inspect products"}
	group.AddCommand(&cobra.Command{Use: "list", Short: "List products", RunE: func(*cobra.Command, []string) error { return nil }})
	a.Root().AddCommand(group)
	enforceUnknownSubcommandErrors(a.Root())

	usage := group.UsageString()
	if strings.Contains(usage, "neetodesk products [flags]") {
		t.Errorf("group help should not advertise its own use line, got:\n%s", usage)
	}
	if !strings.Contains(usage, "neetodesk products [command]") {
		t.Errorf("group help should still advertise its subcommands, got:\n%s", usage)
	}
}

func TestUsageKeepsUseLineForRunnableParents(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	parent := &cobra.Command{Use: "setup", Short: "Set up", RunE: func(*cobra.Command, []string) error { return nil }}
	parent.AddCommand(&cobra.Command{Use: "claude", Short: "Claude", RunE: func(*cobra.Command, []string) error { return nil }})
	a.Root().AddCommand(parent)
	enforceUnknownSubcommandErrors(a.Root())

	usage := parent.UsageString()
	if !strings.Contains(usage, "neetodesk setup [flags]") {
		t.Errorf("a genuinely runnable parent should keep its use line, got:\n%s", usage)
	}
}

func TestUsageLinksToTheDocs(t *testing.T) {
	p := subdomainProduct()
	p.DocsURL = "https://apidocs.neetodesk.com/cli/introduction"
	a, _ := newTestApp(t, p)

	usage := a.Root().UsageString()
	if !strings.Contains(usage, "LEARN MORE") || !strings.Contains(usage, p.DocsURL) {
		t.Errorf("expected the docs URL under a LEARN MORE section, got:\n%s", usage)
	}
}

func TestSubcommandUsageLinksToTheDocs(t *testing.T) {
	p := subdomainProduct()
	p.DocsURL = "https://apidocs.neetodesk.com/cli/introduction"
	a, _ := newTestApp(t, p)

	if usage := newTestSubcommand(t, a).UsageString(); !strings.Contains(usage, p.DocsURL) {
		t.Errorf("expected the docs URL in subcommand help, got:\n%s", usage)
	}
}

func TestUsageOmitsLearnMoreWithoutADocsURL(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	if strings.Contains(a.Root().UsageString(), "LEARN MORE") {
		t.Error("help should omit LEARN MORE when the product has no docs URL")
	}
}

func TestLongFlagUsageWrapsInsteadOfRunningPastTheTerminal(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	child := newTestSubcommand(t, a)
	child.Flags().String("time-zone", "", "Time zone as an IANA name, e.g. America/New_York "+
		"(full list: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)")

	usage := child.UsageString()
	for _, line := range strings.Split(usage, "\n") {
		if len(line) > output.TerminalWidth() {
			t.Errorf("help line runs past the terminal at %d columns:\n%q", output.TerminalWidth(), line)
		}
	}
	if !strings.Contains(usage, "List_of_tz_database_time_zones") {
		t.Errorf("wrapping dropped the URL:\n%s", usage)
	}
}
