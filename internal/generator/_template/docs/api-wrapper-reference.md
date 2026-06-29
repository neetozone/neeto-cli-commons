# API wrapper reference

## `internal/auth`

```go
// BaseURL returns the host for a subdomain. Honors the *_BASE_URL env var.
auth.BaseURL(subdomain string) string

// Login runs the browser-based auth flow and persists the token.
auth.Login(subdomain string) (*auth.Credentials, error)

// SelectCredentials returns the credentials a command should use.
//   - subdomain != ""   : the matching entry (error if missing).
//   - subdomain == ""   : the one entry if exactly one is authenticated.
//   - subdomain == "" + >1 authenticated : error asking for --subdomain.
//   - empty store       : "Not authenticated" error.
auth.SelectCredentials(subdomain string) (*auth.Credentials, error)

// Store manipulation (rarely needed by command code).
auth.LoadStore() (*auth.Store, error)
auth.SaveStore(s *auth.Store) error
```

## `internal/client`

```go
// Constructors
client.New(creds *auth.Credentials) *client.Client

// HTTP methods — all return json.RawMessage (nil for 204).
c.Get(path string, params url.Values) (json.RawMessage, error)
c.Post(path string, body interface{}) (json.RawMessage, error)
c.Put(path string, body interface{}) (json.RawMessage, error)
c.Patch(path string, body interface{}) (json.RawMessage, error)
c.Delete(path string) error

// Pagination helpers
client.AddPaginationParams(params url.Values, page, pageSize int)

// Errors
client.APIError{StatusCode int, Message string, Errors []string, Suggestion string}
```

4xx and 5xx responses are returned as `*client.APIError` with the server's
message and a contextual suggestion for common codes (401, 403, 404, 422,
429).

## `internal/output`

```go
output.Print(data json.RawMessage, breadcrumbs []output.Breadcrumb)
output.PrintWithPagination(data, pagination json.RawMessage, breadcrumbs []output.Breadcrumb)
output.PrintMessage(msg string)           // "success" in --quiet mode
output.PrintQuiet(data json.RawMessage, breadcrumbs []output.Breadcrumb) // prints sid/id in --quiet
```

Global toggles (set by root's `PersistentPreRun`):

```go
output.ForceJSON // --json
output.QuietMode // --quiet
output.ToonMode  // --toon
```

Precedence: `--toon` > `--quiet` > `--json` > pretty (TTY default).

## `internal/commands` helpers

```go
getClient(cmd *cobra.Command) (*client.Client, error)  // resolves --subdomain
printList(data, resourceKey, breadcrumbs)              // unwraps list responses
printResource(data, breadcrumbs)                       // for show
printActionResult(data, breadcrumbs)                   // for create/update
paginationParams(cmd) url.Values                       // page/page-size
addPaginationFlags(cmd)                                // for list commands
readJSONFile(path string) (map[string]interface{}, error)
```
