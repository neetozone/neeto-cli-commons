package generator

import "embed"

// templateFS carries the genericized CLI skeleton. `all:` so dotfiles
// (.goreleaser.yml, .githooks, .neetoci, .claude-plugin, .scripts,
// .gitignore) and the underscore-prefixed root are included.
//
// The directory is named _template (underscore prefix) so Go's build tooling
// ignores the .go files inside — they are source for the generated repo,
// not for this generator.
//
//go:embed all:_template
var templateFS embed.FS
