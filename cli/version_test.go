package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVersion_JSONCarriesTheInjectedBuildInfo(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	a.SetBuildInfo("1.4.2", "abc1234", "2026-09-07")

	if err := run(t, a, "version", "--json"); err != nil {
		t.Fatalf("version --json: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("version --json is not JSON: %v\n%s", err, out.String())
	}

	want := map[string]string{
		"binary":  "neetodesk",
		"version": "1.4.2",
		"commit":  "abc1234",
		"date":    "2026-09-07",
	}
	if len(payload) != len(want) {
		t.Errorf("payload keys = %v, want exactly %v", payload, want)
	}
	for key, value := range want {
		if payload[key] != value {
			t.Errorf("%s = %q, want %q", key, payload[key], value)
		}
	}
}

func TestVersion_DefaultsBeforeBuildInfoIsInjected(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())

	if err := run(t, a, "version", "--json"); err != nil {
		t.Fatalf("version --json: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("version --json is not JSON: %v", err)
	}
	if payload["version"] != "dev" || payload["commit"] != "none" || payload["date"] != "unknown" {
		t.Errorf("unexpected defaults: %v", payload)
	}
}

func TestVersionLine(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	a.SetBuildInfo("1.4.2", "abc1234", "2026-09-07")

	if want := "neetodesk 1.4.2 (commit: abc1234, built: 2026-09-07)"; a.versionLine() != want {
		t.Errorf("versionLine() = %q, want %q", a.versionLine(), want)
	}
}

func TestSetBuildInfo_EnablesTheVersionFlag(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	a.SetBuildInfo("1.4.2", "abc1234", "2026-09-07")

	if a.Root().Version != "1.4.2" {
		t.Errorf("root.Version = %q, want %q", a.Root().Version, "1.4.2")
	}
}

func TestVersion_BinaryNameComesFromTheProduct(t *testing.T) {
	a, out := newTestApp(t, singleHostProduct())

	if err := run(t, a, "version", "--json"); err != nil {
		t.Fatalf("version --json: %v", err)
	}
	if !strings.Contains(out.String(), `"binary":"neetodeploy"`) {
		t.Errorf("expected the product binary name, got %s", out.String())
	}
}
