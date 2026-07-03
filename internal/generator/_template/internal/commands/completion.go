package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var completionShells = []string{"zsh", "bash", "fish", "powershell"}

var completionCmd = &cobra.Command{
	Use:   "completion [zsh|bash|fish|powershell]",
	Short: "Install shell completion",
}

func init() {
	name := rootCmd.Name()
	completionCmd.Long = fmt.Sprintf(
		"Install shell completion for %s.\n\n"+
			"Running \"%s completion <shell>\" writes the completion script under\n"+
			"~/.config/%s/completions and wires your shell to load it on the next start —\n"+
			"no manual sourcing needed. Re-running refreshes the script and shell config.\n"+
			"Pass --print to emit the raw script to standard output instead.",
		name, name, name,
	)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	for _, shell := range completionShells {
		completionCmd.AddCommand(completionShellCmd(shell))
	}
	rootCmd.AddCommand(completionCmd)
}

func completionShellCmd(shell string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Install %s completion", shell),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printOnly, _ := cmd.Flags().GetBool("print")
			if printOnly {
				return generateCompletion(cmd.Root(), shell, cmd.OutOrStdout())
			}
			return installCompletion(cmd.Root(), shell, cmd.OutOrStdout())
		},
	}
	cmd.Flags().Bool("print", false, "Print the completion script to stdout instead of installing it")
	return cmd
}

func generateCompletion(root *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "zsh":
		return root.GenZshCompletion(w)
	case "bash":
		return root.GenBashCompletionV2(w, true)
	case "fish":
		return root.GenFishCompletion(w, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("Unsupported shell: %s", shell)
	}
}

func completionsDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", name, "completions"), nil
}

func writeCompletionScript(root *cobra.Command, shell, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return generateCompletion(root, shell, f)
}

func atomicWrite(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// upsertBlock ensures the file at path contains exactly one block delimited by
// start/end marker lines, with the given body. Any pre-existing block is
// removed and a fresh one is appended, so re-running overwrites the config
// without ever duplicating it. Content outside the block is preserved. Returns
// true when an existing block was replaced.
func upsertBlock(path, start, end, body string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	var kept []string
	existed := false
	if len(data) > 0 {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		inBlock := false
		for _, ln := range lines {
			trimmed := strings.TrimSpace(ln)
			if trimmed == start {
				inBlock = true
				existed = true
				continue
			}
			if inBlock {
				if trimmed == end {
					inBlock = false
				}
				continue
			}
			kept = append(kept, ln)
		}
	}

	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	var sb strings.Builder
	if len(kept) > 0 {
		sb.WriteString(strings.Join(kept, "\n"))
		sb.WriteString("\n\n")
	}
	sb.WriteString(start + "\n")
	sb.WriteString(strings.TrimRight(body, "\n") + "\n")
	sb.WriteString(end + "\n")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return existed, err
	}
	return existed, atomicWrite(path, []byte(sb.String()))
}

func installCompletion(root *cobra.Command, shell string, w io.Writer) error {
	name := root.Name()
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Could not determine home directory: %w", err)
	}
	dir, err := completionsDir(name)
	if err != nil {
		return err
	}
	start := "# " + name + " completions"
	end := "# end " + name + " completions"

	switch shell {
	case "zsh":
		script := filepath.Join(dir, "_"+name)
		if err := writeCompletionScript(root, shell, script); err != nil {
			return err
		}
		rc := filepath.Join(home, ".zshrc")
		body := fmt.Sprintf("fpath=(\"$HOME/.config/%s/completions\" $fpath)\nautoload -Uz compinit && compinit", name)
		refreshed, err := upsertBlock(rc, start, end, body)
		if err != nil {
			return err
		}
		return reportInstall(w, name, shell, script, rc, refreshed)

	case "bash":
		script := filepath.Join(dir, name+".bash")
		if err := writeCompletionScript(root, shell, script); err != nil {
			return err
		}
		rc := filepath.Join(home, ".bashrc")
		body := fmt.Sprintf("[ -f \"$HOME/.config/%s/completions/%s.bash\" ] && source \"$HOME/.config/%s/completions/%s.bash\"", name, name, name, name)
		refreshed, err := upsertBlock(rc, start, end, body)
		if err != nil {
			return err
		}
		return reportInstall(w, name, shell, script, rc, refreshed)

	case "fish":
		fishDir := filepath.Join(home, ".config", "fish", "completions")
		script := filepath.Join(fishDir, name+".fish")
		if err := writeCompletionScript(root, shell, script); err != nil {
			return err
		}
		fmt.Fprintf(w, "Installed %s completion:\n", shell)
		fmt.Fprintf(w, "  script: %s (overwritten)\n", script)
		fmt.Fprintln(w, "fish loads it automatically. Start a new shell to use it.")
		fmt.Fprintf(w, "Re-run \"%s completion %s\" after upgrading to keep completions current with the latest commands.\n", name, shell)
		return nil

	case "powershell":
		script := filepath.Join(dir, name+".ps1")
		if err := writeCompletionScript(root, shell, script); err != nil {
			return err
		}
		profile := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		body := fmt.Sprintf(". \"$HOME/.config/%s/completions/%s.ps1\"", name, name)
		refreshed, err := upsertBlock(profile, start, end, body)
		if err != nil {
			return err
		}
		return reportInstall(w, name, shell, script, profile, refreshed)

	default:
		return fmt.Errorf("Unsupported shell: %s", shell)
	}
}

func reportInstall(w io.Writer, name, shell, script, rc string, refreshed bool) error {
	fmt.Fprintf(w, "Installed %s completion:\n", shell)
	fmt.Fprintf(w, "  script: %s (overwritten)\n", script)
	if refreshed {
		fmt.Fprintf(w, "  loader: refreshed in %s\n", rc)
	} else {
		fmt.Fprintf(w, "  loader: added to %s\n", rc)
	}
	fmt.Fprintf(w, "Start a new shell (or run: source %s) to use it.\n", rc)
	fmt.Fprintf(w, "Re-run \"%s completion %s\" after upgrading to keep completions current with the latest commands.\n", name, shell)
	return nil
}
