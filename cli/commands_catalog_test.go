package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func catalogFor(t *testing.T, a *App) []map[string]any {
	t.Helper()
	a.Printer.Out.(interface{ Reset() }).Reset()
	if err := run(t, a, "commands"); err != nil {
		t.Fatalf("commands: %v", err)
	}
	var raw []map[string]any
	if err := json.Unmarshal([]byte(a.Printer.Out.(interface{ String() string }).String()), &raw); err != nil {
		t.Fatalf("catalog is not a JSON array: %v", err)
	}
	return raw
}

func entryFor(t *testing.T, entries []map[string]any, command string) map[string]any {
	t.Helper()
	for _, entry := range entries {
		if entry["command"] == command {
			return entry
		}
	}
	t.Fatalf("catalog has no %q entry", command)
	return nil
}

func addWidgetsCommand(t *testing.T, a *App) {
	t.Helper()
	widgets := &cobra.Command{Use: "widgets", Short: "Manage widgets"}
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a widget",
		Run:   func(*cobra.Command, []string) {},
	}
	create.Flags().StringP("name", "n", "", "Widget name")
	create.Flags().Int("size", 3, "Widget size")
	MarkFlagsRequired(create, "name")
	widgets.AddCommand(create)

	a.Root().AddCommand(widgets)
	t.Cleanup(func() { a.Root().RemoveCommand(widgets) })
}

func TestCatalog_ShapeIsTheDocumentedContract(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	addWidgetsCommand(t, a)

	if err := run(t, a, "commands"); err != nil {
		t.Fatalf("commands: %v", err)
	}

	var entries []map[string]any
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("catalog is not a JSON array: %v\n%s", err, out.String())
	}

	widgets := entryFor(t, entries, "neetodesk widgets")
	for _, key := range []string{"command", "description", "subcommands"} {
		if _, ok := widgets[key]; !ok {
			t.Errorf("widgets entry missing %q key: %v", key, widgets)
		}
	}

	subcommands, ok := widgets["subcommands"].([]any)
	if !ok || len(subcommands) != 1 {
		t.Fatalf("expected one subcommand, got %v", widgets["subcommands"])
	}
	create := subcommands[0].(map[string]any)
	if create["command"] != "neetodesk widgets create" || create["description"] != "Create a widget" {
		t.Fatalf("unexpected subcommand entry: %v", create)
	}

	flags, ok := create["flags"].([]any)
	if !ok || len(flags) != 2 {
		t.Fatalf("expected two flags, got %v", create["flags"])
	}
	name := flags[0].(map[string]any)
	want := map[string]any{
		"name":        "name",
		"shorthand":   "n",
		"type":        "string",
		"description": "Widget name",
		"required":    true,
	}
	for key, value := range want {
		if name[key] != value {
			t.Errorf("flag %s = %v, want %v (entry: %v)", key, name[key], value, name)
		}
	}
	if _, ok := name["default"]; ok {
		t.Errorf("an empty default must be omitted: %v", name)
	}

	size := flags[1].(map[string]any)
	if size["default"] != "3" || size["type"] != "int" {
		t.Errorf("unexpected size flag: %v", size)
	}
	if _, ok := size["required"]; ok {
		t.Errorf("required must be omitted when false: %v", size)
	}
}

func TestCatalog_DropsPersistentAndHelpFlags(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	addWidgetsCommand(t, a)

	if err := run(t, a, "commands"); err != nil {
		t.Fatalf("commands: %v", err)
	}
	for _, flag := range []string{`"name":"json"`, `"name":"quiet"`, `"name":"toon"`, `"name":"subdomain"`, `"name":"help"`} {
		if strings.Contains(strings.ReplaceAll(out.String(), " ", ""), flag) {
			t.Errorf("catalog should not list %s:\n%s", flag, out.String())
		}
	}
}

func TestCatalog_OmitsItsOwnPlumbing(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())

	if err := run(t, a, "commands"); err != nil {
		t.Fatalf("commands: %v", err)
	}

	var entries []map[string]any
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("catalog is not a JSON array: %v", err)
	}

	listed := map[string]bool{}
	for _, entry := range entries {
		listed[entry["command"].(string)] = true
	}
	for _, hidden := range []string{"neetodesk commands", "neetodesk completion", "neetodesk help"} {
		if listed[hidden] {
			t.Errorf("%q must not appear in the catalog", hidden)
		}
	}
	for _, shown := range []string{"neetodesk login", "neetodesk logout", "neetodesk whoami", "neetodesk version", "neetodesk doctor", "neetodesk update", "neetodesk setup"} {
		if !listed[shown] {
			t.Errorf("%q is missing from the catalog", shown)
		}
	}
}

func TestCatalog_RequiredSurvivesAJSONFileOverride(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	addWidgetsCommand(t, a)

	entries := catalogFor(t, a)
	widgets := entryFor(t, entries, "neetodesk widgets")
	create := widgets["subcommands"].([]any)[0].(map[string]any)
	if create["flags"].([]any)[0].(map[string]any)["required"] != true {
		t.Errorf("expected --name to be required: %v", create)
	}
}

func TestCatalog_SingleHostHasNoSubdomainFlag(t *testing.T) {
	a, out := newTestApp(t, singleHostProduct())

	if err := run(t, a, "commands"); err != nil {
		t.Fatalf("commands: %v", err)
	}
	if strings.Contains(out.String(), "subdomain") {
		t.Errorf("single_host catalog must not mention a subdomain:\n%s", out.String())
	}
}
