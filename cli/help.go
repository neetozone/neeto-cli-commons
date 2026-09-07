package cli

import (
	"os"
	"strings"
	"sync"

	"github.com/neetozone/neeto-cli-commons/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const requiredFlagSuffix = " (required)"

// templateFuncs are registered process-wide by cobra, so every App would
// otherwise re-register them.
var templateFuncs sync.Once

func registerTemplateFuncs() {
	templateFuncs.Do(func() {
		cobra.AddTemplateFunc("bold", bold)
		cobra.AddTemplateFunc("flagUsages", flagUsages)
	})
}

// bold emphasises help section headers, the way gh does. Plain text when the
// output is piped or when NO_COLOR is set (https://no-color.org).
func bold(s string) string {
	if !output.IsTTY() || os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

// flagUsages appends the "(required)" marker at render time, so the marker
// never reaches the `commands` catalog the docs sites generate from.
func flagUsages(flags *pflag.FlagSet) string {
	var marked []*pflag.Flag
	flags.VisitAll(func(f *pflag.Flag) {
		if isRequired(f) {
			f.Usage += requiredFlagSuffix
			marked = append(marked, f)
		}
	})
	defer func() {
		for _, f := range marked {
			f.Usage = strings.TrimSuffix(f.Usage, requiredFlagSuffix)
		}
	}()

	return strings.TrimRight(flags.FlagUsages(), " \t\n")
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
{{flagUsages .Flags}}{{end}}{{if .HasExample}}

{{bold "EXAMPLES"}}
{{.Example}}{{end}}{{if .HasHelpSubCommands}}

{{bold "ADDITIONAL HELP TOPICS"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`
