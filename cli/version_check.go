package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	skipVersionCheckEnvVar = "NEETO_SKIP_VERSION_CHECK"
	versionCacheTTL        = 24 * time.Hour
	versionCheckTimeout    = 2 * time.Second
	versionCacheFile       = "version-check.json"
)

var versionCheckExemptCommands = map[string]bool{
	"update":                        true,
	"version":                       true,
	"doctor":                        true,
	"completion":                    true,
	"help":                          true,
	cobra.ShellCompRequestCmd:       true,
	cobra.ShellCompNoDescRequestCmd: true,
}

type versionCache struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checked_at"`
}

func (a *App) checkVersion(cmd *cobra.Command) error {
	if os.Getenv(skipVersionCheckEnvVar) != "" || a.versionCheckExempt(cmd) {
		return nil
	}
	installed, ok := canonicalVersion(a.Build.Version)
	if !ok {
		return nil
	}
	latest, ok := a.latestVersion()
	if !ok || semver.Compare(installed, latest) >= 0 {
		return nil
	}
	bin := a.Product.BinaryName
	return fmt.Errorf("%s %s is outdated (latest: %s). Run '%s update' to continue.",
		bin, a.Build.Version, strings.TrimPrefix(latest, "v"), bin)
}

func (a *App) versionCheckExempt(cmd *cobra.Command) bool {
	for cmd.HasParent() && cmd.Parent() != a.root {
		cmd = cmd.Parent()
	}
	return cmd.Parent() == a.root && versionCheckExemptCommands[cmd.Name()]
}

func (a *App) latestVersion() (string, bool) {
	path, pathErr := a.versionCachePath()
	if pathErr == nil {
		if cached, ok := readVersionCache(path); ok && time.Since(cached.CheckedAt) < versionCacheTTL {
			if latest, ok := canonicalVersion(cached.Latest); ok {
				return latest, true
			}
		}
	}

	raw, ok := a.fetchLatestVersion()
	if !ok {
		return "", false
	}
	latest, ok := canonicalVersion(raw)
	if !ok {
		return "", false
	}
	if pathErr == nil {
		writeVersionCache(path, versionCache{Latest: raw, CheckedAt: time.Now()})
	}
	return latest, true
}

func (a *App) fetchLatestVersion() (string, bool) {
	client := &http.Client{Timeout: versionCheckTimeout}
	resp, err := client.Get(a.latestVersionURL)
	if err != nil {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(body)), true
}

func (a *App) versionCachePath() (string, error) {
	dir, err := a.Auth.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, versionCacheFile), nil
}

func readVersionCache(path string) (versionCache, bool) {
	var cache versionCache
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &cache) != nil {
		return versionCache{}, false
	}
	return cache, true
}

func writeVersionCache(path string, cache versionCache) {
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

func canonicalVersion(v string) (string, bool) {
	v = "v" + strings.TrimPrefix(strings.TrimSpace(v), "v")
	if !semver.IsValid(v) {
		return "", false
	}
	return semver.Canonical(v), true
}
