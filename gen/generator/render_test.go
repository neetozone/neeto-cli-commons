package generator

import (
	"path/filepath"
	"testing"

	"github.com/neetozone/neeto-cli-commons/gen/vars"
)

func TestRenderPath(t *testing.T) {
	v := varsFor("testapp")

	cases := map[string]string{
		"cmd/__BINARY__/main.go.tmpl":        filepath.Join("cmd", "testapp", "main.go"),
		"internal/commands/register.go.tmpl": filepath.Join("internal", "commands", "register.go"),
		"skills/__BINARY__/SKILL.md.tmpl":    filepath.Join("skills", "testapp", "SKILL.md"),
		".goreleaser.yml.tmpl":               ".goreleaser.yml",
		".gitignore.tmpl":                    ".gitignore",
		".neeto-cli.yml.tmpl":                ".neeto-cli.yml",
	}

	for in, want := range cases {
		if got := renderPath(in, v); got != want {
			t.Errorf("renderPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderPath_MultipleBinaryPlaceholders(t *testing.T) {
	v := varsFor("acme")
	got := renderPath("a/__BINARY__/b/__BINARY__/c.go.tmpl", v)
	want := filepath.Join("a", "acme", "b", "acme", "c.go")
	if got != want {
		t.Errorf("renderPath = %q, want %q", got, want)
	}
}

func varsFor(binary string) *vars.Variables {
	v := &vars.Variables{}
	v.BinaryName = binary
	return v
}
