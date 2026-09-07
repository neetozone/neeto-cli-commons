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

func Effective() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	return strings.TrimSpace(embeddedVersion)
}
