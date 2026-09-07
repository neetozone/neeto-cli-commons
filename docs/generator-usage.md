# Generator usage

## Install

```bash
go install github.com/neetozone/neeto-cli-commons/gen/cmd/neeto-cli-gen@latest
```

Or from a local checkout:

```bash
make build
./neeto-cli-gen ...
```

## Commands

### `neeto-cli-gen new`

Generate a new CLI repo.

```
Flags:
  --config string     YAML answers file (non-interactive)
  -o, --output string Target directory (default: ./<repo-name>)
  --non-interactive   Fail instead of prompting (requires --config)
  --skip-tidy         Skip `go mod tidy` post-generation
  --skip-git-init     Skip `git init` + initial commit
```

The generator refuses to generate into a directory that already contains
files. Use `--output` to pick a new path, or remove the existing contents
first.

### `neeto-cli-gen version`

Print generator version.

## Interactive mode

```bash
neeto-cli-gen new
```

You will be prompted for every field. Default values are derived from the
product display name (e.g., `NeetoForm` yields `neetoform`,
`neetoform.com`, `NEETOFORM_BASE_URL`, etc.).

## Non-interactive mode

Create a YAML file, then run:

```bash
neeto-cli-gen new --config answers.yml --output ./my-cli --non-interactive
```

### Answers file schema

```yaml
# Identity
pretty_name: NeetoForm              # required
binary_name: neetoform              # derived from pretty_name if omitted

# Module / repo
github_org: neetozone
repo_name: neetoform-cli            # derived: <binary>-cli
module_path: github.com/neetozone/neetoform-cli

# Metadata
company_name: BigBinary
support_email: support@bigbinary.com
short_description: NeetoForm CLI
long_description: A command-line interface for NeetoForm.

# Runtime
domain: neetoform.com
api_base_path: /api/external/v2
env_var: NEETOFORM_BASE_URL
config_dir: .config/neetoform

# Release
homebrew_tap: neetozone/homebrew-tap
s3_bucket: neeto-downloads
s3_path_prefix: cli/NeetoForm

# Tooling
go_version: 1.26.1

# Post-gen
run_tidy: true
run_git_init: true
```

Every field is optional in the YAML — missing fields are derived from
`pretty_name` and defaults.

## What the generator creates

A complete Cobra-based Go CLI with:

- Entry point at `cmd/<binary>/main.go`
- Global flags: `--json`, `--quiet`, `--toon`, `--subdomain`
- Browser-based login, multi-subdomain credential store
- HTTP client (`Get`/`Post`/`Put`/`Patch`/`Delete`)
- Output formatting (pretty / JSON / quiet / TOON)
- `doctor`, `version`, `commands`, `setup` (claude/cursor/windsurf/copilot/gemini/codex)
- `.neetoci/` pipelines, `.scripts/release.sh`, `installers/`
- `.claude-plugin/` with embedded `SKILL.md`

After generation the only thing you write yourself is the
product-specific resource commands under `internal/commands/`. See the
generated repo's `docs/adding-commands.md`.
