package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newSetupCommand() *cobra.Command {
	pretty := a.Product.PrettyName
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up " + pretty + " for AI coding assistants",
		Long: "Set up " + pretty + " for AI coding assistants.\n\n" +
			"Every subcommand except \"claude\" writes into the current project directory,\n" +
			"so run it from the root of the project you want the assistant to use " + pretty + " in.",
	}

	cmd.AddCommand(
		a.newSetupClaudeCommand(),
		a.newSetupCursorCommand(),
		a.newSetupWindsurfCommand(),
		a.newSetupCopilotCommand(),
		a.newSetupGeminiCommand(),
		a.newSetupCodexCommand(),
	)
	return cmd
}

func (a *App) newSetupClaudeCommand() *cobra.Command {
	pretty := a.Product.PrettyName
	return &cobra.Command{
		Use:   "claude",
		Short: "Register " + pretty + " plugin with Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("Could not determine home directory: %w", err)
			}

			if _, err := os.Stat(filepath.Join(home, ".claude")); os.IsNotExist(err) {
				return fmt.Errorf("Claude Code not found (~/.claude/ does not exist).")
			}

			dest := filepath.Join(home, filepath.FromSlash(a.Product.ConfigDir), "claude-plugin")
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("Could not clean destination: %w", err)
			}
			if err := a.Plugin.Extract(dest); err != nil {
				return fmt.Errorf("Could not extract plugin: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s plugin extracted to %s\n", pretty, dest)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "To finish installation, open Claude Code and run these slash commands:")
			fmt.Fprintf(out, "  /plugin marketplace add %s\n", dest)
			fmt.Fprintf(out, "  /plugin install %s@%s\n", a.Plugin.Name(), a.Plugin.MarketplaceName())
			fmt.Fprintln(out)
			fmt.Fprintln(out, "(Claude Code installs plugins via interactive slash commands — there is no shell equivalent today.)")
			return nil
		},
	}
}

func (a *App) newSetupCursorCommand() *cobra.Command {
	target := filepath.Join(".cursor", "rules", a.Product.BinaryName+".mdc")
	return &cobra.Command{
		Use:   "cursor",
		Short: "Write " + a.Product.PrettyName + " rules for Cursor IDE",
		Long:  a.ruleFileHelp(target),
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeRuleFile(cmd.OutOrStdout(), target, a.cursorContent())
		},
	}
}

func (a *App) cursorContent() string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %q\n", a.Product.PrettyName+" CLI")
	b.WriteString("alwaysApply: true\n")
	b.WriteString("---\n\n")
	b.WriteString(a.Plugin.SkillBody())
	return b.String()
}

func (a *App) newSetupWindsurfCommand() *cobra.Command {
	target := filepath.Join(".windsurf", "rules", a.Product.BinaryName+".md")
	return &cobra.Command{
		Use:   "windsurf",
		Short: "Write " + a.Product.PrettyName + " rules for Windsurf IDE",
		Long:  a.ruleFileHelp(target),
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeRuleFile(cmd.OutOrStdout(), target, a.windsurfContent())
		},
	}
}

func (a *App) windsurfContent() string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("trigger: always_on\n")
	fmt.Fprintf(&b, "description: %q\n", a.Product.PrettyName+" CLI")
	b.WriteString("---\n\n")
	b.WriteString(a.Plugin.SkillBody())
	return b.String()
}

func (a *App) newSetupCopilotCommand() *cobra.Command {
	target := filepath.Join(".github", "copilot-instructions.md")
	return &cobra.Command{
		Use:   "copilot",
		Short: "Add " + a.Product.PrettyName + " instructions for GitHub Copilot",
		Long:  a.sectionHelp(target),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.writeSection(cmd.OutOrStdout(), target, a.Plugin.SkillBody())
		},
	}
}

func (a *App) newSetupGeminiCommand() *cobra.Command {
	const target = "GEMINI.md"
	return &cobra.Command{
		Use:   "gemini",
		Short: "Add " + a.Product.PrettyName + " instructions for Gemini CLI",
		Long:  a.sectionHelp(target),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.writeSection(cmd.OutOrStdout(), target, a.Plugin.SkillBody())
		},
	}
}

func (a *App) newSetupCodexCommand() *cobra.Command {
	const target = "AGENTS.md"
	return &cobra.Command{
		Use:   "codex",
		Short: "Add " + a.Product.PrettyName + " instructions for OpenAI Codex",
		Long:  a.sectionHelp(target),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.writeSection(cmd.OutOrStdout(), target, a.Plugin.SkillBody())
		},
	}
}

func (a *App) sectionHelp(target string) string {
	return fmt.Sprintf(
		"Add a %s section to %s in the current project directory.\n\n"+
			"Existing content in the file is kept. Re-running replaces the %s section\n"+
			"instead of adding a duplicate, so run it again after every upgrade.",
		a.Product.PrettyName, target, a.Product.PrettyName,
	)
}

func (a *App) ruleFileHelp(target string) string {
	return fmt.Sprintf(
		"Write the %s rule file to %s in the current project directory.\n\n"+
			"Re-running overwrites the file, so run it again after every upgrade to pick up the latest rules.",
		a.Product.PrettyName, target,
	)
}

func writeRuleFile(w io.Writer, target, content string) error {
	_, statErr := os.Stat(target)

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := atomicWrite(target, []byte(content)); err != nil {
		return err
	}

	reportWrite(w, target, statErr == nil)
	return nil
}

func (a *App) sectionMarkers() (string, string) {
	name := a.root.Name()
	return "<!-- " + name + ":start -->", "<!-- " + name + ":end -->"
}

func (a *App) writeSection(w io.Writer, target, body string) error {
	_, statErr := os.Stat(target)
	start, end := a.sectionMarkers()
	if _, err := upsertBlock(target, start, end, "## "+a.Product.PrettyName+" CLI\n\n"+body); err != nil {
		return err
	}

	reportWrite(w, target, statErr == nil)
	return nil
}

func reportWrite(w io.Writer, target string, existed bool) {
	if existed {
		fmt.Fprintf(w, "Updated %s\n", target)
		return
	}
	fmt.Fprintf(w, "Wrote %s\n", target)
}
