package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallZshWritesScriptAndWiresRc(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	var out bytes.Buffer
	if err := a.installCompletion("zsh", &out); err != nil {
		t.Fatalf("installCompletion returned error: %v", err)
	}

	script := filepath.Join(home, ".config", "neetodesk", "completions", "_neetodesk")
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("expected completion script at %s: %v", script, err)
	}
	if !strings.HasPrefix(string(data), "#compdef neetodesk") {
		t.Fatalf("script missing zsh compdef header, got: %.40q", string(data))
	}

	rc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("expected .zshrc to be created: %v", err)
	}
	for _, want := range []string{"# neetodesk completions", "# end neetodesk completions", `fpath=("$HOME/.config/neetodesk/completions" $fpath)`} {
		if !strings.Contains(string(rc), want) {
			t.Fatalf(".zshrc missing %q:\n%s", want, rc)
		}
	}
	if !strings.Contains(out.String(), "added to") {
		t.Fatalf("first install should report loader added, got: %s", out.String())
	}
}

func TestReinstallRefreshesWithoutDuplicating(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := a.installCompletion("zsh", &bytes.Buffer{}); err != nil {
		t.Fatalf("first install error: %v", err)
	}
	var out bytes.Buffer
	if err := a.installCompletion("zsh", &out); err != nil {
		t.Fatalf("second install error: %v", err)
	}

	rc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if n := strings.Count(string(rc), "# neetodesk completions"); n != 1 {
		t.Fatalf("expected start marker exactly once, found %d:\n%s", n, rc)
	}
	if !strings.Contains(out.String(), "refreshed in") {
		t.Fatalf("re-run should report loader refreshed, got: %s", out.String())
	}
}

func TestReinstallRestoresTamperedConfig(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := a.installCompletion("zsh", &bytes.Buffer{}); err != nil {
		t.Fatalf("install error: %v", err)
	}

	script := filepath.Join(home, ".config", "neetodesk", "completions", "_neetodesk")
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(script, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcPath, []byte("# neetodesk completions\ngarbage-line\n# end neetodesk completions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.installCompletion("zsh", &bytes.Buffer{}); err != nil {
		t.Fatalf("reinstall error: %v", err)
	}

	data, _ := os.ReadFile(script)
	if !strings.HasPrefix(string(data), "#compdef neetodesk") {
		t.Fatalf("script was not regenerated, got: %.40q", string(data))
	}
	rc, _ := os.ReadFile(rcPath)
	if strings.Contains(string(rc), "garbage-line") {
		t.Fatalf("stale block not overwritten:\n%s", rc)
	}
	if !strings.Contains(string(rc), `fpath=("$HOME/.config/neetodesk/completions" $fpath)`) {
		t.Fatalf("fresh block not written:\n%s", rc)
	}
	if n := strings.Count(string(rc), "# neetodesk completions"); n != 1 {
		t.Fatalf("expected start marker exactly once, found %d:\n%s", n, rc)
	}
}

func TestInstallPreservesExistingRcContent(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	rcPath := filepath.Join(os.Getenv("HOME"), ".zshrc")
	if err := os.WriteFile(rcPath, []byte("export EDITOR=vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.installCompletion("zsh", &bytes.Buffer{}); err != nil {
		t.Fatalf("install error: %v", err)
	}
	rc, _ := os.ReadFile(rcPath)
	if !strings.Contains(string(rc), "export EDITOR=vim") {
		t.Fatalf("existing rc content was lost:\n%s", rc)
	}
	if !strings.Contains(string(rc), "# neetodesk completions") {
		t.Fatalf("marker not appended:\n%s", rc)
	}
}

func TestInstallBashWiresBashrc(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := a.installCompletion("bash", &bytes.Buffer{}); err != nil {
		t.Fatalf("install error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "neetodesk", "completions", "neetodesk.bash")); err != nil {
		t.Fatalf("expected the bash script: %v", err)
	}
	rc, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatalf("expected .bashrc: %v", err)
	}
	if !strings.Contains(string(rc), `source "$HOME/.config/neetodesk/completions/neetodesk.bash"`) {
		t.Fatalf(".bashrc missing the loader:\n%s", rc)
	}
}

func TestInstallPowerShellWiresTheProfile(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := a.installCompletion("powershell", &bytes.Buffer{}); err != nil {
		t.Fatalf("install error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "neetodesk", "completions", "neetodesk.ps1")); err != nil {
		t.Fatalf("expected the powershell script: %v", err)
	}
	profile, err := os.ReadFile(filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"))
	if err != nil {
		t.Fatalf("expected the PowerShell profile: %v", err)
	}
	if !strings.Contains(string(profile), `. "$HOME/.config/neetodesk/completions/neetodesk.ps1"`) {
		t.Fatalf("profile missing the loader:\n%s", profile)
	}
}

func TestInstallFishNeedsNoRcEdit(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := a.installCompletion("fish", &bytes.Buffer{}); err != nil {
		t.Fatalf("install error: %v", err)
	}
	script := filepath.Join(home, ".config", "fish", "completions", "neetodesk.fish")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("expected fish completion at %s: %v", script, err)
	}
}

func TestInstallReportsReRunTip(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	var out bytes.Buffer
	if err := a.installCompletion("zsh", &out); err != nil {
		t.Fatalf("install error: %v", err)
	}
	if !strings.Contains(out.String(), "Re-run") || !strings.Contains(out.String(), "latest commands") {
		t.Fatalf("output should nudge re-running to stay current, got: %s", out.String())
	}
}

func TestInstallRejectsAnUnknownShell(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	if err := a.installCompletion("nushell", &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error for an unsupported shell")
	}
}

func TestGenerateWritesEachShell(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	for _, shell := range completionShells() {
		var buf bytes.Buffer
		if err := generateCompletion(a.Root(), shell, &buf); err != nil {
			t.Fatalf("generate %s error: %v", shell, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("generate %s produced empty script", shell)
		}
	}
}

func TestCompletionRegistersEveryShellSubcommand(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	completion, _, err := a.Root().Find([]string{"completion"})
	if err != nil {
		t.Fatalf("completion not found: %v", err)
	}
	got := map[string]bool{}
	for _, name := range commandNames(completion) {
		got[name] = true
	}
	for _, shell := range completionShells() {
		if !got[shell] {
			t.Errorf("completion is missing the %q subcommand, has %v", shell, commandNames(completion))
		}
	}
}

func TestPrintDoesNotTouchFilesystem(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	var buf bytes.Buffer
	if err := generateCompletion(a.Root(), "zsh", &buf); err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Fatalf("print path should not create ~/.config, stat err: %v", err)
	}
}

func TestPrintFlagOnCommandWritesNoFiles(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")

	if err := run(t, a, "completion", "zsh", "--print"); err != nil {
		t.Fatalf("running completion zsh --print: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("expected the completion script on stdout")
	}
	if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
		t.Fatalf("--print via the command must not create ~/.config, stat err: %v", err)
	}
}

func TestUpsertBlockRefusesUnterminatedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rc")
	original := "# begin block\nstale line\n\nexport EDITOR=vim\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := upsertBlock(path, "# begin block", "# end block", "fresh line"); err == nil {
		t.Fatal("expected an error for a start marker without an end marker")
	}
	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Fatalf("file was modified despite the error:\n%s", data)
	}
}

func TestCompletionHelpDoesNotDoubleTheCompletionsDirectory(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	for _, c := range a.Root().Commands() {
		if c.Name() != "completion" {
			continue
		}
		if strings.Contains(c.Long, "completions/completions") {
			t.Errorf("completion help documents a doubled path:\n%s", c.Long)
		}
		if !strings.Contains(c.Long, "~/.config/neetodesk/completions") {
			t.Errorf("completion help lost the real path:\n%s", c.Long)
		}
	}
}
