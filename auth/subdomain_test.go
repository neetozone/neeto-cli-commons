package auth

import (
	"strings"
	"testing"
)

// BaseURL interpolates the subdomain into the URL host, so anything that can
// end a host or start a new one has to be refused before it gets there.
func TestLoginRejectsSubdomainsThatEscapeTheHost(t *testing.T) {
	hostile := []string{
		"evil.example#",
		"evil.example/x?",
		"evil.example:8443#",
		"acme.evil.example#",
		"acme/../../evil",
		"acme\\@evil.example",
		"-acme",
		"acme-",
		"ACME",
		"acme_corp",
		"acme.corp",
	}

	for _, subdomain := range hostile {
		if subdomainLabel.MatchString(subdomain) {
			t.Errorf("subdomainLabel matched %q, want no match", subdomain)
			continue
		}

		if _, err := New(subdomainProduct()).Login(subdomain); err == nil {
			t.Errorf("Login(%q) accepted a hostile subdomain", subdomain)
		} else if !strings.Contains(err.Error(), "not a valid subdomain") {
			t.Errorf("Login(%q) failed for the wrong reason: %v", subdomain, err)
		}
	}
}

func TestSubdomainLabelAcceptsRealWorkspaceNames(t *testing.T) {
	for _, subdomain := range []string{"acme", "acme-corp", "a", "a1", "spinkart", "neeto-zone-1"} {
		if !subdomainLabel.MatchString(subdomain) {
			t.Errorf("subdomainLabel did not match %q", subdomain)
		}
	}
}
