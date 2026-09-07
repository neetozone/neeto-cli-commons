package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRawAuthFile(t *testing.T, a *Auth, contents string) {
	t.Helper()
	dir, err := a.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, authFile), []byte(contents), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func readRawAuthFile(t *testing.T, a *Auth) []byte {
	t.Helper()
	path, err := a.AuthFilePath()
	if err != nil {
		t.Fatalf("AuthFilePath() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return raw
}

func TestConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := New(subdomainProduct()).ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	if want := filepath.Join(home, ".config/neetodesk"); dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestLoadStore_FileAbsent_ReturnsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store, err := New(subdomainProduct()).LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty store, got %d entries", len(store.Credentials))
	}
}

func TestLoadStore_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	writeRawAuthFile(t, a, `{not valid json`)

	if _, err := a.LoadStore(); err == nil {
		t.Error("LoadStore() expected error for invalid JSON")
	}
}

func TestSaveLoadStore_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())

	store := &Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok-a"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "tok-b"},
	}}
	if err := a.SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	loaded, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(loaded.Credentials) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Credentials))
	}
	if loaded.Credentials[0].Subdomain != "acme" || loaded.Credentials[1].Subdomain != "beta" {
		t.Errorf("order not preserved: %+v", loaded.Credentials)
	}

	var shape map[string]json.RawMessage
	if err := json.Unmarshal(readRawAuthFile(t, a), &shape); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := shape["credentials"]; !ok {
		t.Errorf("subdomain store missing 'credentials' key, got %v", keys(shape))
	}
}

func TestSaveStore_EmptyRemovesFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	a := New(subdomainProduct())

	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("initial SaveStore() error = %v", err)
	}
	if err := a.SaveStore(&Store{}); err != nil {
		t.Fatalf("SaveStore(empty) error = %v", err)
	}

	path := filepath.Join(tmp, ".config/neetodesk", authFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected auth file removed, stat err = %v", err)
	}
}

func TestSaveStore_CreatesDirectoryAndPermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	a := New(subdomainProduct())

	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, ".config/neetodesk", authFile))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestLoadStore_LegacySingleObjectMigrates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	writeRawAuthFile(t, a, `{"subdomain":"acme","email":"a@acme.com","session_token":"tok-a"}`)

	store, err := a.LoadStore()
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

	if err := a.SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(readRawAuthFile(t, a), &shape); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := shape["credentials"]; !ok {
		t.Errorf("migrated file missing 'credentials' key, got keys %v", keys(shape))
	}
}

func TestLoadStore_LegacyEmptyToken_ReturnsEmptyStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	writeRawAuthFile(t, a, `{"subdomain":"acme","email":"a@acme.com","session_token":""}`)

	store, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty store, got %d entries", len(store.Credentials))
	}
}

func TestSingleHost_RoundTripUsesFlatShape(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(singleHostProduct())

	store := &Store{Credentials: []Credentials{
		{Email: "ops@example.com", SessionToken: "tok-deploy"},
	}}
	if err := a.SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	var shape map[string]json.RawMessage
	if err := json.Unmarshal(readRawAuthFile(t, a), &shape); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := shape["credentials"]; ok {
		t.Errorf("single_host file should be flat, got keys %v", keys(shape))
	}
	if _, ok := shape["subdomain"]; ok {
		t.Errorf("single_host file should carry no subdomain, got keys %v", keys(shape))
	}
	if _, ok := shape["session_token"]; !ok {
		t.Errorf("single_host file missing 'session_token', got keys %v", keys(shape))
	}

	loaded, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(loaded.Credentials) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Credentials))
	}
	if loaded.Credentials[0].SessionToken != "tok-deploy" || loaded.Credentials[0].Subdomain != "" {
		t.Errorf("loaded = %+v", loaded.Credentials[0])
	}
}

func TestSingleHost_MigratesLegacyMultiStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(singleHostProduct())
	writeRawAuthFile(t, a, `{"credentials":[
		{"subdomain":"old","email":"stale@example.com","session_token":""},
		{"subdomain":"acme","email":"ops@example.com","session_token":"tok-deploy"}
	]}`)

	store, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(store.Credentials))
	}
	c := store.Credentials[0]
	if c.Subdomain != "" || c.Email != "ops@example.com" || c.SessionToken != "tok-deploy" {
		t.Errorf("migrated entry = %+v", c)
	}

	if err := a.SaveStore(store); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(readRawAuthFile(t, a), &shape); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, ok := shape["credentials"]; ok {
		t.Errorf("collapsed file should be flat, got keys %v", keys(shape))
	}
}

func TestSingleHost_LegacyFlatFileWithStaleSubdomain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(singleHostProduct())
	writeRawAuthFile(t, a, `{"subdomain":"stale","email":"ops@example.com","session_token":"tok"}`)

	store, err := a.LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if len(store.Credentials) != 1 || store.Credentials[0].Subdomain != "" {
		t.Errorf("store = %+v", store.Credentials)
	}
}

func TestSingleHost_SaveClearsWhenTokenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	a := New(singleHostProduct())

	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Email: "ops@example.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}
	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Email: "ops@example.com"},
	}}); err != nil {
		t.Fatalf("SaveStore(tokenless) error = %v", err)
	}

	path := filepath.Join(tmp, ".config/neetodeploy", authFile)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected auth file removed, stat err = %v", err)
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

func TestStore_RemoveAndSubdomains(t *testing.T) {
	store := &Store{Credentials: []Credentials{
		{Subdomain: "acme"}, {Subdomain: "beta"},
	}}
	if got := strings.Join(store.Subdomains(), ","); got != "acme,beta" {
		t.Errorf("Subdomains() = %q, want %q", got, "acme,beta")
	}
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

	_, err := New(subdomainProduct()).SelectCredentials("")
	if err == nil {
		t.Fatal("expected error when store is empty")
	}
	if !strings.Contains(err.Error(), "Not authenticated. Run 'neetodesk login' to authenticate.") {
		t.Errorf("error = %q, want the canonical not-authenticated message", err.Error())
	}
}

func TestSelectCredentials_SingleEntryIsDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	creds, err := a.SelectCredentials("")
	if err != nil {
		t.Fatalf("SelectCredentials(\"\") error = %v", err)
	}
	if creds.Subdomain != "acme" {
		t.Errorf("got subdomain %q, want %q", creds.Subdomain, "acme")
	}
}

func TestSelectCredentials_MultipleRequireFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	_, err := a.SelectCredentials("")
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

func TestSelectCredentials_ExplicitMatchAndMiss(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(subdomainProduct())
	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Subdomain: "acme", Email: "a@acme.com", SessionToken: "t1"},
		{Subdomain: "beta", Email: "b@beta.com", SessionToken: "t2"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	creds, err := a.SelectCredentials("beta")
	if err != nil {
		t.Fatalf("SelectCredentials(beta) error = %v", err)
	}
	if creds.SessionToken != "t2" {
		t.Errorf("got token %q, want %q", creds.SessionToken, "t2")
	}

	_, err = a.SelectCredentials("ghost")
	if err == nil {
		t.Fatal("expected error for unknown subdomain")
	}
	if !strings.Contains(err.Error(), "acme") {
		t.Errorf("error = %q, want list including 'acme'", err.Error())
	}
}

func TestSelectCredentials_SingleHostIgnoresSubdomain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := New(singleHostProduct())
	if err := a.SaveStore(&Store{Credentials: []Credentials{
		{Email: "ops@example.com", SessionToken: "tok"},
	}}); err != nil {
		t.Fatalf("SaveStore() error = %v", err)
	}

	for _, arg := range []string{"", "anything"} {
		creds, err := a.SelectCredentials(arg)
		if err != nil {
			t.Fatalf("SelectCredentials(%q) error = %v", arg, err)
		}
		if creds.SessionToken != "tok" {
			t.Errorf("SelectCredentials(%q) token = %q, want tok", arg, creds.SessionToken)
		}
	}
}

func TestSelectCredentials_SingleHostNotLoggedIn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := New(singleHostProduct()).SelectCredentials("")
	if err == nil || !strings.Contains(err.Error(), "Run 'neetodeploy login'") {
		t.Errorf("error = %v, want the neetodeploy not-authenticated message", err)
	}
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
