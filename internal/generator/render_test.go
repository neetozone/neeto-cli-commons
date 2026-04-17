package generator

import (
	"path/filepath"
	"testing"

	"github.com/neetozone/neeto-cli-template/internal/vars"
)

func TestRenderPath(t *testing.T) {
	v := &vars.Variables{BinaryName: "testapp"}

	cases := map[string]string{
		"cmd/__BINARY__/main.go.tmpl":     filepath.Join("cmd", "testapp", "main.go"),
		"internal/auth/auth.go.tmpl":      filepath.Join("internal", "auth", "auth.go"),
		"internal/client/pagination.go":   filepath.Join("internal", "client", "pagination.go"),
		"skills/__BINARY__/SKILL.md.tmpl": filepath.Join("skills", "testapp", "SKILL.md"),
		".goreleaser.yml.tmpl":            ".goreleaser.yml",
		".gitignore.tmpl":                 ".gitignore",
		".neetoci/default.yml":            filepath.Join(".neetoci", "default.yml"),
	}

	for in, want := range cases {
		if got := renderPath(in, v); got != want {
			t.Errorf("renderPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderPath_MultipleBinaryPlaceholders(t *testing.T) {
	v := &vars.Variables{BinaryName: "acme"}
	got := renderPath("a/__BINARY__/b/__BINARY__/c.go.tmpl", v)
	want := filepath.Join("a", "acme", "b", "acme", "c.go")
	if got != want {
		t.Errorf("renderPath = %q, want %q", got, want)
	}
}
