package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRawAuthFile(t *testing.T, contents string) {
	t.Helper()
	dir, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, authFile), []byte(contents), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestLoadStore_FileAbsent_ReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty store, got %d entries", len(store.Credentials))
	}
}

func TestLoadStore_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeRawAuthFile(t, `{not valid json`)

	if _, err := LoadStore(); err == nil {
		t.Error("LoadStore() expected error for invalid JSON")
	}
}

func TestSaveLoadStore_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store := &Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok-a"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "tok-b"},
	}}

	if err := SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(loaded.Credentials) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Credentials))
	}
	if loaded.Credentials[0].Subdomain != "acme" || loaded.Credentials[1].Subdomain != "beta" {
		t.Errorf("order not preserved: %+v", loaded.Credentials)
	}
}

func TestSaveStore_EmptyRemovesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("initial SaveStore() error = %v", err)
	}

	if err := SaveStore(&Store{}); err != nil {
		t.Fatalf("SaveStore(empty) error = %v", err)
	}

	path := filepath.Join(tmp, configDir, authFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected auth file removed, stat err = %v", err)
	}
}

func TestSaveStore_CreatesDirectoryAndPermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	store := &Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}
	if err := SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	path := filepath.Join(tmp, configDir, authFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestLoadStore_LegacySingleObjectMigrates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeRawAuthFile(t, `{"subdomain":"acme","email":"a@acme.com","session_token":"tok-a"}`)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 1 {
		t.Fatalf("expected 1 migrated entry, got %d", len(store.Credentials))
	}
	c := store.Credentials[0]
	if c.Subdomain != "acme" || c.Email != "a@acme.com" || c.SessionToken != "tok-a" {
		t.Errorf("migrated entry = %+v", c)
	}

	if err := SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}
	path, _ := authFilePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(raw, &shape); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := shape["credentials"]; !ok {
		t.Errorf("migrated file missing 'credentials' key, got keys %v", keys(shape))
	}
}

func TestLoadStore_LegacyEmptyToken_ReturnsEmptyStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeRawAuthFile(t, `{"subdomain":"acme","email":"a@acme.com","session_token":""}`)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty store, got %d entries", len(store.Credentials))
	}
}

func TestStore_UpsertAndFind(t *testing.T) {
	store := &Store{}
	store.Upsert(Credentials{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"})
	store.Upsert(Credentials{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"})
	if len(store.Credentials) != 2 {
		t.Fatalf("expected 2 entries after two Upserts, got %d", len(store.Credentials))
	}

	store.Upsert(Credentials{Subdomain: "acme", Email: "new@acme.com", SessionToken: "t3"})
	if len(store.Credentials) != 2 {
		t.Fatalf("Upsert on same subdomain appended, got %d entries", len(store.Credentials))
	}
	got, ok := store.Find("acme")
	if !ok {
		t.Fatalf("Find(acme) missing after Upsert")
	}
	if got.Email != "new@acme.com" || got.SessionToken != "t3" {
		t.Errorf("Upsert did not replace fields: %+v", got)
	}

	if _, ok := store.Find("missing"); ok {
		t.Error("Find(missing) returned ok=true")
	}
}

func TestStore_Remove(t *testing.T) {
	store := &Store{Credentials: []Credentials{
		{Subdomain: "acme"}, {Subdomain: "beta"},
	}}
	if !store.Remove("acme") {
		t.Error("Remove(acme) returned false")
	}
	if len(store.Credentials) != 1 || store.Credentials[0].Subdomain != "beta" {
		t.Errorf("after Remove, store = %+v", store.Credentials)
	}
	if store.Remove("acme") {
		t.Error("Remove(acme) second time returned true")
	}
}

func TestSelectCredentials_NotLoggedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := SelectCredentials("")
	if err == nil {
		t.Fatal("expected error when store is empty")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error = %q, want contains 'not logged in'", err.Error())
	}
}

func TestSelectCredentials_SingleEntryIsDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	creds, err := SelectCredentials("")
	if err != nil {
		t.Fatalf("SelectCredentials(\"\") error = %v", err)
	}
	if creds.Subdomain != "acme" {
		t.Errorf("got subdomain %q, want %q", creds.Subdomain, "acme")
	}
}

func TestSelectCredentials_MultipleRequireFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	_, err := SelectCredentials("")
	if err == nil {
		t.Fatal("expected error with multiple subdomains and no flag")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--subdomain") {
		t.Errorf("error = %q, want mention of --subdomain", msg)
	}
	if !strings.Contains(msg, "acme") || !strings.Contains(msg, "beta") {
		t.Errorf("error = %q, want list of logged-in subdomains", msg)
	}
}

func TestSelectCredentials_ExplicitMatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	creds, err := SelectCredentials("beta")
	if err != nil {
		t.Fatalf("SelectCredentials(beta) error = %v", err)
	}
	if creds.SessionToken != "t2" {
		t.Errorf("got token %q, want %q", creds.SessionToken, "t2")
	}
}

func TestSelectCredentials_ExplicitMiss(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	_, err := SelectCredentials("ghost")
	if err == nil {
		t.Fatal("expected error for unknown subdomain")
	}
	if !strings.Contains(err.Error(), "acme") {
		t.Errorf("error = %q, want list including 'acme'", err.Error())
	}
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
