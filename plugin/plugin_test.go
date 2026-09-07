package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

const skillFixture = `---
name: neetodesk
description: Manage NeetoDesk from the command line.
---

## Prerequisites

Run ` + "`neetodesk doctor`" + ` to check authentication.
`

func product(t *testing.T, pretty string) config.Product {
	t.Helper()
	p := config.Product{PrettyName: pretty, SkillMD: []byte(skillFixture)}
	p.ApplyDerivations()
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	return p
}

func TestExtractRoundTrip(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	dest := t.TempDir()
	if err := pl.Extract(dest); err != nil {
		t.Fatalf("Extract() = %v", err)
	}

	wantPluginJSON := `{
  "name": "neetodesk",
  "description": "NeetoDesk CLI",
  "author": {
    "name": "BigBinary",
    "email": "support@bigbinary.com"
  },
  "homepage": "https://github.com/neetozone/neeto-desk-cli",
  "repository": "https://github.com/neetozone/neeto-desk-cli",
  "license": "MIT"
}
`
	wantMarketplaceJSON := `{
  "name": "neetodesk",
  "owner": {
    "name": "BigBinary",
    "email": "support@bigbinary.com"
  },
  "plugins": [
    {
      "name": "neetodesk",
      "source": "./",
      "description": "NeetoDesk CLI"
    }
  ]
}
`
	wantHooksJSON := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
`
	wantSessionStart := `#!/bin/sh
# NeetoDesk CLI — session-start hook for Claude Code
# Lightweight auth liveness check. Always exits 0 (informational).

if ! command -v neetodesk >/dev/null 2>&1; then
  echo "NeetoDesk CLI is not installed or not on PATH."
  exit 0
fi

if neetodesk whoami >/dev/null 2>&1; then
  echo "NeetoDesk plugin active."
else
  echo "NeetoDesk CLI installed but not authenticated. Run 'neetodesk login' to authenticate."
fi

exit 0
`
	wantDoctor := `---
name: neetodesk-doctor
description: Check NeetoDesk CLI health — auth, API connectivity.
invocable: true
---

Run ` + "`neetodesk doctor`" + ` and report the results to the user.
`

	want := map[string]string{
		".claude-plugin/plugin.json":      wantPluginJSON,
		".claude-plugin/marketplace.json": wantMarketplaceJSON,
		"hooks/hooks.json":                wantHooksJSON,
		"hooks/session-start.sh":          wantSessionStart,
		"commands/doctor.md":              wantDoctor,
		"skills/neetodesk/SKILL.md":       skillFixture,
	}

	for rel, body := range want {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("ReadFile(%s) = %v", rel, err)
		}
		if string(got) != body {
			t.Errorf("%s =\n%q\nwant\n%q", rel, got, body)
		}
	}

	var found []string
	err := filepath.WalkDir(dest, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dest, path)
		if relErr != nil {
			return relErr
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() = %v", err)
	}
	if len(found) != len(want) {
		t.Errorf("Extract() wrote %v, want exactly %d files", found, len(want))
	}
	for _, rel := range found {
		if _, ok := want[rel]; !ok {
			t.Errorf("Extract() wrote unexpected file %s", rel)
		}
	}
}

func TestExtractMarksShellHooksExecutable(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	dest := t.TempDir()
	if err := pl.Extract(dest); err != nil {
		t.Fatalf("Extract() = %v", err)
	}

	shells, err := filepath.Glob(filepath.Join(dest, "hooks", "*.sh"))
	if err != nil {
		t.Fatalf("Glob() = %v", err)
	}
	if len(shells) == 0 {
		t.Fatal("Extract() wrote no hooks/*.sh")
	}
	for _, path := range shells {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("Stat(%s) = %v", path, statErr)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("%s mode = %v, want 0755", path, info.Mode().Perm())
		}
	}

	info, err := os.Stat(filepath.Join(dest, "hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("Stat(hooks.json) = %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("hooks.json mode = %v, want 0644", info.Mode().Perm())
	}
}

func TestExtractOverwritesAnExistingTree(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	dest := t.TempDir()
	if err := pl.Extract(dest); err != nil {
		t.Fatalf("first Extract() = %v", err)
	}
	stale := filepath.Join(dest, "hooks", "session-start.sh")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v", err)
	}
	if err := pl.Extract(dest); err != nil {
		t.Fatalf("second Extract() = %v", err)
	}
	info, err := os.Stat(stale)
	if err != nil {
		t.Fatalf("Stat() = %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("re-extracted hook mode = %v, want 0755", info.Mode().Perm())
	}
	body, err := os.ReadFile(stale)
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	if string(body) == "stale" {
		t.Error("Extract() did not overwrite the stale hook")
	}
}

func TestPluginJSONURLsFollowModulePath(t *testing.T) {
	pl := New(product(t, "NeetoKB"))

	data, err := pl.PluginJSON()
	if err != nil {
		t.Fatalf("PluginJSON() = %v", err)
	}
	var got struct {
		Name       string `json:"name"`
		Homepage   string `json:"homepage"`
		Repository string `json:"repository"`
		License    string `json:"license"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}

	const want = "https://github.com/neetozone/neeto-kb-cli"
	if got.Homepage != want {
		t.Errorf("homepage = %q, want %q", got.Homepage, want)
	}
	if got.Repository != want {
		t.Errorf("repository = %q, want %q", got.Repository, want)
	}
	if got.Name != "neetokb" {
		t.Errorf("name = %q, want neetokb", got.Name)
	}
	if got.License != "MIT" {
		t.Errorf("license = %q, want MIT", got.License)
	}
	if strings.Contains(string(data), "neetokb-cli") {
		t.Errorf("plugin.json still carries the broken neetokb-cli repo name:\n%s", data)
	}
}

func TestPluginJSONHonoursAnExplicitModulePath(t *testing.T) {
	p := product(t, "NeetoDeploy")
	p.RepoName = "neeto-deploy-cli-go"
	p.ModulePath = "github.com/neetozone/neeto-deploy-cli-go"
	pl := New(p)

	data, err := pl.PluginJSON()
	if err != nil {
		t.Fatalf("PluginJSON() = %v", err)
	}
	if !strings.Contains(string(data), `"homepage": "https://github.com/neetozone/neeto-deploy-cli-go"`) {
		t.Errorf("plugin.json = %s", data)
	}
}

func TestDescriptionPrefersTheProductShortDescription(t *testing.T) {
	p := product(t, "NeetoCal")
	p.ShortDescription = "NeetoCal integration for Claude Code. Manage meetings, bookings, and scheduling."
	pl := New(p)

	data, err := pl.MarketplaceJSON()
	if err != nil {
		t.Fatalf("MarketplaceJSON() = %v", err)
	}
	if !strings.Contains(string(data), p.ShortDescription) {
		t.Errorf("marketplace.json = %s", data)
	}
}

func TestSkillBodyStripsFrontmatter(t *testing.T) {
	cases := map[string]struct {
		in   string
		want string
	}{
		"frontmatter": {
			in:   "---\nname: neetodesk\n---\n\n# Body\n",
			want: "# Body\n",
		},
		"no frontmatter": {
			in:   "# Body\n\nSecond line.\n",
			want: "# Body\n\nSecond line.\n",
		},
		"unterminated frontmatter": {
			in:   "---\nname: neetodesk\n",
			want: "---\nname: neetodesk\n",
		},
		"horizontal rule in body": {
			in:   "Intro.\n\n---\n\nOutro.\n",
			want: "Intro.\n\n---\n\nOutro.\n",
		},
		"empty": {
			in:   "",
			want: "",
		},
	}
	for name, tc := range cases {
		if got := SkillBody([]byte(tc.in)); got != tc.want {
			t.Errorf("%s: SkillBody() = %q, want %q", name, got, tc.want)
		}
	}
}

func TestSkillBodyMethodReadsProductSkillMD(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	body := pl.SkillBody()
	if strings.HasPrefix(body, "---") {
		t.Errorf("SkillBody() kept the frontmatter: %q", body)
	}
	if !strings.HasPrefix(body, "## Prerequisites") {
		t.Errorf("SkillBody() = %q", body)
	}
}

func TestFilesFailsWithoutSkillContent(t *testing.T) {
	p := product(t, "NeetoDesk")
	p.SkillMD = nil
	pl := New(p)

	if _, err := pl.Files(); err == nil {
		t.Fatal("Files() should fail when skill content is missing")
	}
	if err := pl.Extract(t.TempDir()); err == nil {
		t.Fatal("Extract() should fail when skill content is missing")
	}
}

func TestExtractClaudePluginAcceptsSkillBytes(t *testing.T) {
	p := product(t, "NeetoDesk")
	p.SkillMD = nil
	dest := t.TempDir()

	if err := ExtractClaudePlugin(p, []byte(skillFixture), dest); err != nil {
		t.Fatalf("ExtractClaudePlugin() = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "skills", "neetodesk", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}
	if string(got) != skillFixture {
		t.Errorf("SKILL.md = %q", got)
	}
}

func TestFilesMatchTheIndividualArtefacts(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	files, err := pl.Files()
	if err != nil {
		t.Fatalf("Files() = %v", err)
	}

	byPath := map[string][]byte{}
	for _, f := range files {
		byPath[filepath.ToSlash(f.Path)] = f.Data
	}

	pluginJSON, err := pl.PluginJSON()
	if err != nil {
		t.Fatalf("PluginJSON() = %v", err)
	}
	marketplaceJSON, err := pl.MarketplaceJSON()
	if err != nil {
		t.Fatalf("MarketplaceJSON() = %v", err)
	}
	hooksJSON, err := pl.HooksJSON()
	if err != nil {
		t.Fatalf("HooksJSON() = %v", err)
	}

	want := map[string]string{
		".claude-plugin/plugin.json":      string(pluginJSON),
		".claude-plugin/marketplace.json": string(marketplaceJSON),
		"hooks/hooks.json":                string(hooksJSON),
		"hooks/session-start.sh":          pl.SessionStartScript(),
		"commands/doctor.md":              pl.DoctorCommandMD(),
		"skills/neetodesk/SKILL.md":       string(pl.SkillMD()),
	}
	for rel, body := range want {
		if string(byPath[rel]) != body {
			t.Errorf("Files()[%s] = %q, want %q", rel, byPath[rel], body)
		}
	}
}

func TestManifestsAreNewlineTerminated(t *testing.T) {
	pl := New(product(t, "NeetoDesk"))
	for _, build := range []func() ([]byte, error){pl.PluginJSON, pl.MarketplaceJSON, pl.HooksJSON} {
		data, err := build()
		if err != nil {
			t.Fatalf("build() = %v", err)
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			t.Errorf("manifest is not newline terminated: %q", data)
		}
	}
}

func TestNameAndMarketplaceNameTrackTheBinary(t *testing.T) {
	pl := New(product(t, "NeetoPlaydash"))
	if pl.Name() != "neetoplaydash" {
		t.Errorf("Name() = %q", pl.Name())
	}
	if pl.MarketplaceName() != "neetoplaydash" {
		t.Errorf("MarketplaceName() = %q", pl.MarketplaceName())
	}
	if pl.SkillPath() != filepath.Join("skills", "neetoplaydash", "SKILL.md") {
		t.Errorf("SkillPath() = %q", pl.SkillPath())
	}
}
