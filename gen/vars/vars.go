package vars

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/neetozone/neeto-cli-commons/config"
)

type Variables struct {
	config.Product `yaml:",inline"`

	TemplateVersion string `yaml:"template_version"`
	RunTidy         bool   `yaml:"run_tidy"`
	RunGitInit      bool   `yaml:"run_git_init"`
}

func DeriveBinary(pretty string) string { return config.DeriveBinary(pretty) }

func Defaults(pretty string) Variables {
	v := Variables{}
	v.PrettyName = pretty
	v.ApplyDerivations()
	return v
}

func (v *Variables) ApplyDerivations() {
	v.Product.ApplyDerivations()
	if v.LongDescription == "" && v.PrettyName != "" {
		v.LongDescription = "A command-line interface for " + v.PrettyName + "."
	}
	if v.CommonsVersion == "" {
		v.CommonsVersion = "main"
	}
}

func (v *Variables) HomebrewTapOwner() string { return tapPart(v.HomebrewTap, 0) }
func (v *Variables) HomebrewTapName() string  { return tapPart(v.HomebrewTap, 1) }

func tapPart(tap string, i int) string {
	parts := strings.SplitN(tap, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[i]
}

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
		{"api_base_path", v.APIBasePath, `^/.*`, "must start with /"},
		{"env_prefix", v.EnvPrefix, `^[A-Z][A-Z0-9_]*$`, "UPPER_SNAKE_CASE"},
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

	return v.Product.Validate()
}
