package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type versionServer struct {
	url  string
	hits *atomic.Int32
}

func newVersionServer(t *testing.T, status int, body string) versionServer {
	t.Helper()
	hits := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return versionServer{url: srv.URL + "/VERSION", hits: hits}
}

func newVersionCheckApp(t *testing.T, installed, latestURL string) (*App, *bool) {
	t.Helper()
	t.Setenv(skipVersionCheckEnvVar, "")
	a, _ := newTestApp(t, subdomainProduct())
	a.SetBuildInfo(installed, "abc1234", "2026-10-13")
	a.latestVersionURL = latestURL
	ran := false
	a.root.AddCommand(&cobra.Command{
		Use:  "tickets",
		RunE: func(cmd *cobra.Command, args []string) error { ran = true; return nil },
	})
	return a, &ran
}

func seedVersionCache(t *testing.T, a *App, latest string, checkedAt time.Time) {
	t.Helper()
	path, err := a.versionCachePath()
	if err != nil {
		t.Fatalf("versionCachePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, _ := json.Marshal(versionCache{Latest: latest, CheckedAt: checkedAt})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}

func TestVersionCheck_BlocksCommandWhenNewerVersionIsPublished(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4\n")
	a, ran := newVersionCheckApp(t, "1.4.3", srv.url)

	err := run(t, a, "tickets")

	if err == nil {
		t.Fatal("expected the outdated CLI to be blocked")
	}
	want := "neetodesk 1.4.3 is outdated (latest: 1.4.4). Run 'neetodesk update' to continue."
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if *ran {
		t.Error("command ran despite the CLI being outdated")
	}
}

func TestVersionCheck_ComparesVersionsNumerically(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.10.0")
	a, ran := newVersionCheckApp(t, "1.9.0", srv.url)

	if err := run(t, a, "tickets"); err == nil {
		t.Fatal("expected 1.9.0 to be blocked by 1.10.0")
	}
	if *ran {
		t.Error("command ran despite the CLI being outdated")
	}
}

func TestVersionCheck_RunsCommandWhenUpToDateOrAhead(t *testing.T) {
	for _, installed := range []string{"1.4.4", "1.5.0"} {
		t.Run(installed, func(t *testing.T) {
			srv := newVersionServer(t, http.StatusOK, "1.4.4")
			a, ran := newVersionCheckApp(t, installed, srv.url)

			if err := run(t, a, "tickets"); err != nil {
				t.Fatalf("tickets: %v", err)
			}
			if !*ran {
				t.Error("command did not run")
			}
		})
	}
}

func TestVersionCheck_RunsCommandWhenLatestVersionIsUnavailable(t *testing.T) {
	cases := map[string]func(t *testing.T) string{
		"server error": func(t *testing.T) string { return newVersionServer(t, http.StatusInternalServerError, "1.9.9").url },
		"not found":    func(t *testing.T) string { return newVersionServer(t, http.StatusForbidden, "").url },
		"garbage body": func(t *testing.T) string {
			return newVersionServer(t, http.StatusOK, "<Error>AccessDenied</Error>").url
		},
		"unreachable": func(t *testing.T) string { return "http://127.0.0.1:1/VERSION" },
	}
	for name, url := range cases {
		t.Run(name, func(t *testing.T) {
			a, ran := newVersionCheckApp(t, "1.4.3", url(t))

			if err := run(t, a, "tickets"); err != nil {
				t.Fatalf("tickets: %v", err)
			}
			if !*ran {
				t.Error("command did not run")
			}
		})
	}
}

func TestVersionCheck_SkipsDevBuilds(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4")
	a, ran := newVersionCheckApp(t, "", srv.url)

	if err := run(t, a, "tickets"); err != nil {
		t.Fatalf("tickets: %v", err)
	}
	if !*ran {
		t.Error("command did not run")
	}
	if srv.hits.Load() != 0 {
		t.Errorf("dev build fetched the latest version %d times", srv.hits.Load())
	}
}

func TestVersionCheck_SkipsWhenEnvVarIsSet(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4")
	a, ran := newVersionCheckApp(t, "1.4.3", srv.url)
	t.Setenv(skipVersionCheckEnvVar, "1")

	if err := run(t, a, "tickets"); err != nil {
		t.Fatalf("tickets: %v", err)
	}
	if !*ran {
		t.Error("command did not run")
	}
	if srv.hits.Load() != 0 {
		t.Errorf("skipped check still fetched the latest version %d times", srv.hits.Load())
	}
}

func TestVersionCheck_ExemptsCommandsNeededToRecover(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4")
	a, _ := newVersionCheckApp(t, "1.4.3", srv.url)

	for _, args := range [][]string{{"update"}, {"version"}, {"doctor"}, {"completion", "zsh"}} {
		cmd, _, err := a.root.Find(args)
		if err != nil {
			t.Fatalf("find %v: %v", args, err)
		}
		if !a.versionCheckExempt(cmd) {
			t.Errorf("%v is not exempt from the version check", args)
		}
	}

	for _, args := range [][]string{{"version"}, {"help"}, {cobra.ShellCompRequestCmd, "tic"}} {
		if err := run(t, a, args...); err != nil {
			t.Errorf("%v was blocked: %v", args, err)
		}
	}
	if srv.hits.Load() != 0 {
		t.Errorf("exempt commands fetched the latest version %d times", srv.hits.Load())
	}
}

func TestVersionCheck_DoesNotExemptProductSubcommandsSharingAnExemptName(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4")
	a, _ := newVersionCheckApp(t, "1.4.3", srv.url)
	apps := &cobra.Command{Use: "apps"}
	apps.AddCommand(&cobra.Command{Use: "update", RunE: func(cmd *cobra.Command, args []string) error { return nil }})
	a.root.AddCommand(apps)

	if err := run(t, a, "apps", "update"); err == nil {
		t.Error("apps update was not blocked")
	}
}

func TestVersionCheck_BlocksBeforeProductPreRun(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.4")
	a, _ := newVersionCheckApp(t, "1.4.3", srv.url)
	preRan := false
	a.PreRun = func(cmd *cobra.Command, args []string) error { preRan = true; return nil }

	if err := run(t, a, "tickets"); err == nil {
		t.Fatal("expected the outdated CLI to be blocked")
	}
	if preRan {
		t.Error("product PreRun ran before the version check blocked")
	}
}

func TestVersionCheck_UsesFreshCacheWithoutFetching(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.3")
	a, ran := newVersionCheckApp(t, "1.4.3", srv.url)
	seedVersionCache(t, a, "1.4.4", time.Now().Add(-time.Hour))

	if err := run(t, a, "tickets"); err == nil {
		t.Fatal("expected the cached newer version to block")
	}
	if *ran {
		t.Error("command ran despite the cached newer version")
	}
	if srv.hits.Load() != 0 {
		t.Errorf("fresh cache still fetched the latest version %d times", srv.hits.Load())
	}
}

func TestVersionCheck_RefetchesWhenCacheIsStale(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.3")
	a, ran := newVersionCheckApp(t, "1.4.3", srv.url)
	seedVersionCache(t, a, "1.4.4", time.Now().Add(-versionCacheTTL-time.Minute))

	if err := run(t, a, "tickets"); err != nil {
		t.Fatalf("tickets: %v", err)
	}
	if !*ran {
		t.Error("command did not run")
	}
	if srv.hits.Load() != 1 {
		t.Errorf("stale cache fetched the latest version %d times, want 1", srv.hits.Load())
	}
}

func TestVersionCheck_CachesFetchedVersion(t *testing.T) {
	srv := newVersionServer(t, http.StatusOK, "1.4.3")
	a, _ := newVersionCheckApp(t, "1.4.3", srv.url)

	for range 2 {
		if err := run(t, a, "tickets"); err != nil {
			t.Fatalf("tickets: %v", err)
		}
	}
	if srv.hits.Load() != 1 {
		t.Errorf("fetched the latest version %d times across two runs, want 1", srv.hits.Load())
	}
}

func TestNew_PointsLatestVersionURLAtTheReleaseBucket(t *testing.T) {
	a, _ := newTestApp(t, subdomainProduct())

	want := "https://neeto-downloads.s3.amazonaws.com/cli/NeetoDesk/latest/VERSION"
	if a.latestVersionURL != want {
		t.Errorf("latestVersionURL = %q, want %q", a.latestVersionURL, want)
	}
	if !strings.HasPrefix(a.latestVersionURL, a.Product.LatestURL()) {
		t.Errorf("latestVersionURL is not under LatestURL %q", a.Product.LatestURL())
	}
}
