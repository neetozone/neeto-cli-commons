package commands

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// bold emphasises help section headers, the way gh does. Plain text when the
// output is piped or when NO_COLOR is set (https://no-color.org).
//
// This checks the terminal directly rather than calling output.IsTTY so the
// file needs no module path, and so stays a plain .go template file: every
// {{ }} in the cobra template below would otherwise need escaping.
func bold(s string) string {
	if !term.IsTerminal(int(os.Stdout.Fd())) || os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

// usageTemplate mirrors gh's help layout: uppercase section headers, two-space
// indented bodies, examples after the flags. It replaces cobra's default,
// which also appends a "Use [command] --help" footer we don't want.
//
// Subcommands inherit this: cobra walks up to the parent when a command has no
// template of its own.
const usageTemplate = `{{bold "USAGE"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{bold "ALIASES"}}
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}

{{bold "COMMANDS"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableFlags}}

{{bold "FLAGS"}}
{{.Flags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

{{bold "EXAMPLES"}}
{{.Example}}{{end}}{{if .HasHelpSubCommands}}

{{bold "ADDITIONAL HELP TOPICS"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`

func init() {
	cobra.AddTemplateFunc("bold", bold)
	rootCmd.SetUsageTemplate(usageTemplate)
}
