package questionnaire

import "testing"

func TestValidateBinary(t *testing.T) {
	good := []string{"acme", "a1", "acme-tool", "neetoform"}
	bad := []string{"", "Acme", "acme_tool", "1acme", "go", "TEST"}
	for _, s := range good {
		if err := validateBinary(s); err != nil {
			t.Errorf("validateBinary(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateBinary(s); err == nil {
			t.Errorf("validateBinary(%q) expected error", s)
		}
	}
}

func TestValidateEnvVar(t *testing.T) {
	good := []string{"ACME_BASE_URL", "A", "A1_B2"}
	bad := []string{"", "1ACME", "acme", "ACME-URL", "acme_base_url"}
	for _, s := range good {
		if err := validateEnvVar(s); err != nil {
			t.Errorf("validateEnvVar(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateEnvVar(s); err == nil {
			t.Errorf("validateEnvVar(%q) expected error", s)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	good := []string{"a@b.co", "support@bigbinary.com"}
	bad := []string{"", "no-at", "a@b", "@b.co", "a@@b.co"}
	for _, s := range good {
		if err := validateEmail(s); err != nil {
			t.Errorf("validateEmail(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateEmail(s); err == nil {
			t.Errorf("validateEmail(%q) expected error", s)
		}
	}
}

func TestValidateTap(t *testing.T) {
	good := []string{"owner/repo", "neetozone/homebrew-tap"}
	bad := []string{"", "owner", "owner/", "/repo", "a b/c"}
	for _, s := range good {
		if err := validateTap(s); err != nil {
			t.Errorf("validateTap(%q) unexpected error: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := validateTap(s); err == nil {
			t.Errorf("validateTap(%q) expected error", s)
		}
	}
}
