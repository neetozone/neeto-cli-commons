package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			if a.Printer.UseJSON() {
				payload, _ := json.Marshal(map[string]string{
					"binary":  a.Product.BinaryName,
					"version": a.Build.Version,
					"commit":  a.Build.Commit,
					"date":    a.Build.Date,
				})
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), a.versionLine())
		},
	}
}

func (a *App) versionLine() string {
	return fmt.Sprintf("%s %s (commit: %s, built: %s)",
		a.Product.BinaryName, a.Build.Version, a.Build.Commit, a.Build.Date)
}
