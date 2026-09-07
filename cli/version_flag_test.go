package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/config"
)

func TestVersionFlagCarriesBuildInfo(t *testing.T) {
	p := config.Product{PrettyName: "NeetoDesk"}
	p.ApplyDerivations()
	a := New(p)
	a.SetBuildInfo("1.4.2", "abc1234", "2026-09-07T00:00:00Z")

	var buf bytes.Buffer
	a.Root().SetOut(&buf)
	a.Root().SetArgs([]string{"--version"})
	if err := a.Root().Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"neetodesk", "1.4.2", "abc1234", "2026-09-07T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("--version output %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "dev") || strings.Contains(got, "unknown") {
		t.Errorf("--version output %q still carries placeholder build info", got)
	}
}
