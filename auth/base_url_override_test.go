package auth

import (
	"bytes"
	"strings"
	"testing"
)

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

	a := newTestAuth(t, p)
	var err bytes.Buffer
	a.Err = &err

	if got := a.BaseURL("acme"); got != "https://acme.neetodesk.com" {
		t.Errorf("BaseURL = %q, want the real product host", got)
	}
	if want := "Ignoring NEETODESK_BASE_URL"; !strings.Contains(err.String(), want) {
		t.Errorf("Err = %q, want it to mention %q", err.String(), want)
	}
}

// The warning is the only thing telling a developer their token is leaving the
// product host, and it must not pollute the stream a script parses.
func TestBaseURLWarnsOnErrNotOut(t *testing.T) {
	p := subdomainProduct()
	t.Setenv(p.BaseURLEnvVar(), "http://acme.lvh.me:8980")

	a := newTestAuth(t, p)
	var out, err bytes.Buffer
	a.Out, a.Err = &out, &err

	if got := a.BaseURL("acme"); got != "http://acme.lvh.me:8980" {
		t.Errorf("BaseURL = %q, want the override", got)
	}
	if want := "send your credentials to http://acme.lvh.me:8980"; !strings.Contains(err.String(), want) {
		t.Errorf("Err = %q, want it to mention %q", err.String(), want)
	}
	if out.Len() != 0 {
		t.Errorf("Out = %q, want nothing on the machine-readable stream", out.String())
	}
}
