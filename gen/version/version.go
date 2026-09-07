// Package version exposes build-time metadata injected via ldflags and the
// fallback semver from VERSION at the repo root (baked in at build time).
package version

import (
	_ "embed"
	"strings"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

//go:embed embedded_version.txt
var embeddedVersion string

// Effective returns Version when an ldflag-injected value is present, else
// the semver baked in from VERSION at build time.
func Effective() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	return strings.TrimSpace(embeddedVersion)
}
