package cli

import (
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/v2/auth"
	"github.com/neetozone/neeto-cli-commons/v2/config"
	"github.com/spf13/cobra"
)

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Token  string
	Body   map[string]any
}

// roleServer answers every request with status and body and records what it got.
func roleServer(t *testing.T, status int, body string) (*httptest.Server, *[]recordedRequest) {
	t.Helper()
	requests := &[]recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded := recordedRequest{
			Method: r.Method,
			Path:   r.URL.EscapedPath(),
			Query:  r.URL.RawQuery,
			Token:  r.Header.Get("Session-Token"),
		}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			if err := json.Unmarshal(raw, &recorded.Body); err != nil {
				t.Errorf("request body is not JSON: %v: %s", err, raw)
			}
		}
		*requests = append(*requests, recorded)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server, requests
}

// roleApp returns a logged-in app with the role commands added, talking to server.
func roleApp(t *testing.T, p config.Product, server *httptest.Server) (*App, *strings.Builder) {
	t.Helper()
	a, buf := newTestApp(t, p)
	t.Setenv(p.BaseURLEnvVar(), server.URL)
	writeCredentials(t, a, auth.Credentials{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"})
	a.AddRoleCommands()

	out := &strings.Builder{}
	a.Printer.Out = out
	a.Printer.Err = buf
	return a, out
}

func onlyRequest(t *testing.T, requests *[]recordedRequest) recordedRequest {
	t.Helper()
	if len(*requests) != 1 {
		t.Fatalf("expected 1 request, got %d: %+v", len(*requests), *requests)
	}
	return (*requests)[0]
}

func TestRoleCommandsAreOptIn(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	names := commandNames(a.Root())
	if slices.Contains(names, "roles") || slices.Contains(names, "permissions") {
		t.Fatalf("role commands must not be added until the product asks: %v", names)
	}

	a.AddRoleCommands()

	names = commandNames(a.Root())
	if !slices.Contains(names, "roles") || !slices.Contains(names, "permissions") {
		t.Errorf("AddRoleCommands did not add roles and permissions: %v", names)
	}
}

func TestAddRoleCommandsTwiceAddsThemOnce(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	a.AddRoleCommands()
	a.AddRoleCommands()

	count := 0
	for _, name := range commandNames(a.Root()) {
		if name == "roles" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected one roles command, got %d", count)
	}
}

func TestAddRoleCommandsKeepsTheProductsOwnRolesCommand(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	own := &cobra.Command{Use: "roles", Short: "Product roles"}
	a.Root().AddCommand(own)

	a.AddRoleCommands()

	roles, _, err := a.Root().Find([]string{"roles"})
	if err != nil || roles != own {
		t.Errorf("the product's roles command was replaced: %v", err)
	}
	if !slices.Contains(commandNames(a.Root()), "permissions") {
		t.Error("permissions should still be added when only roles is taken")
	}
}

func TestRolesListUsesTheCommonsPathWhateverTheProductBasePath(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"roles":[{"id":"r1","name":"Admin","permission_ids":["p1"]}]}`)
	product := subdomainProduct()
	product.APIBasePath = "/api/external/v1"
	a, out := roleApp(t, product, server)

	if err := run(t, a, "roles", "list", "--json"); err != nil {
		t.Fatalf("roles list: %v", err)
	}

	request := onlyRequest(t, requests)
	if request.Method != http.MethodGet || request.Path != "/api/external/v2/roles" {
		t.Errorf("got %s %s, want GET /api/external/v2/roles", request.Method, request.Path)
	}
	if request.Token != "tok" {
		t.Errorf("Session-Token = %q, want the saved token", request.Token)
	}
	if !strings.Contains(out.String(), `"name": "Admin"`) {
		t.Errorf("roles were not printed: %s", out.String())
	}
	if !strings.Contains(out.String(), "neetodesk roles update") {
		t.Errorf("breadcrumbs should use the product's binary name: %s", out.String())
	}
}

func TestPermissionsListPrintsThePermissions(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"permissions":[{"id":"p1","category":"Members","description":"View members"}]}`)
	a, out := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "permissions", "list", "--quiet"); err != nil {
		t.Fatalf("permissions list: %v", err)
	}

	request := onlyRequest(t, requests)
	if request.Path != "/api/external/v2/permissions" {
		t.Errorf("path = %s, want /api/external/v2/permissions", request.Path)
	}
	if !strings.Contains(out.String(), `"description":"View members"`) || strings.Contains(out.String(), "breadcrumbs") {
		t.Errorf("--quiet should print just the permissions: %s", out.String())
	}
}

func TestRolesCreateSendsTheFlags(t *testing.T) {
	server, requests := roleServer(t, http.StatusCreated, `{"role":{"id":"r2","name":"Support"}}`)
	a, out := roleApp(t, subdomainProduct(), server)

	err := run(t, a, "roles", "create", "--name", "Support", "--description", "First line",
		"--permission-ids", "p1, p2,,", "--quiet")
	if err != nil {
		t.Fatalf("roles create: %v", err)
	}

	request := onlyRequest(t, requests)
	if request.Method != http.MethodPost || request.Path != "/api/external/v2/roles" {
		t.Errorf("got %s %s, want POST /api/external/v2/roles", request.Method, request.Path)
	}
	want := map[string]any{"name": "Support", "description": "First line", "permission_ids": []any{"p1", "p2"}}
	if !jsonEqual(request.Body, want) {
		t.Errorf("body = %v, want %v", request.Body, want)
	}
	if strings.TrimSpace(out.String()) != "r2" {
		t.Errorf("--quiet should print the new role's id, got %q", out.String())
	}
}

func TestRolesCreateLeavesOutFlagsThatWereNotPassed(t *testing.T) {
	server, requests := roleServer(t, http.StatusCreated, `{"role":{"id":"r2","name":"Support"}}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "create", "--name", "Support"); err != nil {
		t.Fatalf("roles create: %v", err)
	}

	if body := onlyRequest(t, requests).Body; !jsonEqual(body, map[string]any{"name": "Support"}) {
		t.Errorf("body = %v, want only the name", body)
	}
}

func TestRolesCreateRequiresAName(t *testing.T) {
	server, requests := roleServer(t, http.StatusCreated, `{}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	err := run(t, a, "roles", "create", "--description", "No name")

	if err == nil || !strings.Contains(err.Error(), `"name"`) {
		t.Errorf("expected a missing --name error, got %v", err)
	}
	if len(*requests) != 0 {
		t.Errorf("no request should be sent without a name, got %+v", *requests)
	}
}

func TestRolesUpdateSendsOnlyTheChangedFlags(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"role":{"id":"r1","name":"Helpdesk"}}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "update", "r1", "--name", "Helpdesk"); err != nil {
		t.Fatalf("roles update: %v", err)
	}

	request := onlyRequest(t, requests)
	if request.Method != http.MethodPatch || request.Path != "/api/external/v2/roles/r1" {
		t.Errorf("got %s %s, want PATCH /api/external/v2/roles/r1", request.Method, request.Path)
	}
	if !jsonEqual(request.Body, map[string]any{"name": "Helpdesk"}) {
		t.Errorf("body = %v, want only the name", request.Body)
	}
}

func TestRolesUpdateWithEmptyPermissionIDsClearsThem(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"role":{"id":"r1"}}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "update", "r1", "--permission-ids", ""); err != nil {
		t.Fatalf("roles update: %v", err)
	}

	if body := onlyRequest(t, requests).Body; !jsonEqual(body, map[string]any{"permission_ids": []any{}}) {
		t.Errorf("body = %v, want an empty permission_ids list (not null)", body)
	}
}

func TestRolesUpdateWithoutChangesSendsNothing(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	err := run(t, a, "roles", "update", "r1")

	if err == nil || !strings.Contains(err.Error(), "Nothing to update") {
		t.Errorf("expected a nothing-to-update error, got %v", err)
	}
	if len(*requests) != 0 {
		t.Errorf("no request should be sent, got %+v", *requests)
	}
}

func TestRolesUpdateEscapesTheID(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"role":{"id":"x"}}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "update", "../permissions", "--name", "x"); err != nil {
		t.Fatalf("roles update: %v", err)
	}

	if path := onlyRequest(t, requests).Path; path != "/api/external/v2/roles/..%2Fpermissions" {
		t.Errorf("path = %s; the id must stay one escaped segment", path)
	}
}

func TestRolesUpdateShowsTheAPIError(t *testing.T) {
	server, _ := roleServer(t, http.StatusUnprocessableEntity, `{"error":"The Admin role cannot be changed."}`)
	a, _ := roleApp(t, subdomainProduct(), server)

	err := run(t, a, "roles", "update", "admin", "--permission-ids", "")

	if err == nil || !strings.Contains(err.Error(), "The Admin role cannot be changed.") {
		t.Errorf("expected the API's message, got %v", err)
	}
}

func TestRolesDeleteMovesTeamMembersToTheNewRole(t *testing.T) {
	server, requests := roleServer(t, http.StatusNoContent, ``)
	a, out := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "delete", "r1", "--new-role-id", "r2", "--quiet"); err != nil {
		t.Fatalf("roles delete: %v", err)
	}

	request := onlyRequest(t, requests)
	if request.Method != http.MethodDelete || request.Path != "/api/external/v2/roles/r1" || request.Query != "new_role_id=r2" {
		t.Errorf("got %s %s?%s, want DELETE /api/external/v2/roles/r1?new_role_id=r2", request.Method, request.Path, request.Query)
	}
	if strings.TrimSpace(out.String()) != "success" {
		t.Errorf("--quiet delete should print success, got %q", out.String())
	}
}

func TestRolesDeleteWithoutANewRoleSendsNoQuery(t *testing.T) {
	server, requests := roleServer(t, http.StatusNoContent, ``)
	a, out := roleApp(t, subdomainProduct(), server)

	if err := run(t, a, "roles", "delete", "r1"); err != nil {
		t.Fatalf("roles delete: %v", err)
	}

	if query := onlyRequest(t, requests).Query; query != "" {
		t.Errorf("query = %q, want none", query)
	}
	if !strings.Contains(out.String(), "Role deleted.") {
		t.Errorf("expected a confirmation, got %q", out.String())
	}
}

func TestRoleCommandsNeedALogin(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())
	a.AddRoleCommands()

	err := run(t, a, "roles", "list")

	if err == nil || !strings.Contains(err.Error(), "Not authenticated") {
		t.Errorf("expected a not-logged-in error, got %v", err)
	}
}

func TestRoleCommandsWorkForASingleHostProduct(t *testing.T) {
	server, requests := roleServer(t, http.StatusOK, `{"roles":[]}`)
	a, _ := roleApp(t, singleHostProduct(), server)

	if err := run(t, a, "roles", "list"); err != nil {
		t.Fatalf("roles list: %v", err)
	}

	if path := onlyRequest(t, requests).Path; path != "/api/external/v2/roles" {
		t.Errorf("path = %s; a single-host product's /api/cli/v1 base path must not apply", path)
	}
}

func jsonEqual(got, want any) bool {
	a, errA := json.Marshal(got, json.Deterministic(true))
	b, errB := json.Marshal(want, json.Deterministic(true))
	return errA == nil && errB == nil && string(a) == string(b)
}
