package render_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/neetozone/neeto-cli-commons/render"
)

func product(t *testing.T, extra string) config.Product {
	t.Helper()
	yaml := extra
	if !strings.Contains(yaml, "pretty_name:") {
		yaml = "pretty_name: NeetoCal\nbinary_name: neetocal\nshort_description: NeetoCal CLI\n" + yaml
	}
	p, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return *p
}

func render1(t *testing.T, p config.Product, name string) string {
	t.Helper()
	b, err := render.File(p, name)
	if err != nil {
		t.Fatalf("File(%q): %v", name, err)
	}
	return string(b)
}

func TestAllProducesEveryRepoFile(t *testing.T) {
	files, err := render.All(product(t, ""))
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	want := []string{
		".claude-plugin/marketplace.json", ".claude-plugin/plugin.json",
		".githooks/pre-commit", ".gitignore", ".golangci.yml", ".goreleaser.yml",
		".neetoci/release.yml", ".neetoci/verify.yml",
		"Makefile", "bin/setup", "commands/doctor.md",
		"docs/adding-commands.md",
		"docs/api-wrapper-reference.md",
		"hooks/hooks.json", "hooks/session-start.sh",
		"installers/install.cmd", "installers/install.ps1", "installers/install.sh",
		"mise.toml",
	}
	got := make([]string, 0, len(files))
	for name := range files {
		got = append(got, name)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("All keys = %v, want %v", got, want)
	}
	if _, ok := files["README.md"]; ok {
		t.Error("All must not own README.md; only its marked sections are generated")
	}
}

func TestNoUnrenderedActionsRemain(t *testing.T) {
	p := product(t, "")
	for _, name := range render.Names() {
		body := render1(t, p, name)
		for _, field := range []string{
			".BinaryName", ".PrettyName", ".ModulePath", ".RepoName", ".Domain",
			".EnvPrefix", ".GoVersion", ".ShortDescription", ".S3PathPrefix",
			".HomebrewTap", ".LatestURL", ".BrewFormula", ".InstallDirEnvVar",
			"__BINARY__",
		} {
			if strings.Contains(body, field) {
				t.Errorf("%s still contains the unrendered placeholder %s", name, field)
			}
		}
	}
}

func TestGoreleaserTargetsMainVersionVars(t *testing.T) {
	body := render1(t, product(t, ""), "goreleaser.yml")
	for _, want := range []string{
		"-X main.version={{.Version}}",
		"-X main.commit={{.ShortCommit}}",
		"-X main.date={{.Date}}",
		"owner: neetozone",
		"name: homebrew-tap",
		"description: \"NeetoCal CLI\"",
		"bin.install \"neetocal\"",
		"directory: \"cli/NeetoCal/v{{ .Version }}\"",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("goreleaser.yml is missing %q", want)
		}
	}
	if strings.Contains(body, "internal/commands") {
		t.Error("goreleaser.yml still points ldflags at internal/commands")
	}
}

func TestMakefileTargetsMainVersionVars(t *testing.T) {
	body := render1(t, product(t, ""), "makefile")
	if !strings.Contains(body, "-X main.version=$(VERSION)") {
		t.Error("Makefile ldflags do not target main.version")
	}
	if strings.Contains(body, "internal/commands") {
		t.Error("Makefile still points ldflags at internal/commands")
	}
}

func TestInstallersUseTheS3LatestURL(t *testing.T) {
	p := product(t, "")
	want := "https://neeto-downloads.s3.amazonaws.com/cli/NeetoCal/latest"
	for _, name := range []string{"installers/install.sh", "installers/install.ps1", "installers/install.cmd"} {
		body := render1(t, p, name)
		if !strings.Contains(body, want) {
			t.Errorf("%s does not point at %s", name, want)
		}
		if strings.Contains(body, "https://neetocal.com/cli/") {
			t.Errorf("%s still points at the product domain, which 404s for several products", name)
		}
	}
	sh := render1(t, p, "installers/install.sh")
	if !strings.Contains(sh, `INSTALL_DIR="${NEETOCAL_INSTALL_DIR:-/usr/local/bin}"`) {
		t.Error("install.sh did not render the install-dir env var")
	}
	if !strings.Contains(sh, "SHA256SUMS") {
		t.Error("install.sh lost its checksum verification")
	}
}

func TestReadmeInstallationUsesTheInstallableBrewFormula(t *testing.T) {
	body := render1(t, product(t, ""), "readme-sections/installation")
	if !strings.Contains(body, "brew install neetozone/tap/neetocal") {
		t.Error("installation section does not use the installable tap form")
	}
	if strings.Contains(body, "homebrew-tap/") || strings.Contains(body, "brew trust") {
		t.Error("installation section still carries the old brew lines")
	}
	if !strings.Contains(body, "curl -fsSL https://neeto-downloads.s3.amazonaws.com/cli/NeetoCal/latest/install.sh | sh") {
		t.Error("installation section does not use the S3 install URL")
	}
}

func TestReadmeReleaseDescribesThePushToMain(t *testing.T) {
	body := render1(t, product(t, ""), "readme-sections/release")
	if strings.Contains(body, "bump PR") {
		t.Error("release section still claims the pipeline opens a bump PR")
	}
	if !strings.Contains(body, "pushes the version bump commit\nstraight to `main`") {
		t.Error("release section does not describe the push to main")
	}
}

func TestReadmePrerequisitesUseTheConfiguredGoVersion(t *testing.T) {
	body := render1(t, product(t, "go_version: \"1.27.3\"\n"), "readme-sections/prerequisites")
	if !strings.Contains(body, "1.27.3+") {
		t.Error("prerequisites section does not use .GoVersion")
	}
}

func TestGlobalFlagsDropSubdomainForSingleHost(t *testing.T) {
	single := render1(t, product(t, "tenancy: single_host\napi_host: app.neetodeploy.com\n"), "readme-sections/global-flags")
	if strings.Contains(single, "--subdomain") {
		t.Error("single_host global flags must not advertise --subdomain")
	}
	if strings.Contains(single, "|---|---|\n\n|") {
		t.Error("single_host global flags left a blank line inside the table")
	}
	multi := render1(t, product(t, ""), "readme-sections/global-flags")
	if !strings.Contains(multi, "--subdomain") {
		t.Error("subdomain global flags must advertise --subdomain")
	}
}

func TestBinSetupNamesTheRepository(t *testing.T) {
	body := render1(t, product(t, "pretty_name: NeetoKB\nbinary_name: neetokb\nshort_description: NeetoKB CLI\n"), "bin-setup")
	if !strings.Contains(body, "Setting up neeto-kb-cli for local development") {
		t.Error("bin/setup does not use the derived repo name")
	}
	if strings.Contains(body, "neetokb-cli") {
		t.Error("bin/setup still carries the neetokb-cli typo")
	}
}

func TestNeetociUsesVerifyAndTheReleaseShim(t *testing.T) {
	p := product(t, "commons_version: 1.4.0\n")
	verify := render1(t, p, "neetoci/verify.yml")
	if !strings.Contains(verify, "neetoci-version go 1.26") {
		t.Error("verify.yml does not derive the Go line from .GoVersion")
	}
	if !strings.Contains(verify, "neeto-cli-sync --check") {
		t.Error("verify.yml does not run the sync check")
	}
	if !strings.Contains(verify, "paths:") {
		t.Error("verify.yml lost its paths filter")
	}

	release := render1(t, p, "neetoci/release.yml")
	if !strings.Contains(release, "neeto-cli-commons/v1.4.0/assets/release.sh") {
		t.Error("release.yml does not pin the commons version it fetches")
	}
	if !strings.Contains(release, render.ReleaseScriptSHA256()) {
		t.Error("release.yml does not carry the release script digest")
	}
	if strings.Contains(release, ".scripts/release.sh") {
		t.Error("release.yml still runs a per-repo release script")
	}
	if !strings.Contains(release, `-H "Authorization: token ${GITHUB_TOKEN}"`) {
		t.Error("release.yml fetches the release script without a token; neeto-cli-commons is private, so raw.githubusercontent.com answers 404")
	}
}

func TestCommonsRefFallsBackToMain(t *testing.T) {
	body := render1(t, product(t, ""), "neetoci/release.yml")
	if !strings.Contains(body, "neeto-cli-commons/main/assets/release.sh") {
		t.Error("release.yml does not fall back to main when no commons_version is set")
	}
}

func TestReleaseScriptDigestMatchesSidecar(t *testing.T) {
	sidecar, err := os.ReadFile("../assets/release.sh.sha256")
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	want := strings.Fields(string(sidecar))
	if len(want) < 2 || want[1] != "release.sh" {
		t.Fatalf("sidecar is not a sha256sum line: %q", string(sidecar))
	}
	if want[0] != render.ReleaseScriptSHA256() {
		t.Errorf("assets/release.sh.sha256 is stale: %s, want %s", want[0], render.ReleaseScriptSHA256())
	}
	sum := sha256.Sum256(render.ReleaseScript())
	if hex.EncodeToString(sum[:]) != render.ReleaseScriptSHA256() {
		t.Error("ReleaseScriptSHA256 does not digest ReleaseScript")
	}
}

func TestReleaseScriptCarriesNoProductStrings(t *testing.T) {
	body := string(render.ReleaseScript())
	for _, banned := range []string{"NeetoCal", "neetocal", "NeetoDeploy", "neeto-cal-cli", "cli/Neeto"} {
		if strings.Contains(body, banned) {
			t.Errorf("release.sh hardcodes the product string %q", banned)
		}
	}
	for _, want := range []string{
		"Bump version to ",
		"GORELEASER_CURRENT_TAG",
		"TAP_GITHUB_TOKEN",
		".neeto-cli.yml",
		"--config \"$CONFIG_FILE\" --out \"$RENDER_DIR\"",
		"goreleaser release --clean --config",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("release.sh is missing %q", want)
		}
	}
	if strings.Contains(body, "apt install gh") {
		t.Error("release.sh still reinstalls gh from apt")
	}
	if strings.Contains(body, "go test ./...") {
		t.Error("release.sh still re-runs the test suite verify.yml already ran")
	}
	if strings.Contains(body, "latest/checksums.txt") {
		t.Error("release.sh still uploads checksums.txt to latest/")
	}
	if strings.Contains(body, "aws s3 rm \"${S3_BASE}/latest/\" --recursive") {
		t.Error("release.sh still empties latest/ before uploading")
	}
}

func TestMergeREADMEReplacesOnlyMarkedSections(t *testing.T) {
	p := product(t, "")
	existing := []byte("# NeetoCal CLI\n\nHouse prose.\n\n" +
		"<!-- neeto-cli-commons:release:start -->\nstale\n<!-- neeto-cli-commons:release:end -->\n\nMore prose.\n")
	merged, err := render.MergeREADME(p, existing)
	if err != nil {
		t.Fatalf("MergeREADME: %v", err)
	}
	got := string(merged)
	if !strings.Contains(got, "House prose.") || !strings.Contains(got, "More prose.") {
		t.Error("MergeREADME dropped prose outside the markers")
	}
	if strings.Contains(got, "stale") {
		t.Error("MergeREADME did not replace the marked section")
	}
	if !strings.Contains(got, "## Release") {
		t.Error("MergeREADME did not insert the rendered section")
	}
	if !strings.HasSuffix(got, "\n\nMore prose.\n") {
		t.Errorf("MergeREADME changed the spacing after the end marker: %q", got[len(got)-24:])
	}

	missing := render.MissingSections(existing)
	if len(missing) != len(render.SectionNames())-1 {
		t.Errorf("MissingSections = %v", missing)
	}

	again, err := render.MergeREADME(p, merged)
	if err != nil {
		t.Fatalf("MergeREADME (second pass): %v", err)
	}
	if string(again) != got {
		t.Error("MergeREADME is not idempotent")
	}
}

func TestMergeREADMERejectsAnUnclosedSection(t *testing.T) {
	_, err := render.MergeREADME(product(t, ""), []byte("<!-- neeto-cli-commons:release:start -->\n"))
	if err == nil {
		t.Error("MergeREADME accepted a section with no end marker")
	}
}

func TestIsExecutable(t *testing.T) {
	for _, p := range []string{"bin/setup", ".githooks/pre-commit", "installers/install.sh"} {
		if !render.IsExecutable(p) {
			t.Errorf("%s should be executable", p)
		}
	}
	for _, p := range []string{"Makefile", ".gitignore", "installers/install.ps1", "commands/doctor.md"} {
		if render.IsExecutable(p) {
			t.Errorf("%s should not be executable", p)
		}
	}
}

func TestShellEnvQuotesEveryValue(t *testing.T) {
	got := render.ShellEnv(product(t, ""))
	for _, want := range []string{
		"BINARY_NAME='neetocal'",
		"REPO_NAME='neeto-cal-cli'",
		"S3_BUCKET='neeto-downloads'",
		"S3_BASE='s3://neeto-downloads/cli/NeetoCal'",
		"S3_HTTPS_BASE='https://neeto-downloads.s3.amazonaws.com/cli/NeetoCal'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ShellEnv is missing %q, got:\n%s", want, got)
		}
	}
}

func TestFileRejectsAnUnknownAsset(t *testing.T) {
	if _, err := render.File(product(t, ""), "does-not-exist"); err == nil {
		t.Error("File accepted an unknown asset")
	}
}

func TestNamesCoverEverySection(t *testing.T) {
	names := render.Names()
	index := map[string]bool{}
	for _, n := range names {
		index[n] = true
	}
	for _, s := range render.SectionNames() {
		if !index["readme-sections/"+s] {
			t.Errorf("Names is missing readme-sections/%s", s)
		}
	}
	for _, n := range []string{"skill-frontmatter", "release.sh", "goreleaser.yml"} {
		if !index[n] {
			t.Errorf("Names is missing %s", n)
		}
	}
}

func TestAllDerivesPluginURLsFromTheModulePath(t *testing.T) {
	// neeto-kb-cli shipped plugin.json pointing at github.com/neetozone/neetokb-cli,
	// which 404s. Generating it from the module path makes that impossible.
	p := product(t, "pretty_name: NeetoKB\nbinary_name: neetokb\n")
	files, err := render.All(p)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	manifest := string(files[".claude-plugin/plugin.json"])
	if !strings.Contains(manifest, "github.com/neetozone/neeto-kb-cli") {
		t.Errorf("plugin.json does not carry the real repo URL:\n%s", manifest)
	}
	if strings.Contains(manifest, "neetokb-cli") {
		t.Errorf("plugin.json still carries the broken repo name:\n%s", manifest)
	}
}
