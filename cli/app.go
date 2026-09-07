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

// App owns the root command and every generic command. Products add their
// domain commands to Root() and call Execute.
type App struct {
	Product config.Product
	Printer *output.Printer
	Auth    *auth.Auth
	Plugin  *plugin.Plugin
	Build   BuildInfo

	// PreRun runs before every command. NeetoDeploy uses it to gate on a
	// Teleport session.
	PreRun func(cmd *cobra.Command, args []string) error

	// FormatError renders an error for the user. NeetoDeploy uses it to turn
	// cobra's raw "required flag not set" into a multi-section message with
	// discovery hints. Returning "" falls back to the default rendering.
	FormatError func(cmd *cobra.Command, err error) string

	// ExitStatus maps an error to a process exit code and whether to print it.
	// NeetoDeploy uses it to propagate tsh's own exit code without repeating
	// the message tsh already printed.
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
	// cobra renders the version template at --version time from a string fixed
	// when it was set, so it has to be re-set once the real build info lands.
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
