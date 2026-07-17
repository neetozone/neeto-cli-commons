# neeto-cli-template

Template repository and generator for building product CLIs that follow the
`neeto-cal-cli` architecture: Cobra-based command tree, browser login with
multi-subdomain support, HTTP client, `--json`/`--quiet`/`--toon` output, Claude
plugin scaffolding, `.neetoci` release pipeline, and cross-platform installers.

Running the generator produces a **brand new repository** ready to be filled in
with product-specific commands. The generator never modifies an existing
directory.

## Install

```bash
go install github.com/neetozone/neeto-cli-template/cmd/neeto-cli-gen@latest
```

## Generate a new CLI

Interactive:

```bash
neeto-cli-gen new
```

From a YAML answers file:

```bash
neeto-cli-gen new --config answers.yml --output ./my-cli
```

Full flag reference is in [`docs/generator-usage.md`](docs/generator-usage.md).

## Layout

- `cmd/neeto-cli-gen/` — generator entrypoint
- `internal/vars/` — template variable struct + defaults
- `internal/questionnaire/` — interactive form + YAML loader
- `internal/generator/` — path/content rendering, permissions, post-gen
- `template/` — genericized CLI skeleton; embedded into the generator binary

## Development

```bash
bin/setup        # installs golangci-lint, wires git hooks
make build       # builds ./neeto-cli-gen
make test        # runs unit tests
make check       # fmt + vet + test
```

## Docs

- [`docs/generator-usage.md`](docs/generator-usage.md) — generator CLI
  reference
- [`docs/template-architecture.md`](docs/template-architecture.md) — how
  `template/` renders
- [`docs/releasing.md`](docs/releasing.md) — cutting a new template version
