package vars

import (
	"strings"
	"testing"
)

func TestDeriveBinary(t *testing.T) {
	cases := map[string]string{
		"NeetoCal":    "neetocal",
		"NeetoForm":   "neetoform",
		"Big Widget":  "bigwidget",
		"Acme-Tool":   "acme-tool",
		"foo_bar baz": "foobarbaz",
		"123Start":    "123start",
		"":            "",
	}
	for in, want := range cases {
		if got := DeriveBinary(in); got != want {
			t.Errorf("DeriveBinary(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaults_FromPrettyName(t *testing.T) {
	v := Defaults("NeetoForm")

	checks := map[string]string{
		"BinaryName":   "neetoform",
		"RepoName":     "neetoform-cli",
		"ModulePath":   "github.com/neetozone/neetoform-cli",
		"Domain":       "neetoform.com",
		"EnvVar":       "NEETOFORM_BASE_URL",
		"ConfigDir":    ".config/neetoform",
		"HomebrewTap":  "neetozone/homebrew-tap",
		"S3Bucket":     "neeto-downloads",
		"S3PathPrefix": "cli/NeetoForm",
		"GithubOrg":    "neetozone",
		"CompanyName":  "BigBinary",
		"SupportEmail": "support@bigbinary.com",
		"ApiBasePath":  "/api/external/v2",
		"GoVersion":    "1.26.1",
	}

	got := map[string]string{
		"BinaryName":   v.BinaryName,
		"RepoName":     v.RepoName,
		"ModulePath":   v.ModulePath,
		"Domain":       v.Domain,
		"EnvVar":       v.EnvVar,
		"ConfigDir":    v.ConfigDir,
		"HomebrewTap":  v.HomebrewTap,
		"S3Bucket":     v.S3Bucket,
		"S3PathPrefix": v.S3PathPrefix,
		"GithubOrg":    v.GithubOrg,
		"CompanyName":  v.CompanyName,
		"SupportEmail": v.SupportEmail,
		"ApiBasePath":  v.ApiBasePath,
		"GoVersion":    v.GoVersion,
	}

	for name, want := range checks {
		if got[name] != want {
			t.Errorf("Defaults(...)%s = %q, want %q", name, got[name], want)
		}
	}

	if v.ShortDescription != "NeetoForm CLI" {
		t.Errorf("ShortDescription = %q", v.ShortDescription)
	}
	if !strings.Contains(v.LongDescription, "NeetoForm") {
		t.Errorf("LongDescription = %q", v.LongDescription)
	}
}

func TestApplyDerivations_RespectsUserOverrides(t *testing.T) {
	v := Variables{
		PrettyName: "NeetoForm",
		BinaryName: "nform",
		Domain:     "neetoform.io",
	}
	v.ApplyDerivations()

	if v.BinaryName != "nform" {
		t.Errorf("BinaryName overridden to %q, want preserved nform", v.BinaryName)
	}
	if v.Domain != "neetoform.io" {
		t.Errorf("Domain overridden to %q, want preserved neetoform.io", v.Domain)
	}
	// EnvVar derives from user-set BinaryName "nform".
	if v.EnvVar != "NFORM_BASE_URL" {
		t.Errorf("EnvVar = %q, want NFORM_BASE_URL (derived from user BinaryName)", v.EnvVar)
	}
}

func TestValidate_Valid(t *testing.T) {
	v := Defaults("NeetoForm")
	if err := v.Validate(); err != nil {
		t.Fatalf("Validate() defaults should be valid: %v", err)
	}
}

func TestValidate_InvalidBinary(t *testing.T) {
	v := Defaults("NeetoForm")
	v.BinaryName = "Bad-Upper"
	if err := v.Validate(); err == nil {
		t.Error("Validate() should reject uppercase binary name")
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	v := Defaults("NeetoForm")
	v.ModulePath = ""
	if err := v.Validate(); err == nil {
		t.Error("Validate() should reject missing module path")
	}
}

func TestValidate_EnvVarWrongCase(t *testing.T) {
	v := Defaults("NeetoForm")
	v.EnvVar = "neetoform_base_url"
	if err := v.Validate(); err == nil {
		t.Error("Validate() should reject lowercase env var")
	}
}

func TestValidate_HyphenInBinaryBecomesUnderscoreInEnvVar(t *testing.T) {
	v := Defaults("Acme-Tool")
	if v.BinaryName != "acme-tool" {
		t.Fatalf("BinaryName = %q, want acme-tool", v.BinaryName)
	}
	if v.EnvVar != "ACME_TOOL_BASE_URL" {
		t.Errorf("EnvVar = %q, want ACME_TOOL_BASE_URL", v.EnvVar)
	}
}
