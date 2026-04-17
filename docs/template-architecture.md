# Template architecture

## Layout

```
internal/generator/
├── _template/          # the skeleton the generator renders
├── embed.go            # //go:embed all:_template
├── generator.go        # Run orchestrator
├── render.go           # renderPath, renderFile
├── permissions.go      # .exec-manifest handling
├── postgen.go          # go mod tidy, git init
├── testdata/golden_answers.yml
└── *_test.go
```

## Why `_template/`

The template contains plain `.go` files (for the generated repo). If this
directory were called `template/` at the module root, `go build` and
`go mod tidy` would try to compile it as part of the template repo itself
— polluting dependencies and breaking builds. Directories whose name starts
with `_` or `.` are ignored by Go tooling, so we name it `_template/`. The
`all:` prefix on the `go:embed` directive re-enables inclusion for dotfiles
*and* underscore-prefixed paths.

## Rendering model

### Two-stage render

1. **Path rendering** (`renderPath`): every `__BINARY__` segment becomes the
   binary name; a single trailing `.tmpl` is stripped.
2. **Content rendering** (`renderFile`): files ending in `.tmpl` are passed
   through Go `text/template` with the `Variables` struct as `.`; other
   files are copied verbatim.

### GoReleaser escape

GoReleaser config (`.goreleaser.yml`, `.scripts/release.sh`) also uses
`{{ … }}` syntax. We escape each native GoReleaser token in the template
source as `{{"{{"}}.Version}}`, which renders back to a literal
`{{.Version}}` that GoReleaser interprets at release time.

### Permissions

`embed.FS` strips executable bits. `_template/.exec-manifest` lists every
path that needs `0755` restored; the generator reads it, path-renders each
entry, and chmods post-write. The manifest itself is not copied to output.

## Variables

Single source of truth: `internal/vars/vars.go`. The struct carries YAML
tags so `LoadConfig` can parse an answers file. `ApplyDerivations` fills
empty fields from `PrettyName` (binary, repo, module, domain, env-var,
config dir, S3 prefix, etc.). `Validate` enforces regex checks.

## Adding a new template variable

1. Add a field to `Variables` with a yaml tag and any derivation in
   `ApplyDerivations`.
2. Add validation (if applicable) in both `Variables.Validate` and
   `internal/questionnaire/validate.go`.
3. Add an input to `questionnaire.Ask` and a row to
   `docs/generator-usage.md`.
4. Reference the new field as `{{.FieldName}}` anywhere in
   `internal/generator/_template/`.
5. Update `internal/generator/testdata/golden_answers.yml` if the field
   is required.

## Releasing the generator

See `docs/releasing.md`.
