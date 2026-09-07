package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Tenancy string

const (
	TenancySubdomain  Tenancy = "subdomain"
	TenancySingleHost Tenancy = "single_host"
)

type Product struct {
	PrettyName string `yaml:"pretty_name"`
	BinaryName string `yaml:"binary_name"`

	GithubOrg  string `yaml:"github_org"`
	RepoName   string `yaml:"repo_name"`
	ModulePath string `yaml:"module_path"`

	CompanyName      string `yaml:"company_name"`
	License          string `yaml:"license"`
	SupportEmail     string `yaml:"support_email"`
	ShortDescription string `yaml:"short_description"`
	LongDescription  string `yaml:"long_description"`

	Domain      string `yaml:"domain"`
	APIBasePath string `yaml:"api_base_path"`
	EnvPrefix   string `yaml:"env_prefix"`
	ConfigDir   string `yaml:"config_dir"`

	Tenancy   Tenancy `yaml:"tenancy"`
	APIHost   string  `yaml:"api_host"`
	LoginHost string  `yaml:"login_host"`

	RootExamples   []string `yaml:"root_examples"`
	PriorityFields []string `yaml:"priority_fields"`

	HomebrewTap   string `yaml:"homebrew_tap"`
	S3Bucket      string `yaml:"s3_bucket"`
	S3PathPrefix  string `yaml:"s3_path_prefix"`
	InstallShURL  string `yaml:"install_sh_url"`
	InstallPS1URL string `yaml:"install_ps1_url"`

	GoVersion      string `yaml:"go_version"`
	CommonsVersion string `yaml:"commons_version"`

	SkillMD []byte `yaml:"-"`
}

var binarySafe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func DeriveBinary(pretty string) string {
	var b strings.Builder
	for _, r := range pretty {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func DeriveRepoName(binary string) string {
	if binary == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(binary, "neeto"); ok && rest != "" {
		return "neeto-" + rest + "-cli"
	}
	return binary + "-cli"
}

func (p *Product) ApplyDerivations() {
	if p.BinaryName == "" && p.PrettyName != "" {
		p.BinaryName = DeriveBinary(p.PrettyName)
	}
	if p.GithubOrg == "" {
		p.GithubOrg = "neetozone"
	}
	if p.RepoName == "" {
		p.RepoName = DeriveRepoName(p.BinaryName)
	}
	if p.ModulePath == "" && p.RepoName != "" {
		p.ModulePath = "github.com/" + p.GithubOrg + "/" + p.RepoName
	}
	if p.CompanyName == "" {
		p.CompanyName = "BigBinary"
	}
	if p.SupportEmail == "" {
		p.SupportEmail = "support@bigbinary.com"
	}
	if p.Domain == "" && p.BinaryName != "" {
		p.Domain = p.BinaryName + ".com"
	}
	if p.APIBasePath == "" {
		p.APIBasePath = "/api/external/v2"
	}
	if p.EnvPrefix == "" && p.BinaryName != "" {
		p.EnvPrefix = strings.ToUpper(strings.ReplaceAll(p.BinaryName, "-", "_"))
	}
	if p.ConfigDir == "" && p.BinaryName != "" {
		p.ConfigDir = ".config/" + p.BinaryName
	}
	if p.Tenancy == "" {
		p.Tenancy = TenancySubdomain
	}
	if p.HomebrewTap == "" {
		p.HomebrewTap = p.GithubOrg + "/tap"
	}
	if p.S3Bucket == "" {
		p.S3Bucket = "neeto-downloads"
	}
	if p.S3PathPrefix == "" && p.PrettyName != "" {
		p.S3PathPrefix = "cli/" + p.PrettyName
	}
	if p.InstallShURL == "" {
		p.InstallShURL = p.LatestURL() + "/install.sh"
	}
	if p.InstallPS1URL == "" {
		p.InstallPS1URL = p.LatestURL() + "/install.ps1"
	}
	if p.ShortDescription == "" && p.PrettyName != "" {
		p.ShortDescription = p.PrettyName + " CLI"
	}
	if p.GoVersion == "" {
		p.GoVersion = "1.26.1"
	}
	if len(p.PriorityFields) == 0 {
		p.PriorityFields = DefaultPriorityFields()
	}
	p.PriorityFields = normalisePriorityFields(p.PriorityFields)
}

func DefaultPriorityFields() []string {
	return []string{
		"sid", "id", "name", "title", "email", "first_name", "last_name",
		"status", "state", "kind", "type", "slug",
	}
}

func normalisePriorityFields(fields []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(fields)+2)
	for _, f := range []string{"sid", "id"} {
		out = append(out, f)
		seen[f] = true
	}
	for _, f := range fields {
		if !seen[f] {
			out = append(out, f)
			seen[f] = true
		}
	}
	return out
}

func (p Product) RootShort() string {
	title := p.PrettyName + " CLI"
	d := strings.TrimSpace(p.ShortDescription)
	switch {
	case d == "" || d == title:
		return title
	case strings.HasPrefix(d, title):
		return d
	default:
		return title + " — " + d
	}
}

func (p Product) BaseURLEnvVar() string    { return p.EnvPrefix + "_BASE_URL" }
func (p Product) InstallDirEnvVar() string { return p.EnvPrefix + "_INSTALL_DIR" }
func (p Product) S3URLBase() string {
	return "https://" + p.S3Bucket + ".s3.amazonaws.com/" + p.S3PathPrefix
}
func (p Product) LatestURL() string   { return p.S3URLBase() + "/latest" }
func (p Product) BrewFormula() string { return p.HomebrewTap + "/" + p.BinaryName }
func (p Product) UserAgent(version string) string {
	return p.BinaryName + "-cli/" + version
}

func (p Product) Validate() error {
	if p.PrettyName == "" {
		return fmt.Errorf("pretty_name is required")
	}
	if !binarySafe.MatchString(p.BinaryName) {
		return fmt.Errorf("binary_name %q must match %s", p.BinaryName, binarySafe)
	}
	if p.Tenancy != TenancySubdomain && p.Tenancy != TenancySingleHost {
		return fmt.Errorf("tenancy %q must be %q or %q", p.Tenancy, TenancySubdomain, TenancySingleHost)
	}
	if p.Tenancy == TenancySingleHost && p.APIHost == "" {
		return fmt.Errorf("api_host is required when tenancy is %q", TenancySingleHost)
	}
	if !strings.HasPrefix(p.APIBasePath, "/") || len(p.APIBasePath) < 2 {
		return fmt.Errorf("api_base_path %q must start with /", p.APIBasePath)
	}
	return nil
}

func Load(path string) (*Product, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Could not read %s: %w", path, err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Product, error) {
	var p Product
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("Invalid product config: %w", err)
	}
	p.ApplyDerivations()
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}
