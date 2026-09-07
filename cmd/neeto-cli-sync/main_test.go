package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const productYAML = `pretty_name: NeetoCal
binary_name: neetocal
short_description: NeetoCal CLI
`

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".neeto-cli.yml"), []byte(productYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func sync(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	var out strings.Builder
	full := append([]string{"--config", filepath.Join(dir, ".neeto-cli.yml")}, args...)
	err := run(full, &out)
	return out.String(), err
}

func TestWriteThenCheckIsClean(t *testing.T) {
	dir := repo(t)
	if _, err := sync(t, dir, "--out", dir); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := sync(t, dir, "--out", dir, "--check"); err != nil {
		t.Fatalf("check after write: %v", err)
	}
}

func TestCheckFailsOnDrift(t *testing.T) {
	dir := repo(t)
	if _, err := sync(t, dir, "--out", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("drifted\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "mise.toml")); err != nil {
		t.Fatal(err)
	}
	_, err := sync(t, dir, "--out", dir, "--check")
	if err == nil {
		t.Fatal("check passed despite drift")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Makefile (out of date)") || !strings.Contains(msg, "mise.toml (missing)") {
		t.Errorf("check did not report the drift: %s", msg)
	}
}

func TestCheckFailsOnALostExecutableBit(t *testing.T) {
	dir := repo(t)
	if _, err := sync(t, dir, "--out", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, "bin", "setup"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := sync(t, dir, "--out", dir, "--check")
	if err == nil || !strings.Contains(err.Error(), "bin/setup (mode") {
		t.Errorf("check did not catch the lost executable bit: %v", err)
	}
}

func TestWriteSetsExecutableBits(t *testing.T) {
	dir := repo(t)
	if _, err := sync(t, dir, "--out", dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin/setup", ".githooks/pre-commit", "installers/install.sh"} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s is not executable (%o)", name, info.Mode().Perm())
		}
	}
	info, err := os.Stat(filepath.Join(dir, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Errorf("Makefile should not be executable (%o)", info.Mode().Perm())
	}
}

func TestRepositoryWriteMergesTheREADME(t *testing.T) {
	dir := repo(t)
	readme := "# NeetoCal CLI\n\nHouse prose.\n\n" +
		"<!-- neeto-cli-commons:release:start -->\nstale\n<!-- neeto-cli-commons:release:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	out, err := sync(t, dir)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !strings.Contains(out, "README.md has no markers for: installation") {
		t.Errorf("sync did not report the missing markers: %s", out)
	}
	merged, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(merged), "stale") || !strings.Contains(string(merged), "House prose.") {
		t.Errorf("README merge went wrong:\n%s", merged)
	}
	if _, err := sync(t, dir, "--check"); err != nil {
		t.Fatalf("check after a repository write: %v", err)
	}
}

func TestEnvPrintsShellAssignments(t *testing.T) {
	dir := repo(t)
	out, err := sync(t, dir, "--env")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"BINARY_NAME='neetocal'", "REPO_NAME='neeto-cal-cli'", "S3_BASE='s3://neeto-downloads/cli/NeetoCal'"} {
		if !strings.Contains(out, want) {
			t.Errorf("--env is missing %q, got:\n%s", want, out)
		}
	}
}

func TestMissingConfigIsAnError(t *testing.T) {
	if err := run([]string{"--config", filepath.Join(t.TempDir(), "nope.yml")}, io.Discard); err == nil {
		t.Error("run accepted a missing config")
	}
}
