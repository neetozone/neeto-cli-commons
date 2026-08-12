# Spinning a new product CLI

End-to-end checklist from nothing to a released v1.0.0. Every step below was
walked while creating [`neeto-planner-cli`](https://github.com/neetozone/neeto-planner-cli);
use it as the reference repo when something here is ambiguous.

For the flag-by-flag generator reference see
[`generator-usage.md`](generator-usage.md). This document is the surrounding
process — the repo, labels, CI and secrets the generator does *not* create.

## 1. Build the generator

```bash
go install github.com/neetozone/neeto-cli-template/cmd/neeto-cli-gen@latest
```

Or from a checkout, which is what you want if you are also changing the
template:

```bash
make build      # produces ./neeto-cli-gen
```

## 2. Write the answers file

Prefer a YAML answers file over the interactive prompt — it is reviewable,
and you will usually regenerate at least once.

```yaml
pretty_name: NeetoPlanner
binary_name: neetoplanner

github_org: neetozone
repo_name: neeto-planner-cli
module_path: github.com/neetozone/neeto-planner-cli

company_name: BigBinary
support_email: support@bigbinary.com
short_description: NeetoPlanner CLI
long_description: A command-line interface for NeetoPlanner.

domain: neetoplanner.com
api_base_path: /api/external/v2
env_var: NEETOPLANNER_BASE_URL
config_dir: .config/neetoplanner

homebrew_tap: neetozone/homebrew-tap
s3_bucket: neeto-downloads
s3_path_prefix: cli/NeetoPlanner

go_version: 1.26.1

run_tidy: true
run_git_init: true
```

**`repo_name` almost always needs an explicit value.** It defaults to
`<binary_name>-cli`, which yields `neetoplanner-cli`, but the fleet convention
is `neeto-<product>-cli` — `neeto-desk-cli`, `neeto-cal-cli`,
`neeto-planner-cli`. Set it, and set `module_path` to match, or the Go module
path will disagree with the repo URL.

Where the runtime values come from, using the product's web repo:

| Field | Source |
| --- | --- |
| `domain` | `config/constants.yml` — the production `host`, minus any `app.` prefix. Confirm the product actually serves per-org subdomains (`organizations.subdomain` in `db/schema.rb`); the template's login flow assumes `<subdomain>.<domain>`. |
| `api_base_path` | The versioned external API namespace, `app/controllers/api/external/v2/`. Current products are all on `/api/external/v2`. |
| `s3_path_prefix` | `cli/<PrettyName>`, matching the siblings under `s3://neeto-downloads/cli/`. |

## 3. Generate and verify

```bash
neeto-cli-gen new --config answers.yml --output ./neeto-planner-cli --non-interactive
cd neeto-planner-cli
make check      # fmt + vet + test
make build
./neetoplanner --help
```

The generator refuses to write into a directory that already has files, so
regenerating means deleting the output directory first.

Before going further, confirm the login round-trip works against a real
subdomain — it exercises the browser flow, the credential store and the base
URL together, and it is the one thing a wrong `domain` value breaks:

```bash
./neetoplanner login
./neetoplanner whoami
```

## 4. Create the GitHub repo

The fleet is private. `--source=.` wires the remote to the repo the generator
already `git init`-ed.

```bash
gh repo create neetozone/neeto-planner-cli --private \
  --description "NeetoPlanner CLI" --source=. --remote=origin
git push -u origin main
```

Push the generated commit to `main` on its own, before any hand-written code.
It keeps "what the template produced" separate from "what we wrote", which is
what makes a later template upgrade diffable.

## 5. Create the release labels

```bash
gh label create major --description "Releases breaking changes." --color c4550b
gh label create minor --description "Releases non-breaking noteworthy changes with backward compatible." --color 00917d
gh label create patch --description "Releases backward compatible bug fixes." --color 0e8a16
```

Do not skip this. `.scripts/release.sh` reads the last merged PR's label to
pick the version bump and **exits successfully doing nothing** when no
`major`/`minor`/`patch` label is present. A new repo has none of them, so
without this step the first merge to `main` looks green and ships nothing.

## 6. Wire up NeetoCI

`.neetoci/verify.yml` and `.neetoci/release.yml` ship with the generated repo.
Once `.neetoci/` is on `main`, NeetoCI picks the repo up on its own — the first
pull request gets a `ci/neeto-ci/pull_request/default` check without any
dashboard registration.

Releases need credentials that the generator cannot provision. Open an
existing CLI project in the NeetoCI dashboard — `neeto-desk-cli` or
`neeto-cal-cli` — and copy its environment variables across:

| Variable | Used for |
| --- | --- |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | uploading archives and installers to `s3://neeto-downloads/cli/<PrettyName>/` |
| `GITHUB_TOKEN` | creating the GitHub release via goreleaser, and pushing the formula to `neetozone/homebrew-tap` |

Set these **before** merging anything with a version label. A merge without
them still tags and pushes `v<version>`, then fails at goreleaser, leaving a
tag with no release behind it.

## 7. Add the product commands

Branch, then follow the generated `docs/adding-commands.md`. Two conventions
worth keeping across the fleet:

- Resource commands are `<binary> <noun> <verb>` and stop at three tokens.
  Parent scoping is a flag (`--project`), not another level of nesting.
- `neetoplanner commands` builds its JSON catalog by walking the live Cobra
  tree, so anything registered is automatically discoverable by the Claude
  plugin. Nothing to update by hand.

If the product's API is not ready, it is still worth landing the full command
tree with real flags and help text and a `RunE` that names the endpoint it is
waiting on. The surface becomes reviewable and the plugin skill can describe
it. Say so in the README, and in `internal/plugin/skill.md` tell the agent not
to treat those failures as bugs to work around.

## 8. Cut the first release

Generated `VERSION` is `0.1.0`, so merging a PR labelled `major` produces
`v1.0.0`. Verify both install paths afterwards:

```bash
brew install neetozone/homebrew-tap/neetoplanner
curl -fsSL https://neetoplanner.com/cli/install.sh | sh
neetoplanner version
```

The generated README points users at `https://<domain>/cli/install.{sh,ps1,cmd}`,
but `release.sh` only uploads those installers to
`s3://neeto-downloads/cli/<PrettyName>/latest/`. Something has to serve the
product-domain path from the bucket, and it is not a Rails route in the
product's web repo — check how an existing CLI's domain is wired at the CDN
and mirror it. Until that exists the shell installer 404s even though the
release succeeded, so test the URL rather than assuming it resolves.

## Known gotchas

**The generated scaffold does not pass `make lint`.** A freshly generated repo
reports around 20 `golangci-lint` findings — unchecked `resp.Body.Close()`,
capitalized error strings, and helpers in `internal/commands/helpers.go` that
are unused until you add resource commands. `make check` and
`.neetoci/verify.yml` run only fmt, vet and test, so this surfaces at the
first commit, where the `.githooks/pre-commit` hook runs `make lint` and
blocks. Until the template is cleaned up, commit the first product change with
`--no-verify` rather than absorbing 20 unrelated fixes into a new repo and
drifting it from the template.

**`.template-version` records which generator built the repo.** It is not the
repo's own version — that is `VERSION`. Check it before assuming a repo has a
given template fix:

```bash
cat neeto-planner-cli/.template-version
```
