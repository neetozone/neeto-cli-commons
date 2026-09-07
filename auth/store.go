package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const authFile = "auth.json"

type Credentials struct {
	Subdomain    string `json:"subdomain,omitempty"`
	Email        string `json:"email"`
	SessionToken string `json:"session_token"`
}

// Store is the persisted collection of all logged-in subdomains. Under
// single_host tenancy it holds exactly one entry with an empty Subdomain.
type Store struct {
	Credentials []Credentials `json:"credentials"`
}

func (s *Store) Find(subdomain string) (*Credentials, bool) {
	for i := range s.Credentials {
		if s.Credentials[i].Subdomain == subdomain {
			return &s.Credentials[i], true
		}
	}
	return nil, false
}

func (s *Store) Upsert(creds Credentials) {
	for i := range s.Credentials {
		if s.Credentials[i].Subdomain == creds.Subdomain {
			s.Credentials[i] = creds
			return
		}
	}
	s.Credentials = append(s.Credentials, creds)
}

func (s *Store) Remove(subdomain string) bool {
	for i := range s.Credentials {
		if s.Credentials[i].Subdomain == subdomain {
			s.Credentials = append(s.Credentials[:i], s.Credentials[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Store) Subdomains() []string {
	out := make([]string, len(s.Credentials))
	for i, c := range s.Credentials {
		out[i] = c.Subdomain
	}
	return out
}

// ConfigDir returns the directory holding all persisted CLI state.
func (a *Auth) ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Could not determine home directory: %w", err)
	}
	return filepath.Join(home, a.product.ConfigDir), nil
}

func (a *Auth) AuthFilePath() (string, error) {
	dir, err := a.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, authFile), nil
}

// LoadStore reads the saved sessions, returning an empty store when the user is
// not logged in. It transparently migrates the older on-disk shapes: the flat
// single-credential object that the subdomain CLIs wrote before multi-login, and
// the multi-credential store that NeetoDeploy wrote before it dropped subdomains.
func (a *Auth) LoadStore() (*Store, error) {
	path, err := a.AuthFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("Could not read credentials: %w", err)
	}

	var store Store
	if err := json.Unmarshal(data, &store); err == nil && store.Credentials != nil {
		if a.RequiresSubdomain() {
			return &store, nil
		}
		return collapseToSingle(store.Credentials), nil
	}

	var legacy Credentials
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("Invalid credentials file: %w", err)
	}
	if legacy.SessionToken == "" {
		return &Store{}, nil
	}
	if !a.RequiresSubdomain() {
		legacy.Subdomain = ""
	}
	return &Store{Credentials: []Credentials{legacy}}, nil
}

// SaveStore writes the credentials, removing the file when there are none.
// Subdomain products write the multi-credential store; single_host products
// write the flat single-object shape the NeetoDeploy CLI has always used.
func (a *Auth) SaveStore(store *Store) error {
	path, err := a.AuthFilePath()
	if err != nil {
		return err
	}

	payload, ok := a.storePayload(store)
	if !ok {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("Could not remove credentials: %w", err)
		}
		return nil
	}

	dir, err := a.ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("Could not create config directory: %w", err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("Could not serialize credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("Could not write credentials: %w", err)
	}
	return nil
}

func (a *Auth) storePayload(store *Store) (any, bool) {
	if store == nil || len(store.Credentials) == 0 {
		return nil, false
	}
	if a.RequiresSubdomain() {
		return store, true
	}
	single := collapseToSingle(store.Credentials)
	if len(single.Credentials) == 0 {
		return nil, false
	}
	return single.Credentials[0], true
}

func collapseToSingle(creds []Credentials) *Store {
	for i := len(creds) - 1; i >= 0; i-- {
		if creds[i].SessionToken != "" {
			c := creds[i]
			c.Subdomain = ""
			return &Store{Credentials: []Credentials{c}}
		}
	}
	return &Store{}
}

func (a *Auth) SelectCredentials(subdomain string) (*Credentials, error) {
	store, err := a.LoadStore()
	if err != nil {
		return nil, err
	}
	if len(store.Credentials) == 0 {
		return nil, fmt.Errorf("Not authenticated. Run '%s login' to authenticate.", a.product.BinaryName)
	}
	if !a.RequiresSubdomain() {
		return &store.Credentials[0], nil
	}
	if subdomain != "" {
		creds, ok := store.Find(subdomain)
		if !ok {
			return nil, fmt.Errorf("Not authenticated for %q. Authenticated subdomains: %s.",
				subdomain, strings.Join(store.Subdomains(), ", "))
		}
		return creds, nil
	}
	if len(store.Credentials) == 1 {
		return &store.Credentials[0], nil
	}
	return nil, fmt.Errorf("Multiple subdomains authenticated (%s); specify --subdomain.",
		strings.Join(store.Subdomains(), ", "))
}
