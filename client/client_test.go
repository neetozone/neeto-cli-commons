package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neetozone/neeto-cli-commons/auth"
	"github.com/neetozone/neeto-cli-commons/config"
)

func subdomainProduct() config.Product {
	p := config.Product{
		PrettyName: "NeetoDesk",
		BinaryName: "neetodesk",
		Domain:     "neetodesk.com",
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
	}
	p.ApplyDerivations()
	return p
}

func newTestClient(server *httptest.Server) *Client {
	return &Client{
		Host:         server.URL,
		BaseURL:      server.URL,
		SessionToken: "test_token",
		HTTPClient:   server.Client(),
		UserAgent:    "neetodesk-cli/1.2.3",
		ProductName:  "NeetoDesk",
		BinaryName:   "neetodesk",
	}
}

func TestNew_Subdomain(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "")

	c := New(subdomainProduct(), &auth.Credentials{Subdomain: "acme", SessionToken: "tok_123"}, "1.2.3")

	if want := "https://acme.neetodesk.com/api/external/v2"; c.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, want)
	}
	if want := "https://acme.neetodesk.com"; c.Host != want {
		t.Errorf("Host = %q, want %q", c.Host, want)
	}
	if c.SessionToken != "tok_123" {
		t.Errorf("SessionToken = %q, want %q", c.SessionToken, "tok_123")
	}
	if want := "neetodesk-cli/1.2.3"; c.UserAgent != want {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, want)
	}
}

func TestNew_AlternateBasePaths(t *testing.T) {
	t.Setenv("NEETOPLANNER_BASE_URL", "")
	t.Setenv("NEETOCI_BASE_URL", "")

	planner := config.Product{PrettyName: "NeetoPlanner", BinaryName: "neetoplanner", APIBasePath: "/api/external/v1"}
	planner.ApplyDerivations()
	c := New(planner, &auth.Credentials{Subdomain: "acme"}, "9.9.9")
	if want := "https://acme.neetoplanner.com/api/external/v1"; c.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, want)
	}

	ci := config.Product{PrettyName: "NeetoCI", BinaryName: "neetoci", APIBasePath: "/api/v2"}
	ci.ApplyDerivations()
	c = New(ci, &auth.Credentials{Subdomain: "acme"}, "9.9.9")
	if want := "https://acme.neetoci.com/api/v2"; c.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, want)
	}
}

func TestNew_SingleHostIgnoresSubdomain(t *testing.T) {
	t.Setenv("NEETODEPLOY_BASE_URL", "")

	c := New(singleHostProduct(), &auth.Credentials{SessionToken: "tok_123"}, "2.0.0")

	if want := "https://app.neetodeploy.com/api/cli/v1"; c.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, want)
	}
	if want := "https://app.neetodeploy.com"; c.Host != want {
		t.Errorf("Host = %q, want %q", c.Host, want)
	}
	if want := "neetodeploy-cli/2.0.0"; c.UserAgent != want {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, want)
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/items" {
			t.Errorf("path = %q, want /items", r.URL.Path)
		}
		if r.URL.Query().Get("page_number") != "2" {
			t.Errorf("page_number param = %q, want 2", r.URL.Query().Get("page_number"))
		}
		if _, err := w.Write([]byte(`{"items":[]}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	c := newTestClient(server)
	data, err := c.Get("/items", url.Values{"page_number": {"2"}})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(data) != `{"items":[]}` {
		t.Errorf("Get() = %s", string(data))
	}
}

func TestGet_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query string %q", r.URL.RawQuery)
		}
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	if _, err := newTestClient(server).Get("/items", nil); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestGetV2(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	c := newTestClient(server)
	c.BaseURL = server.URL + "/api/cli/v1"
	c.V2BasePath = "/api/cli/v2"

	if _, err := c.GetV2("/apps/demo", nil); err != nil {
		t.Fatalf("GetV2() error = %v", err)
	}
	if want := "/api/cli/v2/apps/demo"; gotPath != want {
		t.Errorf("GetV2 path = %q, want %q", gotPath, want)
	}
}

func TestGetV2_NotConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("GetV2 should not issue a request when V2BasePath is empty")
	}))
	defer server.Close()

	if _, err := newTestClient(server).GetV2("/apps", nil); err == nil {
		t.Fatal("GetV2() expected error when V2BasePath is empty")
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]string
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("Unmarshal error: %v", err)
		}
		if parsed["name"] != "Demo" {
			t.Errorf("body name = %q, want Demo", parsed["name"])
		}
		if _, err := w.Write([]byte(`{"sid":"abc"}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	data, err := newTestClient(server).Post("/items", map[string]string{"name": "Demo"})
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if string(data) != `{"sid":"abc"}` {
		t.Errorf("Post() = %s", string(data))
	}
}

func TestPutAndPatch(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				t.Errorf("method = %q, want %q", r.Method, method)
			}
			if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
				t.Errorf("Write error: %v", err)
			}
		}))

		c := newTestClient(server)
		var err error
		if method == http.MethodPut {
			_, err = c.Put("/items/abc", map[string]string{"name": "Updated"})
		} else {
			_, err = c.Patch("/items/abc", map[string]string{"name": "Patched"})
		}
		server.Close()
		if err != nil {
			t.Fatalf("%s error = %v", method, err)
		}
	}
}

func TestDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := newTestClient(server).Delete("/items/abc"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestDeleteWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if got := r.URL.Query().Get("email"); got != "sam@example.com" {
			t.Errorf("email param = %q, want sam@example.com", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := newTestClient(server).DeleteWithParams("/items/abc", url.Values{"email": {"sam@example.com"}})
	if err != nil {
		t.Fatalf("DeleteWithParams() error = %v", err)
	}
}

func TestDeleteWithParams_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query string %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := newTestClient(server).DeleteWithParams("/items/abc", nil); err != nil {
		t.Fatalf("DeleteWithParams() error = %v", err)
	}
}

func TestDeleteWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string][]string
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("Unmarshal error: %v", err)
		}
		if len(parsed["keys"]) != 1 || parsed["keys"][0] != "SECRET" {
			t.Errorf("body = %v, want keys [SECRET]", parsed)
		}
		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	_, err := newTestClient(server).DeleteWithBody("/apps/demo/env", map[string][]string{"keys": {"SECRET"}})
	if err != nil {
		t.Fatalf("DeleteWithBody() error = %v", err)
	}
}

func TestRequestHeaders_CarryInjectedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Session-Token"); got != "tok_123" {
			t.Errorf("Session-Token = %q, want tok_123", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		got := r.Header.Get("User-Agent")
		if got != "neetodesk-cli/4.5.6" {
			t.Errorf("User-Agent = %q, want neetodesk-cli/4.5.6", got)
		}
		if strings.Contains(got, "dev") {
			t.Errorf("User-Agent = %q, should not fall back to dev", got)
		}
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	t.Setenv("NEETODESK_BASE_URL", server.URL)
	c := New(subdomainProduct(), &auth.Credentials{Subdomain: "acme", SessionToken: "tok_123"}, "4.5.6")
	c.HTTPClient = server.Client()

	if _, err := c.Get("/test", nil); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`{"error":"Token expired"}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	_, err := newTestClient(server).Get("/items", nil)
	if err == nil {
		t.Fatal("Get() expected error for 401 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Suggestion, "neetodesk login") {
		t.Errorf("Suggestion = %q, want it to name the binary", apiErr.Suggestion)
	}
}

func TestHTTPError_UsesClientSuggestionHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"error":"App not found"}`)); err != nil {
			t.Errorf("Write error: %v", err)
		}
	}))
	defer server.Close()

	c := newTestClient(server)
	c.SuggestionFor = func(status int, message string) string {
		if status == 404 && strings.Contains(message, "App") {
			return "No app with that slug."
		}
		return ""
	}

	_, err := c.Get("/apps/demo", nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Suggestion != "No app with that slug." {
		t.Errorf("Suggestion = %q, want the hook's text", apiErr.Suggestion)
	}
}

func TestNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	data, err := newTestClient(server).Get("/test", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if data != nil {
		t.Errorf("Get() = %s, want nil for 204", string(data))
	}
}

func TestConnectionError_NamesProduct(t *testing.T) {
	c := &Client{
		BaseURL:     "http://127.0.0.1:1/api",
		HTTPClient:  &http.Client{},
		ProductName: "NeetoDesk",
	}
	_, err := c.Get("/items", nil)
	if err == nil {
		t.Fatal("Get() expected a connection error")
	}
	if !strings.Contains(err.Error(), "Could not connect to NeetoDesk.") {
		t.Errorf("error = %q, want it to name NeetoDesk", err.Error())
	}
}
