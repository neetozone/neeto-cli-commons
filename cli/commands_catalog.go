package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// catalogFlag and catalogEntry are a published contract: the docs sites run
// `<binary> commands` and generate their command reference from this shape.
type catalogFlag struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

type catalogEntry struct {
	Command     string         `json:"command"`
	Description string         `json:"description"`
	Flags       []catalogFlag  `json:"flags,omitempty"`
	Subcommands []catalogEntry `json:"subcommands,omitempty"`
}

func (a *App) newCommandsCatalogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "commands",
		Short: "List all available commands as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := json.MarshalIndent(buildCatalog(a.root), "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
}

func buildCatalog(cmd *cobra.Command) []catalogEntry {
	var entries []catalogEntry

	for _, sub := range cmd.Commands() {
		if skipInCatalog(sub) {
			continue
		}

		entry := catalogEntry{
			Command:     sub.CommandPath(),
			Description: sub.Short,
		}

		sub.Flags().VisitAll(func(f *pflag.Flag) {
			if skipFlagInCatalog(f) {
				return
			}
			entry.Flags = append(entry.Flags, catalogFlag{
				Name:        f.Name,
				Shorthand:   f.Shorthand,
				Type:        f.Value.Type(),
				Default:     f.DefValue,
				Description: f.Usage,
				Required:    isRequired(f),
			})
		})

		if sub.HasSubCommands() {
			entry.Subcommands = buildCatalog(sub)
		}

		entries = append(entries, entry)
	}

	return entries
}

func skipInCatalog(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", "completion", "commands":
		return true
	}
	return cmd.Hidden
}

func skipFlagInCatalog(f *pflag.Flag) bool {
	switch f.Name {
	case "help", "json", "quiet", "toon", "subdomain":
		return true
	}
	return f.Hidden
}
