// Package render turns a config.Product into the concrete repository files
// every neeto CLI ships. It is the single source of truth for the installers,
// the CI pipelines, the build config and the machine-generated README sections.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/neetozone/neeto-cli-commons/assets"
	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/neetozone/neeto-cli-commons/plugin"
)

const markerPrefix = "neeto-cli-commons:"

const releaseScriptAsset = "release.sh"

type target struct {
	asset string
	dest  string
}

var repoFiles = []target{
	{"goreleaser.yml", ".goreleaser.yml"},
	{"gitignore", ".gitignore"},
	{"golangci.yml", ".golangci.yml"},
	{"makefile", "Makefile"},
	{"mise.toml", "mise.toml"},
	{"bin-setup", "bin/setup"},
	{"githooks/pre-commit", ".githooks/pre-commit"},
	{"neetoci/verify.yml", ".neetoci/verify.yml"},
	{"neetoci/release.yml", ".neetoci/release.yml"},
	{"installers/install.sh", "installers/install.sh"},
	{"installers/install.ps1", "installers/install.ps1"},
	{"installers/install.cmd", "installers/install.cmd"},
	{"commands/doctor.md", "commands/doctor.md"},
	{"docs/adding-commands.md", "docs/adding-commands.md"},
	{"docs/api-wrapper-reference.md", "docs/api-wrapper-reference.md"},
}

// All renders every file this package owns in a product repository, keyed by
// its repository-relative path. README.md is not included: only the marked
// sections inside it are generated, so it is merged through MergeREADME.
func All(p config.Product) (map[string][]byte, error) {
	out := make(map[string][]byte, len(repoFiles))
	for _, t := range repoFiles {
		data, err := File(p, t.asset)
		if err != nil {
			return nil, err
		}
		out[t.dest] = data
	}

	// The Claude plugin manifests and hooks are owned by the plugin package so
	// that the copies on disk and the ones `setup claude` emits come from one
	// source. They were hand-maintained before, and had already drifted.
	pl := plugin.New(p)
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
	out[".claude-plugin/plugin.json"] = pluginJSON
	out[".claude-plugin/marketplace.json"] = marketplaceJSON
	out["hooks/hooks.json"] = hooksJSON
	out["hooks/session-start.sh"] = []byte(pl.SessionStartScript())

	return out, nil
}

// File renders one asset by name. Names are the asset's path under assets/
// without the .tmpl suffix, for example "installers/install.sh",
// "readme-sections/release" or "skill-frontmatter".
func File(p config.Product, name string) ([]byte, error) {
	if name == releaseScriptAsset {
		return ReleaseScript(), nil
	}

	raw, err := fs.ReadFile(assets.FS, name+".tmpl")
	if err != nil {
		return nil, fmt.Errorf("Unknown asset %q: %w", name, err)
	}

	tmpl, err := template.New(name).Funcs(funcs()).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("Could not parse asset %q: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, p); err != nil {
		return nil, fmt.Errorf("Could not render asset %q: %w", name, err)
	}
	return buf.Bytes(), nil
}

// Names lists every asset File accepts, sorted.
func Names() []string {
	var names []string
	err := fs.WalkDir(assets.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".tmpl") {
			names = append(names, strings.TrimSuffix(p, ".tmpl"))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	names = append(names, releaseScriptAsset)
	sort.Strings(names)
	return names
}

// SectionNames lists the machine-generated README sections, in the order they
// appear in a freshly generated README.
func SectionNames() []string {
	return []string{
		"installation",
		"verify-installation",
		"prerequisites",
		"make-targets",
		"global-flags",
		"release",
		"ai-coding-assistants",
	}
}

// MergeREADME replaces every marked section of an existing README with the
// freshly rendered one. Prose outside the markers, and sections whose markers
// are absent, are left untouched.
func MergeREADME(p config.Product, existing []byte) ([]byte, error) {
	out := existing
	for _, name := range SectionNames() {
		block, err := File(p, "readme-sections/"+name)
		if err != nil {
			return nil, err
		}
		start, end := markers(name)
		i := bytes.Index(out, start)
		if i < 0 {
			continue
		}
		j := bytes.Index(out[i:], end)
		if j < 0 {
			return nil, fmt.Errorf("README section %q has a start marker but no end marker", name)
		}
		j += i + len(end)

		merged := make([]byte, 0, len(out)+len(block))
		merged = append(merged, out[:i]...)
		merged = append(merged, bytes.TrimRight(block, "\n")...)
		merged = append(merged, out[j:]...)
		out = merged
	}
	return out, nil
}

// MissingSections reports the generated sections an existing README carries no
// markers for, so a sync can tell the user what it left alone.
func MissingSections(existing []byte) []string {
	var missing []string
	for _, name := range SectionNames() {
		start, _ := markers(name)
		if !bytes.Contains(existing, start) {
			missing = append(missing, name)
		}
	}
	return missing
}

// ReleaseScript is the product-agnostic release script every repo runs from CI.
func ReleaseScript() []byte {
	data, err := fs.ReadFile(assets.FS, releaseScriptAsset)
	if err != nil {
		panic(err)
	}
	return data
}

// ReleaseScriptSHA256 is the digest the CI shim checks the fetched script
// against.
func ReleaseScriptSHA256() string {
	sum := sha256.Sum256(ReleaseScript())
	return hex.EncodeToString(sum[:])
}

// IsExecutable reports whether a rendered file must land on disk as 0755.
func IsExecutable(repoPath string) bool {
	switch repoPath {
	case "bin/setup", ".githooks/pre-commit":
		return true
	}
	return strings.HasSuffix(repoPath, ".sh")
}

// ShellEnv renders the product values the release script needs as POSIX shell
// assignments, so no shell has to parse .neeto-cli.yml itself.
func ShellEnv(p config.Product) string {
	pairs := [][2]string{
		{"BINARY_NAME", p.BinaryName},
		{"REPO_NAME", p.RepoName},
		{"PRETTY_NAME", p.PrettyName},
		{"S3_BUCKET", p.S3Bucket},
		{"S3_PREFIX", p.S3PathPrefix},
		{"S3_BASE", "s3://" + p.S3Bucket + "/" + p.S3PathPrefix},
		{"S3_HTTPS_BASE", p.S3URLBase()},
	}
	var b strings.Builder
	for _, kv := range pairs {
		fmt.Fprintf(&b, "%s=%s\n", kv[0], shellQuote(kv[1]))
	}
	return b.String()
}

func markers(name string) (start, end []byte) {
	return []byte("<!-- " + markerPrefix + name + ":start -->"),
		[]byte("<!-- " + markerPrefix + name + ":end -->")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func funcs() template.FuncMap {
	return template.FuncMap{
		"tapOwner": func(p config.Product) string {
			owner, _ := splitTap(p.HomebrewTap)
			return owner
		},
		"tapRepo": func(p config.Product) string {
			_, repo := splitTap(p.HomebrewTap)
			return repo
		},
		"goMinor":          goMinor,
		"commonsRawBase":   commonsRawBase,
		"releaseScriptSHA": ReleaseScriptSHA256,
	}
}

// splitTap turns the installable tap name ("neetozone/tap") back into the
// GitHub coordinates GoReleaser needs ("neetozone", "homebrew-tap").
func splitTap(tap string) (owner, repo string) {
	owner, repo, found := strings.Cut(tap, "/")
	if !found || repo == "" {
		return owner, "homebrew-tap"
	}
	if !strings.HasPrefix(repo, "homebrew-") {
		repo = "homebrew-" + repo
	}
	return owner, repo
}

func goMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}

var bareSemver = regexp.MustCompile(`^\d+\.\d+`)

func commonsRawBase(p config.Product) string {
	ref := p.CommonsVersion
	switch {
	case ref == "":
		ref = "main"
	case bareSemver.MatchString(ref):
		// "1.2.0" is a version; "main" or "v1.2.0" is already a ref.
		ref = "v" + ref
	}
	return "https://raw.githubusercontent.com/neetozone/neeto-cli-commons/" + ref
}
