package commands

import (
	"strings"
	"testing"
)

func TestUsageTemplateSections(t *testing.T) {
	usage := rootCmd.UsageString()

	for _, header := range []string{"USAGE", "COMMANDS", "FLAGS", "EXAMPLES"} {
		if !strings.Contains(usage, header) {
			t.Errorf("expected a %q section in the root help", header)
		}
	}
}

// Cobra's default template ends with a "Use [command] --help" tip. Ours must
// not: the help output already lists every command.
func TestUsageOmitsTrailingTip(t *testing.T) {
	if strings.Contains(rootCmd.UsageString(), "for more information about a command") {
		t.Error("help output should not end with the [command] --help tip")
	}
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
