package assets

import "embed"

//go:embed bin-setup.tmpl gitignore.tmpl golangci.yml.tmpl goreleaser.yml.tmpl makefile.tmpl
//go:embed mise.toml.tmpl skill-frontmatter.tmpl release.sh release.sh.sha256
//go:embed commands docs githooks installers neetoci readme-sections
var FS embed.FS
