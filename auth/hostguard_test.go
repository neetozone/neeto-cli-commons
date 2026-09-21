package auth

import (
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

func guardProduct() config.Product {
	return config.Product{
		PrettyName: "NeetoTest",
		BinaryName: "neetotest",
		Domain:     "neetotest.com",
		Tenancy:    config.TenancySubdomain,
	}
}

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
		"",
	}

	for _, subdomain := range hostile {
		if subdomain == "" {
			// Login treats empty as "no subdomain given" and the caller
			// prompts, so only the non-empty cases are guarded here.
			continue
		}
		if subdomainLabel.MatchString(subdomain) {
			t.Errorf("subdomainLabel.MatchString(%q) = true, want false", subdomain)
			continue
		}

		a := New(guardProduct())
		if _, err := a.Login(subdomain); err == nil {
			t.Errorf("Login(%q) accepted a hostile subdomain", subdomain)
		} else if !strings.Contains(err.Error(), "not a valid subdomain") {
			t.Errorf("Login(%q) failed for the wrong reason: %v", subdomain, err)
		}
	}
}

func TestValidSubdomainAcceptsRealWorkspaceNames(t *testing.T) {
	for _, subdomain := range []string{"acme", "acme-corp", "a", "a1", "spinkart", "neeto-zone-1"} {
		if !subdomainLabel.MatchString(subdomain) {
			t.Errorf("subdomainLabel.MatchString(%q) = false, want true", subdomain)
		}
	}
}

// The override redirects every request, the ones carrying the stored session
// token included, so plaintext is only acceptable against loopback.
func TestCheckOverrideRejectsPlaintextRemoteHosts(t *testing.T) {
	refused := []string{
		"http://evil.example",
		"http://192.168.1.10:3000",
		"ftp://evil.example",
		"file:///etc/passwd",
		"evil.example",
		"://nonsense",
	}
	for _, raw := range refused {
		if err := checkOverride(raw); err == nil {
			t.Errorf("checkOverride(%q) = nil, want an error", raw)
		}
	}

	allowed := []string{
		"https://acme.neetotest.com",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://acme.lvh.me:8980",
		"http://app.lvh.me:9029",
	}
	for _, raw := range allowed {
		if err := checkOverride(raw); err != nil {
			t.Errorf("checkOverride(%q) = %v, want nil", raw, err)
		}
	}
}

func TestBaseURLIgnoresAnInvalidOverride(t *testing.T) {
	t.Setenv(guardProduct().BaseURLEnvVar(), "http://evil.example")

	if got := New(guardProduct()).BaseURL("acme"); got != "https://acme.neetotest.com" {
		t.Errorf("BaseURL = %q, want the real product host", got)
	}
}
