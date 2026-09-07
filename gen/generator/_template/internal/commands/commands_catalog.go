package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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

var commandsCatalogCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all available commands as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		catalog := buildCatalog(rootCmd)

		out, err := json.MarshalIndent(catalog, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	},
}

func buildCatalog(cmd *cobra.Command) []catalogEntry {
	var entries []catalogEntry

	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" || sub.Name() == "commands" {
			continue
		}

		entry := catalogEntry{
			Command:     sub.CommandPath(),
			Description: sub.Short,
		}

		sub.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden || f.Name == "help" || f.Name == "json" || f.Name == "quiet" || f.Name == "toon" || f.Name == "subdomain" {
				return
			}

			cf := catalogFlag{
				Name:        f.Name,
				Shorthand:   f.Shorthand,
				Type:        f.Value.Type(),
				Default:     f.DefValue,
				Description: f.Usage,
			}

			annotations := f.Annotations
			if annotations != nil {
				if _, ok := annotations[cobra.BashCompOneRequiredFlag]; ok {
					cf.Required = true
				}
			}

			entry.Flags = append(entry.Flags, cf)
		})

		if sub.HasSubCommands() {
			entry.Subcommands = buildCatalog(sub)
		}

		entries = append(entries, entry)
	}

	return entries
}

func init() {
	rootCmd.AddCommand(commandsCatalogCmd)
}
