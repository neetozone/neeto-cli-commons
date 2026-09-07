package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test; skip under -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "testapp-cli")

	err := Run(Options{
		OutputDir:       out,
		ConfigPath:      "testdata/golden_answers.yml",
		SkipTidy:        false,
		SkipGitInit:     true,
		TemplateVersion: "test",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	mustExist := []string{
		".neeto-cli.yml",
		"product.go",
		"cmd/testapp/main.go",
		"internal/commands/register.go",
		"internal/commands/example.go",
		"skills/testapp/SKILL.md",
		".goreleaser.yml",
		".neetoci/verify.yml",
		".neetoci/release.yml",
		".claude-plugin/plugin.json",
		".claude-plugin/marketplace.json",
		"commands/doctor.md",
		"hooks/hooks.json",
		"hooks/session-start.sh",
		"installers/install.sh",
		"installers/install.ps1",
		"installers/install.cmd",
		"mise.toml",
		".gitignore",
		".template-version",
		"VERSION",
		"Makefile",
		"README.md",
		"go.mod",
	}
	for _, f := range mustExist {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("expected %s: %v", f, err)
		}
	}

	forbidden := []string{
		"internal/auth/auth.go",
		"internal/client/client.go",
		"internal/output/output.go",
		"internal/plugin/embed.go",
		"internal/commands/root.go",
		".scripts/release.sh",
		".neetoci/default.yml",
		".exec-manifest",
	}
	for _, f := range forbidden {
		if _, err := os.Stat(filepath.Join(out, f)); !os.IsNotExist(err) {
			t.Errorf("%s must not be generated any more; err=%v", f, err)
		}
	}

	execFiles := []string{
		".githooks/pre-commit",
		"hooks/session-start.sh",
		"bin/setup",
		"installers/install.sh",
	}
	for _, f := range execFiles {
		info, err := os.Stat(filepath.Join(out, f))
		if err != nil {
			t.Errorf("stat %s: %v", f, err)
			continue
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s not executable (mode %v)", f, info.Mode().Perm())
		}
	}

	tv, err := os.ReadFile(filepath.Join(out, ".template-version"))
	if err != nil {
		t.Fatalf("read .template-version: %v", err)
	}
	if strings.TrimSpace(string(tv)) != "test" {
		t.Errorf(".template-version = %q, want %q", string(tv), "test")
	}

	grl, _ := os.ReadFile(filepath.Join(out, ".goreleaser.yml"))
	if !strings.Contains(string(grl), "{{ .Version }}") {
		t.Errorf(".goreleaser.yml is missing literal {{ .Version }}; contents: %s", string(grl))
	}
	if strings.Contains(string(grl), "{{\"{{\"}}") {
		t.Errorf(".goreleaser.yml still has un-rendered escape sequence")
	}

	mainGo, _ := os.ReadFile(filepath.Join(out, "cmd/testapp/main.go"))
	if !strings.Contains(string(mainGo), "github.com/neetozone/testapp-cli/internal/commands") {
		t.Errorf("main.go module path wrong: %s", string(mainGo))
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = out
	tidy.Env = append(os.Environ(), "GOPRIVATE=github.com/neetozone/*")
	if combined, err := tidy.CombinedOutput(); err != nil {
		t.Skipf("go mod tidy failed (network?): %v\n%s", err, combined)
	}

	build := exec.Command("go", "build", "-o", "testapp", "./cmd/testapp/")
	build.Dir = out
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, combined)
	}

	help, err := exec.Command(filepath.Join(out, "testapp"), "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, help)
	}
	for _, flag := range []string{"--json", "--quiet", "--toon", "--subdomain"} {
		if !strings.Contains(string(help), flag) {
			t.Errorf("--help missing %s:\n%s", flag, string(help))
		}
	}

	cat, err := exec.Command(filepath.Join(out, "testapp"), "commands").CombinedOutput()
	if err != nil {
		t.Fatalf("commands failed: %v\n%s", err, cat)
	}
	for _, want := range []string{"testapp login", "testapp logout", "testapp whoami", "testapp doctor", "testapp setup", "testapp version", "testapp update"} {
		if !strings.Contains(string(cat), want) {
			t.Errorf("catalog missing %q", want)
		}
	}
}

func TestRun_RefusesNonEmpty(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "occupied")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "placeholder"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Run(Options{
		OutputDir:       out,
		ConfigPath:      "testdata/golden_answers.yml",
		SkipTidy:        true,
		SkipGitInit:     true,
		TemplateVersion: "test",
	})
	if err == nil {
		t.Fatal("expected error when target dir is non-empty")
	}
	if !strings.Contains(err.Error(), "refusing to generate") {
		t.Errorf("error = %q, want 'refusing to generate'", err.Error())
	}
}
