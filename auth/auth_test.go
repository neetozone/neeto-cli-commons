package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func newTestAuth(t *testing.T, p config.Product) *Auth {
	t.Helper()
	a := New(p)
	a.Out = io.Discard
	a.Err = io.Discard
	a.OpenBrowser = func(string) error { return nil }
	a.PollInterval = time.Millisecond
	a.PollTimeout = 10 * time.Second
	return a
}

func TestBaseURL_Subdomain_Default(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "")

	got := New(subdomainProduct()).BaseURL("acme")
	if want := "https://acme.neetodesk.com"; got != want {
		t.Errorf("BaseURL(\"acme\") = %q, want %q", got, want)
	}
}

func TestBaseURL_Subdomain_Override(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "http://acme.lvh.me:8980")

	got := newTestAuth(t, subdomainProduct()).BaseURL("acme")
	if want := "http://acme.lvh.me:8980"; got != want {
		t.Errorf("BaseURL(\"acme\") = %q, want %q", got, want)
	}
}

func TestBaseURL_Subdomain_OverrideStripsTrailingSlash(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "http://acme.lvh.me:8980/")

	got := newTestAuth(t, subdomainProduct()).BaseURL("acme")
	if want := "http://acme.lvh.me:8980"; got != want {
		t.Errorf("BaseURL(\"acme\") = %q, want %q", got, want)
	}
}

func TestBaseURL_SingleHost_IgnoresSubdomain(t *testing.T) {
	t.Setenv("NEETODEPLOY_BASE_URL", "")

	a := New(singleHostProduct())
	if got, want := a.BaseURL(""), "https://app.neetodeploy.com"; got != want {
		t.Errorf("BaseURL(\"\") = %q, want %q", got, want)
	}
	if got, want := a.BaseURL("acme"), "https://app.neetodeploy.com"; got != want {
		t.Errorf("BaseURL(\"acme\") = %q, want %q", got, want)
	}
}

func TestBaseURL_SingleHost_Override(t *testing.T) {
	t.Setenv("NEETODEPLOY_BASE_URL", "http://app.lvh.me:9029/")

	got := newTestAuth(t, singleHostProduct()).BaseURL("")
	if want := "http://app.lvh.me:9029"; got != want {
		t.Errorf("BaseURL(\"\") = %q, want %q", got, want)
	}
}

func TestRequiresSubdomain(t *testing.T) {
	if !New(subdomainProduct()).RequiresSubdomain() {
		t.Error("subdomain product should require a subdomain")
	}
	if New(singleHostProduct()).RequiresSubdomain() {
		t.Error("single_host product should not require a subdomain")
	}
}

func TestLoginURL_Subdomain(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "")

	got := New(subdomainProduct()).LoginURL("acme", "tok123")
	want := "https://acme.neetodesk.com/api/cli/v1/login?token=tok123"
	if got != want {
		t.Errorf("LoginURL = %q, want %q", got, want)
	}
}

func TestLoginURL_SingleHost_UsesLoginHost(t *testing.T) {
	t.Setenv("NEETODEPLOY_BASE_URL", "")

	got := New(singleHostProduct()).LoginURL("", "tok123")
	want := "https://neetozone.neetodeploy.com/admin/cli/login?token=tok123"
	if got != want {
		t.Errorf("LoginURL = %q, want %q", got, want)
	}
}

func TestLoginURL_SingleHost_Override(t *testing.T) {
	t.Setenv("NEETODEPLOY_BASE_URL", "http://app.lvh.me:9029")

	got := newTestAuth(t, singleHostProduct()).LoginURL("", "tok123")
	want := "http://app.lvh.me:9029/admin/cli/login?token=tok123"
	if got != want {
		t.Errorf("LoginURL = %q, want %q", got, want)
	}
}

func TestLoginURL_EscapesToken(t *testing.T) {
	t.Setenv("NEETODESK_BASE_URL", "")
	t.Setenv("NEETODEPLOY_BASE_URL", "")

	raw := "a b&c=d/e?f#g"

	got := New(subdomainProduct()).LoginURL("acme", raw)
	want := "https://acme.neetodesk.com/api/cli/v1/login?token=a+b%26c%3Dd%2Fe%3Ff%23g"
	if got != want {
		t.Errorf("subdomain LoginURL = %q, want %q", got, want)
	}

	got = New(singleHostProduct()).LoginURL("", raw)
	want = "https://neetozone.neetodeploy.com/admin/cli/login?token=a+b%26c%3Dd%2Fe%3Ff%23g"
	if got != want {
		t.Errorf("single_host LoginURL = %q, want %q", got, want)
	}
}

func TestCheckStatus_EscapesTokenInPath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "pending"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	a := newTestAuth(t, subdomainProduct())
	if _, _, _, err := a.checkStatus(server.URL, "a b/c"); err != nil {
		t.Fatalf("checkStatus() error = %v", err)
	}
	want := "/api/cli/v1/sessions/a%20b%2Fc/status"
	if gotPath != want {
		t.Errorf("status path = %q, want %q", gotPath, want)
	}
}

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/v1/sessions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept header = %q, want application/json", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type header = %q, want application/json", got)
		}
		if err := json.NewEncoder(w).Encode(map[string]string{"login_token": "abc123"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	token, err := newTestAuth(t, subdomainProduct()).createSession(server.URL)
	if err != nil {
		t.Fatalf("createSession() error = %v", err)
	}
	if token != "abc123" {
		t.Errorf("createSession() = %q, want %q", token, "abc123")
	}
}

func TestCreateSession_Subdomain_NotFoundStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := newTestAuth(t, subdomainProduct()).createSession(server.URL)
	if err == nil {
		t.Fatal("createSession() expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "Subdomain not found") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "Subdomain not found")
	}
	if want := "if your NeetoDesk URL is acme.neetodesk.com then enter 'acme'."; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
	if strings.Count(err.Error(), "For example") != 1 || strings.Contains(err.Error(), "Similarly") {
		t.Errorf("error = %q, want exactly one example", err.Error())
	}
}

func TestCreateSession_Subdomain_Redirect(t *testing.T) {
	errorPage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte("<!DOCTYPE html><html><body>error</body></html>")); err != nil {
			t.Errorf("write error: %v", err)
		}
	}))
	defer errorPage.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, errorPage.URL+"/error", http.StatusFound)
	}))
	defer server.Close()

	_, err := newTestAuth(t, subdomainProduct()).createSession(server.URL)
	if err == nil {
		t.Fatal("createSession() expected error for unknown subdomain")
	}
	if !strings.Contains(err.Error(), "Subdomain not found") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "Subdomain not found")
	}
}

func TestCreateSession_SingleHost_NotFoundAndRedirectDiffer(t *testing.T) {
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()

	errorPage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("<html></html>")); err != nil {
			t.Errorf("write error: %v", err)
		}
	}))
	defer errorPage.Close()

	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, errorPage.URL+"/error", http.StatusFound)
	}))
	defer redirecting.Close()

	a := newTestAuth(t, singleHostProduct())

	_, err := a.createSession(notFound.URL)
	if err == nil || !strings.Contains(err.Error(), "Authentication endpoint not found") {
		t.Errorf("404 error = %v, want it to mention 'Authentication endpoint not found'", err)
	}

	_, err = a.createSession(redirecting.URL)
	if err == nil || !strings.Contains(err.Error(), "was redirected to") {
		t.Errorf("redirect error = %v, want it to mention 'was redirected to'", err)
	}
}

func TestCreateSession_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := newTestAuth(t, subdomainProduct()).createSession(server.URL); err == nil {
		t.Error("createSession() expected error for 500 response")
	}
}

func TestCheckStatus_Authenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/v1/sessions/tok123/status" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":        "authenticated",
			"email":         "alice@example.com",
			"session_token": "sess_xyz",
		}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	status, email, sessionToken, err := newTestAuth(t, subdomainProduct()).checkStatus(server.URL, "tok123")
	if err != nil {
		t.Fatalf("checkStatus() error = %v", err)
	}
	if status != "authenticated" {
		t.Errorf("status = %q, want %q", status, "authenticated")
	}
	if email != "alice@example.com" {
		t.Errorf("email = %q, want %q", email, "alice@example.com")
	}
	if sessionToken != "sess_xyz" {
		t.Errorf("sessionToken = %q, want %q", sessionToken, "sess_xyz")
	}
}

func TestCheckStatus_PendingAndExpired(t *testing.T) {
	for _, want := range []string{"pending", "expired"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewEncoder(w).Encode(map[string]string{"status": want}); err != nil {
				t.Errorf("encode error: %v", err)
			}
		}))

		status, _, _, err := newTestAuth(t, subdomainProduct()).checkStatus(server.URL, "tok123")
		server.Close()
		if err != nil {
			t.Fatalf("checkStatus() error = %v", err)
		}
		if status != want {
			t.Errorf("status = %q, want %q", status, want)
		}
	}
}

func loginServer(t *testing.T, statusHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/cli/v1/sessions" {
			if err := json.NewEncoder(w).Encode(map[string]string{"login_token": "tok123"}); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}
		statusHandler(w, r)
	}))
}

func TestLogin_BailsAfterTooManyConsecutiveErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var statusCalls int32
	server := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&statusCalls, 1)
		if _, err := w.Write([]byte("<html>not json</html>")); err != nil {
			t.Errorf("write error: %v", err)
		}
	})
	defer server.Close()

	t.Setenv("NEETODESK_BASE_URL", server.URL)
	a := newTestAuth(t, subdomainProduct())

	_, err := a.Login("acme")
	if err == nil {
		t.Fatal("Login() expected error after repeated status failures")
	}
	want := fmt.Sprintf("Could not check authentication status after %d attempts", maxConsecutiveErrors)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to mention %d attempts", err.Error(), maxConsecutiveErrors)
	}
	if got := atomic.LoadInt32(&statusCalls); got != int32(maxConsecutiveErrors) {
		t.Errorf("status polls = %d, want %d", got, maxConsecutiveErrors)
	}
}

func TestLogin_ResetsErrorCountOnSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var statusCalls int32
	server := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&statusCalls, 1)
		switch {
		case n <= 4, n >= 6 && n <= 9:
			if _, err := w.Write([]byte("<html>not json</html>")); err != nil {
				t.Errorf("write error: %v", err)
			}
		case n == 5:
			if err := json.NewEncoder(w).Encode(map[string]string{"status": "pending"}); err != nil {
				t.Errorf("encode error: %v", err)
			}
		default:
			if err := json.NewEncoder(w).Encode(map[string]string{
				"status":        "authenticated",
				"email":         "alice@example.com",
				"session_token": "sess_xyz",
			}); err != nil {
				t.Errorf("encode error: %v", err)
			}
		}
	})
	defer server.Close()

	t.Setenv("NEETODESK_BASE_URL", server.URL)
	a := newTestAuth(t, subdomainProduct())

	creds, err := a.Login("acme")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if creds.Subdomain != "acme" || creds.Email != "alice@example.com" || creds.SessionToken != "sess_xyz" {
		t.Errorf("credentials = %+v", creds)
	}

	store, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 1 || store.Credentials[0].SessionToken != "sess_xyz" {
		t.Errorf("store = %+v", store.Credentials)
	}
}

func TestLogin_Expired(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "expired"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	})
	defer server.Close()

	t.Setenv("NEETODESK_BASE_URL", server.URL)

	_, err := newTestAuth(t, subdomainProduct()).Login("acme")
	if err == nil || !strings.Contains(err.Error(), "Authentication session expired") {
		t.Errorf("error = %v, want it to mention an expired session", err)
	}
}

func TestLogin_SingleHost_StoresWithoutSubdomain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":        "authenticated",
			"email":         "ops@example.com",
			"session_token": "sess_deploy",
		}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	})
	defer server.Close()

	t.Setenv("NEETODEPLOY_BASE_URL", server.URL)
	a := newTestAuth(t, singleHostProduct())

	creds, err := a.Login("ignored")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if creds.Subdomain != "" {
		t.Errorf("Subdomain = %q, want empty", creds.Subdomain)
	}

	loaded, err := a.SelectCredentials("")
	if err != nil {
		t.Fatalf("SelectCredentials() error = %v", err)
	}
	if loaded.SessionToken != "sess_deploy" || loaded.Email != "ops@example.com" {
		t.Errorf("loaded = %+v", loaded)
	}
}

func TestLogin_TimesOut(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := loginServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "pending"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	})
	defer server.Close()

	t.Setenv("NEETODESK_BASE_URL", server.URL)
	a := newTestAuth(t, subdomainProduct())
	a.PollTimeout = 20 * time.Millisecond

	_, err := a.Login("acme")
	if err == nil || !strings.Contains(err.Error(), "NeetoDesk CLI authentication timed out") {
		t.Errorf("error = %v, want a NeetoDesk timeout message", err)
	}
}
