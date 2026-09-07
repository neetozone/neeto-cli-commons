package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/neetozone/neeto-cli-commons/auth"
	"github.com/neetozone/neeto-cli-commons/config"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	Host         string
	BaseURL      string
	SessionToken string
	HTTPClient   *http.Client

	UserAgent   string
	ProductName string
	BinaryName  string

	V2BasePath string

	SuggestionFor func(status int, message string) string
}

func New(p config.Product, creds *auth.Credentials, version string) *Client {
	subdomain := ""
	if creds != nil {
		subdomain = creds.Subdomain
	}
	host := auth.New(p).BaseURL(subdomain)

	c := &Client{
		Host:        host,
		BaseURL:     host + p.APIBasePath,
		HTTPClient:  &http.Client{Timeout: defaultTimeout},
		UserAgent:   p.UserAgent(version),
		ProductName: p.PrettyName,
		BinaryName:  p.BinaryName,
	}
	if creds != nil {
		c.SessionToken = creds.SessionToken
	}
	return c
}

func (c *Client) Get(path string, params url.Values) (json.RawMessage, error) {
	return c.get(c.BaseURL, path, params)
}

func (c *Client) GetV2(path string, params url.Values) (json.RawMessage, error) {
	if c.V2BasePath == "" {
		return nil, fmt.Errorf("client: V2BasePath is not configured")
	}
	return c.get(c.Host+c.V2BasePath, path, params)
}

func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.doWithBody(http.MethodPost, path, body)
}

func (c *Client) Put(path string, body any) (json.RawMessage, error) {
	return c.doWithBody(http.MethodPut, path, body)
}

func (c *Client) Patch(path string, body any) (json.RawMessage, error) {
	return c.doWithBody(http.MethodPatch, path, body)
}

func (c *Client) Delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

func (c *Client) DeleteWithBody(path string, body any) (json.RawMessage, error) {
	return c.doWithBody(http.MethodDelete, path, body)
}

func (c *Client) get(base, path string, params url.Values) (json.RawMessage, error) {
	u := base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) doWithBody(method, path string, body any) (json.RawMessage, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("Could not encode request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, c.BaseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (json.RawMessage, error) {
	req.Header.Set("Session-Token", c.SessionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to %s. Check your internet connection: %w", c.ProductName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not read response: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody, errorOptions{
			binaryName:    c.BinaryName,
			suggestionFor: c.SuggestionFor,
		})
	}

	return json.RawMessage(respBody), nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return &http.Client{Timeout: defaultTimeout}
	}
	return c.HTTPClient
}
