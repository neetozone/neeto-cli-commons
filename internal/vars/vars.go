// Package vars defines the template variable struct and default derivations
// used by both the questionnaire and the generator. It is the single source
// of truth for which values the template consumes.
package vars

import (
	"fmt"
	"regexp"
	"strings"
)

// Variables holds every value the template files reference. Fields are
// grouped by purpose; see `Defaults` for how blank fields are derived.
type Variables struct {
	// Identity.
	PrettyName string `yaml:"pretty_name"`
	BinaryName string `yaml:"binary_name"`

	// Module / repo.
	GithubOrg  string `yaml:"github_org"`
	RepoName   string `yaml:"repo_name"`
	ModulePath string `yaml:"module_path"`

	// Company / metadata.
	CompanyName      string `yaml:"company_name"`
	SupportEmail     string `yaml:"support_email"`
	ShortDescription string `yaml:"short_description"`
	LongDescription  string `yaml:"long_description"`

	// Runtime integrations.
	Domain      string `yaml:"domain"`
	ApiBasePath string `yaml:"api_base_path"`
	EnvVar      string `yaml:"env_var"`
	ConfigDir   string `yaml:"config_dir"`

	// Release pipeline.
	HomebrewTap      string `yaml:"homebrew_tap"`
	HomebrewTapOwner string `yaml:"-"`
	HomebrewTapName  string `yaml:"-"`
	S3Bucket         string `yaml:"s3_bucket"`
	S3PathPrefix     string `yaml:"s3_path_prefix"`

	// Tooling.
	GoVersion string `yaml:"go_version"`

	// Injected, not user-set.
	TemplateVersion string `yaml:"template_version"`

	// Post-gen actions.
	RunTidy    bool `yaml:"run_tidy"`
	RunGitInit bool `yaml:"run_git_init"`
}

// DeriveBinary lowercases and strips spaces to produce a command name from
// a display name. "NeetoForm" -> "neetoform", "Big Widget" -> "bigwidget".
func DeriveBinary(pretty string) string {
	b := strings.Builder{}
	for _, r := range pretty {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-':
			b.WriteRune('-')
			// spaces, underscores, punctuation are dropped
		}
	}
	return b.String()
}

// Defaults returns a Variables populated with derived defaults for a given
// pretty name. Empty string is allowed — callers fill fields progressively.
func Defaults(pretty string) Variables {
	v := Variables{PrettyName: pretty}
	v.ApplyDerivations()
	return v
}

// ApplyDerivations fills any empty derivable field from its upstream inputs.
// Fields the user has set (non-empty) are left alone.
func (v *Variables) ApplyDerivations() {
	if v.BinaryName == "" && v.PrettyName != "" {
		v.BinaryName = DeriveBinary(v.PrettyName)
	}
	if v.GithubOrg == "" {
		v.GithubOrg = "neetozone"
	}
	if v.RepoName == "" && v.BinaryName != "" {
		v.RepoName = v.BinaryName + "-cli"
	}
	if v.ModulePath == "" && v.GithubOrg != "" && v.RepoName != "" {
		v.ModulePath = "github.com/" + v.GithubOrg + "/" + v.RepoName
	}
	if v.CompanyName == "" {
		v.CompanyName = "BigBinary"
	}
	if v.SupportEmail == "" {
		v.SupportEmail = "support@bigbinary.com"
	}
	if v.Domain == "" && v.BinaryName != "" {
		v.Domain = v.BinaryName + ".com"
	}
	if v.ApiBasePath == "" {
		v.ApiBasePath = "/api/external/v2"
	}
	if v.EnvVar == "" && v.BinaryName != "" {
		v.EnvVar = strings.ToUpper(strings.ReplaceAll(v.BinaryName, "-", "_")) + "_BASE_URL"
	}
	if v.ConfigDir == "" && v.BinaryName != "" {
		v.ConfigDir = ".config/" + v.BinaryName
	}
	if v.HomebrewTap == "" && v.GithubOrg != "" {
		v.HomebrewTap = v.GithubOrg + "/homebrew-tap"
	}
	if parts := strings.SplitN(v.HomebrewTap, "/", 2); len(parts) == 2 {
		v.HomebrewTapOwner = parts[0]
		v.HomebrewTapName = parts[1]
	}
	if v.S3Bucket == "" {
		v.S3Bucket = "neeto-downloads"
	}
	if v.S3PathPrefix == "" && v.PrettyName != "" {
		v.S3PathPrefix = "cli/" + strings.ReplaceAll(v.PrettyName, " ", "")
	}
	if v.GoVersion == "" {
		v.GoVersion = "1.26.1"
	}
	if v.ShortDescription == "" && v.PrettyName != "" {
		v.ShortDescription = v.PrettyName + " CLI"
	}
	if v.LongDescription == "" && v.PrettyName != "" {
		v.LongDescription = "A command-line interface for " + v.PrettyName + "."
	}
}

// Validate checks that every required field is present and well-formed.
// Returns the first error encountered; callers should address errors
// iteratively.
func (v *Variables) Validate() error {
	checks := []struct {
		name, value, pattern, desc string
	}{
		{"pretty_name", v.PrettyName, `^[A-Za-z][A-Za-z0-9 \-]{1,39}$`, "2-40 chars, letters/digits/spaces/hyphens, must start with a letter"},
		{"binary_name", v.BinaryName, `^[a-z][a-z0-9-]{1,31}$`, "lowercase letters/digits/hyphens, 2-32 chars, must start with a letter"},
		{"github_org", v.GithubOrg, `^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`, "valid GitHub org name"},
		{"repo_name", v.RepoName, `^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`, "valid GitHub repo name"},
		{"module_path", v.ModulePath, `^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`, "valid Go module path"},
		{"domain", v.Domain, `^[a-z0-9][a-z0-9.-]*[a-z0-9]$`, "valid host name"},
		{"api_base_path", v.ApiBasePath, `^/.*`, "must start with /"},
		{"env_var", v.EnvVar, `^[A-Z][A-Z0-9_]*$`, "UPPER_SNAKE_CASE"},
		{"homebrew_tap", v.HomebrewTap, `^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`, "owner/repo"},
		{"go_version", v.GoVersion, `^\d+\.\d+(\.\d+)?$`, "semver-ish"},
	}

	for _, c := range checks {
		if c.value == "" {
			return fmt.Errorf("%s is required", c.name)
		}
		ok, err := regexp.MatchString(c.pattern, c.value)
		if err != nil {
			return fmt.Errorf("%s: validation regex error: %w", c.name, err)
		}
		if !ok {
			return fmt.Errorf("%s=%q is invalid (%s)", c.name, c.value, c.desc)
		}
	}

	if v.CompanyName == "" {
		return fmt.Errorf("company_name is required")
	}
	if len(v.ShortDescription) < 5 || len(v.ShortDescription) > 120 {
		return fmt.Errorf("short_description must be 5-120 chars")
	}
	if len(v.LongDescription) < 10 || len(v.LongDescription) > 240 {
		return fmt.Errorf("long_description must be 10-240 chars")
	}

	return nil
}
