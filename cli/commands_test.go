package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/neetozone/neeto-cli-commons/auth"
	"github.com/neetozone/neeto-cli-commons/config"
	"github.com/spf13/cobra"
)

const skillFixture = `---
name: neetodesk
description: NeetoDesk CLI
---

Use the neetodesk CLI to manage tickets.
`

func subdomainProduct() config.Product {
	p := config.Product{
		PrettyName: "NeetoDesk",
		BinaryName: "neetodesk",
		Domain:     "neetodesk.com",
		SkillMD:    []byte(skillFixture),
	}
	p.ApplyDerivations()
	return p
}

func singleHostProduct() config.Product {
	p := config.Product{
		PrettyName:  "NeetoDeploy",
		BinaryName:  "neetodeploy",
		Domain:      "neetodeploy.com",
		Tenancy:     config.TenancySingleHost,
		APIHost:     "https://app.neetodeploy.com",
		LoginHost:   "https://neetozone.neetodeploy.com",
		APIBasePath: "/api/cli/v1",
		SkillMD:     []byte(skillFixture),
	}
	p.ApplyDerivations()
	return p
}

func newTestApp(t *testing.T, p config.Product) (*App, *bytes.Buffer) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(p.BaseURLEnvVar(), "")

	a := New(p)
	out := &bytes.Buffer{}
	a.Printer.Out = out
	a.Printer.Err = out
	a.root.SetOut(out)
	a.root.SetErr(out)
	return a, out
}

func run(t *testing.T, a *App, args ...string) error {
	t.Helper()
	a.root.SetArgs(args)
	return a.root.Execute()
}

func writeCredentials(t *testing.T, a *App, creds ...auth.Credentials) {
	t.Helper()
	if err := a.Auth.SaveStore(&auth.Store{Credentials: creds}); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}
}

func commandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	return names
}

func TestRegisterCommonCommands_RegistersEveryGenericCommand(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	got := map[string]bool{}
	for _, name := range commandNames(a.Root()) {
		got[name] = true
	}

	for _, want := range []string{"login", "logout", "whoami", "version", "doctor", "update", "completion", "setup", "commands"} {
		if !got[want] {
			t.Errorf("root is missing the %q command, has %v", want, commandNames(a.Root()))
		}
	}
}

func TestRegisterCommonCommands_SetupHasEveryTarget(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	setup, _, err := a.Root().Find([]string{"setup"})
	if err != nil {
		t.Fatalf("setup command not found: %v", err)
	}

	got := map[string]bool{}
	for _, name := range commandNames(setup) {
		got[name] = true
	}
	for _, want := range []string{"claude", "cursor", "windsurf", "copilot", "gemini", "codex"} {
		if !got[want] {
			t.Errorf("setup is missing the %q target, has %v", want, commandNames(setup))
		}
	}
}

func TestRegisterCommonCommands_DisablesCobrasOwnCompletionCommand(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	if !a.Root().CompletionOptions.DisableDefaultCmd {
		t.Error("cobra's default completion command should be replaced by ours")
	}
}

func TestSingleHostDropsSubdomainSurfaces(t *testing.T) {
	a, _ := newTestApp(t, singleHostProduct())

	if a.Root().PersistentFlags().Lookup("subdomain") != nil {
		t.Error("single_host products must not register --subdomain")
	}

	logout, _, err := a.Root().Find([]string{"logout"})
	if err != nil {
		t.Fatalf("logout not found: %v", err)
	}
	if logout.Flags().Lookup("all") != nil {
		t.Error("single_host logout must not offer --all")
	}

	whoami, _, err := a.Root().Find([]string{"whoami"})
	if err != nil {
		t.Fatalf("whoami not found: %v", err)
	}
	if want := "Show current authenticated user"; whoami.Short != want {
		t.Errorf("whoami Short = %q, want %q", whoami.Short, want)
	}
}

func TestSubdomainProductKeepsPluralWhoami(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	whoami, _, err := a.Root().Find([]string{"whoami"})
	if err != nil {
		t.Fatalf("whoami not found: %v", err)
	}
	if want := "Show current authenticated user(s)"; whoami.Short != want {
		t.Errorf("whoami Short = %q, want %q", whoami.Short, want)
	}
}

func TestWhoamiReportsTheSavedCredentials(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	writeCredentials(t, a, auth.Credentials{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"})

	if err := run(t, a, "whoami"); err != nil {
		t.Fatalf("whoami: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("whoami did not emit a JSON message off a terminal: %q", out.String())
	}
	if want := "Authenticated as a@acme.com on acme.neetodesk.com (default)."; payload["message"] != want {
		t.Errorf("message = %q, want %q", payload["message"], want)
	}
}

func TestLogoutAllClearsEverySubdomain(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	writeCredentials(t, a,
		auth.Credentials{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
		auth.Credentials{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"},
	)

	if err := run(t, a, "logout", "--all"); err != nil {
		t.Fatalf("logout --all: %v", err)
	}

	store, err := a.Auth.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected no saved credentials, got %d", len(store.Credentials))
	}

	path, _ := a.Auth.AuthFilePath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed, stat err: %v", filepath.Base(path), err)
	}
}
