package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSectionCreatesFileWithMarkers(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	target := filepath.Join(t.TempDir(), "AGENTS.md")

	var out bytes.Buffer
	if err := a.writeSection(&out, target, "body text"); err != nil {
		t.Fatalf("writeSection returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected %s to be created: %v", target, err)
	}
	for _, want := range []string{"<!-- neetodesk:start -->", "## NeetoDesk CLI", "body text", "<!-- neetodesk:end -->"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("file missing %q:\n%s", want, data)
		}
	}
	if !strings.Contains(out.String(), "Wrote") {
		t.Fatalf("first run should report the file was written, got: %s", out.String())
	}
}

func TestWriteSectionRerunDoesNotDuplicate(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	target := filepath.Join(t.TempDir(), "AGENTS.md")
	body := a.Plugin.SkillBody()

	if err := a.writeSection(&bytes.Buffer{}, target, body); err != nil {
		t.Fatalf("first run error: %v", err)
	}
	first, _ := os.ReadFile(target)

	var out bytes.Buffer
	if err := a.writeSection(&out, target, body); err != nil {
		t.Fatalf("second run error: %v", err)
	}
	second, _ := os.ReadFile(target)

	if !bytes.Equal(first, second) {
		t.Fatalf("re-run changed the file from %d to %d bytes", len(first), len(second))
	}
	if n := strings.Count(string(second), "<!-- neetodesk:start -->"); n != 1 {
		t.Fatalf("expected start marker exactly once, found %d:\n%s", n, second)
	}
	if n := strings.Count(string(second), "## NeetoDesk CLI"); n != 1 {
		t.Fatalf("expected section heading exactly once, found %d", n)
	}
	if !strings.Contains(out.String(), "Updated") {
		t.Fatalf("re-run should report the file was updated, got: %s", out.String())
	}
}

func TestWriteSectionPreservesExistingContent(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	target := filepath.Join(t.TempDir(), "GEMINI.md")
	existing := "# My project\n\nHouse rules for this repo.\n"
	if err := os.WriteFile(target, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.writeSection(&bytes.Buffer{}, target, "body text"); err != nil {
		t.Fatalf("writeSection returned error: %v", err)
	}

	data, _ := os.ReadFile(target)
	if !strings.HasPrefix(string(data), existing) {
		t.Fatalf("existing content was not preserved at the top:\n%s", data)
	}
	if !strings.Contains(string(data), "## NeetoDesk CLI\n\nbody text") {
		t.Fatalf("section not appended after existing content:\n%s", data)
	}
}

func TestWriteSectionReplacesStaleBlock(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	target := filepath.Join(t.TempDir(), "copilot-instructions.md")
	stale := "# Notes\n\n<!-- neetodesk:start -->\n## NeetoDesk CLI\n\nstale body\n<!-- neetodesk:end -->\n\n## Testing\n\nRun make test.\n"
	if err := os.WriteFile(target, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.writeSection(&bytes.Buffer{}, target, "fresh body"); err != nil {
		t.Fatalf("writeSection returned error: %v", err)
	}

	data, _ := os.ReadFile(target)
	if strings.Contains(string(data), "stale body") {
		t.Fatalf("stale block was not replaced:\n%s", data)
	}
	if !strings.Contains(string(data), "fresh body") {
		t.Fatalf("fresh body not written:\n%s", data)
	}
	if !strings.HasPrefix(string(data), "# Notes\n") {
		t.Fatalf("content outside the block was lost:\n%s", data)
	}
	if !strings.Contains(string(data), "## Testing\n\nRun make test.\n") {
		t.Fatalf("content after the block was lost:\n%s", data)
	}
	if strings.Index(string(data), "## Testing") > strings.Index(string(data), "<!-- neetodesk:start -->") {
		t.Fatalf("refreshed block should follow the rest of the file:\n%s", data)
	}
}

func TestWriteSectionCreatesParentDirectories(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	target := filepath.Join(t.TempDir(), ".github", "copilot-instructions.md")

	if err := a.writeSection(&bytes.Buffer{}, target, "body text"); err != nil {
		t.Fatalf("writeSection returned error: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected %s to be created: %v", target, err)
	}
}

func TestSectionMarkersFollowTheBinaryName(t *testing.T) {
	a, _ := newTestApp(t, singleHostProduct())

	start, end := a.sectionMarkers()
	if start != "<!-- neetodeploy:start -->" || end != "<!-- neetodeploy:end -->" {
		t.Errorf("markers = %q / %q", start, end)
	}
}

func TestWriteRuleFileCreatesFileWithContent(t *testing.T) {
	target := filepath.Join(t.TempDir(), ".cursor", "rules", "rules.mdc")

	var out bytes.Buffer
	if err := writeRuleFile(&out, target, "first rules"); err != nil {
		t.Fatalf("writeRuleFile returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected %s to be created: %v", target, err)
	}
	if string(data) != "first rules" {
		t.Fatalf("unexpected content: %q", data)
	}
	if !strings.Contains(out.String(), "Wrote") {
		t.Fatalf("first run should report the file was written, got: %s", out.String())
	}
}

func TestWriteRuleFileOverwritesExistingFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "rules.md")
	if err := writeRuleFile(&bytes.Buffer{}, target, "old rules"); err != nil {
		t.Fatalf("first run error: %v", err)
	}

	var out bytes.Buffer
	if err := writeRuleFile(&out, target, "new rules"); err != nil {
		t.Fatalf("second run error: %v", err)
	}

	data, _ := os.ReadFile(target)
	if string(data) != "new rules" {
		t.Fatalf("re-run should overwrite the file, got: %q", data)
	}
	if !strings.Contains(out.String(), "Updated") {
		t.Fatalf("re-run should report the file was updated, got: %s", out.String())
	}
}

func TestRuleContentsCarryTheSkillBodyAndFrontmatter(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	cursor := a.cursorContent()
	if !strings.HasPrefix(cursor, "---\ndescription: \"NeetoDesk CLI\"\nalwaysApply: true\n---\n\n") {
		t.Errorf("unexpected cursor frontmatter:\n%s", cursor)
	}
	if !strings.Contains(cursor, "Use the neetodesk CLI to manage tickets.") {
		t.Errorf("cursor rules should carry the skill body:\n%s", cursor)
	}
	if strings.Contains(cursor, "name: neetodesk") {
		t.Errorf("the skill's own frontmatter must be stripped:\n%s", cursor)
	}

	windsurf := a.windsurfContent()
	if !strings.HasPrefix(windsurf, "---\ntrigger: always_on\ndescription: \"NeetoDesk CLI\"\n---\n\n") {
		t.Errorf("unexpected windsurf frontmatter:\n%s", windsurf)
	}
}

func TestSetupClaudeExtractsThePlugin(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	home := os.Getenv("HOME")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := run(t, a, "setup", "claude"); err != nil {
		t.Fatalf("setup claude: %v", err)
	}

	dest := filepath.Join(home, ".config", "neetodesk", "claude-plugin")
	for _, rel := range []string{
		filepath.Join(".claude-plugin", "plugin.json"),
		filepath.Join(".claude-plugin", "marketplace.json"),
		filepath.Join("skills", "neetodesk", "SKILL.md"),
	} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Errorf("expected %s in the extracted plugin: %v", rel, err)
		}
	}
	if !strings.Contains(out.String(), "/plugin install neetodesk@neetodesk") {
		t.Errorf("expected the install instructions, got:\n%s", out.String())
	}
}

func TestSetupClaudeRefusesWithoutClaudeCode(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	err := run(t, a, "setup", "claude")
	if err == nil || !strings.Contains(err.Error(), "Claude Code not found") {
		t.Fatalf("expected a missing-Claude-Code error, got %v", err)
	}
}
