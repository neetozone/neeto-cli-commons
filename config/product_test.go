package config

import "testing"

func TestDeriveRepoNameUsesHouseConvention(t *testing.T) {
	cases := map[string]string{
		"neetokb":  "neeto-kb-cli",
		"neetocal": "neeto-cal-cli",
		"widget":   "widget-cli",
	}
	for in, want := range cases {
		if got := DeriveRepoName(in); got != want {
			t.Errorf("DeriveRepoName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPriorityFieldsAlwaysLeadWithIdentity(t *testing.T) {
	p := Product{PrettyName: "NeetoDesk", PriorityFields: []string{"status", "kind"}}
	p.ApplyDerivations()
	if p.PriorityFields[0] != "sid" || p.PriorityFields[1] != "id" {
		t.Fatalf("identity fields must lead, got %v", p.PriorityFields[:2])
	}
	seen := map[string]int{}
	for _, f := range p.PriorityFields {
		seen[f]++
	}
	for f, n := range seen {
		if n > 1 {
			t.Errorf("%q duplicated %d times", f, n)
		}
	}
}

func TestHomebrewTapIsInstallableForm(t *testing.T) {
	p := Product{PrettyName: "NeetoCal"}
	p.ApplyDerivations()
	if p.HomebrewTap != "neetozone/tap" {
		t.Errorf("HomebrewTap = %q, want neetozone/tap", p.HomebrewTap)
	}
	if p.BrewFormula() != "neetozone/tap/neetocal" {
		t.Errorf("BrewFormula() = %q", p.BrewFormula())
	}
}

func TestValidateRejectsSingleHostWithoutAPIHost(t *testing.T) {
	p := Product{PrettyName: "NeetoDeploy", Tenancy: TenancySingleHost}
	p.ApplyDerivations()
	if err := p.Validate(); err == nil {
		t.Fatal("expected an error when api_host is missing")
	}
}

func TestDerivationsForSubdomainProduct(t *testing.T) {
	p := Product{PrettyName: "NeetoCal"}
	p.ApplyDerivations()
	checks := map[string]string{
		p.BinaryName:         "neetocal",
		p.RepoName:           "neeto-cal-cli",
		p.ModulePath:         "github.com/neetozone/neeto-cal-cli",
		p.Domain:             "neetocal.com",
		p.EnvPrefix:          "NEETOCAL",
		p.ConfigDir:          ".config/neetocal",
		p.BaseURLEnvVar():    "NEETOCAL_BASE_URL",
		p.InstallDirEnvVar(): "NEETOCAL_INSTALL_DIR",
		p.S3PathPrefix:       "cli/NeetoCal",
		string(p.Tenancy):    "subdomain",
	}
	for got, want := range checks {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestRootShortDoesNotDoubleTheProductName(t *testing.T) {
	cases := []struct{ short, want string }{
		{"", "NeetoCal CLI"},
		{"NeetoCal CLI", "NeetoCal CLI"},
		{"NeetoCal CLI — manage your calendar", "NeetoCal CLI — manage your calendar"},
		{"manage your calendar from the terminal", "NeetoCal CLI — manage your calendar from the terminal"},
	}
	for _, c := range cases {
		p := Product{PrettyName: "NeetoCal", ShortDescription: c.short}
		if got := p.RootShort(); got != c.want {
			t.Errorf("ShortDescription %q -> RootShort %q, want %q", c.short, got, c.want)
		}
	}
}
