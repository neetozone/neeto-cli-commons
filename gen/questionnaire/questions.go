// Package questionnaire drives interactive and file-based collection of
// template variables from a user. The output is always a fully populated
// vars.Variables that has passed vars.Validate.
package questionnaire

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/neetozone/neeto-cli-commons/gen/vars"
)

// Ask runs the full interactive questionnaire. The returned Variables is
// validated before return.
func Ask(templateVersion string) (*vars.Variables, error) {
	if !isTTY() {
		return nil, fmt.Errorf("interactive mode requires a TTY; use --config answers.yml for non-interactive runs")
	}

	v := vars.Variables{TemplateVersion: templateVersion}

	// Group 1: collect PrettyName first so we can derive defaults before
	// the rest of the form renders.
	prettyForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Product name (display)").
				Description("Shown in help, docs, and error messages. e.g. NeetoForm").
				Value(&v.PrettyName).
				Validate(validatePretty),
		),
	)
	if err := prettyForm.Run(); err != nil {
		return nil, err
	}
	v.ApplyDerivations()

	mainForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Binary / command name").
				Description("Invoked on the shell. lowercase, digits, hyphens.").
				Value(&v.BinaryName).
				Validate(validateBinary),
			huh.NewInput().
				Title("GitHub org").
				Value(&v.GithubOrg).
				Validate(validateOrg),
			huh.NewInput().
				Title("GitHub repo name").
				Value(&v.RepoName).
				Validate(validateRepo),
			huh.NewInput().
				Title("Go module path").
				Description("e.g. github.com/acme/acme-cli").
				Value(&v.ModulePath).
				Validate(validateModule),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Product domain").
				Description("Base URL computed from subdomain + this, e.g. acme.com").
				Value(&v.Domain).
				Validate(validateDomain),
			huh.NewInput().
				Title("API base path").
				Value(&v.ApiBasePath).
				Validate(validateAPIBase),
			huh.NewInput().
				Title("Env-var override name").
				Description("e.g. ACME_BASE_URL. Points the CLI at a staging/local server.").
				Value(&v.EnvVar).
				Validate(validateEnvVar),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Company / author name").
				Value(&v.CompanyName).
				Validate(validateNonEmpty("company_name")),
			huh.NewInput().
				Title("Support email").
				Value(&v.SupportEmail).
				Validate(validateEmail),
			huh.NewInput().
				Title("Short description").
				Description("1-line, shown in cobra Short").
				Value(&v.ShortDescription).
				Validate(validateShortDesc),
			huh.NewInput().
				Title("Long description").
				Description("Up to 240 chars, shown in cobra Long and plugin manifest").
				Value(&v.LongDescription).
				Validate(validateLongDesc),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Homebrew tap (owner/repo)").
				Value(&v.HomebrewTap).
				Validate(validateTap),
			huh.NewInput().
				Title("S3 bucket for release uploads").
				Value(&v.S3Bucket).
				Validate(validateNonEmpty("s3_bucket")),
			huh.NewInput().
				Title("S3 path prefix").
				Description("e.g. cli/Acme").
				Value(&v.S3PathPrefix).
				Validate(validateNonEmpty("s3_path_prefix")),
			huh.NewInput().
				Title("Go version").
				Value(&v.GoVersion).
				Validate(validateGoVersion),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run `go mod tidy` after generation?").
				Value(&v.RunTidy).
				Affirmative("Yes").
				Negative("Skip"),
			huh.NewConfirm().
				Title("Initialize git repo with an initial commit?").
				Value(&v.RunGitInit).
				Affirmative("Yes").
				Negative("Skip"),
		),
	)

	if err := mainForm.Run(); err != nil {
		return nil, err
	}

	if err := v.Validate(); err != nil {
		return nil, err
	}

	return &v, nil
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
