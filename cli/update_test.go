package cli

import (
	"strings"
	"testing"
)

func TestResolveUpdate(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	cases := []struct {
		name     string
		goos     string
		homebrew bool
		method   string
		contains string
	}{
		{"windows re-runs the PowerShell installer", "windows", false, "Windows", "install.ps1"},
		{"windows ignores the homebrew flag", "windows", true, "Windows", "install.ps1"},
		{"homebrew upgrades the formula", "darwin", true, "Homebrew", "brew upgrade neetozone/tap/neetodesk"},
		{"non-brew unix re-runs install.sh", "darwin", false, "shell-script", "install.sh"},
		{"linux without brew re-runs install.sh", "linux", false, "shell-script", "install.sh"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method, command := a.resolveUpdate(tc.goos, tc.homebrew)
			if method != tc.method {
				t.Errorf("method = %q, want %q", method, tc.method)
			}
			if !strings.Contains(command, tc.contains) {
				t.Errorf("command %q does not contain %q", command, tc.contains)
			}
		})
	}
}

func TestResolveUpdate_ShellScriptCleansUpItsTempFile(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	_, command := a.resolveUpdate("linux", false)
	if !strings.Contains(command, `trap 'rm -f "$f"' EXIT`) {
		t.Errorf("the shell path must not leak its temp file: %s", command)
	}
	if !strings.Contains(command, `curl -fsSL`) || !strings.Contains(command, `-o "$f"`) {
		t.Errorf("the installer must be downloaded before it runs: %s", command)
	}
}

func TestResolveUpdate_WindowsCommandIsNotDoubleWrapped(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	_, command := a.resolveUpdate("windows", false)
	if strings.Contains(command, "powershell") {
		t.Errorf("runShell already invokes PowerShell, so the command must not: %s", command)
	}
	if !strings.HasPrefix(command, "irm ") || !strings.HasSuffix(command, " | iex") {
		t.Errorf("unexpected Windows command: %s", command)
	}
}

func TestResolveUpdate_URLsComeFromTheProduct(t *testing.T) {
	a, _ := newTestApp(t, singleHostProduct())

	_, shell := a.resolveUpdate("linux", false)
	if !strings.Contains(shell, "cli/NeetoDeploy/latest/install.sh") {
		t.Errorf("install.sh URL should follow the product: %s", shell)
	}
	_, windows := a.resolveUpdate("windows", false)
	if !strings.Contains(windows, "cli/NeetoDeploy/latest/install.ps1") {
		t.Errorf("install.ps1 URL should follow the product: %s", windows)
	}
	_, brew := a.resolveUpdate("darwin", true)
	if !strings.Contains(brew, "neetozone/tap/neetodeploy") {
		t.Errorf("brew formula should follow the product: %s", brew)
	}
}

func TestIsUsageError(t *testing.T) {
	usage := []string{
		"unknown flag: --version",
		"unknown shorthand flag: 'x' in -x",
		`unknown command "frobnicate" for "neetodesk"`,
		"flag needs an argument: --app",
		`required flag(s) "app" not set`,
		"requires at least 1 arg(s), only received 0",
		"accepts at most 2 arg(s), received 3",
		"accepts 1 arg(s), received 0",
	}
	for _, msg := range usage {
		if !isUsageError(errString(msg)) {
			t.Errorf("%q should be a usage error", msg)
		}
	}

	notUsage := []string{
		"not logged in",
		"request failed with status 500",
	}
	for _, msg := range notUsage {
		if isUsageError(errString(msg)) {
			t.Errorf("%q should not be a usage error", msg)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }
