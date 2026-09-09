package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/neetozone/neeto-cli-commons/auth"
	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/neetozone/neeto-cli-commons/output"
	"github.com/neetozone/neeto-cli-commons/plugin"
	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type App struct {
	Product config.Product
	Printer *output.Printer
	Auth    *auth.Auth
	Plugin  *plugin.Plugin
	Build   BuildInfo

	PreRun func(cmd *cobra.Command, args []string) error

	FormatError func(cmd *cobra.Command, err error) string

	ExitStatus func(err error) (code int, report bool)

	root *cobra.Command
}

func New(p config.Product) *App {
	a := &App{
		Product: p,
		Printer: output.New(p),
		Auth:    auth.New(p),
		Plugin:  plugin.New(p),
		Build:   BuildInfo{Version: "dev", Commit: "none", Date: "unknown"},
	}
	a.root = a.newRootCommand()
	a.registerCommonCommands()
	return a
}

func (a *App) SetBuildInfo(version, commit, date string) {
	if version != "" {
		a.Build.Version = version
	}
	if commit != "" {
		a.Build.Commit = commit
	}
	if date != "" {
		a.Build.Date = date
	}
	a.root.Version = a.Build.Version
	a.root.SetVersionTemplate(fmt.Sprintf("%s %s (commit: %s, built: %s)\n",
		a.Product.BinaryName, a.Build.Version, a.Build.Commit, a.Build.Date))
}

func (a *App) Root() *cobra.Command { return a.root }

func (a *App) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           a.Product.BinaryName,
		Short:         a.Product.RootShort(),
		Long:          a.Product.LongDescription,
		Example:       a.examples(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			quietFlag, _ := cmd.Flags().GetBool("quiet")
			toonFlag, _ := cmd.Flags().GetBool("toon")
			a.Printer.ForceJSON = jsonFlag
			a.Printer.Quiet = quietFlag
			a.Printer.Toon = toonFlag
			if a.PreRun != nil {
				return a.PreRun(cmd, args)
			}
			return nil
		},
	}

	root.PersistentFlags().Bool("json", false, "Output as JSON")
	root.PersistentFlags().Bool("quiet", false, "Output raw data only (no envelope)")
	root.PersistentFlags().Bool("toon", false, "Output in TOON format (token-optimized for AI agents)")
	if a.Auth.RequiresSubdomain() {
		root.PersistentFlags().String("subdomain", "", "Override saved subdomain")
	}
	root.SuggestionsMinimumDistance = 2
	return root
}

func (a *App) examples() string {
	if len(a.Product.RootExamples) == 0 {
		return ""
	}
	var b strings.Builder
	for i, ex := range a.Product.RootExamples {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "  $ %s", ex)
	}
	return b.String()
}

func (a *App) Execute() {
	enforceUnknownSubcommandErrors(a.root)

	cmd, err := a.root.ExecuteC()
	if err == nil {
		return
	}
	code, report := a.exitStatus(err)
	if report {
		if a.FormatError != nil {
			if msg := a.FormatError(cmd, err); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
				os.Exit(code)
			}
		}
		fmt.Fprintln(os.Stderr, err)
		if isUsageError(err) {
			fmt.Fprintf(os.Stderr, "Run '%s --help' for usage.\n", cmd.CommandPath())
		}
	}
	os.Exit(code)
}

func (a *App) exitStatus(err error) (int, bool) {
	if a.ExitStatus != nil {
		return a.ExitStatus(err)
	}
	return 1, true
}

func enforceUnknownSubcommandErrors(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		if sub.HasSubCommands() && !sub.Runnable() {
			markGroupCommand(sub)
			sub.RunE = func(cmd *cobra.Command, _ []string) error {
				return cmd.Help()
			}
			if sub.Args == nil {
				sub.Args = unknownSubcommandArgs
			}
		}
		enforceUnknownSubcommandErrors(sub)
	}
}

func markGroupCommand(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[groupCommandAnnotation] = "true"
}

func unknownSubcommandArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf("unknown command %q for %q%s", args[0], cmd.CommandPath(), subcommandSuggestions(cmd, args[0]))
}

func subcommandSuggestions(cmd *cobra.Command, name string) string {
	if cmd.DisableSuggestions {
		return ""
	}
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	suggestions := cmd.SuggestionsFor(name)
	if len(suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nDid you mean this?\n")
	for _, s := range suggestions {
		fmt.Fprintf(&b, "\t%s\n", s)
	}
	return b.String()
}

func isUsageError(err error) bool {
	msg := err.Error()
	prefixes := []string{
		"unknown flag",
		"unknown shorthand flag",
		"unknown command",
		"invalid argument",
		"flag needs an argument",
		"required flag",
		"requires at least",
		"accepts ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	return false
}
