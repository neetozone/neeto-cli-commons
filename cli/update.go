package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update the CLI to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			method, command := a.resolveUpdate(runtime.GOOS, isHomebrewInstall())
			fmt.Fprintf(cmd.OutOrStdout(), "Detected %s install.\n", method)
			fmt.Fprintf(cmd.OutOrStdout(), "Running: %s\n", command)
			return runShell(command)
		},
	}
}

func (a *App) resolveUpdate(goos string, homebrew bool) (method, command string) {
	switch {
	case goos == "windows":
		return "Windows", fmt.Sprintf("irm %s | iex", a.Product.InstallPS1URL)
	case homebrew:
		return "Homebrew", fmt.Sprintf("brew update && brew upgrade %s", a.Product.BrewFormula())
	default:
		// Download to a temp file before executing so a failed download
		// (404/DNS) surfaces as a non-zero exit instead of being swallowed
		// by the pipe (`curl | sh` reports sh's exit code, not curl's).
		return "shell-script", fmt.Sprintf(`f="$(mktemp)" && trap 'rm -f "$f"' EXIT && curl -fsSL %s -o "$f" && sh "$f"`, a.Product.InstallShURL)
	}
}

// isHomebrewInstall reports whether the running binary lives inside a Homebrew
// Cellar, which is the canonical marker of a formula-managed install on both
// Apple Silicon (/opt/homebrew/Cellar) and Intel (/usr/local/Cellar).
func isHomebrewInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return strings.Contains(exe, "/Cellar/")
}

func runShell(command string) error {
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("powershell", "-NoProfile", "-Command", command)
	} else {
		c = exec.Command("sh", "-c", command)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
