package vars

import "testing"

func TestDefaultsDerivesTheRepoNameAndDescriptions(t *testing.T) {
	v := Defaults("NeetoWidget")

	if v.BinaryName != "neetowidget" {
		t.Errorf("BinaryName = %q, want neetowidget", v.BinaryName)
	}
	if v.RepoName != "neeto-widget-cli" {
		t.Errorf("RepoName = %q, want neeto-widget-cli", v.RepoName)
	}
	if v.EnvPrefix != "NEETOWIDGET" {
		t.Errorf("EnvPrefix = %q, want NEETOWIDGET", v.EnvPrefix)
	}
	if v.LongDescription != "A command-line interface for NeetoWidget." {
		t.Errorf("LongDescription = %q", v.LongDescription)
	}
	if v.CommonsVersion != "main" {
		t.Errorf("CommonsVersion = %q, want main", v.CommonsVersion)
	}
}

func TestApplyDerivationsLeavesSuppliedValuesAlone(t *testing.T) {
	v := Variables{}
	v.PrettyName = "NeetoWidget"
	v.LongDescription = "Something the author wrote."
	v.CommonsVersion = "v1.2.0"
	v.ApplyDerivations()

	if v.LongDescription != "Something the author wrote." {
		t.Errorf("LongDescription was overwritten: %q", v.LongDescription)
	}
	if v.CommonsVersion != "v1.2.0" {
		t.Errorf("CommonsVersion was overwritten: %q", v.CommonsVersion)
	}
}

func TestHomebrewTapSplitsIntoOwnerAndName(t *testing.T) {
	v := Defaults("NeetoWidget")

	if got := v.HomebrewTapOwner(); got != "neetozone" {
		t.Errorf("HomebrewTapOwner() = %q, want neetozone", got)
	}
	if got := v.HomebrewTapName(); got != "tap" {
		t.Errorf("HomebrewTapName() = %q, want tap", got)
	}
}

func TestValidateRejectsIncompleteAnswers(t *testing.T) {
	v := Variables{}
	if err := v.Validate(); err == nil {
		t.Fatal("Validate() on an empty Variables returned no error")
	}

	good := Defaults("NeetoWidget")
	good.ShortDescription = "manage widgets from the terminal"
	if err := good.Validate(); err != nil {
		t.Fatalf("Validate() on a derived Variables failed: %v", err)
	}
}

func TestValidateRejectsABadBinaryName(t *testing.T) {
	v := Defaults("NeetoWidget")
	v.ShortDescription = "manage widgets from the terminal"
	v.BinaryName = "Neeto Widget"

	if err := v.Validate(); err == nil {
		t.Error("Validate() accepted a binary name with a space and a capital")
	}
}
