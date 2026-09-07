package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/auth"
)

func decodeChecks(t *testing.T, raw []byte) []checkResult {
	t.Helper()
	var checks []checkResult
	if err := json.Unmarshal(raw, &checks); err != nil {
		t.Fatalf("doctor output is not a JSON array of checks: %v\n%s", err, raw)
	}
	return checks
}

func TestRenderChecks_GlyphsOnATerminal(t *testing.T) {
	var buf bytes.Buffer
	renderChecks(&buf, []checkResult{
		{Name: "Authentication", OK: true, Detail: "authenticated as a@acme.com on acme.neetodesk.com"},
		{Name: "API connection", Skipped: true, Detail: "skipped"},
		{Name: "CLI version", Detail: "boom"},
	}, true)

	want := "✓ Authentication: authenticated as a@acme.com on acme.neetodesk.com\n" +
		"• API connection: skipped\n" +
		"✗ CLI version: boom\n"
	if buf.String() != want {
		t.Errorf("rendered\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestRenderChecks_FallsBackToWordsOffATerminal(t *testing.T) {
	var buf bytes.Buffer
	renderChecks(&buf, []checkResult{
		{Name: "Authentication", OK: true, Detail: "ok"},
		{Name: "API connection", Skipped: true, Detail: "skipped"},
		{Name: "CLI version", Detail: "boom"},
	}, false)

	want := "OK Authentication: ok\n" +
		"- API connection: skipped\n" +
		"FAIL CLI version: boom\n"
	if buf.String() != want {
		t.Errorf("rendered\n%q\nwant\n%q", buf.String(), want)
	}
	for _, glyph := range []string{"✓", "✗", "•"} {
		if strings.Contains(buf.String(), glyph) {
			t.Errorf("non-TTY output must not carry %q:\n%s", glyph, buf.String())
		}
	}
}

func TestDoctor_PlainOutputSkipsTheProbeWithoutASubdomain(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())

	if err := run(t, a, "doctor"); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "FAIL Authentication: Not authenticated.") {
		t.Errorf("expected a failed authentication check, got:\n%s", got)
	}
	if !strings.Contains(got, "- API connection: skipped (no subdomain") {
		t.Errorf("expected the probe to be skipped, got:\n%s", got)
	}
	if !strings.Contains(got, "OK CLI version: dev") {
		t.Errorf("expected the version check, got:\n%s", got)
	}
}

func TestDoctor_JSONEmitsStructuredChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	product := subdomainProduct()
	a, out := newTestApp(t, product)
	t.Setenv(product.BaseURLEnvVar(), server.URL)
	writeCredentials(t, a, auth.Credentials{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"})

	if err := run(t, a, "doctor", "--json"); err != nil {
		t.Fatalf("doctor --json: %v", err)
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("doctor --json did not emit an envelope: %v\n%s", err, out.String())
	}

	checks := decodeChecks(t, envelope.Data)
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d: %s", len(checks), envelope.Data)
	}
	for _, check := range checks {
		if !check.OK {
			t.Errorf("check %q should have passed, detail: %s", check.Name, check.Detail)
		}
	}
	if checks[0].Name != "Authentication" || !strings.Contains(checks[0].Detail, "a@acme.com") {
		t.Errorf("unexpected authentication check: %+v", checks[0])
	}
	if checks[1].Name != "API connection" || !strings.Contains(checks[1].Detail, server.URL) {
		t.Errorf("unexpected connection check: %+v", checks[1])
	}
	if strings.ContainsAny(string(envelope.Data), "✓✗•") {
		t.Errorf("structured output must carry no glyphs: %s", envelope.Data)
	}
}

func TestDoctor_QuietEmitsTheRawChecks(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())

	if err := run(t, a, "doctor", "--quiet"); err != nil {
		t.Fatalf("doctor --quiet: %v", err)
	}

	checks := decodeChecks(t, out.Bytes())
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d: %s", len(checks), out.String())
	}
	if !checks[1].Skipped {
		t.Errorf("expected the connection check to be skipped, got %+v", checks[1])
	}
}

func TestDoctor_ToonEmitsAStructuredDocument(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())

	if err := run(t, a, "doctor", "--toon"); err != nil {
		t.Fatalf("doctor --toon: %v", err)
	}
	if strings.ContainsAny(out.String(), "✓✗•") {
		t.Errorf("TOON output must carry no glyphs:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Authentication") {
		t.Errorf("TOON output should name each check:\n%s", out.String())
	}
}

func TestDoctor_SingleHostAlwaysProbes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	product := singleHostProduct()
	a, out := newTestApp(t, product)
	t.Setenv(product.BaseURLEnvVar(), server.URL)

	if err := run(t, a, "doctor", "--quiet"); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	checks := decodeChecks(t, out.Bytes())
	if checks[1].Skipped {
		t.Errorf("a single_host product knows its host, so the probe must run: %+v", checks[1])
	}
	if !checks[1].OK {
		t.Errorf("probe should have succeeded: %+v", checks[1])
	}
}

func TestDoctorShortIsTheCanonicalWording(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	doctor, _, err := a.Root().Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("doctor not found: %v", err)
	}
	if want := "Check CLI health and connectivity"; doctor.Short != want {
		t.Errorf("doctor Short = %q, want %q", doctor.Short, want)
	}
}
