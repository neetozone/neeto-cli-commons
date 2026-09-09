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

const groupCommandAnnotation = "neeto-cli/group-command"

var templateFuncs sync.Once

func registerTemplateFuncs() {
	templateFuncs.Do(func() {
		cobra.AddTemplateFunc("bold", bold)
		cobra.AddTemplateFunc("flagUsages", flagUsages)
		cobra.AddTemplateFunc("isGroupCommand", isGroupCommand)
	})
}

func bold(s string) string {
	if !output.IsTTY() || os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func isGroupCommand(cmd *cobra.Command) bool {
	return cmd.Annotations[groupCommandAnnotation] == "true"
}

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

const usageTemplate = `{{bold "USAGE"}}{{if and .Runnable (not (isGroupCommand .))}}
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
