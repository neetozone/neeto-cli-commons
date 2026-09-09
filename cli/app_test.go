package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newGroupCommand(name, short string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
	}
}

func newLeafCommand(name, short string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE:  func(*cobra.Command, []string) error { return nil },
	}
}

func addProductsGroup(t *testing.T, a *App) *cobra.Command {
	t.Helper()
	products := newGroupCommand("products", "Inspect products and their available roles")
	products.AddCommand(
		newLeafCommand("enable", "Enable a product"),
		newLeafCommand("disable", "Disable a product"),
	)
	a.Root().AddCommand(products)
	t.Cleanup(func() { a.Root().RemoveCommand(products) })
	return products
}

func TestEnforceUnknownSubcommandErrors_RejectsUnknownNestedSubcommand(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	addProductsGroup(t, a)
	enforceUnknownSubcommandErrors(a.Root())

	err := run(t, a, "products", "cal", "enable")
	if err == nil {
		t.Fatal("expected an error for an unknown nested subcommand, got nil")
	}

	want := `unknown command "cal" for "neetodesk products"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestEnforceUnknownSubcommandErrors_BareGroupCommandStillPrintsHelp(t *testing.T) {
	a, out := newTestApp(t, subdomainProduct())
	products := addProductsGroup(t, a)
	enforceUnknownSubcommandErrors(a.Root())

	if err := run(t, a, "products"); err != nil {
		t.Fatalf("bare group command should exit cleanly, got: %v", err)
	}
	if !strings.Contains(out.String(), products.Short) {
		t.Errorf("expected help output to contain %q, got:\n%s", products.Short, out.String())
	}
}

func TestEnforceUnknownSubcommandErrors_LeavesExistingArgsValidatorAlone(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	products := addProductsGroup(t, a)
	products.Args = cobra.MaximumNArgs(1)
	enforceUnknownSubcommandErrors(a.Root())

	err := run(t, a, "products", "cal", "enable")
	if err == nil {
		t.Fatal("expected the pre-existing Args validator to still run, got nil error")
	}
	if want := "accepts at most 1 arg(s), received 2"; err.Error() != want {
		t.Errorf("error = %q, want %q (the product's own validator, not ours)", err.Error(), want)
	}

	if err := run(t, a, "products"); err != nil {
		t.Errorf("bare group command should still print help and exit cleanly, got: %v", err)
	}
}

func TestEnforceUnknownSubcommandErrors_SuggestsNearMisses(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	addProductsGroup(t, a)
	enforceUnknownSubcommandErrors(a.Root())

	err := run(t, a, "products", "enabl")
	if err == nil {
		t.Fatal("expected an error for a near-miss subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "Did you mean this?") {
		t.Errorf("expected a suggestion header, got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "\tenable") {
		t.Errorf("expected \"enable\" to be suggested, got: %q", err.Error())
	}
}

func TestEnforceUnknownSubcommandErrors_RunnableLeafCommandUnaffected(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	called := false
	leaf := &cobra.Command{
		Use:  "leaf",
		Args: cobra.ExactArgs(2),
		RunE: func(*cobra.Command, []string) error {
			called = true
			return nil
		},
	}
	a.Root().AddCommand(leaf)
	t.Cleanup(func() { a.Root().RemoveCommand(leaf) })

	enforceUnknownSubcommandErrors(a.Root())

	err := run(t, a, "leaf", "one")
	if err == nil {
		t.Fatal("expected the pre-existing Args validator to still run, got nil error")
	}
	if want := "accepts 2 arg(s), received 1"; err.Error() != want {
		t.Errorf("error = %q, want %q (the leaf's own validator, not ours)", err.Error(), want)
	}

	if err := run(t, a, "leaf", "one", "two"); err != nil {
		t.Fatalf("leaf command should run normally, got: %v", err)
	}
	if !called {
		t.Error("expected the leaf command's own RunE to have been called")
	}
}
