package generator

import (
	"fmt"
	"os"
	"os/exec"
)

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGoModTidy(dir string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go not found on PATH; skip with --skip-tidy: %w", err)
	}
	return runCmd(dir, "go", "mod", "tidy")
}

func runGitInit(dir, templateVersion string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found on PATH; skip with --skip-git-init: %w", err)
	}
	if err := runCmd(dir, "git", "init", "-q"); err != nil {
		return err
	}
	if err := runCmd(dir, "git", "add", "."); err != nil {
		return err
	}
	return runCmd(dir, "git", "-c", "commit.gpgsign=false", "commit", "-q",
		"-m", "Initial commit from neeto-cli-template "+templateVersion)
}
