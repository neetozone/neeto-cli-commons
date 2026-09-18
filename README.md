# neeto-cli-commons

Shared module and generator for the neeto product CLIs. Every product imports
this module for its command tree, browser login with multi-subdomain support,
HTTP client, `--json`/`--quiet`/`--toon` output and Claude plugin scaffolding,
and every non-Go file it ships — installers, release pipeline, build config —
is rendered from here by `neeto-cli-sync`.

Running the generator produces a **brand new repository** ready to be filled
in with product-specific commands. The generator never modifies an existing
directory.

## Install

```bash
go install github.com/neetozone/neeto-cli-commons/gen/cmd/neeto-cli-gen@latest
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

- [`docs/generator-usage.md`](docs/generator-usage.md) — generator CLI reference
- [`docs/template-architecture.md`](docs/template-architecture.md) — how `template/` renders
- [`docs/releasing.md`](docs/releasing.md) — cutting a new template version
