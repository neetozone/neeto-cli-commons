package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"

	"github.com/neetozone/neeto-cli-commons/config"
)

const (
	defaultPollInterval  = 2 * time.Second
	defaultPollTimeout   = 5 * time.Minute
	maxConsecutiveErrors = 5
)

type Auth struct {
	product config.Product

	Out          io.Writer
	OpenBrowser  func(string) error
	HTTPClient   *http.Client
	PollInterval time.Duration
	PollTimeout  time.Duration
}

func New(p config.Product) *Auth {
	return &Auth{
		product:      p,
		Out:          os.Stdout,
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

// LoginURL points at the page the user opens in a browser. Under single_host
// tenancy that page is served by the dashboard host, not by the API host.
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

func (a *Auth) Login(subdomain string) (*Credentials, error) {
	if !a.RequiresSubdomain() {
		subdomain = ""
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

// postJSON issues a POST that both sends and accepts JSON. The Accept header is
// the important part: Go's http.Post only sets Content-Type, so without it Rails
// content-negotiates to HTML and the login endpoints return 500.
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
	return strings.TrimRight(os.Getenv(a.product.BaseURLEnvVar()), "/")
}

func (a *Auth) out() io.Writer {
	if a.Out == nil {
		return os.Stdout
	}
	return a.Out
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
