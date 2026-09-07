package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func completionShells() []string { return []string{"zsh", "bash", "fish", "powershell"} }

func (a *App) newCompletionCommand() *cobra.Command {
	name := a.root.Name()
	cmd := &cobra.Command{
		Use:   "completion [zsh|bash|fish|powershell]",
		Short: "Install shell completion",
		Long: fmt.Sprintf(
			"Install shell completion for %s.\n\n"+
				"Running \"%s completion <shell>\" writes the completion script under\n"+
				"~/%s and wires your shell to load it on the next start —\n"+
				"no manual sourcing needed. Re-running refreshes the script and shell config.\n"+
				"Pass --print to emit the raw script to standard output instead.",
			name, name, a.completionsSubdir(),
		),
	}

	for _, shell := range completionShells() {
		cmd.AddCommand(a.newCompletionShellCommand(shell))
	}
	return cmd
}

func (a *App) newCompletionShellCommand(shell string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Install %s completion", shell),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printOnly, _ := cmd.Flags().GetBool("print")
			if printOnly {
				return generateCompletion(cmd.Root(), shell, cmd.OutOrStdout())
			}
			return a.installCompletion(shell, cmd.OutOrStdout())
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

func (a *App) completionsSubdir() string {
	return filepath.ToSlash(a.Product.ConfigDir) + "/completions"
}

func (a *App) completionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, filepath.FromSlash(a.completionsSubdir())), nil
}

func writeCompletionScript(root *cobra.Command, shell, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
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
		if inBlock {
			return false, fmt.Errorf("%s has %q without a closing %q. Fix or remove the markers and re-run.", path, start, end)
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

func (a *App) installCompletion(shell string, w io.Writer) error {
	name := a.root.Name()
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Could not determine home directory: %w", err)
	}
	dir, err := a.completionsDir()
	if err != nil {
		return err
	}
	completions := a.completionsSubdir()
	start := "# " + name + " completions"
	end := "# end " + name + " completions"

	switch shell {
	case "zsh":
		script := filepath.Join(dir, "_"+name)
		if err := writeCompletionScript(a.root, shell, script); err != nil {
			return err
		}
		rc := filepath.Join(home, ".zshrc")
		body := fmt.Sprintf("fpath=(\"$HOME/%s\" $fpath)\nautoload -Uz compinit && compinit", completions)
		refreshed, err := upsertBlock(rc, start, end, body)
		if err != nil {
			return err
		}
		return reportInstall(w, name, shell, script, rc, refreshed)

	case "bash":
		script := filepath.Join(dir, name+".bash")
		if err := writeCompletionScript(a.root, shell, script); err != nil {
			return err
		}
		rc := filepath.Join(home, ".bashrc")
		body := fmt.Sprintf("[ -f \"$HOME/%s/%s.bash\" ] && source \"$HOME/%s/%s.bash\"", completions, name, completions, name)
		refreshed, err := upsertBlock(rc, start, end, body)
		if err != nil {
			return err
		}
		return reportInstall(w, name, shell, script, rc, refreshed)

	case "fish":
		fishDir := filepath.Join(home, ".config", "fish", "completions")
		script := filepath.Join(fishDir, name+".fish")
		if err := writeCompletionScript(a.root, shell, script); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(w, "Installed %s completion:\n", shell)
		_, _ = fmt.Fprintf(w, "  script: %s (overwritten)\n", script)
		_, _ = fmt.Fprintln(w, "fish loads it automatically. Start a new shell to use it.")
		_, _ = fmt.Fprintf(w, "Re-run \"%s completion %s\" after upgrading to keep completions current with the latest commands.\n", name, shell)
		return nil

	case "powershell":
		script := filepath.Join(dir, name+".ps1")
		if err := writeCompletionScript(a.root, shell, script); err != nil {
			return err
		}
		profile := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		body := fmt.Sprintf(". \"$HOME/%s/%s.ps1\"", completions, name)
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
	_, _ = fmt.Fprintf(w, "Installed %s completion:\n", shell)
	_, _ = fmt.Fprintf(w, "  script: %s (overwritten)\n", script)
	if refreshed {
		_, _ = fmt.Fprintf(w, "  loader: refreshed in %s\n", rc)
	} else {
		_, _ = fmt.Fprintf(w, "  loader: added to %s\n", rc)
	}
	_, _ = fmt.Fprintf(w, "Start a new shell (or run: source %s) to use it.\n", rc)
	_, _ = fmt.Fprintf(w, "Re-run \"%s completion %s\" after upgrading to keep completions current with the latest commands.\n", name, shell)
	return nil
}
