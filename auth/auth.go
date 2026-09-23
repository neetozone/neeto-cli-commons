package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/browser"

	"github.com/neetozone/neeto-cli-commons/config"
)

const (
	defaultPollInterval  = 2 * time.Second
	defaultPollTimeout   = 5 * time.Minute
	maxConsecutiveErrors = 30
)

type Auth struct {
	product config.Product

	Out          io.Writer
	Err          io.Writer
	OpenBrowser  func(string) error
	HTTPClient   *http.Client
	PollInterval time.Duration
	PollTimeout  time.Duration
}

func New(p config.Product) *Auth {
	return &Auth{
		product:      p,
		Out:          os.Stdout,
		Err:          os.Stderr,
		OpenBrowser:  browser.OpenURL,
		HTTPClient:   http.DefaultClient,
		PollInterval: defaultPollInterval,
		PollTimeout:  defaultPollTimeout,
	}
}

func (a *Auth) Product() config.Product { return a.product }

func (a *Auth) RequiresSubdomain() bool {
	return a.product.Tenancy != config.TenancySingleHost
}

func (a *Auth) BaseURL(subdomain string) string {
	if override := a.baseURLOverride(); override != "" {
		return override
	}
	if a.RequiresSubdomain() {
		return fmt.Sprintf("https://%s.%s", subdomain, a.product.Domain)
	}
	return strings.TrimRight(a.product.APIHost, "/")
}

func (a *Auth) LoginURL(subdomain, loginToken string) string {
	token := url.QueryEscape(loginToken)
	if a.RequiresSubdomain() {
		return fmt.Sprintf("%s/api/cli/v1/login?token=%s", a.BaseURL(subdomain), token)
	}
	host := strings.TrimRight(a.product.LoginHost, "/")
	if override := a.baseURLOverride(); override != "" {
		host = override
	}
	return fmt.Sprintf("%s/admin/cli/login?token=%s", host, token)
}

// A workspace subdomain is one DNS label. Anything else is refused rather than
// interpolated, because "evil.example#" pasted into https://%s.%s makes
// evil.example the host that receives the login exchange.
var subdomainLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func (a *Auth) Login(subdomain string) (*Credentials, error) {
	if !a.RequiresSubdomain() {
		subdomain = ""
	}
	if subdomain != "" && !subdomainLabel.MatchString(subdomain) {
		return nil, fmt.Errorf(
			"%q is not a valid subdomain. Enter only the workspace name: if your %s URL is acme.%s then enter 'acme'.",
			subdomain, a.product.PrettyName, a.product.Domain)
	}
	baseURL := a.BaseURL(subdomain)

	loginToken, err := a.createSession(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Could not create authentication session: %w", err)
	}

	loginURL := a.LoginURL(subdomain, loginToken)
	_, _ = fmt.Fprintln(a.out(), "Opening browser for authentication...")
	_, _ = fmt.Fprintf(a.out(), "If the browser doesn't open, visit: %s\n", loginURL)
	if err := a.openBrowser(loginURL); err != nil {
		_, _ = fmt.Fprintf(a.out(), "Could not open browser: %v\n", err)
	}

	_, _ = fmt.Fprint(a.out(), "Waiting for authentication")
	deadline := time.Now().Add(a.pollTimeout())
	consecutiveErrors := 0
	var lastErr error

	for time.Now().Before(deadline) {
		time.Sleep(a.pollInterval())
		_, _ = fmt.Fprint(a.out(), ".")

		status, email, sessionToken, err := a.checkStatus(baseURL, loginToken)
		if err != nil {
			consecutiveErrors++
			lastErr = err
			if consecutiveErrors >= maxConsecutiveErrors {
				_, _ = fmt.Fprintln(a.out())
				return nil, fmt.Errorf("Could not check authentication status after %d attempts: %w", consecutiveErrors, lastErr)
			}
			continue
		}
		consecutiveErrors = 0

		switch status {
		case "authenticated":
			_, _ = fmt.Fprintln(a.out(), " done!")
			creds := Credentials{
				Subdomain:    subdomain,
				Email:        email,
				SessionToken: sessionToken,
			}
			store, err := a.LoadStore()
			if err != nil {
				return nil, fmt.Errorf("Authenticated but could not read credentials: %w", err)
			}
			store.Upsert(creds)
			if err := a.SaveStore(store); err != nil {
				return nil, fmt.Errorf("Authenticated but could not save credentials: %w", err)
			}
			return &creds, nil
		case "expired":
			_, _ = fmt.Fprintln(a.out())
			return nil, fmt.Errorf("Authentication session expired. Please try again.")
		}
	}

	_, _ = fmt.Fprintln(a.out())
	return nil, fmt.Errorf("%s CLI authentication timed out after %d minutes. Please try again.",
		a.product.PrettyName, int(a.pollTimeout()/time.Minute))
}

func (a *Auth) createSession(baseURL string) (string, error) {
	resp, err := a.postJSON(baseURL+"/api/cli/v1/sessions", nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := a.sessionEndpointError(baseURL, resp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected status %d.", resp.StatusCode)
	}

	var result struct {
		LoginToken string `json:"login_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.LoginToken, nil
}

func (a *Auth) sessionEndpointError(baseURL string, resp *http.Response) error {
	if a.RequiresSubdomain() {
		if resp.StatusCode == http.StatusNotFound || redirectedAway(baseURL, resp) {
			return fmt.Errorf("Subdomain not found. Please check that you entered the correct subdomain.\nFor example, if your %s URL is acme.%s then enter 'acme'.",
				a.product.PrettyName, a.product.Domain)
		}
		return nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Authentication endpoint not found (HTTP 404) at %s — the API may be unavailable, or this CLI may be out of date. Try updating to the latest version.", baseURL)
	}
	if redirectedAway(baseURL, resp) {
		return fmt.Errorf("Authentication request to %s was redirected to %s and returned an unexpected response — the API may be unavailable, or this CLI may be out of date. Try updating to the latest version.",
			baseURL, resp.Request.URL.Host)
	}
	return nil
}

func (a *Auth) checkStatus(baseURL, loginToken string) (status, email, sessionToken string, err error) {
	statusURL := fmt.Sprintf("%s/api/cli/v1/sessions/%s/status", baseURL, url.PathEscape(loginToken))
	resp, err := a.postJSON(statusURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return "", "", "", fmt.Errorf("status check returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var result struct {
		Status       string `json:"status"`
		Email        string `json:"email"`
		SessionToken string `json:"session_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", err
	}
	return result.Status, result.Email, result.SessionToken, nil
}

func (a *Auth) postJSON(rawURL string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return a.httpClient().Do(req)
}

func redirectedAway(requestedURL string, resp *http.Response) bool {
	requested, err := url.Parse(requestedURL)
	if err != nil || resp.Request == nil {
		return false
	}
	return resp.Request.URL.Host != requested.Host
}

func (a *Auth) baseURLOverride() string {
	envVar := a.product.BaseURLEnvVar()
	raw := strings.TrimRight(os.Getenv(envVar), "/")
	if raw == "" {
		return ""
	}
	if err := checkOverride(raw); err != nil {
		_, _ = fmt.Fprintf(a.errw(), "Ignoring %s: %v\n", envVar, err)
		return ""
	}
	// This variable redirects every request, including the ones that carry the
	// stored session token, so say out loud where the credentials are going.
	_, _ = fmt.Fprintf(a.errw(),
		"Warning: %s is set, so %s will send your credentials to %s instead of %s.\n",
		envVar, a.product.BinaryName, raw, a.product.Domain)
	return raw
}

func checkOverride(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%q is not a valid base URL", raw)
	}
	if u.Scheme == "https" || (u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil
	}
	return fmt.Errorf("%q must use https; http is accepted only for localhost", raw)
}

// Names that reach the developer's own machine. lvh.me is third-party DNS that
// resolves to 127.0.0.1, and every neeto product's dev setup points at it, so
// refusing it would only push people to unset the check entirely.
func isLoopback(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1", "lvh.me":
		return true
	}
	return strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".lvh.me")
}

func (a *Auth) out() io.Writer {
	if a.Out == nil {
		return os.Stdout
	}
	return a.Out
}

// Warnings stay off Out so they never land in the middle of the machine-readable
// stream a script is parsing.
func (a *Auth) errw() io.Writer {
	if a.Err == nil {
		return os.Stderr
	}
	return a.Err
}

func (a *Auth) openBrowser(rawURL string) error {
	if a.OpenBrowser == nil {
		return browser.OpenURL(rawURL)
	}
	return a.OpenBrowser(rawURL)
}

func (a *Auth) httpClient() *http.Client {
	if a.HTTPClient == nil {
		return http.DefaultClient
	}
	return a.HTTPClient
}

func (a *Auth) pollInterval() time.Duration {
	if a.PollInterval <= 0 {
		return defaultPollInterval
	}
	return a.PollInterval
}

func (a *Auth) pollTimeout() time.Duration {
	if a.PollTimeout <= 0 {
		return defaultPollTimeout
	}
	return a.PollTimeout
}
