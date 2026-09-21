package auth

import "testing"

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
		"https://acme.neetodesk.com",
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
	p := subdomainProduct()
	t.Setenv(p.BaseURLEnvVar(), "http://evil.example")

	if got := New(p).BaseURL("acme"); got != "https://acme.neetodesk.com" {
		t.Errorf("BaseURL = %q, want the real product host", got)
	}
}
