package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/neetozone/neeto-cli-commons/output"
	"github.com/spf13/cobra"
)

const doctorProbeTimeout = 10 * time.Second

type checkResult struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Skipped bool   `json:"skipped,omitempty"`
}

func (a *App) newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check CLI health and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := a.runChecks(cmd)

			if a.Printer.ForceJSON || a.Printer.Quiet || a.Printer.Toon {
				payload, err := json.Marshal(checks)
				if err != nil {
					return err
				}
				a.Printer.Print(payload, nil)
				return nil
			}

			renderChecks(cmd.OutOrStdout(), checks, output.IsTTY())
			return nil
		},
	}
}

// runChecks runs every check independently — one failure does not skip the next.
func (a *App) runChecks(cmd *cobra.Command) []checkResult {
	subdomain := ""
	if a.Auth.RequiresSubdomain() {
		subdomain, _ = cmd.Flags().GetString("subdomain")
	}

	creds, credsErr := a.Auth.SelectCredentials(subdomain)
	checks := []checkResult{{Name: "Authentication"}}
	if credsErr != nil {
		checks[0].Detail = credsErr.Error()
	} else {
		checks[0].OK = true
		checks[0].Detail = fmt.Sprintf("authenticated as %s on %s", creds.Email, a.hostFor(creds.Subdomain))
	}

	probeSubdomain := subdomain
	if creds != nil {
		probeSubdomain = creds.Subdomain
	}
	checks = append(checks, a.connectionCheck(probeSubdomain))

	return append(checks, checkResult{Name: "CLI version", OK: true, Detail: a.Build.Version})
}

func (a *App) connectionCheck(subdomain string) checkResult {
	check := checkResult{Name: "API connection"}

	if a.Auth.RequiresSubdomain() && subdomain == "" {
		check.Skipped = true
		check.Detail = "skipped (no subdomain — pass --subdomain or authenticate)"
		return check
	}

	baseURL := a.Auth.BaseURL(subdomain)
	httpClient := &http.Client{Timeout: doctorProbeTimeout}
	start := time.Now()
	resp, err := httpClient.Get(baseURL)
	elapsed := time.Since(start)
	if err != nil {
		check.Detail = fmt.Sprintf("could not reach %s\n  Error: %v", baseURL, err)
		return check
	}
	defer func() { _ = resp.Body.Close() }()

	check.OK = true
	check.Detail = fmt.Sprintf("%s (responding in %dms)", baseURL, elapsed.Milliseconds())
	return check
}

func renderChecks(w io.Writer, checks []checkResult, tty bool) {
	for _, check := range checks {
		fmt.Fprintf(w, "%s %s: %s\n", checkGlyph(check, tty), check.Name, check.Detail)
	}
}

func checkGlyph(check checkResult, tty bool) string {
	switch {
	case check.Skipped:
		if tty {
			return "•"
		}
		return "-"
	case check.OK:
		if tty {
			return "✓"
		}
		return "OK"
	default:
		if tty {
			return "✗"
		}
		return "FAIL"
	}
}
