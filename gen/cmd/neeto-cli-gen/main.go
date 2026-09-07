package main

import (
	"fmt"
	"os"

	"github.com/neetozone/neeto-cli-commons/gen/generator"
	"github.com/neetozone/neeto-cli-commons/gen/version"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "neeto-cli-gen",
		Short: "Scaffold a new product CLI from the neeto-cli-commons",
	}

	rootCmd.AddCommand(newCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCmd() *cobra.Command {
	var opts generator.Options

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a new CLI repo",
		Long: `Interactively (or via --config) collect answers and render the CLI
skeleton into a brand-new target directory. Refuses to generate into a
directory that already contains files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TemplateVersion = version.Effective()
			return generator.Run(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "YAML answers file (non-interactive)")
	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", "", "Target directory (default: ./<repo-name>)")
	cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Fail instead of prompting")
	cmd.Flags().BoolVar(&opts.SkipTidy, "skip-tidy", false, "Skip 'go mod tidy' post-generation")
	cmd.Flags().BoolVar(&opts.SkipGitInit, "skip-git-init", false, "Skip 'git init' + initial commit")

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print generator version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("neeto-cli-gen %s (commit: %s, built: %s)\n",
				version.Effective(), version.Commit, version.Date)
		},
	}
}
