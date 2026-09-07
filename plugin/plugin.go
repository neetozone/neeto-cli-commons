package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neetozone/neeto-cli-commons/config"
)

const (
	sessionStartHookPath = "hooks/session-start.sh"
)

const sessionStartScriptTemplate = `#!/bin/sh
# {{pretty}} CLI — session-start hook for Claude Code
# Lightweight auth liveness check. Always exits 0 (informational).

if ! command -v {{binary}} >/dev/null 2>&1; then
  echo "{{pretty}} CLI is not installed or not on PATH."
  exit 0
fi

if {{binary}} whoami >/dev/null 2>&1; then
  echo "{{pretty}} plugin active."
else
  echo "{{pretty}} CLI installed but not authenticated. Run '{{binary}} login' to authenticate."
fi

exit 0
`

const doctorCommandTemplate = `---
name: {{binary}}-doctor
description: Check {{pretty}} CLI health — auth, API connectivity.
invocable: true
---

Run ` + "`{{binary}} doctor`" + ` and report the results to the user.
`

type owner struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type manifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      owner  `json:"author"`
	Homepage    string `json:"homepage"`
	Repository  string `json:"repository"`
	License     string `json:"license,omitempty"`
}

type marketplaceEntry struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type marketplace struct {
	Name    string             `json:"name"`
	Owner   owner              `json:"owner"`
	Plugins []marketplaceEntry `json:"plugins"`
}

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type hookMatcher struct {
	Hooks []hookCommand `json:"hooks"`
}

type hookEvents struct {
	SessionStart []hookMatcher `json:"SessionStart"`
}

type hooksManifest struct {
	Hooks hookEvents `json:"hooks"`
}

type File struct {
	Path string
	Mode os.FileMode
	Data []byte
}

type Plugin struct {
	product config.Product
	skillMD []byte
}

func New(p config.Product) *Plugin {
	return &Plugin{product: p, skillMD: p.SkillMD}
}

func SkillBody(skillMD []byte) string {
	content := string(skillMD)
	if strings.HasPrefix(content, "---") {
		if idx := strings.Index(content[3:], "---"); idx != -1 {
			content = strings.TrimLeft(content[3+idx+3:], "\n")
		}
	}
	return content
}

func ExtractClaudePlugin(p config.Product, skillMD []byte, dest string) error {
	pl := New(p)
	if len(skillMD) > 0 {
		pl.skillMD = skillMD
	}
	return pl.Extract(dest)
}

func (pl *Plugin) Name() string { return pl.product.BinaryName }

func (pl *Plugin) MarketplaceName() string { return pl.product.BinaryName }

func (pl *Plugin) SkillBody() string { return SkillBody(pl.skillMD) }

func (pl *Plugin) SkillMD() []byte { return pl.skillMD }

func (pl *Plugin) SkillPath() string {
	return filepath.Join("skills", pl.Name(), "SKILL.md")
}

func (pl *Plugin) PluginJSON() ([]byte, error) {
	return marshal(manifest{
		Name:        pl.Name(),
		Description: pl.description(),
		Author:      pl.owner(),
		Homepage:    pl.repositoryURL(),
		Repository:  pl.repositoryURL(),
		License:     pl.product.License,
	})
}

func (pl *Plugin) MarketplaceJSON() ([]byte, error) {
	return marshal(marketplace{
		Name:  pl.MarketplaceName(),
		Owner: pl.owner(),
		Plugins: []marketplaceEntry{{
			Name:        pl.Name(),
			Source:      "./",
			Description: pl.description(),
		}},
	})
}

func (pl *Plugin) HooksJSON() ([]byte, error) {
	return marshal(hooksManifest{
		Hooks: hookEvents{
			SessionStart: []hookMatcher{{
				Hooks: []hookCommand{{
					Type:    "command",
					Command: "${CLAUDE_PLUGIN_ROOT}/" + sessionStartHookPath,
					Timeout: 5,
				}},
			}},
		},
	})
}

func (pl *Plugin) SessionStartScript() string {
	return pl.substitute(sessionStartScriptTemplate)
}

func (pl *Plugin) DoctorCommandMD() string {
	return pl.substitute(doctorCommandTemplate)
}

func (pl *Plugin) Files() ([]File, error) {
	if len(pl.skillMD) == 0 {
		return nil, fmt.Errorf("Could not build the %s plugin: skill content is empty", pl.Name())
	}

	pluginJSON, err := pl.PluginJSON()
	if err != nil {
		return nil, err
	}
	marketplaceJSON, err := pl.MarketplaceJSON()
	if err != nil {
		return nil, err
	}
	hooksJSON, err := pl.HooksJSON()
	if err != nil {
		return nil, err
	}

	return []File{
		{Path: filepath.Join(".claude-plugin", "plugin.json"), Mode: 0o644, Data: pluginJSON},
		{Path: filepath.Join(".claude-plugin", "marketplace.json"), Mode: 0o644, Data: marketplaceJSON},
		{Path: filepath.FromSlash(sessionStartHookPath), Mode: 0o755, Data: []byte(pl.SessionStartScript())},
		{Path: filepath.Join("hooks", "hooks.json"), Mode: 0o644, Data: hooksJSON},
		{Path: filepath.Join("commands", "doctor.md"), Mode: 0o644, Data: []byte(pl.DoctorCommandMD())},
		{Path: pl.SkillPath(), Mode: 0o644, Data: pl.skillMD},
	}, nil
}

func (pl *Plugin) Extract(dest string) error {
	files, err := pl.Files()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		if err := writeFile(filepath.Join(dest, f.Path), f.Data, f.Mode); err != nil {
			return err
		}
	}
	return nil
}

func (pl *Plugin) owner() owner {
	return owner{Name: pl.product.CompanyName, Email: pl.product.SupportEmail}
}

func (pl *Plugin) repositoryURL() string { return "https://" + pl.product.ModulePath }

func (pl *Plugin) description() string {
	if pl.product.ShortDescription != "" {
		return pl.product.ShortDescription
	}
	return pl.product.PrettyName + " CLI"
}

func (pl *Plugin) substitute(body string) string {
	return strings.NewReplacer(
		"{{binary}}", pl.product.BinaryName,
		"{{pretty}}", pl.product.PrettyName,
	).Replace(body)
}

func marshal(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}
